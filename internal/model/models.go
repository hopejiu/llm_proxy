package model

import (
	"strings"
	"time"
)

// DeletedProviderID 当 Provider 被删除时，关联日志的 ProviderID 置为此值
const DeletedProviderID uint = 999

// ProviderConfig 第三方LLM服务商配置
type ProviderConfig struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:100;not null"`
	AutoSuffix  bool      `json:"auto_suffix" gorm:"default:false"`
	UrlSuffix   string    `json:"url_suffix" gorm:"size:200;default:''"`
	BaseURL     string    `json:"base_url" gorm:"size:500;not null"`
	APIKey      string    `json:"api_key" gorm:"size:500;not null"`
	Model       string    `json:"model" gorm:"size:100"`
	Alias       string    `json:"alias" gorm:"size:200"`
	ExtraParams string    `json:"extra_params" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MaskAPIKey 返回脱敏后的 API Key，只显示前4位和后4位
func (p *ProviderConfig) MaskAPIKey() string {
	key := p.APIKey
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// GetDisplayName 获取用于模型列表显示的名称，优先别名
func (p *ProviderConfig) GetDisplayName() string {
	if p.Alias != "" {
		return strings.TrimSpace(strings.Split(p.Alias, ",")[0])
	}
	return p.Model
}

// GetModelNames 获取所有可用于请求的模型名称列表（别名 + 模型名）
func (p *ProviderConfig) GetModelNames() []string {
	var names []string
	if p.Alias != "" {
		for _, a := range strings.Split(p.Alias, ",") {
			if trimmed := strings.TrimSpace(a); trimmed != "" {
				names = append(names, trimmed)
			}
		}
	}
	names = append(names, p.Model)
	return names
}

// GetRequestURL 根据 AutoSuffix 设置返回实际请求 URL
func (p *ProviderConfig) GetRequestURL() string {
	if !p.AutoSuffix {
		return p.BaseURL
	}
	baseURL := strings.TrimRight(p.BaseURL, "/")
	suffix := p.UrlSuffix
	if suffix != "" && !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	return baseURL + suffix
}

// RequestLog 请求日志记录
type RequestLog struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	ProviderID      uint           `json:"provider_id" gorm:"index:idx_provider_id"`
	Provider        ProviderConfig `json:"provider" gorm:"-"` // 不再使用外键关联，手动处理
	Model           string         `json:"model" gorm:"size:100"`
	InputTokens     int            `json:"input_tokens"`
	OutputTokens    int            `json:"output_tokens"`
	TotalTokens     int            `json:"total_tokens"`
	CachedTokens    int            `json:"cached_tokens"`                // 缓存token数
	RequestBody     string         `json:"request_body" gorm:"type:longtext"`     // 完整请求JSON
	ResponseBody    string         `json:"response_body" gorm:"type:longtext"`    // 完整响应JSON
	ResponseContent string         `json:"response_content" gorm:"type:longtext"` // 解析后的可读响应内容（stream类型拼接后的完整内容）
	ThinkingContent string         `json:"thinking_content" gorm:"type:longtext"` // 推理/thinking内容
	Status          string         `json:"status" gorm:"size:20;index:idx_created_at_status"` // success/error
	ErrorMessage    string         `json:"error_message" gorm:"size:1000"`
	Duration        int64          `json:"duration"` // 请求耗时(毫秒)
	Aggregated      bool           `json:"aggregated" gorm:"default:false;index:idx_aggregated"` // 是否已汇总到hourly_stats
	CreatedAt       time.Time      `json:"created_at" gorm:"index:idx_created_at;index:idx_created_at_status"`
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

// HourlyStat 每小时汇总统计（仅统计成功请求）
type HourlyStat struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Hour          time.Time `json:"hour" gorm:"uniqueIndex;not null"` // 小时起始时间，唯一索引
	InputTokens   int64     `json:"input_tokens"`
	OutputTokens  int64     `json:"output_tokens"`
	TotalTokens   int64     `json:"total_tokens"`
	CachedTokens  int64     `json:"cached_tokens"`
	RequestCount  int64     `json:"request_count"`  // 成功请求数
	TotalDuration int64     `json:"total_duration"` // 总耗时(ms)，用于计算平均耗时
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// HourlyStatsResult 分时统计结果
type HourlyStatsResult struct {
	Hour         int   `json:"hour"`
	RequestCount int64 `json:"request_count"`
	TotalTokens  int64 `json:"total_tokens"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	CachedTokens int64 `json:"cached_tokens"`
}
