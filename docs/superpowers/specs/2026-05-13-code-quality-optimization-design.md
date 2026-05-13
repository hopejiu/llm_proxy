# 代码质量与体验优化设计

## 概述

对项目进行分批优化，涵盖并发安全修复、代码重构、用户体验改进三个方面，共 3 批 15 项改进。

---

## 第一批：关键修复

### L1 — Provider 缓存并发安全

**问题**：`getAllProvidersCached()` 返回 slice 浅拷贝，调用者修改返回值会影响缓存；`GetProviderByModel()` 返回指针，外部可修改缓存中的结构体。

**方案**：
- `getAllProvidersCached()` 返回时做深拷贝：`result := make([]model.ProviderConfig, len(s.providerCache)); copy(result, s.providerCache)`
- `GetProviderByModel()` 返回值类型而非指针，避免外部修改缓存

**文件**：`internal/service/proxy_service.go`

### L3 — Stream retry body 消费

**问题**：`PrepareRequestBody()` 返回 `io.NopCloser(bytes.NewReader(body))`，首次请求消费 body 后，重试时 body 已为空。

**方案**：在 `ExecuteStreamWithRetry` 中保存原始 body bytes，每次重试前重新构造 reader：
```go
// 在重试循环中
bodyReader := io.NopCloser(bytes.NewReader(originalBody))
req.Body = bodyReader
```

**文件**：`internal/handler/base_handler.go`（stream_handler 部分拆分后）

### R1 — SetupCodeBuddy 统一

**问题**：app.go 和 web_handler.go 各有一份 SetupCodeBuddy 实现，逻辑完全相同。web_handler.go 中端口硬编码为 8888，用户修改端口后会写入错误 URL。

**方案**：
- 新建 `internal/service/codebuddy_service.go`，提供 `SetupCodeBuddy(dataDir, targetURL string) (*CodeBuddyResult, error)`
- app.go 调用时传入 `fmt.Sprintf("http://localhost:%s/v1", a.cfg.ProxyPort)`
- web_handler.go 调用时传入 `fmt.Sprintf("http://localhost:%s/v1", cfg.ProxyPort)`（从配置读取）
- 删除两处的重复实现

**文件**：新建 `internal/service/codebuddy_service.go`，修改 `app.go`、`internal/handler/web_handler.go`

### R4 — 错误码统一

**问题**：`app_error.go` 定义 `ErrBadRequest` 等 AppError 实例，`response.go` 定义同名 `ErrBadRequest` 字符串常量，命名冲突但含义不同。

**方案**：
- `response.go` 中的字符串常量重命名为 `CodeBadRequest`、`CodeNotFound`、`CodeInternal`、`CodeInvalidInput`
- `app_error.go` 中的预定义错误变量保持不变（它们是 AppError 类型实例）
- 更新所有引用 `response.go` 常量的代码

**文件**：`app_error.go`、`internal/handler/response.go`、所有引用错误码的 handler 文件

---

## 第二批：结构优化

### R2 — base_handler.go 拆分

**问题**：795 行，职责混杂（HTTP 请求、流式读取、SSE 解析、日志保存、body 清洗）。

**方案**：拆为 6 个文件，同属 `handler` 包，无需改 import：

| 新文件 | 内容 | 预估行数 |
|--------|------|----------|
| `base_handler.go` | BaseHandler 结构体、NewBaseHandler、HandlerConfig、UpstreamError、ResolveUpstreamError | ~80 |
| `request_context.go` | requestIDKey、generateRequestID、contextWithRequestID、requestIDFromContext、ProxyRequestInfo、ParseRequestFunc、HandleRequestFunc、HandleProxyRequest | ~100 |
| `http_client.go` | buildHTTPRequest、SendStreamRequest、SendStreamRequestWithHeaders、SendRequest、SendRequestWithHeaders、SendRequestWithRetry、ReadBody | ~130 |
| `stream_handler.go` | StreamRetryConfig、DefaultStreamRetryConfig、StreamLineProcessor、StreamTokens、ExecuteStreamWithRetry、StreamError、readStreamWithFirstByteTimeout、detectStreamError | ~260 |
| `stream_parser.go` | parseStreamResponse、parseContentFromChunk、parseToolCallsFromChunk、buildParsedResult、maxBodySize、sanitizeBody、truncateBody | ~160 |
| `request_log.go` | LogRequest、SaveRequestLog、CreateRequestLog、GetProviderByModel、GetAllProviders、PrepareRequestBody、ExtractUsage | ~65 |

