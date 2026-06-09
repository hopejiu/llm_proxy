# ADR-0005: 基于文本标记的会话追踪

需要为代理网关增加会话（Session）追踪能力，使第三方能够识别连续对话中的请求属于同一会话，从而支持按会话维度统计请求次数、token 用量和成本。

## 方案对比

| 方案 | 描述 | 评估 |
|---|---|---|
| **HTTP Header 透传** ✅（非最终选择） | 客户端在请求头传 `X-Session-ID`，网关响应头回传 | 不依赖请求体，但要求客户端支持自定义 Header，且在 SDK/客户端中不可见 |
| **请求体自定义字段** | body 中加 `session_id` 字段，转发前移除 | 与上游 API 规范不兼容，需要在每个协议的 parser/forwarder 中处理 |
| **Assistant content 文本标记** ✅ **（选定）** | `content` 末尾加 `[SESSION]ID[/SESSION]`，第三方通过 messages 回传 | 零侵入协议，完全兼容所有 API 格式，客户端 SDK 天然支持（messages 透传） |

选定方案：**文本标记（Text Mark）**。

## 决策

### 机制

1. **会话 ID 生成**：自增数字（uint），记录在独立表 `chat_sessions` 的 `id` 主键中。每请求自动 INSERT 获得 ID。

2. **标记格式**：在 assistant 响应的 `content` 末尾追加 `[SESSION]ID[/SESSION]`。

3. **识别逻辑**：
   - 新请求到来时，扫描 messages 数组。匹配第一条 role=assistant 的消息，在 content 中提取 `[SESSION]数字[/SESSION]`。
   - 成功提取 → 关联到已有会话；未提取到 → 自动创建新会话。

4. **仅首次注入**：会话标记只在「新建会话」（首次请求，没有标记）时注入。后续请求命中已有标记时，response 中不再重复注入，避免累积出 `[SESSION]1[/SESSION][SESSION]1[/SESSION]`。

5. **转发前清理**：请求中如果检测到 `[SESSION]` 标记，转发前从 content 中移除，避免污染上游 API 请求。

6. **流式处理**：仅在 final chunk（`finish_reason: "stop"`）的 content 后追加一次标记，不污染中间 chunk。

### 协议一致性

三种协议行为一致：
- **OpenAI**：`choices[0].message.content` 追加/提取/清理
  - 兼容 `content` 为 null 的场景（如仅有 `reasoning_content` 或 `tool_calls` 时）：强行创建 `content` 字段放入 `[SESSION]ID[/SESSION]`
- **Anthropic**：`content[0].text` 追加/提取/清理
  - 兼容 `text` 不存在（如 `tool_use` 类型块）的场景：强行创建 `text` 字段
- **Ollama**：`message.content` 追加/提取/清理
  - 兼容 `content` 为 null/空的场景：强行创建 `content` 字段

### 数据库

**`chat_sessions` 新增表**：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | uint (PK, auto) | 自增主键，即会话 ID |
| `models` | text (JSON) | 该会话使用过的模型列表（去重） |
| `request_count` | int64 | 累计请求次数 |
| `total_tokens` | int64 | 累计总 token |
| `total_cost` | float64 | 累计总成本 |
| `created_at` | datetime | 首次请求时间 |
| `updated_at` | datetime | 最后请求时间 |

**`request_logs` 表改动**：
- 新增 `session_id` 列（uint, nullable）
- 增加索引 `idx_session_id`

### 统计更新策略

每次请求结束后，按 `session_id` 重新聚合 `request_logs`：
- `request_count` = COUNT
- `total_tokens` = SUM(total_tokens)
- `total_cost` = SUM(各请求 token × 对应模型单价)
- `models` = 去重后的模型名列表

成本按 `ModelEntry.InputPrice/CachePrice/OutputPrice` 计算：
```
cost = (input_tokens / 1_000_000) * input_price
     + (output_tokens / 1_000_000) * output_price
     + (cached_tokens / 1_000_000) * cache_price
```

### 改动影响

- **model 层**：新增 `ChatSession` GORM 模型
- **repository 层**：新增 `ChatSessionRepository`，`RequestLogRepository` 加 SessionID 更新方法
- **handler 层**：`HandleProxyRequest` 模板方法体新增 3 个 Hook：
  1. 解析 messages 提取 session_id（pre-process）
  2. 转发前移除 content 中的标记
  3. 响应后注入标记 + 更新会话统计
- **main.go**：新增 Repository 创建 + migrate 注册 `&model.ChatSession{}`
- **前端**：可视化会话数据的面板

## 决策日期

2026-06-09
