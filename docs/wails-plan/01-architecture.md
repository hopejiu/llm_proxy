# 架构设计

## 架构图

```
┌─────────────────────────────────────────────────────────┐
│                   Wails 桌面应用进程                      │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │              WebView2 窗口                        │   │
│  │  ┌──────────┐  ┌──────────────────────────────┐  │   │
│  │  │ 侧边栏    │  │ 内容容器 #content            │  │   │
│  │  │ 导航菜单  │  │ ← fetch pages/*.html 动态加载 │  │   │
│  │  │ 代理状态  │  │ ← Wails runtime 调用 Go 方法  │  │   │
│  │  │          │  │ ← Wails Events 监听实时推送    │  │   │
│  │  └──────────┘  └──────────────────────────────┘  │   │
│  │  ┌──────────────────────────────────────────────┐ │   │
│  │  │ 共享模态框（日志详情、活跃请求详情）           │ │   │
│  │  └──────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Go 后端                              │   │
│  │  App (Wails 绑定层) → VO → Service → Repository  │   │
│  │  LogRingBuffer → Wails Events 推送               │   │
│  │  ActiveRequestTracker → Wails Events 推送        │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │         代理 HTTP 服务器 (:8888)                  │   │
│  │  Gin Engine + ProxyHandler/Anthropic/Ollama      │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │              后台任务                              │   │
│  │  CleanupService (定时汇总+清理)                   │   │
│  │  LogEventPusher (日志→Wails Events)              │   │
│  │  ActiveEventPusher (活跃请求→Wails Events)       │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │              系统托盘                              │   │
│  │  显示窗口 / 停止代理 / 退出应用                    │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
└─────────────────────────────────────────────────────────┘

LLM 客户端 ──HTTP──→ 代理 HTTP 服务器 (:8888)
```

## 目录结构

```
llm_statistic/
├── main.go                          # [NEW] Wails 应用入口
├── app.go                           # [NEW] Wails App 绑定层
├── vo.go                            # [NEW] VO 定义（ProviderVO, RequestLogVO 等）
├── app_error.go                     # [NEW] AppError 统一错误类型
├── wails.json                       # [NEW] Wails 项目配置
├── build/                           # [NEW] Wails 构建资源
│   ├── appicon.png
│   └── windows/
│       ├── icon.ico
│       └── info.json
├── frontend/                        # [NEW] Vanilla JS 前端
│   ├── index.html                   # 主框架（侧边栏 + 内容容器 + 共享模态框）
│   ├── pages/                       # 页面 HTML 片段（不含 <script>，不含共享模态框）
│   │   ├── providers.html           # 配置管理页内容
│   │   ├── realtime.html            # 实时监控页内容
│   │   ├── stats.html               # 统计报表页内容
│   │   └── logs.html                # [NEW] 运行日志页内容
│   ├── js/
│   │   ├── common.js                # 公共工具 + Wails runtime 封装 + 共享模态框逻辑
│   │   ├── router.js                # [NEW] 页面路由：loadPage/destroyPage 管理
│   │   ├── providers.js             # 配置管理页逻辑（导出 initProviders/destroyProviders）
│   │   ├── realtime.js              # 实时监控页逻辑（导出 initRealtime/destroyRealtime）
│   │   ├── stats.js                 # 统计报表页逻辑（导出 initStats/destroyStats）
│   │   └── logs.js                  # [NEW] 运行日志页逻辑
│   ├── css/
│   │   ├── style.css                # 自定义样式（迁移，新增侧边栏+日志页样式）
│   │   └── tailwind.min.css         # Tailwind 本地文件（迁移）
│   ├── lib/
│   │   └── echarts.min.js           # ECharts 本地文件（迁移，在 index.html 全局加载）
│   └── wailsjs/                     # Wails 自动生成的绑定 JS
│       ├── go/main/App.js
│       └── runtime/runtime.js
├── internal/                        # [KEEP/MODIFY] 业务逻辑层
│   ├── config/                      # [MODIFY] 移除 WebPort，新增桌面配置项，配置路径改为用户数据目录
│   │   └── config.go
│   ├── converter/                   # [KEEP]
│   ├── handler/                     # [KEEP] 代理 Handler 不变；WebHandler 不再使用但暂保留
│   ├── logger/                      # [MODIFY] 新增 RingBuffer + 自定义 slog Handler + Wails 事件推送
│   │   ├── logger.go               # [MODIFY] 改造 Init，支持 RingBuffer
│   │   ├── ringbuffer.go           # [NEW] 环形缓冲区实现
│   │   └── handler.go              # [NEW] 自定义 slog Handler（同时写文件+RingBuffer）
│   ├── middleware/                  # [KEEP] CORS 仅代理路由使用
│   ├── model/                       # [KEEP]
│   ├── repository/                  # [KEEP]
│   ├── router/                      # [MODIFY] 仅保留 SetupProxy + StartServer，移除 SetupWeb
│   │   └── router.go
│   └── service/                     # [KEEP]
├── cmd/                             # [DELETE] cmd/server/main.go 删除
└── web/                             # [DELETE] 旧前端文件，被 frontend/ 替代
```

## 模块职责

| 模块 | 职责 | 改动程度 |
|------|------|----------|
| `main.go` | Wails 入口，DI 组装，窗口配置，托盘设置 | 新建 |
| `app.go` | Wails 绑定层，所有前端可调用的 Go 方法 | 新建 |
| `vo.go` | VO 定义，Model→VO 转换，脱敏逻辑 | 新建 |
| `app_error.go` | AppError 类型，统一错误码和消息 | 新建 |
| `internal/config` | 配置加载，路径改为用户数据目录 | 修改 |
| `internal/logger` | 日志初始化 + RingBuffer + 自定义 Handler | 修改+新增 |
| `internal/router` | 仅保留代理路由，移除 Web 路由 | 修改 |
| `internal/service` | 业务逻辑 | 不变 |
| `internal/repository` | 数据访问 | 不变 |
| `internal/handler` | 代理 Handler | 不变 |
| `internal/converter` | 协议转换 | 不变 |
| `frontend/*` | 前端全部文件 | 新建（迁移+改造） |