**文件**：`internal/handler/base_handler.go` → 拆分为上述 6 个文件

### R3 — delta 解析提取

**问题**：choices[0].delta 的 content/reasoning_content/tool_calls 解析在 3 个 handler 中重复。

**方案**：
- 新建 `internal/converter/extract_delta.go`，提供：
  ```go
  type DeltaResult struct {
      Content          string
      ReasoningContent string
      ToolCallsDelta   []interface{}
  }
  func ExtractDeltaFromChunk(chunk map[string]interface{}) DeltaResult
  ```
- 三个 handler 都调用该函数

**文件**：新建 `internal/converter/extract_delta.go`，修改 `proxy_handler.go`、`ollama_handler.go`、`anthropic_stream.go`

### R5 — VO 转换简化

**问题**：`RequestLogVO` 和 `RequestLogDetailVO` 有 10 个相同字段，转换函数几乎一致。

**方案**：
- `RequestLogDetailVO` 嵌入 `RequestLogVO`，只追加 `RequestBody`、`ResponseBody` 字段
- 提取 `baseRequestLogToVO(log model.RequestLog) RequestLogVO` 公共转换函数
- `requestLogToDetailVO` 调用 `baseRequestLogToVO` 后追加 detail 字段

**文件**：`vo.go`

### R6 — API Key 脱敏统一

**问题**：`strings.Contains(data.APIKey, "****")` 判断在 app.go 和 web_handler.go 两处重复。

**方案**：
- 在 `ProviderService` 中新增 `PreserveAPIKey(id uint, newKey string) (string, error)` 方法
- 逻辑：如果 newKey 包含 `****`，查询数据库获取原 key 替换；否则直接返回 newKey
- app.go 和 web_handler.go 都调用该方法

**文件**：`internal/service/provider_service.go`、`app.go`、`internal/handler/web_handler.go`

---

## 第三批：体验提升

### U1 — Provider 搜索过滤

**方案**：
- 在配置管理页面顶部添加搜索框，支持按名称/模型/别名模糊搜索
- 添加协议类型过滤标签（全部/OpenAI/Anthropic/Ollama）
- 搜索和过滤在前端完成（数据已在内存中）

**文件**：`frontend/src/pages/providers.js`、`frontend/src/pages/providers.html`

### U2 — 统计报表表格优化

**方案**：
- "最近请求"表格在窄窗口下隐藏次要列（Provider、Input、Output、Cached、Token/s），保留核心列（时间、模型、Total、耗时、状态、操作）
- 使用 CSS `@media` + 在 th/td 上添加 `class="hidden-cell"` 控制显隐
- 点击行可展开查看完整信息（复用已有的 showLogDetail）

**文件**：`frontend/src/pages/stats.html`、`frontend/src/pages/stats.js`、`frontend/src/style.css`

### U3 — 实时监控自动刷新

**方案**：
- 页面激活时每 5 秒自动刷新活跃请求列表
- 离开页面时停止轮询（在 destroy 中 clearInterval）
- 页面标题栏添加暂停/继续按钮

**文件**：`frontend/src/pages/realtime.js`、`frontend/src/pages/realtime.html`

### U4 — API Key 遮罩

**方案**：
- 编辑弹窗中 API Key 输入框默认 type="password"
- 输入框右侧添加眼睛图标按钮，点击切换 type="text"/"password"
- 新增 Provider 时输入框为 type="text"（方便输入）

**文件**：`frontend/src/pages/providers.js`、`frontend/src/pages/providers.html`

### U5 — 日志暂停/继续

**方案**：
- 日志页面控制栏添加暂停/继续按钮
- 暂停时停止 setInterval 轮询
- 继续时先调用 GetNewLogs 补齐暂停期间的日志，再恢复轮询
- 暂停期间新日志到达时在按钮旁显示未读数量提示

**文件**：`frontend/src/pages/logs.js`、`frontend/src/pages/logs.html`

---

## 实施顺序

1. 第一批（关键修复）：L1 → L3 → R1 → R4
2. 第二批（结构优化）：R2 → R3 → R5 → R6
3. 第三批（体验提升）：U1 → U2 → U3 → U4 → U5

每批完成后构建验证，确保无编译错误和功能回归。
