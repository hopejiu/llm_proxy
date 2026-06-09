package model

import (
	"encoding/json"
	"strings"
	"time"
)

// DeletedProviderID 当 Provider 被删除时，关联日志的 ProviderID 置为此值
const DeletedProviderID uint = 999

// ModelEntry 单个模型配置（存储在 ProviderConfig.Models JSON 中）
type ModelEntry struct {
	Name        string   `json:"name"`         // 发送给上游的模型名，如 "gpt-4"
	Aliases     []string `json:"aliases"`      // 客户端可用的模型名列表，如 ["my-gpt4", "gpt4-custom"]
	ExtraParams string   `json:"extra_params"` // 该模型独有扩展参数 (JSON)
	InputPrice  float64  `json:"input_price"`  // 输入价格（元/百万 token），0=未设置
	OutputPrice float64  `json:"output_price"` // 输出价格（元/百万 token），0=未设置
	CachePrice  float64  `json:"cache_price"`  // 缓存命中价格（元/百万 token），0=未设置
}

// ProviderConfig 第三方LLM服务商配置
type ProviderConfig struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Name       string    `json:"name" gorm:"size:100;not null"`
	AutoSuffix bool      `json:"auto_suffix" gorm:"default:false"`
	UrlSuffix  string    `json:"url_suffix" gorm:"size:200;default:''"`
	BaseURL    string    `json:"base_url" gorm:"size:500;not null"`
	APIKey     string    `json:"api_key" gorm:"size:500;not null"`
	Models     string    `json:"models" gorm:"type:text"` // JSON 数组 [ModelEntry, ...]
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// MaskAPIKey 返回脱敏后的 API Key，只显示前4位和后4位
func (p *ProviderConfig) MaskAPIKey() string {
	key := p.APIKey
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// ParseModels 解析 Models JSON 为 ModelEntry 列表
func (p *ProviderConfig) ParseModels() []ModelEntry {
	if p.Models == "" {
		return nil
	}
	var entries []ModelEntry
	if err := json.Unmarshal([]byte(p.Models), &entries); err != nil {
		return nil
	}
	return entries
}

// FindModelEntry 根据客户端模型名查找匹配的 ModelEntry
// 先匹配 aliases，再匹配 name
func (p *ProviderConfig) FindModelEntry(clientName string) *ModelEntry {
	entries := p.ParseModels()
	for i := range entries {
		for _, alias := range entries[i].Aliases {
			if alias == clientName {
				return &entries[i]
			}
		}
	}
	for i := range entries {
		if entries[i].Name == clientName {
			return &entries[i]
		}
	}
	// 兼容旧数据：如果 Models 为空但有 model/alias 旧字段（migration 前），返回 nil
	if len(entries) > 0 {
		return &entries[0] // 兜底返回第一个
	}
	return nil
}

// ResolveModel 将客户端模型名解析为上游模型名
func (p *ProviderConfig) ResolveModel(clientName string) string {
	if entry := p.FindModelEntry(clientName); entry != nil {
		return entry.Name
	}
	return ""
}

// GetModelExtraParams 获取匹配到的 ModelEntry 的扩展参数
func (p *ProviderConfig) GetModelExtraParams(clientName string) string {
	if entry := p.FindModelEntry(clientName); entry != nil {
		return entry.ExtraParams
	}
	return ""
}

// GetDisplayName 获取用于模型列表显示的名称（第一个 entry 的第一个 alias，兜底第一个 entry 的 name）
func (p *ProviderConfig) GetDisplayName() string {
	entries := p.ParseModels()
	if len(entries) == 0 {
		return ""
	}
	if len(entries[0].Aliases) > 0 {
		return entries[0].Aliases[0]
	}
	return entries[0].Name
}

// GetModelNames 获取所有可用于请求的模型名称列表（所有 entry 的 name + aliases）
func (p *ProviderConfig) GetModelNames() []string {
	entries := p.ParseModels()
	seen := make(map[string]bool)
	var names []string
	for _, e := range entries {
		for _, alias := range e.Aliases {
			if alias != "" && !seen[alias] {
				names = append(names, alias)
				seen[alias] = true
			}
		}
		if e.Name != "" && !seen[e.Name] {
			names = append(names, e.Name)
			seen[e.Name] = true
		}
	}
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
	CachedTokens    int            `json:"cached_tokens"`                                           // 缓存token数
	RequestBody     string         `json:"request_body" gorm:"type:longtext"`                       // 完整请求JSON
	ResponseBody    string         `json:"response_body" gorm:"type:longtext"`                      // 完整响应JSON
	ResponseContent string         `json:"response_content" gorm:"type:longtext"`                   // 解析后的可读响应内容（stream类型拼接后的完整内容）
	ThinkingContent string         `json:"thinking_content" gorm:"type:longtext"`                   // 推理/thinking内容
	Status          string         `json:"status" gorm:"size:20;index:idx_created_at_status"`       // success/error
	ErrorMessage    string         `json:"error_message" gorm:"size:1000"`
	Duration        int64          `json:"duration"`                                                   // 请求耗时(毫秒)
	Aggregated      bool           `json:"aggregated" gorm:"default:false;index:idx_aggregated"`       // 是否已汇总到hourly_stats
	SessionID       *uint          `json:"session_id" gorm:"index:idx_session_id;default:null"`        // 所属会话ID（为NULL表示无会话）
	CreatedAt       time.Time      `json:"created_at" gorm:"index:idx_created_at;index:idx_created_at_status"`
}

// ChatSession 会话统计
type ChatSession struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Models       string    `json:"models" gorm:"type:text"`                    // JSON数组，该会话使用过的模型名（去重）
	RequestCount int64     `json:"request_count" gorm:"default:0"`             // 累计请求次数
	TotalTokens  int64     `json:"total_tokens" gorm:"default:0"`              // 累计总token
	TotalCost    float64   `json:"total_cost" gorm:"default:0"`                // 累计总成本（元）
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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
// provider_id=0, model="" 表示全量合计
// provider_id>0, model="" 表示该 provider 所有模型合计
// provider_id>0, model="xxx" 表示该 provider 下具体模型的合计
type HourlyStat struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Hour          time.Time `json:"hour" gorm:"uniqueIndex:idx_hour_provider_model;not null"`                  // 小时起始时间
	ProviderID    uint      `json:"provider_id" gorm:"uniqueIndex:idx_hour_provider_model;not null;default:0"` // 0=全量, >0=具体Provider
	Model         string    `json:"model" gorm:"uniqueIndex:idx_hour_provider_model;size:100;default:''"`      // ""=全量/合计, 否则为具体模型名
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

// HourlyBreakdownItem 分时详细拆分行，包含模型维度（用于堆叠图）
type HourlyBreakdownItem struct {
	Hour         int    `json:"hour"`
	ProviderID   uint   `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	Model        string `json:"model"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	TotalTokens  int64  `json:"total_tokens"`
}
