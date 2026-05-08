package converter

// OpenAI 兼容 API 类型定义

// OpenAIChatRequest OpenAI Chat Completions 请求
type OpenAIChatRequest struct {
	Model       string             `json:"model"`
	Messages    []OpenAIMessage    `json:"messages"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
	Temperature *float64           `json:"temperature,omitempty"`
	TopP        *float64           `json:"top_p,omitempty"`
	Stop        []string           `json:"stop,omitempty"`
	Tools       []OpenAITool       `json:"tools,omitempty"`
	ToolChoice  interface{}        `json:"tool_choice,omitempty"`
	Thinking    interface{}        `json:"thinking,omitempty"`
}

// OpenAIMessage OpenAI 消息
type OpenAIMessage struct {
	Role      string          `json:"role"`
	Content   interface{}     `json:"content"`
	ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"`
}

// OpenAIToolCall OpenAI tool call
type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

// OpenAIFunctionCall OpenAI function call
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAITool OpenAI tool 定义
type OpenAITool struct {
	Type     string       `json:"type"`
	Function OpenAIFuncDef `json:"function"`
}

// OpenAIFuncDef OpenAI function 定义
type OpenAIFuncDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// OpenAISimpleRequest OpenAI 请求（仅解析 model 和 stream）
type OpenAISimpleRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream,omitempty"`
}

// OpenAIResponse OpenAI 兼容响应格式
type OpenAIResponse struct {
	Choices []OpenAIRespChoice `json:"choices"`
	Usage   OpenAIRespUsage    `json:"usage"`
}

// OpenAIRespChoice OpenAI 响应选择
type OpenAIRespChoice struct {
	Index        int              `json:"index"`
	Message      OpenAIRespMessage `json:"message"`
	FinishReason string           `json:"finish_reason"`
}

// OpenAIRespMessage OpenAI 响应消息
type OpenAIRespMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIRespUsage OpenAI token 用量
type OpenAIRespUsage struct {
	PromptTokens        int                    `json:"prompt_tokens"`
	CompletionTokens    int                    `json:"completion_tokens"`
	TotalTokens         int                    `json:"total_tokens"`
	PromptTokensDetails *OpenAIPromptTokensDetails `json:"prompt_tokens_details,omitempty"`
}

// OpenAIPromptTokensDetails prompt token 详情
type OpenAIPromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}
