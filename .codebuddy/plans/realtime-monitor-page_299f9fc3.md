---
name: realtime-monitor-page
overview: 新建实时监控页面，使用 SSE 实时推送活跃请求的流式输出内容，展示活跃请求列表+最近完成请求列表，点击可查看完整详情，并在导航栏添加入口。
design:
  architecture:
    framework: html
  styleKeywords:
    - 罗兰紫主题
    - 实时监控
    - 深色输出区域
    - 脉冲动画
    - 打字机效果
  fontSystem:
    fontFamily: PingFang-SC
    heading:
      size: 20px
      weight: 700
    subheading:
      size: 16px
      weight: 600
    body:
      size: 14px
      weight: 400
  colorSystem:
    primary:
      - "#8B5CF6"
      - "#7C3AED"
      - "#A78BFA"
    background:
      - "#F8FAFC"
      - "#FFFFFF"
      - "#1E293B"
    text:
      - "#1E293B"
      - "#475569"
      - "#94A3B8"
      - "#E2E8F0"
    functional:
      - "#10B981"
      - "#EF4444"
      - "#F59E0B"
todos:
  - id: create-monitor-service
    content: 新建 monitor_service.go，实现活跃请求追踪和 SSE 广播
    status: completed
  - id: modify-base-handler
    content: 修改 base_handler.go，注入监控钩子到流式读取循环
    status: completed
    dependencies:
      - create-monitor-service
  - id: modify-handlers
    content: 修改 proxy/anthropic/ollama handler，请求入口出口调用 Register/Unregister
    status: completed
    dependencies:
      - modify-base-handler
  - id: create-monitor-handler
    content: 新建 monitor_handler.go，实现 SSE 端点和监控 API
    status: completed
    dependencies:
      - create-monitor-service
  - id: modify-router-main
    content: 修改 router.go 和 main.go，注册监控路由并注入依赖
    status: completed
    dependencies:
      - create-monitor-handler
  - id: create-monitor-frontend
    content: 新建 monitor.html 和 monitor.js，实现实时监控前端页面
    status: completed
  - id: update-nav-and-style
    content: 修改 index.html/stats.html 导航栏，添加监控页面 CSS 样式
    status: completed
    dependencies:
      - create-monitor-frontend
---

## 产品概述

新建"实时监控"页面，通过 SSE 实时推送大模型请求信息，用户可实时查看活跃请求的流式输出内容，以及最近完成的请求列表，点击已完成请求可查看完整详情。

## 核心功能

- **活跃请求实时监控**：当大模型请求正在进行时，实时显示请求信息（模型、Provider、开始时间），并流式展示输出内容（打字机效果）
- **最近完成请求列表**：展示最近完成的请求摘要（模型、token用量、耗时、状态等）
- **请求详情查看**：点击已完成请求弹出详情弹窗，查看完整 Request Body / Response Body
- **导航入口**：在现有页面导航栏添加"实时监控"链接，与"统计报表"并列

## 技术栈

- 后端：Go + Gin（复用现有技术栈）
- 前端：HTML + Tailwind CSS + 原生 JS（复用现有技术栈）
- 实时通信：SSE（Server-Sent Events）

## 实现方案

### 核心架构

采用**内存活跃请求注册表 + SSE 广播**模式：

1. 新建 `MonitorService` 作为全局单例，维护活跃请求 map 和 SSE 客户端列表
2. 在 `BaseHandler` 的流式/非流式请求处理流程中，注入监控钩子（Register → UpdateContent → Unregister）
3. 流式请求每读取一行 SSE 数据，就通过 MonitorService 广播增量内容给所有 SSE 客户端
4. 前端通过 EventSource 连接 SSE 端点，实时渲染活跃请求卡片

### 关键技术决策

**为什么在 BaseHandler 注入钩子而非修改每个 Handler？**

- ProxyHandler、AnthropicHandler、OllamaHandler 都组合了 BaseHandler
- 流式读取的核心逻辑在 `BaseHandler.readStreamWithFirstByteTimeout` 中
- 在此统一注入，三种协议的请求都能被监控，无需修改三个 Handler

**SSE 事件类型设计：**

- `request_start`：新请求开始，包含 requestID、model、provider、requestBody
- `request_content`：流式增量内容，包含 requestID、content（增量文本）
- `request_end`：请求完成，包含 requestID、status、duration、token 用量
- `request_error`：请求失败，包含 requestID、errorMessage

**活跃请求超时清理：**

- 设置 10 分钟超时，防止因异常未 Unregister 导致的内存泄漏
- MonitorService 启动后台 goroutine 定期扫描清理

### 数据流

