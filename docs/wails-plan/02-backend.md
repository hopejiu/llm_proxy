# Go 后端改造

## App 绑定层 (app.go)

App 结构体持有 Service 引用，暴露给前端的方法签名不含 gin.Context，返回 VO 而非 Model：

```go
type App struct {
    ctx             context.Context
    providerService *service.ProviderService
    statsService    *service.StatsService
    tracker         *handler.ActiveRequestTracker
    ringBuffer      *logger.RingBuffer
    proxyServer     *http.Server
    cleanupCancel   context.CancelFunc
    cfg             *config.Config
}
```

### 绑定方法清单

| 方法 | 返回类型 | 说明 |
|------|----------|------|
| `GetProviders()` | `[]vo.ProviderVO, error` | 获取列表（API Key 脱敏） |
| `GetProvider(id uint)` | `vo.ProviderVO, error` | 获取单个（API Key 脱敏） |
| `CreateProvider(data vo.ProviderCreateVO)` | `vo.ProviderVO, error` | 创建 |
| `UpdateProvider(id uint, data vo.ProviderUpdateVO)` | `vo.ProviderVO, error` | 更新（含 API Key 保留逻辑） |
| `DeleteProvider(id uint)` | `error` | 删除 |
| `ExportProviders()` | `[]vo.ProviderExportVO, error` | 导出（含完整 API Key） |
| `ImportProviders(data []vo.ProviderImportVO)` | `error` | 导入 |
| `SetupCodeBuddy()` | `vo.CodeBuddyResultVO, error` | 配置 CodeBuddy（适配动态端口） |
| `GetStats()` | `map[string]*model.TokenStats, error` | 仪表盘统计 |
| `GetDailyStats()` | `[]model.TokenStats, error` | 30 天每日统计 |
| `GetHourlyStatsByDate(date string)` | `[]model.HourlyStatsResult, error` | 分时统计 |
| `GetRecentLogs(limit int)` | `[]vo.RequestLogVO, error` | 最近请求日志 |
| `GetLogDetail(id uint)` | `vo.RequestLogDetailVO, error` | 单条日志详情 |
| `GetActiveRequests()` | `[]vo.ActiveRequestVO, error` | 当前活跃请求 |
| `GetActiveRequest(requestID string)` | `vo.ActiveRequestVO, error` | 按 ID 获取单个活跃请求 |
| `GetProxyStatus()` | `vo.ProxyStatusVO, error` | 代理服务状态（starting/running/error） |
| `StartProxy()` | `error` | 启动代理服务 |
| `StopProxy()` | `error` | 停止代理服务 |
| `GetLogHistory()` | `[]vo.LogEntryVO, error` | 获取 RingBuffer 历史日志 |
| `ExportProvidersToFile()` | `string, error` | 导出到文件（使用 SaveFileDialog） |
| `ImportProvidersFromFile()` | `error` | 从文件导入（使用 OpenFileDialog） |

### Wails 绑定规则

- 所有方法必须是 `App` 的指针方法
- 返回值必须是 `(result, error)` 两值
- 参数和返回值必须可 JSON 序列化
- `context.Context` 不出现在方法签名中（通过 `app.ctx` 访问）

## VO 层 (vo.go)

### 为什么需要 VO

- `model.ProviderConfig` 包含 GORM tag，直接序列化会暴露不必要的字段信息
- `model.RequestLog` 的 `Provider ProviderConfig` 字段（`gorm:"-"`）GORM 查询时不填充，直接返回前端拿不到 Provider 名称
- API Key 脱敏逻辑应从 Model 移到 VO 转换层
- 前端创建/更新 Provider 时提交的数据结构与 Model 不同（不含 ID、CreatedAt 等）

### VO 定义

