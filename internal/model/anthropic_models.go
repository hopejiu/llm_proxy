package model

// AnthropicMessagesRequest Anthropic /v1/messages 请求体
type AnthropicMessagesRequest struct {
	Model         string                `json:"model"`
	Messages      []AnthropicMessage    `json:"messages"`
	System        interface{}           `json:"system,omitempty"`
	MaxTokens     int                   `json:"max_tokens"`
	Stream        bool                  `json:"stream,omitempty"`
	Temperature   *float64              `json:"temperature,omitempty"`
	TopP          *float64              `json:"top_p,omitempty"`
	TopK          *int                  `json:"top_k,omitempty"`
	StopSequences []string              `json:"stop_sequences,omitempty"`
	Tools         []AnthropicTool       `json:"tools,omitempty"`
	ToolChoice    interface{}           `json:"tool_choice,omitempty"`
	Metadata      interface{}           `json:"metadata,omitempty"`
	Thinking      *AnthropicThinking    `json:"thinking,omitempty"`
}

// AnthropicThinking 扩展思考配置
type AnthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// AnthropicMessage Anthropic 消息格式
type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// AnthropicContentBlock Anthropic 内容块（支持所有类型）
type AnthropicContentBlock struct {
	Type      string      `json:"type"`
	Text      string      `json:"text,omitempty"`
	ID        string      `json:"id,omitempty"`
	Name      string      `json:"name,omitempty"`
	Input     interface{} `json:"input,omitempty"`
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"` // tool_result 的 content，可以是 string 或 []AnthropicContentBlock
	IsError   *bool       `json:"is_error,omitempty"` // tool_result 的错误标记
}

// AnthropicTool Anthropic 工具定义
type AnthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"input_schema"`
}

// AnthropicMessagesResponse Anthropic /v1/messages 非流式响应
type AnthropicMessagesResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Content      []AnthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason"`
	StopSequence *string                 `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage          `json:"usage"`
}

// AnthropicUsage Anthropic 使用量
type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// AnthropicModelsResponse Anthropic /v1/models 响应
type AnthropicModelsResponse struct {
	Data    []AnthropicModelInfo `json:"data"`
	HasMore bool                 `json:"has_more"`
	FirstID string               `json:"first_id,omitempty"`
	LastID  string               `json:"last_id,omitempty"`
}

// AnthropicModelInfo Anthropic 模型信息
type AnthropicModelInfo struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}