# 代码质量与体验优化 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 分三批修复并发安全 bug、消除重复代码、拆分大文件、优化用户体验

**架构：** 第一批修复关键 bug（L1/L3/R1/R4），第二批重构代码结构（R2/R3/R5/R6），第三批提升前端体验（U1-U5）

**技术栈：** Go 1.x / Wails v2 / Gin / GORM / 原生 JS + Tailwind CSS

---

## 文件结构

### 第一批修改文件

| 文件 | 操作 | 职责 |
|------|------|------|
| `internal/service/proxy_service.go` | 修改 | L1: 缓存深拷贝 + GetProviderByModel 返回值 |
| `internal/handler/base_handler.go` | 修改 | L3: ExecuteStreamWithRetry body 重试安全 |
| `internal/service/codebuddy_service.go` | 新建 | R1: SetupCodeBuddy 核心逻辑 |
| `app.go` | 修改 | R1: 调用 codebuddy_service |
| `internal/handler/web_handler.go` | 修改 | R1: 调用 codebuddy_service + 修复硬编码端口 |
| `app_error.go` | 修改 | R4: 错误码常量重命名 |
| `internal/handler/response.go` | 修改 | R4: 错误码常量重命名为 CodeXxx |

### 第二批修改文件

| 文件 | 操作 | 职责 |
|------|------|------|
| `internal/handler/base_handler.go` | 拆分 | R2: 拆为 6 个文件 |
| `internal/handler/request_context.go` | 新建 | R2: 请求上下文相关 |
| `internal/handler/http_client.go` | 新建 | R2: HTTP 请求发送 |
| `internal/handler/stream_handler.go` | 新建 | R2: 流式处理与重试 |
| `internal/handler/stream_parser.go` | 新建 | R2: SSE 解析与 body 清洗 |
| `internal/handler/request_log.go` | 新建 | R2: 请求日志相关 |
| `internal/converter/extract_delta.go` | 新建 | R3: delta 解析公共函数 |
| `internal/handler/proxy_handler.go` | 修改 | R3: 调用 ExtractDeltaFromChunk |
| `internal/handler/ollama_handler.go` | 修改 | R3: 调用 ExtractDeltaFromChunk |
| `internal/converter/anthropic_stream.go` | 修改 | R3: 调用 ExtractDeltaFromChunk |
| `vo.go` | 修改 | R5: RequestLogDetailVO 嵌入 RequestLogVO |
| `internal/service/provider_service.go` | 修改 | R6: 新增 PreserveAPIKey 方法 |
| `app.go` | 修改 | R6: 调用 PreserveAPIKey |
| `internal/handler/web_handler.go` | 修改 | R6: 调用 PreserveAPIKey |

### 第三批修改文件

| 文件 | 操作 | 职责 |
|------|------|------|
| `frontend/src/pages/providers.html` | 修改 | U1: 搜索框 + 过滤标签 |
| `frontend/src/pages/providers.js` | 修改 | U1: 搜索过滤逻辑 + U4: API Key 遮罩 |
| `frontend/src/pages/stats.html` | 修改 | U2: 表格响应式隐藏列 |
| `frontend/src/pages/stats.js` | 修改 | U2: 行点击展开详情 |
| `frontend/src/style.css` | 修改 | U2: 响应式隐藏列样式 |
| `frontend/src/pages/realtime.html` | 修改 | U3: 自动刷新开关 |
| `frontend/src/pages/realtime.js` | 修改 | U3: 自动刷新逻辑 |
| `frontend/src/pages/logs.html` | 修改 | U5: 暂停/继续按钮 |
| `frontend/src/pages/logs.js` | 修改 | U5: 暂停/继续逻辑 |

---

## 第一批：关键修复

### 任务 1：L1 — Provider 缓存并发安全

**文件：** `internal/service/proxy_service.go`

- [ ] **步骤 1：修改 getAllProvidersCached 返回深拷贝**

将第 38-39 行：
```go
providers := s.providerCache
s.cacheMu.RUnlock()
return providers, nil
```
改为：
```go
providers := make([]model.ProviderConfig, len(s.providerCache))
copy(providers, s.providerCache)
s.cacheMu.RUnlock()
return providers, nil
```

同样修改第 48 行的双重检查返回：
```go
return s.providerCache, nil
```
改为：
```go
result := make([]model.ProviderConfig, len(s.providerCache))
copy(result, s.providerCache)
return result, nil
```

- [ ] **步骤 2：修改 GetProviderByModel 返回值而非指针**