```
用户请求 → ProxyHandler/Anthropic/Ollama
  → BaseHandler.RegisterRequest(requestID, model, provider, requestBody)
  → 流式读取循环中每行:
    → BaseHandler.UpdateRequestContent(requestID, contentDelta)
    → MonitorService.Broadcast("request_content", {requestID, content})
  → 请求完成:
    → BaseHandler.UnregisterRequest(requestID, status, tokens, duration)
    → MonitorService.Broadcast("request_end", {requestID, ...})

SSE客户端 ← MonitorService.Broadcast → 所有已连接的浏览器页面
```

## 实现细节

### 后端修改

#### 1. 新建 `internal/service/monitor_service.go`

- `ActiveRequest` 结构体：RequestID, Model, ProviderName, ProviderID, StartTime, RequestBody, StreamContent(strings.Builder), ThinkingContent(strings.Builder), Status, InputTokens, OutputTokens, TotalTokens, CachedTokens, Duration, IsStream
- `MonitorService` 结构体：activeRequests(sync.Map), sseClients(map + sync.Mutex), broadcast chan
- 方法：RegisterRequest, UnregisterRequest, UpdateRequestContent, GetActiveRequests, GetRecentCompletedRequests, AddSSEClient, RemoveSSEClient, Broadcast, startCleanupGoroutine
- SSE 客户端管理：每个客户端一个 channel（buffered 100），广播时非阻塞发送，满则断开

#### 2. 修改 `internal/handler/base_handler.go`

- BaseHandler 新增 `monitorService *service.MonitorService` 字段（可为 nil，nil 时不监控）
- 新增 `SetMonitorService(ms *service.MonitorService)` 方法
- 在 `readStreamWithFirstByteTimeout` 的 scanner 循环中，每读取一行有效数据后调用 `monitorService.UpdateRequestContent`
- 新增 `RegisterMonitorRequest` / `UnregisterMonitorRequest` 辅助方法
- 在 `CreateRequestLog` 方法附近调用 RegisterMonitorRequest
- 注意：需要在请求处理入口处将 requestID 和 monitor 关联，可通过 context 传递或直接在各 handler 入口调用

#### 3. 修改 `internal/handler/proxy_handler.go`

- 在 `ChatCompletions` 方法中，解析出 model 后调用 `RegisterMonitorRequest`
- 在 `handleNormalRequestOpenAI` 和 `handleStreamRequestOpenAI` 结束时调用 `UnregisterMonitorRequest`

#### 4. 修改 `internal/handler/anthropic_handler.go` 和 `internal/handler/ollama_handler.go`

- 同理在请求入口/出口调用 Register/Unregister

#### 5. 新建 `internal/handler/monitor_handler.go`

- `MonitorHandler` 结构体，持有 `monitorService` 和 `requestLogRepo`
- `MonitorPage(c *gin.Context)`：渲染 monitor.html
- `SSEStream(c *gin.Context)`：SSE 端点，设置 headers，注册客户端 channel，循环读取并写入
- `GetActiveRequests(c *gin.Context)`：返回当前活跃请求列表
- `GetRecentCompleted(c *gin.Context)`：返回最近完成的请求（复用 requestLogRepo.GetRecent）

#### 6. 修改 `internal/router/router.go`

- SetupWeb 中添加：`r.GET("/monitor", monitorHandler.MonitorPage)`
- API 组添加：`api.GET("/monitor/stream", monitorHandler.SSEStream)`、`api.GET("/monitor/active", monitorHandler.GetActiveRequests)`、`api.GET("/monitor/recent", monitorHandler.GetRecentCompleted)`
- SetupWeb 函数签名新增 `monitorHandler *handler.MonitorHandler` 参数

#### 7. 修改 `cmd/server/main.go`

- 创建 `monitorService := service.NewMonitorService()`
- 将 monitorService 注入到各 handler（通过 SetMonitorService 或构造函数）
- 创建 `monitorHandler := handler.NewMonitorHandler(monitorService, requestLogRepo)`
- 传递给 `router.SetupWeb`

### 前端修改

#### 8. 新建 `web/templates/monitor.html`

- 导航栏：与现有页面一致的紫色渐变导航，包含"配置管理""统计报表""实时监控"三个链接
- 活跃请求区域：卡片列表，每个卡片显示模型名、Provider、已耗时、实时输出内容（打字机效果，带闪烁光标）
- 最近完成请求表格：时间、Provider、模型、Input/Output/Cached Tokens、耗时、状态、操作（查看详情按钮）
- 详情弹窗：复用 stats.html 的 logDetailModal 样式，显示 Request Body / Response Body

#### 9. 新建 `web/static/js/monitor.js`

