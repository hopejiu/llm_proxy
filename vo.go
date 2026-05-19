package main

import (
	"llm-proxy/internal/handler"
	"llm-proxy/internal/model"
	"fmt"
)

// ProviderVO 返回给前端的 Provider 视图（API Key 脱敏）
type ProviderVO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	AutoSuffix  bool   `json:"auto_suffix"`
	UrlSuffix   string `json:"url_suffix"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key"`
	Model       string `json:"model"`
	Alias       string `json:"alias"`
	ExtraParams string `json:"extra_params"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ProviderCreateVO 前端创建 Provider 时提交
type ProviderCreateVO struct {
	Name        string `json:"name"`
	AutoSuffix  bool   `json:"auto_suffix"`
	UrlSuffix   string `json:"url_suffix"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key"`
	Model       string `json:"model"`
	Alias       string `json:"alias"`
	ExtraParams string `json:"extra_params"`
}

// ProviderUpdateVO 前端更新 Provider 时提交（API Key 可能是脱敏格式）
type ProviderUpdateVO struct {
	Name        string `json:"name"`
	AutoSuffix  bool   `json:"auto_suffix"`
	UrlSuffix   string `json:"url_suffix"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key"`
	Model       string `json:"model"`
	Alias       string `json:"alias"`
	ExtraParams string `json:"extra_params"`
}

// RequestLogVO 请求日志列表视图
type RequestLogVO struct {
	ID           uint   `json:"id"`
	ProviderID   uint   `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	Model        string `json:"model"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	TotalTokens  int    `json:"total_tokens"`
	CachedTokens int    `json:"cached_tokens"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	Duration     int64  `json:"duration"`
	CreatedAt    string `json:"created_at"`
}

// RequestLogDetailVO 请求日志详情（含请求体和响应体）
type RequestLogDetailVO struct {
	RequestLogVO
	ResponseContent string `json:"response_content"`
	ThinkingContent string `json:"thinking_content"`
	RequestBody     string `json:"request_body"`
	ResponseBody    string `json:"response_body"`
}

// LogEntryVO 日志条目视图
type LogEntryVO struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// ProxyStatusVO 代理服务状态
type ProxyStatusVO struct {
	Status string `json:"status"`
	Port   string `json:"port"`
	Error  string `json:"error,omitempty"`
}

// CodeBuddyResultVO CodeBuddy 配置结果
type CodeBuddyResultVO struct {
	Message string `json:"message"`
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Added   bool   `json:"added"`
	Models  int    `json:"models"`
}

// HourlyStatBreakdownVO 分时按 Provider 拆分的详细视图（用于堆叠图）
type HourlyStatBreakdownVO struct {
	Hour         int    `json:"hour"`
	ProviderID   uint   `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	TotalTokens  int64  `json:"total_tokens"`
}

// ActiveRequestVO 活跃请求视图，直接复用 handler.ActiveRequest
type ActiveRequestVO = handler.ActiveRequest

// Model → VO 转换函数

func providerToVO(p *model.ProviderConfig) ProviderVO {
	return ProviderVO{
		ID:          p.ID,
		Name:        p.Name,
		AutoSuffix:  p.AutoSuffix,
		UrlSuffix:   p.UrlSuffix,
		BaseURL:     p.BaseURL,
		APIKey:      p.APIKey,
		Model:       p.Model,
		Alias:       p.Alias,
		ExtraParams: p.ExtraParams,
		CreatedAt:   p.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func providersToVOs(providers []model.ProviderConfig) []ProviderVO {
	result := make([]ProviderVO, len(providers))
	for i := range providers {
		result[i] = providerToVO(&providers[i])
	}
	return result
}

func requestLogToVO(log *model.RequestLog) RequestLogVO {
	return RequestLogVO{
		ID:           log.ID,
		ProviderID:   log.ProviderID,
		ProviderName: "", // 需要手动填充
		Model:        log.Model,
		InputTokens:  log.InputTokens,
		OutputTokens: log.OutputTokens,
		TotalTokens:  log.TotalTokens,
		CachedTokens: log.CachedTokens,
		Status:       log.Status,
		ErrorMessage: log.ErrorMessage,
		Duration:     log.Duration,
		CreatedAt:    log.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func requestLogToDetailVO(log *model.RequestLog) RequestLogDetailVO {
	return RequestLogDetailVO{
		RequestLogVO:    requestLogToVO(log),
		ResponseContent: log.ResponseContent,
		ThinkingContent: log.ThinkingContent,
		RequestBody:     log.RequestBody,
		ResponseBody:    log.ResponseBody,
	}
}

// createVOToModel 将 ProviderCreateVO 转换为 model.ProviderConfig
func createVOToModel(data ProviderCreateVO) *model.ProviderConfig {
	return &model.ProviderConfig{
		Name:        data.Name,
		AutoSuffix:  data.AutoSuffix,
		UrlSuffix:   data.UrlSuffix,
		BaseURL:     data.BaseURL,
		APIKey:      data.APIKey,
		Model:       data.Model,
		Alias:       data.Alias,
		ExtraParams: data.ExtraParams,
	}
}

// updateVOToModel 将 ProviderUpdateVO 转换为 model.ProviderConfig
func updateVOToModel(id uint, data ProviderUpdateVO) *model.ProviderConfig {
	return &model.ProviderConfig{
		ID:          id,
		Name:        data.Name,
		AutoSuffix:  data.AutoSuffix,
		UrlSuffix:   data.UrlSuffix,
		BaseURL:     data.BaseURL,
		APIKey:      data.APIKey,
		Model:       data.Model,
		Alias:       data.Alias,
		ExtraParams: data.ExtraParams,
	}
}

// formatDuration 格式化耗时
func formatDuration(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}