将函数签名从 `(*model.ProviderConfig, error)` 改为 `(model.ProviderConfig, error)`。

将第 87-93 行的返回值：
```go
return &providers[i], nil
```
改为：
```go
return providers[i], nil
```

两处都改（别名匹配和模型名匹配）。

- [ ] **步骤 3：更新所有 GetProviderByModel 调用方**

搜索所有调用 `GetProviderByModel` 的地方，将 `provider` 从指针改为值类型。主要在：
- `proxy_handler.go` 的 `handleStream` 和 `handleNormal`
- `anthropic_handler.go` 的 `handleStream` 和 `handleNormal`
- `ollama_handler.go` 的 `handleStream` 和 `handleNormal`

将 `provider.Xxx` 调用保持不变（值类型也可以访问字段），但 `provider.GetRequestURL()` 等方法调用需要确认接收者是值接收者还是指针接收者。如果是指针接收者，需要改为值接收者或将返回值取地址。

- [ ] **步骤 4：运行 go vet 验证**

```bash
cd d:\teamsun_project\teamsun\code\other\llm_statistic && go vet ./...
```

---

### 任务 2：L3 — Stream retry body 消费安全

**文件：** `internal/handler/base_handler.go`

- [ ] **步骤 1：确认 body 在重试中的使用方式**

当前 `ExecuteStreamWithRetry` 的 `body []byte` 参数是字节切片，每次调用 `h.SendStreamRequest(ctx, url, body, apiKey)` 时，`SendStreamRequest` 内部会 `io.NopCloser(bytes.NewReader(body))` 创建新 reader。由于 `body` 是 `[]byte`，`bytes.NewReader` 每次从原始字节创建新 reader，**body 不会被消费**。

但需要确认 `SendStreamRequest` 和 `SendStreamRequestWithHeaders` 的实现是否确实每次创建新 reader。如果是，则 L3 实际上不是 bug，只需添加注释说明安全性。

- [ ] **步骤 2：如果 body 确实安全，添加注释说明**

在 `ExecuteStreamWithRetry` 的 `body []byte` 参数旁添加注释：
```go
body []byte, // 原始请求体字节，每次重试会创建新的 Reader，不会被消费
```

- [ ] **步骤 3：如果 body 不安全（SendStreamRequest 修改了 body），修复**

在 `ExecuteStreamWithRetry` 开头保存 body 副本：
```go
originalBody := make([]byte, len(body))
copy(originalBody, body)
```
重试时使用 `originalBody`。

- [ ] **步骤 4：运行 go vet 验证**

---

### 任务 3：R1 — SetupCodeBuddy 统一

**文件：** 新建 `internal/service/codebuddy_service.go`，修改 `app.go`、`internal/handler/web_handler.go`

- [ ] **步骤 1：创建 codebuddy_service.go**

```go
package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CodeBuddyResult SetupCodeBuddy 的返回结果
type CodeBuddyResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	TargetURL    string `json:"targetUrl"`
	ModelName    string `json:"modelName"`
	ProviderName string `json:"providerName"`
}

// SetupCodeBuddy 配置 CodeBuddy 的 models.json
func SetupCodeBuddy(dataDir, targetURL string) (*CodeBuddyResult, error) {
	homeDir, _ := os.UserHomeDir()
	codebuddyDir := filepath.Join(homeDir, ".codebuddy")
	modelsFilePath := filepath.Join(codebuddyDir, "models.json")
	os.MkdirAll(codebuddyDir, 0755)

	modelName := "llm-proxy"
	providerName := "LLM Proxy"

	var models []map[string]interface{}
	if data, err := os.ReadFile(modelsFilePath); err == nil {
		json.Unmarshal(data, &models)
	}

	// 检查是否已存在相同 URL 的配置
	for _, m := range models {
		if baseURL, ok := m["baseUrl"].(string); ok && baseURL == targetURL {
			return &CodeBuddyResult{
				Success:      true,
				Message:      "LLM Proxy 已配置过，无需重复配置",
				TargetURL:    targetURL,
				ModelName:    modelName,
				ProviderName: providerName,
			}, nil
		}
	}

	// 追加新配置
	newModel := map[string]interface{}{
		"baseUrl":      targetURL,
		"modelName":    modelName,
		"providerName": providerName,
		"apiKey":       "sk-placeholder",
	}
	models = append(models, newModel)

	data, err := json.MarshalIndent(models, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.WriteFile(modelsFilePath, data, 0644); err != nil {
		return nil, fmt.Errorf("写入配置文件失败: %w", err)
	}

	return &CodeBuddyResult{
		Success:      true,
		Message:      "配置成功！请在 CodeBuddy 中选择 LLM Proxy 模型",
		TargetURL:    targetURL,
		ModelName:    modelName,
		ProviderName: providerName,
	}, nil
}
```