- SSE 连接管理：创建 EventSource 连接 `/api/monitor/stream`，处理四种事件类型
- 活跃请求卡片：动态创建/更新/移除 DOM 元素，流式内容追加显示
- 最近完成请求：收到 request_end 事件时，将请求从活跃区移到完成列表
- 详情弹窗：点击查看详情，调用 `/api/logs/:id` 获取完整数据
- 断线重连：EventSource 自带重连，额外处理重连后同步活跃请求列表

#### 10. 修改 `web/templates/index.html` 和 `web/templates/stats.html`

- 导航栏添加"实时监控"链接 `<a href="/monitor">实时监控</a>`

#### 11. 修改 `web/static/css/style.css`

- 添加监控页面样式：活跃请求卡片、流式输出区域、闪烁光标动画、脉冲状态指示器

## 目录结构

```
f:\work\2026\llm_proxy\
├── internal/
│   ├── service/
│   │   └── monitor_service.go       # [NEW] 活跃请求追踪 + SSE 广播服务
│   ├── handler/
│   │   ├── base_handler.go          # [MODIFY] 注入监控钩子，流式读取时广播内容
│   │   ├── proxy_handler.go         # [MODIFY] 请求入口/出口调用 Register/Unregister
│   │   ├── anthropic_handler.go     # [MODIFY] 请求入口/出口调用 Register/Unregister
│   │   ├── ollama_handler.go        # [MODIFY] 请求入口/出口调用 Register/Unregister
│   │   └── monitor_handler.go       # [NEW] 监控页面 + SSE 端点 + API
│   └── router/
│       └── router.go                # [MODIFY] 注册监控路由，SetupWeb 签名变更
├── cmd/
│   └── server/
│       └── main.go                  # [MODIFY] 创建 MonitorService/MonitorHandler 并注入
├── web/
│   ├── templates/
│   │   ├── monitor.html             # [NEW] 实时监控页面
│   │   ├── index.html               # [MODIFY] 导航栏添加"实时监控"链接
│   │   └── stats.html               # [MODIFY] 导航栏添加"实时监控"链接
│   └── static/
│       ├── css/
│       │   └── style.css            # [MODIFY] 添加监控页面样式
│       └── js/
│           └── monitor.js           # [NEW] 监控页面 JS（SSE 连接、DOM 渲染）
```

## 性能注意事项

- SSE 客户端 channel 使用 buffered channel（容量100），广播时非阻塞发送，满则断开该客户端，避免慢客户端阻塞
- 活跃请求内容使用 strings.Builder 追加，避免频繁字符串拼接
- MonitorService 的 activeRequests 使用 sync.Map，适合读多写少场景
- 流式广播只发送增量内容（content delta），不发送全量，减少带宽
- 超时清理 goroutine 每 30 秒扫描一次，移除超过 10 分钟的活跃请求

## 设计风格

沿用现有项目的罗兰紫主题风格，保持视觉一致性。页面采用深浅对比布局，活跃请求区域使用动态视觉元素突出实时感。

## 页面布局

### 导航栏

与现有页面一致的紫色渐变导航栏，包含三个链接：配置管理、统计报表、实时监控（当前页高亮）

### 活跃请求区域（页面上半部分）

- 区域标题"活跃请求"右侧显示当前活跃数量徽标（紫色圆点 + 数字）
- 无活跃请求时显示空状态提示"暂无活跃请求，等待新请求接入..."
- 每个活跃请求为一张卡片：
- 顶部：模型名（粗体）+ Provider名（标签）+ 脉冲动画绿色圆点（表示进行中）+ 已耗时（实时更新）
- 中部：流式输出内容区域，深色背景（#1E293B），等宽字体，内容逐字追加，末尾有闪烁光标
- 底部：Input Tokens / Output Tokens 实时计数（流式请求中持续更新）

### 最近完成请求区域（页面下半部分）

- 区域标题"最近完成"右侧显示自动刷新开关
- 表格列：时间、Provider、模型、Input、Output、Cached、Total、耗时、状态、操作
- 状态列：success 绿色标签，error 红色标签
- 操作列："详情"按钮，点击弹出详情弹窗

### 详情弹窗

- 复用 stats.html 的 logDetailModal 样式
- 元信息网格：模型、Provider、耗时、Input/Output/Total Tokens、状态
- Request Body 和 Response Body 两个可折叠/展开的代码块，带复制按钮

## 交互细节

- 活跃请求卡片出现时有 fadeIn 动画
- 请求完成时卡片有淡出过渡效果，然后移入完成列表
- 流式输出区域自动滚动到底部
- SSE 断线时顶部显示黄色重连提示条
- 脉冲动画：活跃请求的绿色圆点持续脉冲，表示请求进行中

## SubAgent

- **code-explorer**: 用于在实现阶段深入搜索跨文件依赖关系，确认 BaseHandler 中流式读取循环的精确注入点，以及验证所有 handler 的请求入口/出口是否完整覆盖