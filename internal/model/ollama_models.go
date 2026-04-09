package model

// OllamaChatRequest Ollama /api/chat 请求格式
type OllamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
	Tools    []OllamaTool    `json:"tools,omitempty"`
	Format   interface{}     `json:"format,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
	Think    interface{}     `json:"think,omitempty"`
}

// OllamaMessage Ollama 消息格式
type OllamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	Images    []string         `json:"images,omitempty"`
	ToolCalls []OllamaToolCall `json:"tool_calls,omitempty"`
}

// OllamaTool Ollama 工具定义
type OllamaTool struct {
	Type     string                 `json:"type"`
	Function OllamaToolFunction     `json:"function"`
}

// OllamaToolFunction Ollama 工具函数
type OllamaToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// OllamaToolCall Ollama 工具调用
type OllamaToolCall struct {
	Function OllamaToolCallFunction `json:"function"`
}

// OllamaToolCallFunction Ollama 工具调用函数
type OllamaToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OllamaChatResponse Ollama /api/chat 响应格式
type OllamaChatResponse struct {
	Model              string         `json:"model"`
	CreatedAt          string         `json:"created_at"`
	Message            OllamaMessage  `json:"message"`
	Done               bool           `json:"done"`
	DoneReason         string         `json:"done_reason,omitempty"`
	TotalDuration      int64          `json:"total_duration,omitempty"`
	LoadDuration       int64          `json:"load_duration,omitempty"`
	PromptEvalCount    int            `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64          `json:"prompt_eval_duration,omitempty"`
	EvalCount          int            `json:"eval_count,omitempty"`
	EvalDuration       int64          `json:"eval_duration,omitempty"`
}

// OllamaTagsResponse Ollama /api/tags 响应格式
type OllamaTagsResponse struct {
	Models []OllamaModelInfo `json:"models"`
}

// OllamaModelInfo Ollama 模型信息
type OllamaModelInfo struct {
	Name       string            `json:"name"`
	Model      string            `json:"model"`
	ModifiedAt string            `json:"modified_at"`
	Size       int64             `json:"size"`
	Digest     string            `json:"digest"`
	Details    OllamaModelDetails `json:"details,omitempty"`
}

// OllamaModelDetails Ollama 模型详情
type OllamaModelDetails struct {
	Format            string   `json:"format,omitempty"`
	Family            string   `json:"family,omitempty"`
	Families          []string `json:"families,omitempty"`
	ParameterSize     string   `json:"parameter_size,omitempty"`
	QuantizationLevel string   `json:"quantization_level,omitempty"`
}
