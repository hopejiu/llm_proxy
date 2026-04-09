package model

import (
	"time"
)

// ProviderConfig 第三方LLM服务商配置
type ProviderConfig struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:100;not null"`
	BaseURL     string    `json:"base_url" gorm:"size:500;not null"`
	APIKey      string    `json:"api_key" gorm:"size:500;not null"`
	Model       string    `json:"model" gorm:"size:100"`                      // 模型名称
	ExtraParams string    `json:"extra_params" gorm:"type:text"`              // 自定义请求参数JSON
	IsActive    bool      `json:"is_active" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RequestLog 请求日志记录
type RequestLog struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	ProviderID      uint      `json:"provider_id"`
	Provider        ProviderConfig `json:"provider" gorm:"foreignKey:ProviderID"`
	Model           string    `json:"model" gorm:"size:100"`
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	TotalTokens     int       `json:"total_tokens"`
	CachedTokens    int       `json:"cached_tokens"`                         // 缓存token数
	RequestBody     string    `json:"request_body" gorm:"type:longtext"`     // 完整请求JSON
	ResponseBody    string    `json:"response_body" gorm:"type:longtext"`    // 完整响应JSON
	ResponseContent string    `json:"response_content" gorm:"type:longtext"` // 解析后的可读响应内容（stream类型拼接后的完整内容）
	ThinkingContent string    `json:"thinking_content" gorm:"type:longtext"` // 推理/thinking内容
	Status          string    `json:"status" gorm:"size:20"`                 // success/error
	ErrorMessage    string    `json:"error_message" gorm:"size:1000"`
	Duration        int64     `json:"duration"` // 请求耗时(毫秒)
	CreatedAt       time.Time `json:"created_at"`
}

// TokenStats Token使用统计
type TokenStats struct {
	Date              string `json:"date" gorm:"column:date"`
	TotalInputTokens  int64  `json:"total_input_tokens" gorm:"column:total_input_tokens"`
	TotalOutputTokens int64  `json:"total_output_tokens" gorm:"column:total_output_tokens"`
	TotalTokens       int64  `json:"total_tokens" gorm:"column:total_tokens"`
	TotalCachedTokens int64  `json:"total_cached_tokens" gorm:"column:total_cached_tokens"`
	RequestCount      int64  `json:"request_count" gorm:"column:request_count"`
}

// OpenAIRequest OpenAI兼容请求格式
type OpenAIRequest struct {
	Model    string    `json:"model"`
	Stream   bool      `json:"stream,omitempty"`
}


// OpenAIResponse OpenAI兼容响应格式
type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}



type Usage struct {
	PromptTokens          int                       `json:"prompt_tokens"`
	CompletionTokens      int                       `json:"completion_tokens"`
	TotalTokens           int                       `json:"total_tokens"`
	PromptTokensDetails   *PromptTokensDetails      `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

// PromptTokensDetails prompt token详情
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

// CompletionTokensDetails completion token详情
type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}
