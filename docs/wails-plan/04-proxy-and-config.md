# 代理服务集成、配置管理、系统托盘

## 代理服务生命周期

### 启动时序

```
main.go 启动
  → 初始化 DB、Service、Repo（复用现有逻辑）
  → 创建 App 实例
  → wails.Run() 启动 Wails 应用
    → onStartup 回调：
      → 启动代理 HTTP 服务器 goroutine
      → 启动 CleanupService goroutine
      → 启动日志事件推送 goroutine
      → 启动活跃请求事件推送 goroutine
      → 设置系统托盘
    → onShutdown 回调：
      → 优雅关闭代理服务器
      → 取消所有后台 goroutine
      → 关闭 ProxyService 缓存
```

### main.go 入口

```go
func main() {
    // 初始化日志（先于一切）
    logger.Init("llm-proxy.log", slog.LevelInfo)

    // 加载配置
    cfg := config.Load()
    logger.Init("llm-proxy.log", logger.ParseLevel(cfg.LogLevel))

    // 初始化数据库
    db := initDB(cfg)

    // 组装依赖（复用现有逻辑）
    providerRepo := repository.NewProviderRepository(db)
    requestLogRepo := repository.NewRequestLogRepository(db, cfg.DBType)
    hourlyStatRepo := repository.NewHourlyStatRepository(db, cfg.DBType)

    cleanupSvc := service.NewCleanupService(hourlyStatRepo, requestLogRepo, cfg.LogCleanupDays)
    cleanupSvc.BackfillMissingHours()

    proxyService := service.NewProxyService(providerRepo, cfg.ProviderCacheTTL)
    providerService := service.NewProviderService(providerRepo, proxyService)
    statsService := service.NewStatsService(hourlyStatRepo, requestLogRepo)
    tracker := handler.NewActiveRequestTracker()
    ringBuffer := logger.NewRingBuffer(1000)

    // 创建 App
    app := NewApp(cfg, providerService, statsService, tracker, ringBuffer, cleanupSvc)

    // Wails 配置
    err := wails.Run(&options.App{
        Title:     "LLM Proxy",
        Width:     1200,
        Height:    800,
        MinWidth:  900,
        MinHeight: 600,
        AssetHandler: &assethandler.AssetHandler{
            AssetDir: "frontend",
        },
        OnStartup:  app.startup,
        OnShutdown: app.shutdown,
        OnBeforeClose: app.beforeClose,
        Bind: []interface{}{
            app,
        },
    })
    if err != nil {
        slog.Error("Wails 启动失败", "error", err)
        os.Exit(1)
    }
}
```

### App.startup / App.shutdown

```go
func (a *App) startup(ctx context.Context) {
    a.ctx = ctx

    // 启动代理服务
    a.startProxyServer()

    // 启动后台任务
    cleanupCtx, cancel := context.WithCancel(ctx)
    a.cleanupCancel = cancel
    go a.cleanupSvc.Start(cleanupCtx)

    // 启动日志事件推送
    go a.startLogEventPusher()

    // 启动活跃请求事件推送
    go a.startActiveEventPusher()

    // 设置系统托盘
    a.setupTray()
}

func (a *App) shutdown(ctx context.Context) {
    // 取消后台任务
    if a.cleanupCancel != nil {
        a.cleanupCancel()
    }

    // 关闭代理服务器
    a.stopProxyServer()

    // 关闭 ProxyService 缓存
    a.proxyService.Close()
}
```

### 代理服务状态管理

```go
type proxyState struct {
    mu     sync.RWMutex
    status string // "starting" | "running" | "stopped" | "error"
    err    string
    server *http.Server
}

func (a *App) startProxyServer() {
    a.proxyState.mu.Lock()
    a.proxyState.status = "starting"
    a.proxyState.mu.Unlock()

    // 检查端口可用性
    ln, err := net.Listen("tcp", ":"+a.cfg.ProxyPort)
    if err != nil {
        a.proxyState.mu.Lock()
        a.proxyState.status = "error"
        a.proxyState.err = fmt.Sprintf("端口 %s 已被占用", a.cfg.ProxyPort)
        a.proxyState.mu.Unlock()
        slog.Error("代理端口被占用", "port", a.cfg.ProxyPort, "error", err)
        return
    }
    ln.Close()

    // 组装代理路由（复用现有 SetupProxy）
    proxyHandler := handler.NewProxyHandler(...)
    anthropicHandler := handler.NewAnthropicHandler(...)
    ollamaHandler := handler.NewOllamaHandler(...)
    proxyEngine := router.SetupProxy(proxyHandler, anthropicHandler, ollamaHandler)

    server := &http.Server{
        Addr:    ":" + a.cfg.ProxyPort,
        Handler: proxyEngine,
    }

    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            a.proxyState.mu.Lock()
            a.proxyState.status = "error"
            a.proxyState.err = err.Error()
            a.proxyState.mu.Unlock()
            slog.Error("代理服务启动失败", "error", err)
        }
    }()

    a.proxyState.mu.Lock()
    a.proxyState.status = "running"
    a.proxyState.server = server
    a.proxyState.mu.Unlock()
}

func (a *App) GetProxyStatus() vo.ProxyStatusVO {
    a.proxyState.mu.RLock()
    defer a.proxyState.mu.RUnlock()
    return vo.ProxyStatusVO{
        Status: a.proxyState.status,
        Port:   a.cfg.ProxyPort,
        Error:  a.proxyState.err,
    }
}
```