注意：以上代码需要从 app.go 的 SetupCodeBuddy 实现中提取完整逻辑，包括所有边界处理。实现时需读取 app.go 第 198-271 行的完整代码来确保逻辑一致。

- [ ] **步骤 2：修改 app.go 调用 codebuddy_service**

将 app.go 的 `SetupCodeBuddy` 方法改为调用 service：
```go
func (a *App) SetupCodeBuddy() (CodeBuddyResultVO, error) {
	targetURL := fmt.Sprintf("http://localhost:%s/v1", a.cfg.ProxyPort)
	result, err := service.SetupCodeBuddy("", targetURL)
	if err != nil {
		return CodeBuddyResultVO{}, NewAppError("INTERNAL", err.Error())
	}
	return CodeBuddyResultVO{
		Success:      result.Success,
		Message:      result.Message,
		TargetURL:    result.TargetURL,
		ModelName:    result.ModelName,
		ProviderName: result.ProviderName,
	}, nil
}
```

- [ ] **步骤 3：修改 web_handler.go 调用 codebuddy_service**

将 web_handler.go 的 `SetupCodeBuddy` 方法改为调用 service，端口从配置读取：
```go
targetURL := fmt.Sprintf("http://localhost:%s/v1", h.cfg.ProxyPort)
result, err := service.SetupCodeBuddy("", targetURL)
```

- [ ] **步骤 4：运行 go vet 验证**

---

### 任务 4：R4 — 错误码统一

**文件：** `internal/handler/response.go`、`app_error.go`、所有引用错误码的 handler 文件

- [ ] **步骤 1：重命名 response.go 中的错误码常量**

将 `response.go` 中的：
```go
const (
	ErrBadRequest   = "BAD_REQUEST"
	ErrNotFound     = "NOT_FOUND"
	ErrInternal     = "INTERNAL_ERROR"
	ErrInvalidInput = "INVALID_INPUT"
)
```
改为：
```go
const (
	CodeBadRequest   = "BAD_REQUEST"
	CodeNotFound     = "NOT_FOUND"
	CodeInternal     = "INTERNAL_ERROR"
	CodeInvalidInput = "INVALID_INPUT"
)
```

- [ ] **步骤 2：更新所有引用这些常量的代码**

搜索所有使用 `handler.ErrBadRequest`、`handler.ErrNotFound`、`handler.ErrInternal`、`handler.ErrInvalidInput` 的地方，替换为 `handler.CodeBadRequest` 等。

- [ ] **步骤 3：运行 go vet 验证**

---

### 任务 5：第一批构建验证

- [ ] **步骤 1：运行 go vet**

```bash
cd d:\teamsun_project\teamsun\code\other\llm_statistic && go vet ./...
```

- [ ] **步骤 2：运行完整构建**

```bash
cd d:\teamsun_project\teamsun\code\other\llm_statistic && powershell -ExecutionPolicy Bypass -File build.ps1
```

---

## 第二批：结构优化

### 任务 6：R2 — base_handler.go 拆分

**文件：** 拆分 `internal/handler/base_handler.go` 为 6 个文件

- [ ] **步骤 1：创建 request_context.go**

从 base_handler.go 提取以下内容到 `internal/handler/request_context.go`：
- `requestIDKey` 变量
- `generateRequestID()` 函数
- `contextWithRequestID()` 函数
- `requestIDFromContext()` 函数
- `ProxyRequestInfo` 结构体
- `ParseRequestFunc` 类型
- `HandleRequestFunc` 类型
- `HandleProxyRequest()` 函数

- [ ] **步骤 2：创建 http_client.go**

从 base_handler.go 提取以下内容到 `internal/handler/http_client.go`：
- `buildHTTPRequest()` 函数
- `SendStreamRequest()` 函数
- `SendStreamRequestWithHeaders()` 函数
- `SendRequest()` 函数
- `SendRequestWithHeaders()` 函数
- `SendRequestWithRetry()` 函数
- `ReadBody()` 函数

- [ ] **步骤 3：创建 stream_handler.go**

从 base_handler.go 提取以下内容到 `internal/handler/stream_handler.go`：
- `StreamRetryConfig` 结构体
- `DefaultStreamRetryConfig()` 函数
- `StreamLineProcessor` 类型
- `StreamTokens` 结构体
- `ExecuteStreamWithRetry()` 函数
- `StreamError` 结构体
- `readStreamWithFirstByteTimeout()` 函数
- `detectStreamError()` 函数

