package handler

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	DefaultHTTPTimeout = 300 * time.Second
	DefaultLogFilePerm = 0666
)

// BaseHandler 公共基类，封装 Handler 的公共字段和方法
type BaseHandler struct {
	proxyService   *service.ProxyService
	requestLogRepo *repository.RequestLogRepository
	httpClient     *http.Client
}

// NewBaseHandler 创建 BaseHandler 实例
func NewBaseHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository) *BaseHandler {
	return &BaseHandler{
		proxyService:   proxyService,
		requestLogRepo: requestLogRepo,
		httpClient: &http.Client{
			Timeout: DefaultHTTPTimeout,
		},
	}
}

// GetProvider 获取第一个活跃的 Provider
func (h *BaseHandler) GetProvider() (*model.ProviderConfig, error) {
	return h.proxyService.GetFirstActiveProvider()
}

// GetProviders 获取所有活跃的 Providers
func (h *BaseHandler) GetProviders() ([]model.ProviderConfig, error) {
	return h.proxyService.GetActiveProviders()
}

// PrepareRequestBody 准备请求体（委托给 ProxyService）
func (h *BaseHandler) PrepareRequestBody(body []byte, provider *model.ProviderConfig) []byte {
	return h.proxyService.PrepareRequestBody(body, provider)
}

// SendStreamRequest 发送流式 HTTP 请求
func (h *BaseHandler) SendStreamRequest(ctx context.Context, url string, body []byte, apiKey string) (*http.Response, error) {
	return h.SendStreamRequestWithHeaders(ctx, url, body, apiKey, nil)
}

// SendStreamRequestWithHeaders 发送流式 HTTP 请求（支持自定义 header）
func (h *BaseHandler) SendStreamRequestWithHeaders(ctx context.Context, url string, body []byte, apiKey string, extraHeaders http.Header) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	// 合并自定义 header
	if extraHeaders != nil {
		for key, values := range extraHeaders {
			for _, value := range values {
				httpReq.Header.Add(key, value)
			}
		}
	}

	httpResp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	return httpResp, nil
}

// SendRequest 发送普通 HTTP 请求（非流式）
func (h *BaseHandler) SendRequest(ctx context.Context, url string, body []byte, apiKey string) ([]byte, error) {
	return h.SendRequestWithHeaders(ctx, url, body, apiKey, nil)
}

// SendRequestWithHeaders 发送普通 HTTP 请求（支持自定义 header）
func (h *BaseHandler) SendRequestWithHeaders(ctx context.Context, url string, body []byte, apiKey string, extraHeaders http.Header) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	// 合并自定义 header
	if extraHeaders != nil {
		for key, values := range extraHeaders {
			for _, value := range values {
				httpReq.Header.Add(key, value)
			}
		}
	}

	httpResp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ReadStreamLines 通用 SSE 流式行读取，对每行调用回调函数
// 返回读取过程中遇到的错误
func (h *BaseHandler) ReadStreamLines(resp *http.Response, onLine func(line string)) error {
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			onLine(line)
		}
	}
	return scanner.Err()
}

// ExtractUsage 从 OpenAI 格式响应数据中提取 token 使用量
func (h *BaseHandler) ExtractUsage(data map[string]interface{}) (inputTokens, outputTokens, totalTokens, cachedTokens int) {
	if usage, ok := data["usage"].(map[string]interface{}); ok {
		if pt, ok := usage["prompt_tokens"].(float64); ok {
			inputTokens = int(pt)
		}
		if ct, ok := usage["completion_tokens"].(float64); ok {
			outputTokens = int(ct)
		}
		if tt, ok := usage["total_tokens"].(float64); ok {
			totalTokens = int(tt)
		}
		// 从 prompt_tokens_details 中提取 cached_tokens
		if ptd, ok := usage["prompt_tokens_details"].(map[string]interface{}); ok {
			if cht, ok := ptd["cached_tokens"].(float64); ok {
				cachedTokens = int(cht)
			}
		}
	}
	return
}

// ExtractAnthropicUsage 从 Anthropic 格式 usage 中提取 token 数
func (h *BaseHandler) ExtractAnthropicUsage(data map[string]interface{}) (inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens int) {
	if usage, ok := data["usage"].(map[string]interface{}); ok {
		if it, ok := usage["input_tokens"].(float64); ok {
			inputTokens = int(it)
		}
		if ot, ok := usage["output_tokens"].(float64); ok {
			outputTokens = int(ot)
		}
		if cc, ok := usage["cache_creation_input_tokens"].(float64); ok {
			cacheCreationTokens = int(cc)
		}
		if cr, ok := usage["cache_read_input_tokens"].(float64); ok {
			cacheReadTokens = int(cr)
		}
	}
	return
}

// ReadBody 读取请求体并重置 Body，允许后续读取
func (h *BaseHandler) ReadBody(c *gin.Context) ([]byte, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body")
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

// SaveRequestLog 保存请求日志
func (h *BaseHandler) SaveRequestLog(reqLog *model.RequestLog) {
	h.requestLogRepo.Create(reqLog)
}

// CreateRequestLog 创建请求日志对象
func (h *BaseHandler) CreateRequestLog(provider *model.ProviderConfig, reqBody string) *model.RequestLog {
	return &model.RequestLog{
		ProviderID:  provider.ID,
		Model:       provider.Model,
		RequestBody: reqBody,
		Status:      "error",
	}
}