```go
// ProviderVO 返回给前端的 Provider 视图（API Key 脱敏）
type ProviderVO struct {
    ID          uint   `json:"id"`
    Name        string `json:"name"`
    AutoSuffix  bool   `json:"auto_suffix"`
    UrlSuffix   string `json:"url_suffix"`
    BaseURL     string `json:"base_url"`
    APIKey      string `json:"api_key"`       // 脱敏后的 Key
    Model       string `json:"model"`
    Alias       string `json:"alias"`
    ExtraParams string `json:"extra_params"`
    CreatedAt   string `json:"created_at"`
    UpdatedAt   string `json:"updated_at"`
}

// ProviderCreateVO 前端创建 Provider 时提交
type ProviderCreateVO struct {
    Name        string `json:"name"`
    AutoSuffix  bool   `json:"auto_suffix"`
    UrlSuffix   string `json:"url_suffix"`
    BaseURL     string `json:"base_url"`
    APIKey      string `json:"api_key"`
    Model       string `json:"model"`
    Alias       string `json:"alias"`
    ExtraParams string `json:"extra_params"`
}

// ProviderUpdateVO 前端更新 Provider 时提交（API Key 可能是脱敏格式）
type ProviderUpdateVO struct {
    // 与 CreateVO 相同，但 api_key 可能含 ****
    ProviderCreateVO
}

// ProviderExportVO 导出时包含完整 API Key
type ProviderExportVO = ProviderCreateVO  // 导出需要完整 Key

// RequestLogVO 请求日志列表视图
type RequestLogVO struct {
    ID           uint   `json:"id"`
    ProviderID   uint   `json:"provider_id"`
    ProviderName string `json:"provider_name"` // 手动填充
    Model        string `json:"model"`
    InputTokens  int    `json:"input_tokens"`
    OutputTokens int    `json:"output_tokens"`
    TotalTokens  int    `json:"total_tokens"`
    CachedTokens int    `json:"cached_tokens"`
    Status       string `json:"status"`
    ErrorMessage string `json:"error_message"`
    Duration     int64  `json:"duration"`
    CreatedAt    string `json:"created_at"`
}

// RequestLogDetailVO 请求日志详情（含请求体和响应体）
type RequestLogDetailVO struct {
    RequestLogVO
    ResponseContent string `json:"response_content"`
    ThinkingContent string `json:"thinking_content"`
    RequestBody     string `json:"request_body"`
    ResponseBody    string `json:"response_body"`
}

// ActiveRequestVO 活跃请求视图
type ActiveRequestVO = handler.ActiveRequest  // 结构已满足需求，直接复用

// LogEntryVO 日志条目视图
type LogEntryVO struct {
    Time    string `json:"time"`
    Level   string `json:"level"`
    Message string `json:"message"`
}

// ProxyStatusVO 代理服务状态
type ProxyStatusVO struct {
    Status  string `json:"status"`  // "starting" | "running" | "stopped" | "error"
    Port    string `json:"port"`
    Error   string `json:"error,omitempty"`
}

// CodeBuddyResultVO CodeBuddy 配置结果
type CodeBuddyResultVO struct {
    Message string `json:"message"`
    Path    string `json:"path"`
    Exists  bool   `json:"exists"`
    Added   bool   `json:"added"`
    Models  int    `json:"models"`
}
```

### Model → VO 转换

在 `vo.go` 中提供转换函数：

```go
func ProviderToVO(p *model.ProviderConfig) ProviderVO { ... }
func RequestLogToVO(log *model.RequestLog) RequestLogVO { ... }
func RequestLogToDetailVO(log *model.RequestLog) RequestLogDetailVO { ... }
```

关键转换逻辑：
- `ProviderVO.APIKey` = `p.MaskAPIKey()`（脱敏）
- `RequestLogVO.ProviderName` = 手动查询 Provider 名称（因为 `gorm:"-"` 不自动填充）
- 时间字段格式化为 `2006-01-02 15:04:05` 字符串

## 统一错误处理 (app_error.go)

```go
type AppError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func (e *AppError) Error() string {
    return e.Message
}

// 预定义错误
var (
    ErrBadRequest   = &AppError{Code: "BAD_REQUEST", Message: "请求参数错误"}
    ErrNotFound     = &AppError{Code: "NOT_FOUND", Message: "资源不存在"}
    ErrInternal     = &AppError{Code: "INTERNAL", Message: "内部错误"}
    ErrProxyRunning = &AppError{Code: "PROXY_RUNNING", Message: "代理服务正在运行"}
    ErrProxyStopped = &AppError{Code: "PROXY_STOPPED", Message: "代理服务未运行"}
)

func NewAppError(code, message string) *AppError {
    return &AppError{Code: code, Message: message}
}
```

前端统一处理：

```javascript
async function callGo(fn) {
    try {
        return await fn();
    } catch (err) {
        // Wails 将 Go error 序列化为字符串
        // AppError 的 JSON 会被序列化为 {"code":"...","message":"..."}
        let appErr;
        try {
            appErr = JSON.parse(err);
        } catch {
            appErr = { code: 'UNKNOWN', message: String(err) };
        }
        showToast(appErr.message || '操作失败', 'error');
        throw appErr;
    }
}
```

## 日志系统改造

### 自定义 slog Handler (logger/handler.go)

实现 `slog.Handler` 接口，同时写入文件 + RingBuffer：

```go
type multiHandler struct {
    fileHandler slog.Handler  // 写入日志文件 + stdout
    ringBuffer  *RingBuffer
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
    // 先写文件（同步，保证日志持久化）
    if err := h.fileHandler.Handle(ctx, r); err != nil {
        return err
    }
    // 再写 RingBuffer（极轻量，仅 append）
    h.ringBuffer.Write(LogEntry{
        Time:    r.Time.Format("15:04:05.000"),
        Level:   r.Level.String(),
        Message: r.Message,
    })
    return nil
}
```