- [ ] **步骤 4：创建 stream_parser.go**

从 base_handler.go 提取以下内容到 `internal/handler/stream_parser.go`：
- `maxBodySize` 变量
- `sanitizeBody()` 函数
- `truncateBody()` 函数
- `parseStreamResponse()` 函数
- `parseContentFromChunk()` 函数
- `parseToolCallsFromChunk()` 函数
- `buildParsedResult()` 函数

- [ ] **步骤 5：创建 request_log.go**

从 base_handler.go 提取以下内容到 `internal/handler/request_log.go`：
- `LogRequest()` 函数
- `SaveRequestLog()` 函数
- `CreateRequestLog()` 函数
- `GetProviderByModel()` 函数
- `GetAllProviders()` 函数
- `PrepareRequestBody()` 函数
- `ExtractUsage()` 函数

- [ ] **步骤 6：精简 base_handler.go**

只保留：
- `BaseHandler` 结构体定义
- `NewBaseHandler()` 构造函数
- `HandlerConfig` 结构体
- `UpstreamError` 结构体
- `ResolveUpstreamError()` 函数

- [ ] **步骤 7：运行 go vet 验证**

---

### 任务 7：R3 — delta 解析提取

**文件：** 新建 `internal/converter/extract_delta.go`，修改 `proxy_handler.go`、`ollama_handler.go`、`anthropic_stream.go`

- [ ] **步骤 1：创建 extract_delta.go**

```go
package converter

// DeltaResult 从 SSE chunk 的 delta 中提取的结果
type DeltaResult struct {
	Content          string
	ReasoningContent string
	ToolCallsDelta   []interface{}
}

// ExtractDeltaFromChunk 从 OpenAI 格式的 SSE chunk 中提取 delta 内容
func ExtractDeltaFromChunk(chunk map[string]interface{}) DeltaResult {
	var result DeltaResult
	choices, ok := chunk["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return result
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return result
	}
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return result
	}
	if content, ok := delta["content"].(string); ok {
		result.Content = content
	}
	if reasoning, ok := delta["reasoning_content"].(string); ok {
		result.ReasoningContent = reasoning
	}
	if toolCalls, ok := delta["tool_calls"].([]interface{}); ok {
		result.ToolCallsDelta = toolCalls
	}
	return result
}
```

- [ ] **步骤 2：修改 proxy_handler.go 使用 ExtractDeltaFromChunk**

替换流式处理中的 delta 解析代码，改为调用 `converter.ExtractDeltaFromChunk(chunk)`。

- [ ] **步骤 3：修改 ollama_handler.go 使用 ExtractDeltaFromChunk**

同上。

- [ ] **步骤 4：修改 anthropic_stream.go 使用 ExtractDeltaFromChunk**

Anthropic 格式不同（使用 `delta.text` 而非 `choices[0].delta.content`），需要评估是否适用。如果格式差异太大，Anthropic handler 可保持原有解析逻辑。

- [ ] **步骤 5：运行 go vet 验证**

---

### 任务 8：R5 — VO 转换简化

**文件：** `vo.go`

- [ ] **步骤 1：修改 RequestLogDetailVO 嵌入 RequestLogVO**

将 `RequestLogDetailVO` 的重复字段替换为嵌入 `RequestLogVO`，只保留 `RequestBody` 和 `ResponseBody` 字段。

- [ ] **步骤 2：提取 baseRequestLogToVO 公共函数**

```go
func baseRequestLogToVO(log model.RequestLog) RequestLogVO {
	return RequestLogVO{
		ID:            log.ID,
		ProviderName:  log.ProviderName,
		// ... 所有公共字段
	}
}
```

- [ ] **步骤 3：修改 requestLogToDetailVO 调用 baseRequestLogToVO**

```go
func requestLogToDetailVO(log model.RequestLog) RequestLogDetailVO {
	return RequestLogDetailVO{
		RequestLogVO: baseRequestLogToVO(log),
		RequestBody:  log.RequestBody,
		ResponseBody: log.ResponseBody,
	}
}
```

- [ ] **步骤 4：运行 go vet 验证**

---

### 任务 9：R6 — API Key 脱敏统一

**文件：** `internal/service/provider_service.go`、`app.go`、`internal/handler/web_handler.go`

- [ ] **步骤 1：在 ProviderService 中新增 PreserveAPIKey 方法**

