package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/wanglejiu/llm-proxy/internal/config"
	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/service"
)

// ProviderService Wails 绑定服务：Provider 管理
type ProviderService struct {
	svc    *service.ProviderService
	cfg    *config.Config
	proxy  *service.ProxyService
	ctx    context.Context
}

func NewProviderService(svc *service.ProviderService, cfg *config.Config, proxy *service.ProxyService) *ProviderService {
	return &ProviderService{svc: svc, cfg: cfg, proxy: proxy}
}

func (s *ProviderService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}

// GetProviders 获取所有 Provider（API Key 脱敏）
func (s *ProviderService) GetProviders() ([]ProviderVO, error) {
	providers, err := s.svc.GetAllProviders()
	if err != nil {
		return nil, NewAppError("INTERNAL", "获取Provider列表失败")
	}
	return providersToVOs(providers), nil
}

// GetProvider 获取单个 Provider（API Key 脱敏）
func (s *ProviderService) GetProvider(id uint) (ProviderVO, error) {
	provider, err := s.svc.GetProvider(id)
	if err != nil {
		return ProviderVO{}, NewAppError("NOT_FOUND", "Provider不存在")
	}
	return providerToVO(provider), nil
}

// CreateProvider 创建 Provider
func (s *ProviderService) CreateProvider(data ProviderCreateVO) (ProviderVO, error) {
	provider := createVOToModel(data)
	if err := s.svc.CreateProvider(provider); err != nil {
		return ProviderVO{}, NewAppError("INTERNAL", "创建Provider失败")
	}
	return providerToVO(provider), nil
}

// UpdateProvider 更新 Provider（含 API Key 保留逻辑）
func (s *ProviderService) UpdateProvider(id uint, data ProviderUpdateVO) (ProviderVO, error) {
	preservedKey, err := s.svc.PreserveAPIKey(id, data.APIKey)
	if err == nil {
		data.APIKey = preservedKey
	}
	provider := updateVOToModel(id, data)
	if err := s.svc.UpdateProvider(provider); err != nil {
		return ProviderVO{}, NewAppError("INTERNAL", "更新Provider失败")
	}
	return providerToVO(provider), nil
}

// DeleteProvider 删除 Provider
func (s *ProviderService) DeleteProvider(id uint) error {
	if err := s.svc.DeleteProvider(id); err != nil {
		return NewAppError("INTERNAL", "删除Provider失败")
	}
	return nil
}

// ExportProvidersToFile 导出到文件（使用 SaveFileDialog）
func (s *ProviderService) ExportProvidersToFile() (string, error) {
	path, err := application.Get().Dialog.SaveFile().
		SetFilename("providers.json").
		AddFilter("JSON", "*.json").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}

	providers, err := s.svc.GetAllProviders()
	if err != nil {
		return "", NewAppError("INTERNAL", "导出失败")
	}

	data, _ := json.MarshalIndent(providers, "", "  ")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", NewAppError("INTERNAL", "写入文件失败")
	}
	return path, nil
}

// ImportProvidersFromFile 从文件导入（使用 OpenFileDialog）
func (s *ProviderService) ImportProvidersFromFile() error {
	path, err := application.Get().Dialog.OpenFile().
		SetTitle("导入 Provider").
		AddFilter("JSON", "*.json").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return NewAppError("INTERNAL", "读取文件失败")
	}

	var providers []model.ProviderConfig
	if err := json.Unmarshal(data, &providers); err != nil {
		return NewAppError("BAD_REQUEST", "文件格式错误")
	}

	for i := range providers {
		providers[i].ID = 0
	}

	if err := s.svc.ImportAll(providers); err != nil {
		return NewAppError("INTERNAL", "导入Provider失败")
	}
	return nil
}

// SetupCodeBuddy 配置 CodeBuddy models.json
func (s *ProviderService) SetupCodeBuddy() (CodeBuddyResultVO, error) {
	targetURL := fmt.Sprintf("http://localhost:%s/v1", s.cfg.GetProxyPort())
	result, err := service.SetupCodeBuddy(targetURL)
	if err != nil {
		return CodeBuddyResultVO{}, NewAppError("INTERNAL", err.Error())
	}
	return CodeBuddyResultVO{
		Message: result.Message,
		Path:    result.Path,
		Exists:  result.Exists,
		Added:   result.Added,
		Models:  result.Models,
	}, nil
}

// FetchProviderModels 查询上游 API 的所有可用模型（OpenAI 兼容格式：GET /v1/models）
func (s *ProviderService) FetchProviderModels(baseURL, apiKey string) ([]string, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	apiURL := strings.TrimRight(baseURL, "/") + "/v1/models"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误(%d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	var models []string
	if dataArr, ok := result["data"].([]interface{}); ok {
		for _, m := range dataArr {
			if modelObj, ok := m.(map[string]interface{}); ok {
				if id, ok := modelObj["id"].(string); ok {
					models = append(models, id)
				}
			}
		}
	}

	return models, nil
}

// TestProviderConnection 测试 Provider 连接是否可用
func (s *ProviderService) TestProviderConnection(baseURL, apiKey, model, urlSuffix string, autoSuffix bool) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	var requestURL string
	if autoSuffix {
		base := strings.TrimRight(baseURL, "/")
		suffix := urlSuffix
		if suffix != "" && !strings.HasPrefix(suffix, "/") {
			suffix = "/" + suffix
		}
		requestURL = base + suffix
	} else {
		requestURL = baseURL
	}

	if requestURL == "" {
		return "", fmt.Errorf("请求URL为空，请填写Base URL")
	}

	testBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "hi"},
		},
		"max_tokens": 5,
		"stream":     false,
	}
	body, err := json.Marshal(testBody)
	if err != nil {
		return "", fmt.Errorf("构建请求体失败: %w", err)
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("连接失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		return "✅ 连接成功，API 返回正常", nil
	}

	var errResp map[string]interface{}
	errMsg := string(respBody)
	if json.Unmarshal(respBody, &errResp) == nil {
		if msg, ok := errResp["error"].(map[string]interface{}); ok {
			if message, ok := msg["message"].(string); ok {
				errMsg = message
			}
		}
	}
	return fmt.Sprintf("❌ API返回错误(%d): %s", resp.StatusCode, errMsg), nil
}