### RingBuffer (logger/ringbuffer.go)

```go
type LogEntry struct {
    Time    string `json:"time"`
    Level   string `json:"level"`
    Message string `json:"message"`
}

type RingBuffer struct {
    mu      sync.RWMutex
    entries []LogEntry
    cap     int
    pos     int  // 写入位置
    full    bool
    // 订阅机制
    subscribers []chan LogEntry
    subMu      sync.RWMutex
}

func NewRingBuffer(cap int) *RingBuffer { ... }
func (rb *RingBuffer) Write(entry LogEntry) { ... }       // 环形写入 + 通知订阅者
func (rb *RingBuffer) GetAll() []LogEntry { ... }          // 获取全部历史
func (rb *RingBuffer) Subscribe() <-chan LogEntry { ... }  // 订阅新日志
func (rb *RingBuffer) Unsubscribe(ch <-chan LogEntry) { ... }
```

### 背压策略

- 订阅 channel 缓冲区大小 64
- 写入时如果 channel 满则丢弃（`select { case ch <- entry: default: }`），不阻塞业务逻辑
- 日志页面暂停时前端停止 EventsOn 处理，channel 中的消息自然被丢弃

### 日志事件推送 goroutine

```go
// 在 app.go 的 startup 中启动
func (a *App) startLogEventPusher() {
    ch := a.ringBuffer.Subscribe()
    defer a.ringBuffer.Unsubscribe(ch)
    
    for {
        select {
        case entry := <-ch:
            runtime.EventsEmit(a.ctx, "logEvent", entry)
        case <-a.ctx.Done():
            return
        }
    }
}
```

### 活跃请求事件推送

类似日志推送，当 ActiveRequestTracker 发生变更时推送事件：

```go
// 在 tracker 中增加回调机制
type TrackerChangeListener interface {
    OnRequestAdd(req *ActiveRequest)
    OnRequestRemove(requestID string)
    OnRequestUpdate(requestID string)
}

// App 实现 Listener，推送 Wails Events
func (a *App) OnRequestAdd(req *handler.ActiveRequest) {
    runtime.EventsEmit(a.ctx, "activeRequestAdd", req)
}
func (a *App) OnRequestRemove(requestID string) {
    runtime.EventsEmit(a.ctx, "activeRequestRemove", requestID)
}
```

前端可选择性使用 Events 推送替代 500ms 轮询，或降低轮询频率到 2-3 秒作为兜底。

## UpdateProvider API Key 保留逻辑

与原 WebHandler 逻辑一致，但在 App 层实现：

```go
func (a *App) UpdateProvider(id uint, data vo.ProviderUpdateVO) (vo.ProviderVO, error) {
    // 如果 API Key 是脱敏格式（包含 ****），保留原有密钥
    if strings.Contains(data.APIKey, "****") {
        existing, err := a.providerService.GetProvider(id)
        if err == nil {
            data.APIKey = existing.APIKey
        }
    }
    // VO → Model 转换
    provider := vo.UpdateVOToModel(id, data)
    err := a.providerService.UpdateProvider(provider)
    ...
}
```

## SetupCodeBuddy 动态端口

```go
func (a *App) SetupCodeBuddy() (vo.CodeBuddyResultVO, error) {
    // 使用实际配置的代理端口，而非硬编码 8888
    targetURL := fmt.Sprintf("http://localhost:%s/v1", a.cfg.ProxyPort)
    ...
}
```

## RequestLog 的 ProviderName 填充

当前 `model.RequestLog` 的 `Provider ProviderConfig` 字段标记 `gorm:"-"`，GORM 不会自动填充。在 App 层手动处理：

```go
func (a *App) GetRecentLogs(limit int) ([]vo.RequestLogVO, error) {
    logs, err := a.statsService.GetRecentLogs(limit)
    if err != nil {
        return nil, NewAppError("INTERNAL", "获取日志列表失败")
    }
    
    result := make([]vo.RequestLogVO, 0, len(logs))
    for _, log := range logs {
        vo := vo.RequestLogToVO(&log)
        // 手动查询 Provider 名称
        if log.ProviderID != model.DeletedProviderID {
            if p, err := a.providerService.GetProvider(log.ProviderID); err == nil {
                vo.ProviderName = p.Name
            }
        }
        result = append(result, vo)
    }
    return result, nil
}
```

> 注意：批量查询时为避免 N+1 问题，可先收集所有 ProviderID，一次性查询后建 map。