## 配置管理改造

### 问题

- 当前 `.env` 文件从工作目录加载，Wails 打包后工作目录不确定
- SQLite 默认路径 `llm_proxy.db` 是相对路径，打包后可能指向错误位置
- `WebPort` 配置不再需要

### 改造方案

```go
// config.go 改造
func Load() *Config {
    // 确定配置目录：优先使用可执行文件所在目录
    exePath, _ := os.Executable()
    baseDir := filepath.Dir(exePath)

    // 尝试从可执行文件目录加载 .env
    envPath := filepath.Join(baseDir, ".env")
    godotenv.Load(envPath)
    // 也尝试从当前工作目录加载（开发模式兼容）
    godotenv.Load()

    cfg := &Config{
        // 移除 WebPort
        ProxyPort: getEnv("PROXY_PORT", "8888"),
        // ...
    }

    // SQLite 路径：如果是相对路径，改为基于可执行文件目录
    if cfg.IsSQLite() && !filepath.IsAbs(cfg.DBPath) {
        cfg.DBPath = filepath.Join(baseDir, cfg.DBPath)
    }

    // 日志文件路径也基于可执行文件目录
    if cfg.LogFilePath == "" {
        cfg.LogFilePath = filepath.Join(baseDir, "llm-proxy.log")
    }

    return cfg
}
```

### 新增配置项

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `WINDOW_WIDTH` | 1200 | 窗口初始宽度 |
| `WINDOW_HEIGHT` | 800 | 窗口初始高度 |
| `MINIMIZE_TO_TRAY` | true | 关闭窗口时是否最小化到托盘 |
| `AUTO_START_PROXY` | true | 启动时是否自动启动代理服务 |

## 系统托盘

### 方案

使用 `getlantern/systray` 或 Wails v2 的系统托盘支持：

```go
func (a *App) setupTray() {
    // Wails v2 支持通过 OnBeforeClose 控制关闭行为
    // 系统托盘使用第三方库
    go func() {
        systray.Run(a.onTrayReady, a.onTrayExit)
    }()
}

func (a *App) onTrayReady() {
    systray.SetTitle("LLM")
    systray.SetTooltip("LLM Proxy 中转服务")

    mShow := systray.AddMenuItem("显示窗口", "显示主窗口")
    systray.AddSeparator()
    mProxyStatus := systray.AddMenuItem("代理: 运行中", "代理服务状态")
    mToggleProxy := systray.AddMenuItem("停止代理", "启动/停止代理")
    systray.AddSeparator()
    mQuit := systray.AddMenuItem("退出", "退出应用")

    go func() {
        for {
            select {
            case <-mShow.ClickedCh:
                // 显示窗口
                runtime.WindowShow(a.ctx)
            case <-mToggleProxy.ClickedCh:
                // 切换代理状态
                status := a.GetProxyStatus()
                if status.Status == "running" {
                    a.StopProxy()
                    mToggleProxy.SetTitle("启动代理")
                    mProxyStatus.SetTitle("代理: 已停止")
                } else {
                    a.StartProxy()
                    mToggleProxy.SetTitle("停止代理")
                    mProxyStatus.SetTitle("代理: 运行中")
                }
            case <-mQuit.ClickedCh:
                systray.Quit()
            }
        }
    }()
}
```

### 窗口关闭行为

```go
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
    minimizeToTray := a.cfg.MinimizeToTray
    if minimizeToTray {
        // 最小化到托盘而非关闭
        runtime.WindowHide(a.ctx)
        return true  // prevent = true 阻止关闭
    }
    return false  // 允许关闭
}
```

## 窗口状态持久化

记住用户上次关闭时的窗口位置和大小：

```go
// 在 App 中保存窗口状态
type windowState struct {
    X      int `json:"x"`
    Y      int `json:"y"`
    Width  int `json:"width"`
    Height int `json:"height"`
}

func (a *App) saveWindowState() {
    x, y := runtime.WindowGetPosition(a.ctx)
    w, h := runtime.WindowGetSize(a.ctx)
    state := windowState{X: x, Y: y, Width: w, Height: h}
    data, _ := json.Marshal(state)
    os.WriteFile(a.getWindowStatePath(), data, 0644)
}

func (a *App) loadWindowState() windowState {
    data, err := os.ReadFile(a.getWindowStatePath())
    if err != nil {
        return windowState{Width: 1200, Height: 800}
    }
    var state windowState
    json.Unmarshal(data, &state)
    return state
}

func (a *App) getWindowStatePath() string {
    exePath, _ := os.Executable()
    baseDir := filepath.Dir(exePath)
    return filepath.Join(baseDir, ".window-state.json")
}
```

在 `onStartup` 中恢复窗口状态，在 `onShutdown` 中保存窗口状态。

## router.go 改造

仅保留代理路由相关函数，移除 `SetupWeb`：

```go
// router.go 改造后
package router

// SetupProxy 注册代理服务路由（不变）
func SetupProxy(h *handler.ProxyHandler, ah *handler.AnthropicHandler, oh *handler.OllamaHandler) *gin.Engine {
    // ... 保持不变
}

// StartServer 启动 HTTP 服务（不变）
func StartServer(port string, engine *gin.Engine) *http.Server {
    // ... 保持不变
}

// SetupWeb 已移除，Web 管理界面由 Wails App 绑定层替代
```
