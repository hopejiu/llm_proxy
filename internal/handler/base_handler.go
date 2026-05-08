package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"llm-proxy/internal/converter"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// UpstreamError 上游 API 返回的错误，携带状态码和响应体
type UpstreamError struct {
	StatusCode int
	Body       string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// HandlerConfig Handler 层可配置参数
type HandlerConfig struct {
	HTTPTimeout            time.Duration
	StreamFirstByteTimeout time.Duration
	StreamMaxRetries       int
	RetryDelayBase         time.Duration
}

// BaseHandler 公共基类，封装 Handler 的公共字段和方法
type BaseHandler struct {
	proxyService   *service.ProxyService
	requestLogRepo *repository.RequestLogRepository
	httpClient     *http.Client
	config         HandlerConfig
}

// NewBaseHandler 创建 BaseHandler 实例
func NewBaseHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository, cfg HandlerConfig) *BaseHandler {
	return &BaseHandler{
		proxyService:   proxyService,
		requestLogRepo: requestLogRepo,
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
		config: cfg,
	}
}

// GetProviderByModel 根据模型名匹配 Provider
func (h *BaseHandler) GetProviderByModel(modelName string) (*model.ProviderConfig, error) {
	return h.proxyService.GetProviderByModel(modelName)
}

// GetAllProviders 获取所有 Provider
func (h *BaseHandler) GetAllProviders() ([]model.ProviderConfig, error) {
	return h.proxyService.GetAllProviders()
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
		// Go 的 http.Client.Do() 会忽略 Header map 中的 Host key，
		// 实际发送的 Host header 取自 httpReq.Host 字段。
		// 部分API网关（如讯飞）的HMAC签名校验要求 host 请求头必须参与签名，
		// 因此需要将 extraHeaders 中的 host 同步到 httpReq.Host。
		if hostValues := extraHeaders.Values("host"); len(hostValues) > 0 {
			httpReq.Host = hostValues[0]
			delete(httpReq.Header, "Host")
		}
	}

	httpResp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		return nil, &UpstreamError{StatusCode: httpResp.StatusCode, Body: string(respBody)}
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
		// Go 的 http.Client.Do() 会忽略 Header map 中的 Host key，
		// 实际发送的 Host header 取自 httpReq.Host 字段。
		// 部分API网关（如讯飞）的HMAC签名校验要求 host 请求头必须参与签名，
		// 因此需要将 extraHeaders 中的 host 同步到 httpReq.Host。
		if hostValues := extraHeaders.Values("host"); len(hostValues) > 0 {
			httpReq.Host = hostValues[0]
			delete(httpReq.Header, "Host")
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
		return nil, &UpstreamError{StatusCode: httpResp.StatusCode, Body: string(respBody)}
	}

	return respBody, nil
}

// ExtractUsage 从 OpenAI 格式响应数据中提取 token 使用量
func (h *BaseHandler) ExtractUsage(data map[string]interface{}) (inputTokens, outputTokens, totalTokens, cachedTokens int) {
	return converter.ExtractUsage(data)
}

// SendRequestWithRetry 发送普通 HTTP 请求（带重试）
func (h *BaseHandler) SendRequestWithRetry(ctx context.Context, url string, body []byte, apiKey string, maxRetries int) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			retryDelay := time.Duration(attempt) * h.config.RetryDelayBase
			slog.Warn("非流式请求重试", "url", url, "attempt", attempt, "maxRetries", maxRetries, "delay", retryDelay)
			time.Sleep(retryDelay)
		}

		respBody, err := h.SendRequest(ctx, url, body, apiKey)
		if err == nil {
			return respBody, nil
		}
		lastErr = err
		slog.Error("非流式请求失败", "url", url, "attempt", attempt, "error", err)
	}
	return nil, lastErr
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

// maxBodySize 请求体/响应体存储的最大字节数（超过截断）
const maxBodySize = 64 * 1024 // 64KB

// truncateBody 截断过长的请求体/响应体
func truncateBody(body string) string {
	if len(body) <= maxBodySize {
		return body
	}
	return body[:maxBodySize] + "\n...[truncated]"
}

