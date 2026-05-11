package service

import (
	"encoding/json"
	"fmt"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// ProxyService 代理服务
type ProxyService struct {
	providerRepo  *repository.ProviderRepository
	cacheMu       sync.RWMutex
	providerCache []model.ProviderConfig
	cacheExpiry   time.Time
	cacheTTL      time.Duration
}

// NewProxyService 创建代理服务实例
func NewProxyService(providerRepo *repository.ProviderRepository, cacheTTL time.Duration) *ProxyService {
	return &ProxyService{
		providerRepo: providerRepo,
		cacheTTL:    cacheTTL,
	}
}

// Close 关闭 ProxyService 持有的资源（当前无资源需要关闭）
func (s *ProxyService) Close() {}

// getAllProvidersCached 获取 Provider 列表（优先读缓存）
func (s *ProxyService) getAllProvidersCached() ([]model.ProviderConfig, error) {
	s.cacheMu.RLock()
	if time.Now().Before(s.cacheExpiry) {
		providers := s.providerCache
		s.cacheMu.RUnlock()
		return providers, nil
	}
	s.cacheMu.RUnlock()

	// 缓存过期，从数据库加载
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// 双重检查
	if time.Now().Before(s.cacheExpiry) {
		return s.providerCache, nil
	}

	providers, err := s.providerRepo.GetAll()
	if err != nil {
		return nil, err
	}

	s.providerCache = providers
	s.cacheExpiry = time.Now().Add(s.cacheTTL)
	return providers, nil
}

// InvalidateCache 主动失效缓存（Provider 增删改时调用）
func (s *ProxyService) InvalidateCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cacheExpiry = time.Time{}
}

// GetAllProviders 获取所有 Provider
func (s *ProxyService) GetAllProviders() ([]model.ProviderConfig, error) {
	return s.getAllProvidersCached()
}

// GetProviderByModel 根据模型名匹配 Provider，优先别名匹配，其次模型名匹配
func (s *ProxyService) GetProviderByModel(modelName string) (*model.ProviderConfig, error) {
	providers, err := s.getAllProvidersCached()
	if err != nil {
		slog.Error("获取Provider列表失败", "error", err)
		return nil, fmt.Errorf("failed to get providers: %v", err)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no provider available")
	}

	// 使用 MatchModelName 方法匹配（内部已实现优先别名、其次模型名）
	for i := range providers {
		if providers[i].Model == "" {
			continue
		}
		if providers[i].Alias == modelName {
			return &providers[i], nil
		}
	}

	for i := range providers {
		if providers[i].Model == "" {
			continue
		}
		if providers[i].Model == modelName {
			return &providers[i], nil
		}
	}

	// 构建可用模型列表用于错误提示
	var available []string
	for _, p := range providers {
		available = append(available, p.GetModelNames()...)
	}
	return nil, fmt.Errorf("no provider found for model: %s, available models: %s", modelName, strings.Join(available, ", "))
}

// PrepareRequestBody 准备请求体，替换model并合并ExtraParams
func (s *ProxyService) PrepareRequestBody(reqBody []byte, provider *model.ProviderConfig) []byte {
	var reqMap map[string]interface{}
	if err := json.Unmarshal(reqBody, &reqMap); err == nil {
		reqMap["model"] = provider.Model

		// 合并ExtraParams
		if provider.ExtraParams != "" {
			var extraParams map[string]interface{}
			if err := json.Unmarshal([]byte(provider.ExtraParams), &extraParams); err == nil {
				for key, value := range extraParams {
					reqMap[key] = value
				}
			} else {
				slog.Warn("解析ExtraParams失败", "error", err)
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