```go
func (s *ProviderService) PreserveAPIKey(id uint, newKey string) (string, error) {
	if !strings.Contains(newKey, "****") {
		return newKey, nil
	}
	existing, err := s.providerRepo.GetByID(id)
	if err != nil {
		return "", err
	}
	return existing.APIKey, nil
}
```

- [ ] **步骤 2：修改 app.go 的 UpdateProvider 调用 PreserveAPIKey**

替换 `strings.Contains(data.APIKey, "****")` 判断为调用 `a.providerService.PreserveAPIKey(data.ID, data.APIKey)`。

- [ ] **步骤 3：修改 web_handler.go 的 UpdateProvider 调用 PreserveAPIKey**

同上。

- [ ] **步骤 4：运行 go vet 验证**

---

### 任务 10：第二批构建验证

- [ ] **步骤 1：运行 go vet**

- [ ] **步骤 2：运行完整构建**

---

## 第三批：体验提升

### 任务 11：U1 — Provider 搜索过滤

**文件：** `frontend/src/pages/providers.html`、`frontend/src/pages/providers.js`

- [ ] **步骤 1：在 providers.html 添加搜索框和过滤标签**

在页面标题栏下方添加搜索输入框和协议类型过滤标签（全部/OpenAI/Anthropic/Ollama）。

- [ ] **步骤 2：在 providers.js 添加搜索过滤逻辑**

- 添加 `filterProviders()` 函数，根据搜索关键词和协议类型过滤 Provider 列表
- 搜索框 `oninput` 事件触发过滤
- 过滤标签 `onclick` 事件切换类型
- 过滤在前端完成（数据已在内存中）

- [ ] **步骤 3：构建验证**

---

### 任务 12：U2 — 统计报表表格优化

**文件：** `frontend/src/pages/stats.html`、`frontend/src/pages/stats.js`、`frontend/src/style.css`

- [ ] **步骤 1：在 stats.html 的"最近请求"表格中为次要列添加 hidden-cell class**

将 Provider、Input、Output、Cached、Token/s 列的 `<th>` 和 `<td>` 添加 `class="hidden-cell"`。

- [ ] **步骤 2：在 style.css 添加响应式隐藏样式**

```css
@media (max-width: 1100px) {
  .hidden-cell {
    display: none;
  }
}
```

- [ ] **步骤 3：在 stats.js 添加行点击展开详情**

点击表格行时调用 `showLogDetail(log.id)` 查看完整信息。

- [ ] **步骤 4：构建验证**

---

### 任务 13：U3 — 实时监控自动刷新

**文件：** `frontend/src/pages/realtime.html`、`frontend/src/pages/realtime.js`

- [ ] **步骤 1：在 realtime.html 标题栏添加自动刷新开关**

添加与统计报表页面相同的 toggle 开关。

- [ ] **步骤 2：在 realtime.js 添加自动刷新逻辑**

- 页面激活时默认开启自动刷新（每 5 秒）
- 离开页面时在 `destroy()` 中 clearInterval
- toggle 开关控制开启/关闭

- [ ] **步骤 3：构建验证**

---

### 任务 14：U4 — API Key 遮罩

**文件：** `frontend/src/pages/providers.js`、`frontend/src/pages/providers.html`

- [ ] **步骤 1：修改编辑弹窗中 API Key 输入框**

- 编辑时 API Key 输入框默认 `type="password"`，值为脱敏后的 `sk-****last4`
- 输入框右侧添加眼睛图标按钮，点击切换 `type="text"/"password"`
- 新增 Provider 时输入框为 `type="text"`（方便输入新 key）

- [ ] **步骤 2：构建验证**

---

### 任务 15：U5 — 日志暂停/继续

**文件：** `frontend/src/pages/logs.html`、`frontend/src/pages/logs.js`

- [ ] **步骤 1：在 logs.html 控制栏添加暂停/继续按钮**

在日志查看器 header 的控制按钮区域添加暂停按钮。

- [ ] **步骤 2：在 logs.js 添加暂停/继续逻辑**

- 暂停时 clearInterval 停止轮询
- 继续时先调用 GetNewLogs 补齐暂停期间的日志，再恢复 setInterval
- 暂停期间新日志到达时在按钮旁显示未读数量提示（可选，通过计数器实现）

- [ ] **步骤 3：构建验证**

---

### 任务 16：最终构建验证

- [ ] **步骤 1：运行 go vet**

- [ ] **步骤 2：运行完整构建**

- [ ] **步骤 3：手动测试关键功能**
