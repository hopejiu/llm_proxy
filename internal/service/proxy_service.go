package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/wanglejiu/llm-proxy/internal/model"
)

// ProxyService 代理服务：Provider 查询与请求体准备
// 缓存由 ProviderCache 独立管理
type ProxyService struct {
	providerCache *ProviderCache
}

// NewProxyService 创建代理服务实例
func NewProxyService(providerCache *ProviderCache) *ProxyService {
	return &ProxyService{
		providerCache: providerCache,
	}
}

// Close 关闭 ProxyService 持有的资源（当前无资源需要关闭）
func (s *ProxyService) Close() {}

// GetAllProviders 获取所有 Provider（通过 ProviderCache 带缓存读取）
func (s *ProxyService) GetAllProviders() ([]model.ProviderConfig, error) {
	return s.providerCache.GetAll()
}

// GetProviderByModel 根据模型名匹配 Provider，遍历所有 Provider 的 Models 配置
func (s *ProxyService) GetProviderByModel(modelName string) (model.ProviderConfig, error) {
	providers, err := s.providerCache.GetAll()
	if err != nil {
		slog.Error("获取Provider列表失败", "error", err)
		return model.ProviderConfig{}, fmt.Errorf("failed to get providers: %v", err)
	}
	if len(providers) == 0 {
		return model.ProviderConfig{}, fmt.Errorf("no provider available")
	}

	for i := range providers {
		for _, m := range providers[i].ParseModels() {
			for _, al := range m.Aliases {
				if al == modelName {
					return providers[i], nil
				}
			}
		}
	}
	for i := range providers {
		for _, m := range providers[i].ParseModels() {
			if modelName == m.Name {
				return providers[i], nil
			}
		}
	}
	// 构建可用模型列表用于错误提示
	var available []string
	for _, p := range providers {
		available = append(available, p.GetModelNames()...)
	}
	return model.ProviderConfig{}, fmt.Errorf("no provider found for model: %s, available models: %s", modelName, strings.Join(available, ", "))
}

// PrepareRequestBody 准备请求体，替换model为匹配的上游模型名，合并模型级ExtraParams
func (s *ProxyService) PrepareRequestBody(reqBody []byte, provider model.ProviderConfig) []byte {
	var reqInfo struct {
		Model string `json:"model"`
	}
	json.Unmarshal(reqBody, &reqInfo)

	var reqMap map[string]interface{}
	if err := json.Unmarshal(reqBody, &reqMap); err == nil {
		// 查找匹配的 ModelEntry，使用上游模型名替换
		if entry := provider.FindModelEntry(reqInfo.Model); entry != nil {
			reqMap["model"] = entry.Name

			// 合并模型级 ExtraParams（仅在启用时）
			if provider.EnableExtraParams && entry.ExtraParams != "" {
				var ep map[string]interface{}
				if err := json.Unmarshal([]byte(entry.ExtraParams), &ep); err == nil {
					for key, value := range ep {
						reqMap[key] = value
					}
				} else {
					slog.Warn("解析模型级ExtraParams失败", "model", entry.Name, "error", err)
				}
			}
		}

		// 流式请求时注入 stream_options 确保上游 API 返回 usage 信息
		if stream, ok := reqMap["stream"].(bool); ok && stream {
			if _, exists := reqMap["stream_options"]; !exists {
				reqMap["stream_options"] = map[string]interface{}{
					"include_usage": true,
				}
			}
		}

		if newBody, err := json.Marshal(reqMap); err == nil {
			return newBody
		}
		slog.Error("序列化请求体失败", "error", err)
	}
	return reqBody
}
