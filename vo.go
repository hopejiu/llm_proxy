package main

import (
	"fmt"

	"github.com/wanglejiu/llm-proxy/internal/handler"
	"github.com/wanglejiu/llm-proxy/internal/model"
)

// ========== Provider VO ==========

type ProviderVO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	AutoSuffix  bool   `json:"auto_suffix"`
	UrlSuffix   string `json:"url_suffix"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key"`
	Models      string `json:"models"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ProviderCreateVO struct {
	Name        string `json:"name"`
	AutoSuffix  bool   `json:"auto_suffix"`
	UrlSuffix   string `json:"url_suffix"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key"`
	Models      string `json:"models"`
}

type ProviderUpdateVO struct {
	Name        string `json:"name"`
	AutoSuffix  bool   `json:"auto_suffix"`
	UrlSuffix   string `json:"url_suffix"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key"`
	Models      string `json:"models"`
}

func providerToVO(p *model.ProviderConfig) ProviderVO {
	return ProviderVO{
		ID: p.ID, Name: p.Name, AutoSuffix: p.AutoSuffix, UrlSuffix: p.UrlSuffix,
		BaseURL: p.BaseURL, APIKey: p.APIKey, Models: p.Models,
		CreatedAt: p.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: p.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func providersToVOs(providers []model.ProviderConfig) []ProviderVO {
	result := make([]ProviderVO, len(providers))
	for i := range providers {
		result[i] = providerToVO(&providers[i])
	}
	return result
}

func createVOToModel(data ProviderCreateVO) *model.ProviderConfig {
	return &model.ProviderConfig{
		Name: data.Name, AutoSuffix: data.AutoSuffix, UrlSuffix: data.UrlSuffix,
		BaseURL: data.BaseURL, APIKey: data.APIKey, Models: data.Models,
	}
}

func updateVOToModel(id uint, data ProviderUpdateVO) *model.ProviderConfig {
	return &model.ProviderConfig{
		ID: id, Name: data.Name, AutoSuffix: data.AutoSuffix, UrlSuffix: data.UrlSuffix,
		BaseURL: data.BaseURL, APIKey: data.APIKey, Models: data.Models,
	}
}

// ========== Stats/Log VO ==========

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

type RequestLogDetailVO struct {
	RequestLogVO
	ResponseContent string `json:"response_content"`
	ThinkingContent string `json:"thinking_content"`
	RequestBody     string `json:"request_body"`
	ResponseBody    string `json:"response_body"`
}

type HourlyStatBreakdownVO struct {
	Hour         int    `json:"hour"`
	ProviderID   uint   `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	TotalTokens  int64  `json:"total_tokens"`
}

type ActiveRequestVO = handler.ActiveRequest

func requestLogToVO(log *model.RequestLog) RequestLogVO {
	return RequestLogVO{
		ID: log.ID, ProviderID: log.ProviderID, ProviderName: "",
		Model: log.Model, InputTokens: log.InputTokens, OutputTokens: log.OutputTokens,
		TotalTokens: log.TotalTokens, CachedTokens: log.CachedTokens,
		Status: log.Status, ErrorMessage: log.ErrorMessage,
		Duration: log.Duration, CreatedAt: log.CreatedAt.Format("2006-01-02 15:04:05"),
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

// ========== Common VO ==========

type LogEntryVO struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type ProxyStatusVO struct {
	Status string `json:"status"`
	Port   string `json:"port"`
	Error  string `json:"error,omitempty"`
}

type CodeBuddyResultVO struct {
	Message string `json:"message"`
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Added   bool   `json:"added"`
	Models  int    `json:"models"`
}

func formatDuration(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}
