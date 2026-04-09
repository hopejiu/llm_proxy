package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	DefaultHTTPTimeout       = 300 * time.Second
	DefaultLogFilePerm       = 0666
	StreamFirstByteTimeout   = 5 * time.Second
	StreamMaxRetries         = 3
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

// StreamRetryConfig 流式重试配置
type StreamRetryConfig struct {
	FirstByteTimeout time.Duration
	MaxRetries       int
}

// DefaultStreamRetryConfig 返回默认的流式重试配置
func DefaultStreamRetryConfig() StreamRetryConfig {
	return StreamRetryConfig{
		FirstByteTimeout: StreamFirstByteTimeout,
		MaxRetries:       StreamMaxRetries,
	}
}

// StreamLineProcessor 处理流式响应行的回调函数
// line: 原始行内容
// tokens: 当前 token 使用量（可读取）
// 返回值: 是否应该停止处理（如遇到 [DONE]）
type StreamLineProcessor func(line string, tokens *StreamTokens) (stop bool)

// StreamTokens 流式请求的 token 使用量
type StreamTokens struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	CachedTokens int
}

// ExecuteStreamWithRetry 执行带超时重试的流式请求
// 返回: 响应内容构建器, token使用量, 最后错误
func (h *BaseHandler) ExecuteStreamWithRetry(
	ctx context.Context,
	provider *model.ProviderConfig,
	body []byte,
	config StreamRetryConfig,
	processor StreamLineProcessor,
) (responseBuilder strings.Builder, tokens StreamTokens, lastErr error) {
	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		slog.Info("stream request attempt", "attempt", attempt, "maxRetries", config.MaxRetries)

		httpResp, err := h.SendStreamRequest(ctx, provider.BaseURL, body, provider.APIKey)
		if err != nil {
			slog.Error("发送HTTP请求失败", "attempt", attempt, "error", err)
			lastErr = err
			continue
		}

		// 尝试读取流，带首次数据超时检测
		success, timeout, err := h.readStreamWithFirstByteTimeout(httpResp, &responseBuilder, &tokens, processor, config.FirstByteTimeout)
		httpResp.Body.Close()

		if success {
			return responseBuilder, tokens, nil
		}

		if timeout {
			slog.Warn("stream首次数据超时，准备重试", "attempt", attempt, "timeout", config.FirstByteTimeout)
			lastErr = fmt.Errorf("stream first byte timeout after %v", config.FirstByteTimeout)
			continue
		}

		slog.Error("读取流式响应失败", "attempt", attempt, "error", err)
		lastErr = err
		continue
	}

	return responseBuilder, tokens, lastErr
}

// readStreamWithFirstByteTimeout 读取流式响应，检测首次数据超时
// 返回: success(是否成功完成), timeout(是否因超时退出), err(错误信息)
func (h *BaseHandler) readStreamWithFirstByteTimeout(
	httpResp *http.Response,
	responseBuilder *strings.Builder,
	tokens *StreamTokens,
	processor StreamLineProcessor,
	firstByteTimeout time.Duration,
) (success bool, timeout bool, err error) {
	// 创建带超时的 context 用于检测首次数据
	firstByteCtx, firstByteCancel := context.WithTimeout(context.Background(), firstByteTimeout)
	defer firstByteCancel()

	// 用于通知首次数据已到达
	firstByteChan := make(chan struct{}, 1)
	// 用于通知读取完成
	doneChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(httpResp.Body)
		firstDataReceived := false

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			// 首次收到数据，通知主 goroutine
			if !firstDataReceived {
				firstDataReceived = true
				select {
				case firstByteChan <- struct{}{}:
				default:
				}
			}

			responseBuilder.WriteString(line + "\n")

			// 提取 token 使用量
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					// 调用处理器处理行
					if processor != nil {
						if stop := processor(line, tokens); stop {
							break
						}
					}
					continue
				}
				var streamResp map[string]interface{}
				if err := json.Unmarshal([]byte(data), &streamResp); err == nil {
					in, out, total, cached := h.ExtractUsage(streamResp)
					if in > 0 {
						tokens.InputTokens = in
					}
					if out > 0 {
						tokens.OutputTokens = out
					}
					if total > 0 {
						tokens.TotalTokens = total
					}
					if cached > 0 {
						tokens.CachedTokens = cached
					}
				}
			}

			// 调用处理器处理行
			if processor != nil {
				if stop := processor(line, tokens); stop {
					break
				}
			}
		}
		doneChan <- scanner.Err()
	}()

	// 等待首次数据或超时
	select {
	case <-firstByteChan:
		// 首次数据已到达，继续等待读取完成
		slog.Debug("stream首次数据已到达")
	case <-firstByteCtx.Done():
		// 首次数据超时
		return false, true, nil
	}

	// 等待读取完成
	err = <-doneChan
	return err == nil, false, err
}
