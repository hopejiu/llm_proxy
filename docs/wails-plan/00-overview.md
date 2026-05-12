# Wails 桌面应用改造方案 - 总览

## 产品概述

将现有 LLM Proxy 项目从"Go 后端 + 浏览器访问 Web 管理界面"改造为 Wails 桌面应用。前端使用 Vanilla JS（无框架），管理界面嵌入桌面窗口，代理服务由同一进程管理并继续对外提供 OpenAI/Anthropic/Ollama 兼容接口。

## 核心功能

- Wails 桌面窗口承载管理界面（Provider 配置、实时监控、统计报表、运行日志），无需浏览器
- 代理服务（端口 8888）在 Wails 进程内启动，继续对外提供 LLM API 中转
- Go 后端业务逻辑（Repository/Service/ProxyHandler）保持不变，仅适配 Wails 绑定层
- 前端从多页面 HTML+fetch 改为单页面 + 动态加载页面片段，通过 Wails runtime 调用 Go 方法
- 左侧固定侧边栏导航 + 右侧内容区布局，更符合桌面应用交互习惯
- 日志文件保留 + 新增应用内完整版日志页面（日志流 + 级别过滤 + 关键词搜索 + 暂停/继续 + 清屏）
- 系统托盘支持：关闭窗口时最小化到托盘，代理服务继续运行
- 统一错误处理：AppError 类型携带 code + message
- 导入/导出使用 Wails 文件对话框
- 页面切换时自动清理资源，防止内存泄漏
- 共享模态框放在 index.html 主框架中，避免 DOM ID 冲突
- VO 层隔离：App 方法返回 VO 而非 Model

## 技术栈

- **桌面框架**: Wails v2 (Go + WebView2)，需要 CGO（Windows 构建需 MinGW-w64）
- **前端**: Vanilla JS + HTML + Tailwind CSS（本地文件）+ ECharts（本地文件），无构建步骤
- **后端**: Go 1.25 + Gin（代理服务保留） + GORM + SQLite/MySQL
- **构建**: Wails CLI（`wails build`），开发模式 `wails dev`
- **系统托盘**: getlantern/systray 或 wails v2 内置支持

## 核心策略：最小侵入式改造

**代理服务完全不动**：ProxyHandler/AnthropicHandler/OllamaHandler + Gin 引擎 + 路由，原样保留。代理路由保留 CORS 中间件（LLM 客户端可能跨域调用），Web 管理界面不再需要 CORS。

**Web 管理逻辑提取为 Wails Binding**：新建 App 结构体，将 WebHandler 中的业务逻辑封装为 Wails 绑定方法，去掉 `*gin.Context` 依赖。App 层直接调用 Service，保持 Service 层不变。引入 VO 层，App 方法返回 VO 而非 Model。

**前端最小改动**：3 个 HTML 页面拆分为页面片段，通过 fetch 动态加载到 index.html 的内容容器中。现有 JS 逻辑直接复用，仅将 `fetch('/api/xxx')` 替换为 Wails runtime 调用。共享模态框放在 index.html 主框架中。

**入口改造**：`cmd/server/main.go` → 根目录 `main.go`（Wails 入口），在 `onStartup` 回调中启动代理服务 goroutine。

## 方案文件索引

| 文件 | 内容 |
|------|------|
| [01-architecture.md](01-architecture.md) | 架构设计、目录结构、模块职责 |
| [02-backend.md](02-backend.md) | Go 后端改造：App 绑定层、VO 层、错误处理、日志 RingBuffer |
| [03-frontend.md](03-frontend.md) | 前端改造：页面架构、JS 生命周期、fetch→Wails 映射、模态框策略 |
| [04-proxy-and-config.md](04-proxy-and-config.md) | 代理服务集成、配置管理、系统托盘、窗口状态 |
| [05-build-and-deploy.md](05-build-and-deploy.md) | 构建环境、CGO 依赖、CI/CD、开发模式 |
| [06-ui-design.md](06-ui-design.md) | UI 设计风格、布局、各页面设计 |
