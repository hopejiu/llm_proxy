package service

import (
	"encoding/json"
	"fmt"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"log/slog"
	"os"
	"sync"
)

type ProxyService struct {
	providerRepo   *repository.ProviderRepository
	requestLogRepo *repository.RequestLogRepository
	reqBodyLogFile *os.File
	reqBodyLogMu   sync.Mutex
}

func NewProxyService(providerRepo *repository.ProviderRepository, requestLogRepo *repository.RequestLogRepository) *ProxyService {
	logFile, err := os.OpenFile("proxy-reqbody.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		slog.Warn("无法创建请求体日志文件", "error", err)
		logFile = os.Stderr
	}

	return &ProxyService{
		providerRepo:   providerRepo,
		requestLogRepo: requestLogRepo,
		reqBodyLogFile: logFile,
	}
}

// Close 关闭 ProxyService 持有的资源
func (s *ProxyService) Close() {
	if s.reqBodyLogFile != nil && s.reqBodyLogFile != os.Stderr {
		s.reqBodyLogFile.Close()
	}
}

// GetActiveProviders 获取活跃的Provider
func (s *ProxyService) GetActiveProviders() ([]model.ProviderConfig, error) {
	return s.providerRepo.GetActive()
}

// GetFirstActiveProvider 获取第一个活跃的Provider
func (s *ProxyService) GetFirstActiveProvider() (*model.ProviderConfig, error) {
	providers, err := s.providerRepo.GetActive()
	if err != nil {
		slog.Error("获取活跃Provider失败", "error", err)
		return nil, fmt.Errorf("failed to get providers: %v", err)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no active provider available")
	}
	return &providers[0], nil
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

		if newBody, err := json.Marshal(reqMap); err == nil {
			return newBody
		}
		slog.Error("序列化请求体失败", "error", err)
	}
	return reqBody
}
