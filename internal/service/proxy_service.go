package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	reqBodyLogFile *os.File
	reqBodyLogMu   sync.Mutex
)

func init() {
	var err error
	reqBodyLogFile, err = os.OpenFile("proxy-reqbody.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("[ProxyService] 无法创建请求体日志文件: %v", err)
		// 如果无法创建文件，使用标准错误输出
		reqBodyLogFile = os.Stderr
	}
}

type ProxyService struct {
	providerRepo   *repository.ProviderRepository
	requestLogRepo *repository.RequestLogRepository
	httpClient     *http.Client
}

func NewProxyService(providerRepo *repository.ProviderRepository, requestLogRepo *repository.RequestLogRepository) *ProxyService {
	return &ProxyService{
		providerRepo:   providerRepo,
		requestLogRepo: requestLogRepo,
		httpClient:     &http.Client{Timeout: 300 * time.Second},
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
		log.Printf("[ProxyService] 获取活跃Provider失败: %v", err)
		return nil, fmt.Errorf("failed to get providers: %v", err)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no active provider available")
	}
	return &providers[0], nil
}

// PrepareRequestBody 准备请求体，替换model并添加enable_thinking，合并ExtraParams
func (s *ProxyService) PrepareRequestBody(reqBody []byte, provider *model.ProviderConfig) []byte {
	var reqMap map[string]interface{}
	if err := json.Unmarshal(reqBody, &reqMap); err == nil {
		reqMap["model"] = provider.Model
		reqMap["enable_thinking"] = true

		// 合并ExtraParams
		if provider.ExtraParams != "" {
			var extraParams map[string]interface{}
			if err := json.Unmarshal([]byte(provider.ExtraParams), &extraParams); err == nil {
				for key, value := range extraParams {
					// 用户自定义参数优先级高于默认参数
					reqMap[key] = value
				}
			} else {
				log.Printf("[ProxyService] 解析ExtraParams失败: %v", err)
			}
		}

		if newBody, err := json.Marshal(reqMap); err == nil {
			return newBody
		}
		log.Printf("[ProxyService] 序列化请求体失败: %v", err)
	}
	return reqBody
}

// ProxyRequest 代理请求到第三方LLM服务（非流式）
func (s *ProxyService) ProxyRequest(reqBody []byte) (*model.RequestLog, error) {
	startTime := time.Now()

	// 解析请求
	var req model.OpenAIRequest
	if err := json.Unmarshal(reqBody, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %v", err)
	}

	// 获取第一个活跃的Provider
	provider, err := s.GetFirstActiveProvider()
	if err != nil {
		return nil, err
	}

	// 准备请求体
	reqBody = s.PrepareRequestBody(reqBody, provider)

	// 创建请求日志
	reqLog := &model.RequestLog{
		ProviderID:  provider.ID,
		Model:       provider.Model,
		RequestBody: string(reqBody),
		Status:      "error",
	}

	// 转发请求
	targetURL := fmt.Sprintf("%s", strings.TrimSuffix(provider.BaseURL, "/"))
	httpReq, err := http.NewRequest("POST", targetURL, bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("[ProxyService] 创建HTTP请求失败: %v", err)
		reqLog.ErrorMessage = err.Error()
		s.requestLogRepo.Create(reqLog)
		return reqLog, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", provider.APIKey))

	// 发送请求
	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("[ProxyService] 发送请求失败: %v", err)
		reqLog.ErrorMessage = err.Error()
		s.requestLogRepo.Create(reqLog)
		return reqLog, err
	}
	defer httpResp.Body.Close()

	// 处理响应
	return s.handleNormalResponse(httpResp, reqLog, startTime)
}

// handleNormalResponse 处理非流式响应
func (s *ProxyService) handleNormalResponse(resp *http.Response, reqLog *model.RequestLog, startTime time.Time) (*model.RequestLog, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ProxyService] 读取响应体失败: %v", err)
		reqLog.ErrorMessage = err.Error()
		s.requestLogRepo.Create(reqLog)
		return reqLog, err
	}

	reqLog.ResponseBody = string(body)
	reqLog.Duration = time.Since(startTime).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		reqLog.ErrorMessage = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		s.requestLogRepo.Create(reqLog)
		return reqLog, fmt.Errorf("provider returned status %d", resp.StatusCode)
	}

	// 解析响应获取token使用情况
	var openAIResp model.OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err == nil {
		reqLog.InputTokens = openAIResp.Usage.PromptTokens
		reqLog.OutputTokens = openAIResp.Usage.CompletionTokens
		reqLog.TotalTokens = openAIResp.Usage.TotalTokens
		if openAIResp.Usage.PromptTokensDetails != nil {
			reqLog.CachedTokens = openAIResp.Usage.PromptTokensDetails.CachedTokens
		}

		// 提取thinking内容
		if len(openAIResp.Choices) > 0 {
			// 检查是否有reasoning_content字段
			var rawResp map[string]interface{}
			if err := json.Unmarshal(body, &rawResp); err == nil {
				if choices, ok := rawResp["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						if msg, ok := choice["message"].(map[string]interface{}); ok {
							if reasoning, ok := msg["reasoning_content"].(string); ok && reasoning != "" {
								reqLog.ThinkingContent = reasoning
							}
						}
					}
				}
			}
		}
	}

	reqLog.Status = "success"
	s.requestLogRepo.Create(reqLog)

	return reqLog, nil
}
