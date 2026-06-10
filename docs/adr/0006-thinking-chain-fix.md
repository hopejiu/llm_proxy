# ADR-0006: 自动修复思维链缺失

DeepSeek 等模型的思维链（thinking）模式下，若发生了工具调用（tool_calls），后续所有请求必须回传 assistant 消息中的 `reasoning_content`，否则 API 返回 400。但多数客户端（如 Cursor、Cline）不会正确回传 `reasoning_content`，导致会话中断。

## 决策

### 方案

代理服务在 Provider 级别提供 `AutoFixThinking` 开关，开启后自动在转发请求前为 messages 中的 assistant 消息补全 `reasoning_content`。

### 核心规则

1. **代理完全接管**：只要 Provider 开启此功能，所有 assistant 消息的 `reasoning_content` 一律由代理从 request_logs 存储中填充，客户端传什么忽略什么。

2. **实时聚合**：每次请求转发前，从 `request_logs` 按 `session_id` 查询历史记录的 `thinking_content`，不额外缓存。单次会话最多 500 条消息，性能可接受。

3. **匹配策略**：
   - 构建有序思维链队列（按 `created_at` 排序），每条包含 `response_content`（纯文本）和 `thinking_content`
   - 游标（cursor）每次新请求从 0 开始
   - 按 content 完全匹配 + 消费标记（每条只匹配一次）
   - 空 content 时按顺序兜底消费
   - 匹配不到则跳过

4. **仅 OpenAI 协议**：只在 OpenAI 协议路径（`ProxyHandler`）中执行，删除 Anthropic/Ollama handler。

5. **执行位置**：在 `PrepareRequestBody` 之后、转发之前，在 handler 层执行。

### 存储统一

- `ResponseContent` 字段：始终存储纯文本（非 JSON），便于匹配
- `ThinkingContent` 字段：始终存储纯文本 reasoning 内容
- 流式响应需修复提取逻辑，分别写入两个字段

### Provider 模型变更

`ProviderConfig` 新增字段：

| 字段 | 类型 | GORM | 默认值 |
|---|---|---|---|
| `auto_fix_thinking` | bool | `default:false` | false |

### 前端变更

ProvidersPage.tsx 在"启用扩展参数"开关下方增加"自动修复思维链缺失"开关。

### 删除范围

- `internal/handler/anthropic_handler.go`
- `internal/handler/ollama_handler.go`
- `internal/model/anthropic_models.go`
- `internal/model/ollama_models.go`
- `internal/converter/anthropic_converter.go`
- `internal/converter/ollama_converter.go`
- Router 中的 Anthropic/Ollama 路由注册
- `AppService` 中对 Anthropic/Ollama handler 的引用

## 决策日期

2026-06-10
