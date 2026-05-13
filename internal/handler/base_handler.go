package handler

import (
	"bufio"
	"bytes"
	"context"
	"unicode/utf8"
	"encoding/json"
	"fmt"
	"io"
	"llm-proxy/internal/config"
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

// requestIDKey 用于在 context 中存储请求 ID
type requestIDKey struct{}

// ProxyRequestInfo 代理请求的公共信息
type ProxyRequestInfo struct {
	Model       string
	Stream      bool
	Protocol    string // "openai" | "anthropic" | "ollama"
}

// ParseRequestFunc 解析请求体的回调，返回请求信息或错误
type ParseRequestFunc func(body []byte) (*ProxyRequestInfo, error)

// HandleRequestFunc 处理请求的回调（流式或非流式）
type HandleRequestFunc func(c *gin.Context, body []byte, startTime time.Time)

// HandleProxyRequest 代理请求的模板方法，封装公共流程
func (h *BaseHandler) HandleProxyRequest(
	c *gin.Context,
	protocol string,
	parseRequest ParseRequestFunc,
	handleStream HandleRequestFunc,
	handleNormal HandleRequestFunc,
) {
	startTime := time.Now()

	// 注入 requestID 到 context
	requestID := generateRequestID()
	ctx := contextWithRequestID(c.Request.Context(), requestID)
	c.Request = c.Request.WithContext(ctx)

	body, err := h.ReadBody(c)
	if err != nil {
		slog.Error(protocol+" request", "requestID", requestID, "status", "ERROR", "error", "failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	reqInfo, err := parseRequest(body)
	if err != nil {
		slog.Error(protocol+" request", "requestID", requestID, "status", "ERROR", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 注册活跃请求
	tracker := h.tracker
	activeReq := &ActiveRequest{
		RequestID:   requestID,
		Model:       reqInfo.Model,
		RequestBody: string(body),
		Status:      "pending",
		StartTime:   startTime,
		Protocol:    reqInfo.Protocol,
		ClientIP:    c.ClientIP(),
	}
	if reqInfo.Stream {
		activeReq.Status = "streaming"
	}
	tracker.Add(activeReq)
	defer tracker.Remove(requestID)

	if reqInfo.Stream {
		handleStream(c, body, startTime)
	} else {
		handleNormal(c, body, startTime)
	}
}

// UpstreamError 上游 API 返回的错误，携带状态码和响应体
type UpstreamError struct {
	StatusCode int
	Body       string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// BaseHandler 公共基类，封装 Handler 的公共字段和方法
type BaseHandler struct {
	proxyService   *service.ProxyService
	requestLogRepo *repository.RequestLogRepository
	httpClient     *http.Client
	cfg            *config.Config
	tracker        *ActiveRequestTracker
}

// NewBaseHandler 创建 BaseHandler 实例
func NewBaseHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository, cfg *config.Config, tracker *ActiveRequestTracker) *BaseHandler {
	transport := &http.Transport{
		DisableCompression:  true, // 禁用自动解压，SSE 流式响应不应被解压
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &BaseHandler{
		proxyService:   proxyService,
		requestLogRepo: requestLogRepo,
		httpClient: &http.Client{
			Transport: transport,
		},
		cfg:     cfg,
		tracker: tracker,
	}
}

// generateRequestID 生成请求 ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// contextWithRequestID 创建携带请求 ID 的 context
func contextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// requestIDFromContext 从 context 中获取请求 ID
func requestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// ResolveUpstreamError 解析上游错误，返回 HTTP 状态码和错误消息
func ResolveUpstreamError(err error) (statusCode int, message string) {
	statusCode = http.StatusInternalServerError
	message = err.Error()
	if upErr, ok := err.(*UpstreamError); ok {
		statusCode = upErr.StatusCode
		message = upErr.Body
	}
	return
}

// LogRequest 记录请求汇总日志
func (h *BaseHandler) LogRequest(c *gin.Context, reqBody []byte, startTime time.Time, status string, errMsg string, provider model.ProviderConfig) {
	duration := time.Since(startTime).Milliseconds()

	var reqInfo struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	json.Unmarshal(reqBody, &reqInfo)

	requestID := requestIDFromContext(c.Request.Context())

	attrs := []any{
		"requestID", requestID,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"clientIP", c.ClientIP(),
		"model", reqInfo.Model,
		"stream", reqInfo.Stream,
		"duration_ms", duration,
		"status", status,
	}
	if provider.ID != 0 {
		attrs = append(attrs, "provider", provider.Name)
	}
	if errMsg != "" {
		attrs = append(attrs, "error", errMsg)
	}

	switch status {
	case "ERROR", "FAILED":
		slog.Error("proxy request", attrs...)
	default:
		slog.Info("proxy request", attrs...)
	}
}

// GetProviderByModel 根据模型名匹配 Provider
func (h *BaseHandler) GetProviderByModel(modelName string) (model.ProviderConfig, error) {
	return h.proxyService.GetProviderByModel(modelName)
}

// GetAllProviders 获取所有 Provider
func (h *BaseHandler) GetAllProviders() ([]model.ProviderConfig, error) {
	return h.proxyService.GetAllProviders()
}

// PrepareRequestBody 准备请求体（委托给 ProxyService）
func (h *BaseHandler) PrepareRequestBody(body []byte, provider model.ProviderConfig) []byte {
	return h.proxyService.PrepareRequestBody(body, provider)
}

// buildHTTPRequest 构建公共 HTTP 请求（消除流式/非流式请求构建的重复代码）
func (h *BaseHandler) buildHTTPRequest(ctx context.Context, url string, body []byte, apiKey string, extraHeaders http.Header, isStream bool) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	if isStream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}

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

	return httpReq, nil
}

// SendStreamRequest 发送流式 HTTP 请求
func (h *BaseHandler) SendStreamRequest(ctx context.Context, url string, body []byte, apiKey string) (*http.Response, error) {
	return h.SendStreamRequestWithHeaders(ctx, url, body, apiKey, nil)
}

// SendStreamRequestWithHeaders 发送流式 HTTP 请求（支持自定义 header）
func (h *BaseHandler) SendStreamRequestWithHeaders(ctx context.Context, url string, body []byte, apiKey string, extraHeaders http.Header) (*http.Response, error) {
	httpReq, err := h.buildHTTPRequest(ctx, url, body, apiKey, extraHeaders, true)
	if err != nil {
		return nil, err
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
	httpReq, err := h.buildHTTPRequest(ctx, url, body, apiKey, extraHeaders, false)
	if err != nil {
		return nil, err
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
			retryDelay := time.Duration(attempt) * h.cfg.GetRetryDelayBase()
			slog.Debug("非流式请求重试", "url", url, "attempt", attempt, "maxRetries", maxRetries, "delay", retryDelay)
			time.Sleep(retryDelay)
		}

		respBody, err := h.SendRequest(ctx, url, body, apiKey)
		if err == nil {
			return respBody, nil
		}
		lastErr = err
		slog.Debug("非流式请求失败", "url", url, "attempt", attempt, "error", err)
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

// sanitizeBody 清理请求体/响应体，确保内容为合法 UTF-8
func sanitizeBody(body string) string {
	if body == "" {
		return ""
	}
	// 转为 []byte 检查是否为合法 UTF-8
	b := []byte(body)
	if !utf8.Valid(b) {
		// 替换无效 UTF-8 序列为 Unicode 替换字符
		b = bytes.ToValidUTF8(b, []byte("\uFFFD"))
	}
	return string(b)
}

// truncateBody 截断过长的请求体/响应体（按字节截断，回退到合法 UTF-8 边界）
func truncateBody(body string) string {
	if len(body) <= maxBodySize {
		return body
	}
	// 按字节截断到 maxBodySize，然后回退确保不在多字节字符中间截断
	end := maxBodySize
	for end > 0 && !utf8.RuneStart(body[end]) {
		end--
	}
	return body[:end] + "\n...[truncated]"
}

// SaveRequestLog 保存请求日志（清理无效 UTF-8，截断过长的请求体/响应体）
func (h *BaseHandler) SaveRequestLog(reqLog *model.RequestLog) {
	reqLog.RequestBody = sanitizeBody(truncateBody(reqLog.RequestBody))
	reqLog.ResponseBody = sanitizeBody(truncateBody(reqLog.ResponseBody))
	reqLog.ResponseContent = sanitizeBody(reqLog.ResponseContent)
	reqLog.ThinkingContent = sanitizeBody(reqLog.ThinkingContent)
	reqLog.ErrorMessage = sanitizeBody(reqLog.ErrorMessage)
	if err := h.requestLogRepo.Create(reqLog); err != nil {
		slog.Error("保存请求日志失败", "error", err)
	}
}

// CreateRequestLog 创建请求日志对象
func (h *BaseHandler) CreateRequestLog(provider model.ProviderConfig, reqBody string) *model.RequestLog {
	return &model.RequestLog{
		ProviderID:  provider.ID,
		Model:       provider.Model,
		RequestBody: reqBody,
		Status:      "error",
	}
}

// StreamRetryConfig 流式重试配置
type StreamRetryConfig struct {
	MaxRetries int
}

// DefaultStreamRetryConfig 返回基于当前配置的流式重试配置
func (h *BaseHandler) DefaultStreamRetryConfig() StreamRetryConfig {
	return StreamRetryConfig{
		MaxRetries: h.cfg.GetStreamMaxRetries(),
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

// ExecuteStreamWithRetry 执行带重试的流式请求
// 返回: 响应内容构建器, token使用量, 最后错误
func (h *BaseHandler) ExecuteStreamWithRetry(
	ctx context.Context,
	provider model.ProviderConfig,
	body []byte, // 原始请求体字节，每次重试会通过 bytes.NewReader 创建新 Reader，不会被消费
	config StreamRetryConfig,
	processor StreamLineProcessor,
) (responseBuilder strings.Builder, tokens StreamTokens, lastErr error) {
	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		if attempt > 1 {
			// 重试前清空上次的部分数据
			responseBuilder.Reset()
			tokens = StreamTokens{}

			// 重试延迟：次数 * 0.5s
			retryDelay := time.Duration(attempt) * h.cfg.GetRetryDelayBase()
			slog.Warn("流式请求重试", "provider", provider.Name, "model", provider.Model, "attempt", attempt, "maxRetries", config.MaxRetries, "delay", retryDelay)
			time.Sleep(retryDelay)
		}

		// 发送HTTP请求
		httpResp, err := h.SendStreamRequest(ctx, provider.GetRequestURL(), body, provider.APIKey)
		if err != nil {
			slog.Debug("发送HTTP请求失败", "provider", provider.Name, "attempt", attempt, "error", err)
			lastErr = err
			continue
		}

		// 读取流式响应，通过 lineChan 传递行数据
		// 使用 cancel 通知读取 goroutine 在 processor 提前退出时停止
		readCtx, readCancel := context.WithCancel(ctx)
		lineChan := make(chan string, 64)
		readDone := make(chan error, 1)

		go func() {
			readDone <- h.readStreamResponse(readCtx, httpResp, &responseBuilder, &tokens, lineChan)
		}()

		// 在独立 goroutine 中调用 processor 写客户端
		processorDone := make(chan struct{})
		go func() {
			defer close(processorDone)
			for line := range lineChan {
				// 客户端已断开，不再写
				if ctx.Err() != nil {
					return
				}
				if processor != nil {
					if stop := processor(line, &tokens); stop {
						// processor 提前退出（如收到 [DONE]），通知读取 goroutine 停止
						readCancel()
						return
					}
				}
			}
		}()

		// 等待读取完成
		readErr := <-readDone
		// 关闭 Body
		httpResp.Body.Close()
		// 等待 processor 完成
		<-processorDone

		// processor 提前退出时 readErr 可能非 nil（context 取消导致读取中断），视为成功
		if readErr == nil || readCtx.Err() != nil {
			slog.Debug("流式请求成功", "provider", provider.Name, "attempt", attempt)
			return responseBuilder, tokens, nil
		}

		slog.Debug("读取流式响应失败", "provider", provider.Name, "attempt", attempt, "error", readErr)
		lastErr = readErr
		continue
	}

	return responseBuilder, tokens, lastErr
}

// StreamError 流式响应中的错误
type StreamError struct {
	Code    interface{} `json:"code"`
	Message string      `json:"message"`
}

// readStreamResponse 读取流式响应，提取 token 使用量，通过 lineChan 传给调用方
// 当 ctx 被取消时（如 processor 提前退出），提前终止读取
func (h *BaseHandler) readStreamResponse(
	ctx context.Context,
	httpResp *http.Response,
	responseBuilder *strings.Builder,
	tokens *StreamTokens,
	lineChan chan<- string,
) error {
	defer close(lineChan)

	// bufio.Scanner 默认 MaxScanTokenSize 仅 64KB，LLM 流式响应的行
	// （如 reasoning_content、tool_calls arguments）可能远超此限制，
	// 导致 scanner.Err() 返回 bufio.ErrTooLong 而提前退出。
	// 增大缓冲区到 10MB 以支持长行。
	const maxScanBufferSize = 10 * 1024 * 1024 // 10MB
	scanner := bufio.NewScanner(httpResp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxScanBufferSize)

	for scanner.Scan() {
		// 检查是否被取消（processor 提前退出场景）
		if ctx.Err() != nil {
			return ctx.Err()
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		responseBuilder.WriteString(line + "\n")

		// 提取 token 使用量
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				lineChan <- line
				continue
			}
			var streamResp map[string]interface{}
			if err := json.Unmarshal([]byte(data), &streamResp); err == nil {
				// 检测流式响应中的错误
				if streamErr := h.detectStreamError(streamResp); streamErr != nil {
					return fmt.Errorf("stream error: code=%v, message=%s", streamErr.Code, streamErr.Message)
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

		lineChan <- line
	}
	return scanner.Err()
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
		parseContentFromChunk(chunk, &contentBuilder, &reasoningBuilder)
		parseToolCallsFromChunk(chunk, toolCalls)
	}

	return buildParsedResult(contentBuilder, reasoningBuilder, toolCalls, lastChunk)
}

// parseContentFromChunk 从 chunk 中提取文本和推理内容
func parseContentFromChunk(chunk map[string]interface{}, contentBuilder, reasoningBuilder *strings.Builder) {
	choices, ok := chunk["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return
	}
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return
	}
	if content, ok := delta["content"].(string); ok {
		contentBuilder.WriteString(content)
	}
	if reasoning, ok := delta["reasoning_content"].(string); ok && reasoning != "" {
		reasoningBuilder.WriteString(reasoning)
	}
}

// parseToolCallsFromChunk 从 chunk 中提取 tool_calls 片段并拼接
func parseToolCallsFromChunk(chunk map[string]interface{}, toolCalls map[int]map[string]interface{}) {
	choices, ok := chunk["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return
	}
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return
	}
	toolCallsDelta, ok := delta["tool_calls"].([]interface{})
	if !ok {
		return
	}
	for _, tc := range toolCallsDelta {
		tcMap, ok := tc.(map[string]interface{})
		if !ok {
			continue
		}
		index := 0
		if idx, ok := tcMap["index"].(float64); ok {
			index = int(idx)
		}
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
		if id, ok := tcMap["id"].(string); ok && id != "" {
			toolCalls[index]["id"] = id
		}
		if t, ok := tcMap["type"].(string); ok && t != "" {
			toolCalls[index]["type"] = t
		}
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

// buildParsedResult 构建格式化的解析结果
func buildParsedResult(contentBuilder, reasoningBuilder strings.Builder, toolCalls map[int]map[string]interface{}, lastChunk map[string]interface{}) string {
	result := map[string]interface{}{
		"content": contentBuilder.String(),
	}
	if reasoningBuilder.Len() > 0 {
		result["reasoning_content"] = reasoningBuilder.String()
	}
	if len(toolCalls) > 0 {
		indices := make([]int, 0, len(toolCalls))
		for idx := range toolCalls {
			indices = append(indices, idx)
		}
		sort.Ints(indices)
		tcArray := make([]map[string]interface{}, len(indices))
		for i, idx := range indices {
			tcArray[i] = toolCalls[idx]
		}
		result["tool_calls"] = tcArray
	}
	if lastChunk != nil {
		if usage, ok := lastChunk["usage"].(map[string]interface{}); ok {
			result["usage"] = usage
		}
		if m, ok := lastChunk["model"].(string); ok {
			result["model"] = m
		}
	}
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return contentBuilder.String()
	}
	return string(jsonBytes)
}

// SafeWriteSSE 写入 SSE 数据
func SafeWriteSSE(c *gin.Context, data string) {
	c.Writer.Write([]byte(data))
	c.Writer.Flush()
}

// CloseClientConnection 标记请求连接在响应完成后关闭
// 仅设置 Connection: close 响应头不够，Go 的 net/http 服务器不会因此关闭 TCP 连接
// 必须同时设置 c.Request.Close = true 才能确保连接被关闭
func CloseClientConnection(c *gin.Context) {
	c.Header("Connection", "close")
	c.Request.Close = true
}
