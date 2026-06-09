# LLM Proxy

一个桌面 LLM API 代理工具，提供 OpenAI/Anthropic/Ollama 兼容接口，负责请求转发、协议转换、用量统计和 Provider 管理。

## 语言

**Provider**:
第三方 LLM 服务商配置，包含 base_url、api_key、model 等，是代理转发的目标。
_避免_: Upstream, backend, service provider

**代理服务 (Proxy Service)**:
本应用内置的 Gin HTTP 服务器，监听本地端口，接收客户端请求并转发到上游 Provider。
_避免_: 后端服务, 代理服务器, 中间层

**请求日志 (Request Log)**:
每一次代理转发请求的记录，包含 token 用量、耗时、状态、请求/响应体。
_避免_: 日志, request record, 调用记录

**聚合统计 (Hourly Stats)**:
按小时汇总的 token 用量和请求计数，用于仪表盘统计报表。
_避免_: 统计分析, 汇总数据, aggregated stats

**活跃请求 (Active Request)**:
当前正在被代理转发、尚未完成的请求，用于实时监控面板。
_避免_: 正在进行的请求, live request

**协议转换 (Protocol Conversion)**:
在 OpenAI / Anthropic / Ollama 三种 API 格式之间进行双向转换。
_避免_: 协议适配, 格式转换

**Wails 绑定服务 (Bound Service)**:
注册到 Wails 应用中的 Go 结构体，其导出方法自动暴露给前端 React 调用。
_避免_: 绑定层, bound struct, frontend API

## 标记的歧义

- **"服务" (Service)**: 在 `internal/service/` 中指业务逻辑层（如 ProviderService、StatsService）；在 Wails 绑定层中指注册到 `application.NewService()` 的结构体。前者是纯 Go 业务逻辑，后者是前端可调用的 API 门面。后续讨论中需加前缀区分：**业务服务** vs **绑定服务**。

**DBManager**:
运行时数据库连接管理器，持有 `*gorm.DB` 和 `dbType`，所有 Repository 通过 `GetDB()` / `GetDBType()` 获取连接。支持运行时 `Replace(newDB, newDBType)` 热切换数据库，旧连接在指定延迟后自动关闭。
_避免_: 直接传递 `*gorm.DB`，DB helper，连接池

**连接测试 (Connection Test)**:
在应用数据库配置变更前，临时创建一个数据库连接并执行 Ping 操作，验证参数可用性（数据库是否存在、能否连通）。测试通过后才允许「应用配置」。
_避免_: 直接应用不验证

**会话 (Session)**:
一组连续的、属于同一轮对话的代理请求。会话通过递增的数字 ID 标识，在 assistant 响应的 content 末尾以 `[SESSION]ID[/SESSION]` 文本后缀传递，第三方在后续请求中通过 messages 数组回传。
_避免_: 对话, conversation, thread, 聊天记录

**会话标记 (Session Mark)**:
嵌入在 assistant 消息 content 末尾的文本标记，格式为 `[SESSION]数字ID[/SESSION]`。第三方的后续请求通过第一条 assistant 消息中的此标记识别会话归属。网关转发前会从 content 中移除该标记，避免污染上游 API。
_避免_: session tag, session token, 会话标识

**会话统计 (Session Stats)**:
按会话聚合的指标，包含请求次数、token 用量和成本。通过独立表 `chat_sessions` 持久化，每次请求结束后按 session_id 重新聚合 `request_logs` 计算。
_避免_: 会话分析, session analytics

## 示例对话

**开发者**: "用户添加了一个新的 Provider，绑定服务应该调用业务服务创建记录，然后刷新前端表格。"

**领域专家**: "对，`ProviderService`（绑定服务）调用 `provider_service.go`（业务服务）的 `Create` 方法，后端持久化完成后，前端自动重新拉取 Provider 列表。`GetProviders` 不会自动触发，需要调用方在写入操作后主动触发刷新。"