// SaveRequestLog 保存请求日志（自动截断过长的请求体/响应体）
func (h *BaseHandler) SaveRequestLog(reqLog *model.RequestLog) {
	reqLog.RequestBody = truncateBody(reqLog.RequestBody)
	reqLog.ResponseBody = truncateBody(reqLog.ResponseBody)
	if err := h.requestLogRepo.Create(reqLog); err != nil {
		slog.Error("保存请求日志失败", "error", err)
	}
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

// DefaultStreamRetryConfig 返回基于当前配置的流式重试配置
func (h *BaseHandler) DefaultStreamRetryConfig() StreamRetryConfig {
	return StreamRetryConfig{
		FirstByteTimeout: h.config.StreamFirstByteTimeout,
		MaxRetries:       h.config.StreamMaxRetries,
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
		attemptStartTime := time.Now()

		if attempt == 1 {
			slog.Info("开始流式请求", "provider", provider.Name, "model", provider.Model)
		} else {
			// 重试延迟：次数 * 0.5s
			retryDelay := time.Duration(attempt) * h.config.RetryDelayBase
			slog.Warn("流式请求失败，正在重试", "provider", provider.Name, "model", provider.Model, "attempt", attempt, "maxRetries", config.MaxRetries, "delay", retryDelay)
			sleepStart := time.Now()
			time.Sleep(retryDelay)
			slog.Debug("重试延迟完成", "actual_delay", time.Since(sleepStart), "expected_delay", retryDelay)
		}

		// 发送HTTP请求
		httpStartTime := time.Now()
		httpResp, err := h.SendStreamRequest(ctx, provider.GetRequestURL(), body, provider.APIKey)
		httpDuration := time.Since(httpStartTime)

		if err != nil {
			slog.Error("发送HTTP请求失败", "provider", provider.Name, "attempt", attempt, "duration", httpDuration, "error", err)
			lastErr = err
			continue
		}
		slog.Debug("HTTP连接建立成功", "provider", provider.Name, "attempt", attempt, "duration", httpDuration)

		// 尝试读取流，带首次数据超时检测
		success, timeout, err := h.readStreamWithFirstByteTimeout(httpResp, &responseBuilder, &tokens, processor, config.FirstByteTimeout)
		httpResp.Body.Close()

		attemptDuration := time.Since(attemptStartTime)

		if success {
			slog.Info("流式请求成功", "provider", provider.Name, "attempt", attempt, "total_duration", attemptDuration)
			return responseBuilder, tokens, nil
		}

		if timeout {
			slog.Warn("首次数据超时", "provider", provider.Name, "attempt", attempt, "timeout", config.FirstByteTimeout, "total_duration", attemptDuration)
			lastErr = fmt.Errorf("stream first byte timeout after %v", config.FirstByteTimeout)
			continue
		}

		slog.Error("读取流式响应失败", "provider", provider.Name, "attempt", attempt, "duration", attemptDuration, "error", err)
		lastErr = err
		continue
	}

	return responseBuilder, tokens, lastErr
}

// StreamError 流式响应中的错误
type StreamError struct {
	Code    interface{} `json:"code"`
	Message string      `json:"message"`
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
	// 用于传递检测到的流式错误
	streamErrChan := make(chan StreamError, 1)

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
					// 检测流式响应中的错误
					if streamErr := h.detectStreamError(streamResp); streamErr != nil {
						select {
						case streamErrChan <- *streamErr:
						default:
						}
						doneChan <- fmt.Errorf("stream error: code=%v, message=%s", streamErr.Code, streamErr.Message)
						return
					}

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
	firstByteStartTime := time.Now()
	select {
	case <-firstByteChan:
		// 首次数据已到达，继续等待读取完成
		slog.Debug("stream首次数据已到达", "wait_duration", time.Since(firstByteStartTime))
	case <-firstByteCtx.Done():
		// 首次数据超时
		slog.Warn("stream首次数据等待超时", "timeout", firstByteTimeout, "waited", time.Since(firstByteStartTime))
		return false, true, nil
	}

	// 等待读取完成
	readStartTime := time.Now()
	err = <-doneChan
	readDuration := time.Since(readStartTime)

	if err == nil {
		slog.Debug("stream读取完成", "read_duration", readDuration)
	} else {
		slog.Debug("stream读取出错", "read_duration", readDuration, "error", err)
	}

	return err == nil, false, err
}

// detectStreamError 检测流式响应中的错误
func (h *BaseHandler) detectStreamError(data map[string]interface{}) *StreamError {
	// 检查 OpenAI 格式的错误
	if errMap, ok := data["error"].(map[string]interface{}); ok {
		streamErr := &StreamError{}
		if code, ok := errMap["code"]; ok {
			streamErr.Code = code
		}
		if msg, ok := errMap["message"].(string); ok {
			streamErr.Message = msg
		}
		// 如果有错误码或错误消息，返回错误
		if streamErr.Code != nil || streamErr.Message != "" {
			return streamErr
		}
	}

	// 检查其他格式的错误（如 error_code, error_msg）
	if errorCode, ok := data["error_code"].(string); ok {
		streamErr := &StreamError{
			Code:    errorCode,
			Message: "",
		}
		if errorMsg, ok := data["error_msg"].(string); ok {
			streamErr.Message = errorMsg
		}
		return streamErr
	}

	return nil
}

// parseStreamResponse 返回格式化的可读 JSON
func parseStreamResponse(sseData string) string {
	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder
	var lastChunk map[string]interface{}

	// tool_calls 拼接：按 index 存储每个 tool_call 的片段
	toolCalls := make(map[int]map[string]interface{})

	lines := strings.Split(sseData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			continue
		}

		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		lastChunk = chunk

		// 提取 choices[0].delta.content
		if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if delta, ok := choice["delta"].(map[string]interface{}); ok {
					// 提取普通内容
					if content, ok := delta["content"].(string); ok {
						contentBuilder.WriteString(content)
					}
					// 提取推理内容
					if reasoning, ok := delta["reasoning_content"].(string); ok && reasoning != "" {
						reasoningBuilder.WriteString(reasoning)
					}
					// 提取 tool_calls
					if toolCallsDelta, ok := delta["tool_calls"].([]interface{}); ok {
						for _, tc := range toolCallsDelta {
							if tcMap, ok := tc.(map[string]interface{}); ok {
								index := 0
								if idx, ok := tcMap["index"].(float64); ok {
									index = int(idx)
								}
								// 初始化该 index 的 tool_call
								if toolCalls[index] == nil {
									toolCalls[index] = map[string]interface{}{
										"id":   "",
										"type": "function",
										"function": map[string]string{
											"name":      "",
											"arguments": "",
										},
									}
								}
								// 拼接 id
								if id, ok := tcMap["id"].(string); ok && id != "" {
									toolCalls[index]["id"] = id
								}
								// 拼接 type
								if t, ok := tcMap["type"].(string); ok && t != "" {
									toolCalls[index]["type"] = t
								}
								// 拼接 function
								if fn, ok := tcMap["function"].(map[string]interface{}); ok {
									fnMap := toolCalls[index]["function"].(map[string]string)
									if name, ok := fn["name"].(string); ok && name != "" {
										fnMap["name"] = name
									}
									if args, ok := fn["arguments"].(string); ok {
										fnMap["arguments"] += args
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 构建可读的 JSON 结果
	result := map[string]interface{}{
		"content": contentBuilder.String(),
	}

	// 如果有推理内容，也加入
	if reasoningBuilder.Len() > 0 {
		result["reasoning_content"] = reasoningBuilder.String()
	}

	// 如果有 tool_calls，按 index 顺序加入
	if len(toolCalls) > 0 {
		indices := make([]int, 0, len(toolCalls))
		for idx := range toolCalls {
			indices = append(indices, idx)
		}
		// 排序
		sort.Ints(indices)
		// 按顺序构建 tool_calls 数组
		tcArray := make([]map[string]interface{}, len(indices))
		for i, idx := range indices {
			tcArray[i] = toolCalls[idx]
		}
		result["tool_calls"] = tcArray
	}

	// 从最后一个 chunk 提取 usage 信息
	if lastChunk != nil {
		if usage, ok := lastChunk["usage"].(map[string]interface{}); ok {
			result["usage"] = usage
		}
		// 提取 model
		if model, ok := lastChunk["model"].(string); ok {
			result["model"] = model
		}
	}

	// 格式化输出
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return contentBuilder.String()
	}
	return string(jsonBytes)
}
