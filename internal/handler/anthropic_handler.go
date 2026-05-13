package handler

import (
	"encoding/json"
	"fmt"
	"llm-proxy/internal/config"
	"llm-proxy/internal/converter"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AnthropicHandler Anthropic API 适配器，复用 BaseHandler
type AnthropicHandler struct {
	*BaseHandler
}

// NewAnthropicHandler 创建 Anthropic 适配器
func NewAnthropicHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository, cfg *config.Config, tracker *ActiveRequestTracker) *AnthropicHandler {
	return &AnthropicHandler{
		BaseHandler: NewBaseHandler(proxyService, requestLogRepo, cfg, tracker),
	}
}

// Messages 处理 /anthropic/v1/messages 请求
func (h *AnthropicHandler) Messages(c *gin.Context) {
	h.HandleProxyRequest(c, "anthropic",
		func(body []byte) (*ProxyRequestInfo, error) {
			var anthropicReq model.AnthropicMessagesRequest
			if err := json.Unmarshal(body, &anthropicReq); err != nil {
				return nil, fmt.Errorf("invalid request body")
			}
			return &ProxyRequestInfo{Model: anthropicReq.Model, Stream: anthropicReq.Stream, Protocol: "anthropic"}, nil
		},
		func(c *gin.Context, body []byte, startTime time.Time) {
			h.handleStreamMessages(c, body, startTime)
		},
		func(c *gin.Context, body []byte, startTime time.Time) {
			h.handleNonStreamMessages(c, body, startTime)
		},
	)
}

// handleNonStreamMessages 处理非流式请求（协议转换）
func (h *AnthropicHandler) handleNonStreamMessages(c *gin.Context, body []byte, startTime time.Time) {
	requestID := requestIDFromContext(c.Request.Context())

	var anthropicReq model.AnthropicMessagesRequest
	json.Unmarshal(body, &anthropicReq)

	provider, err := h.GetProviderByModel(anthropicReq.Model)
	if err != nil {
		slog.Error("anthropic request", "requestID", requestID, "model", anthropicReq.Model, "status", "FAILED", "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": err.Error()},
		})
		return
	}

	h.tracker.UpdateProvider(requestID, provider.ID, provider.Name)

	openAIReq := converter.AnthropicToOpenAI(&anthropicReq, provider.Model)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	respBody, err := h.SendRequest(c.Request.Context(), provider.GetRequestURL(), openAIBody, provider.APIKey)
	if err != nil {
		statusCode, errMsg := ResolveUpstreamError(err)
		slog.Error("anthropic request", "requestID", requestID, "provider", provider.Name, "model", provider.Model, "status", "FAILED", "duration_ms", time.Since(startTime).Milliseconds(), "error", errMsg)
		c.JSON(statusCode, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": errMsg},
		})
		return
	}

	anthropicResp := converter.OpenAIToAnthropic(respBody, provider.Model)

	// 从 OpenAI 响应中提取 reasoning_content
	var thinkingContent string
	var rawResp map[string]interface{}
	if json.Unmarshal(respBody, &rawResp) == nil {
		if choices, ok := rawResp["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if msg, ok := choice["message"].(map[string]interface{}); ok {
					if reasoning, ok := msg["reasoning_content"].(string); ok && reasoning != "" {
						thinkingContent = reasoning
					}
				}
			}
		}
	}

	reqLog := &model.RequestLog{
		ProviderID:      provider.ID,
		Model:           provider.Model,
		RequestBody:     string(openAIBody),
		ResponseBody:    string(respBody),
		ResponseContent: converter.ExtractTextFromAnthropicContent(anthropicResp.Content),
		ThinkingContent: thinkingContent,
		InputTokens:     anthropicResp.Usage.InputTokens,
		OutputTokens:    anthropicResp.Usage.OutputTokens,
		TotalTokens:     anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		CachedTokens:    anthropicResp.Usage.CacheReadInputTokens,
		Status:          "success",
		Duration:        time.Since(startTime).Milliseconds(),
	}
	h.SaveRequestLog(reqLog)
	slog.Info("anthropic request", "requestID", requestID, "provider", provider.Name, "model", provider.Model, "status", "SUCCESS", "duration_ms", time.Since(startTime).Milliseconds())

	// 非流式请求完成后，将响应内容追加到 tracker
	if reqLog.ResponseContent != "" {
		h.tracker.AppendResponse(requestID, reqLog.ResponseContent)
	}
	if reqLog.ThinkingContent != "" {
		h.tracker.AppendResponse(requestID, reqLog.ThinkingContent)
	}

	c.JSON(http.StatusOK, anthropicResp)
}

// handleStreamMessages 处理流式请求（OpenAI 类型 Provider，需要协议转换）
func (h *AnthropicHandler) handleStreamMessages(c *gin.Context, body []byte, startTime time.Time) {
	requestID := requestIDFromContext(c.Request.Context())

	var anthropicReq model.AnthropicMessagesRequest
	json.Unmarshal(body, &anthropicReq)

	provider, err := h.GetProviderByModel(anthropicReq.Model)
	if err != nil {
		slog.Error("anthropic request", "requestID", requestID, "model", anthropicReq.Model, "status", "FAILED", "error", err.Error())
		h.sendAnthropicSSEError(c, err.Error())
		return
	}

	h.tracker.UpdateProvider(requestID, provider.ID, provider.Name)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	CloseClientConnection(c)
	
	openAIReq := converter.AnthropicToOpenAI(&anthropicReq, provider.Model)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	state := newAnthropicStreamState(h, c, provider, requestID)

	responseBuilder, tokens, lastErr := h.ExecuteStreamWithRetry(
		c.Request.Context(),
		provider,
		openAIBody,
		h.DefaultStreamRetryConfig(),
		func(line string, currentTokens *StreamTokens) bool {
			return state.processLine(line, currentTokens)
		},
	)

	if lastErr != nil {
		slog.Error("anthropic request", "requestID", requestID, "provider", provider.Name, "model", provider.Model, "status", "FAILED", "duration_ms", time.Since(startTime).Milliseconds(), "error", lastErr.Error())
		h.sendAnthropicSSEError(c, lastErr.Error())
		// 超时/错误时也必须关闭流状态并发送 [DONE]，否则客户端会一直挂起等待
		state.finalize(&tokens)
		SafeWriteSSE(c, "data: [DONE]\n\n")
		return
	}

	// 确保发送 [DONE] 标记（某些上游可能不发送，导致客户端一直等待）
	SafeWriteSSE(c, "data: [DONE]\n\n")

	// 如果流结束时没有收到 finish_reason，补发结束事件
	state.finalize(&tokens)

	reqLog := &model.RequestLog{
		ProviderID:      provider.ID,
		Model:           provider.Model,
		RequestBody:     string(openAIBody),
		ResponseBody:    responseBuilder.String(),
		ResponseContent: state.fullContent.String(),
		ThinkingContent: state.thinkingContent.String(),
		InputTokens:     tokens.InputTokens,
		OutputTokens:    tokens.OutputTokens,
		TotalTokens:     tokens.TotalTokens,
		CachedTokens:    tokens.CachedTokens,
		Status:          "success",
		Duration:        time.Since(startTime).Milliseconds(),
	}

	h.SaveRequestLog(reqLog)

	slog.Info("anthropic request", "requestID", requestID, "provider", provider.Name, "model", provider.Model, "status", "STREAM_END", "duration_ms", time.Since(startTime).Milliseconds())
}

// Models 处理 /anthropic/v1/models 请求
func (h *AnthropicHandler) Models(c *gin.Context) {
	providers, err := h.GetAllProviders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": err.Error()},
		})
		return
	}

	var models []model.AnthropicModelInfo
	for _, provider := range providers {
		for _, name := range provider.GetModelNames() {
			models = append(models, model.AnthropicModelInfo{
				ID:          name,
				Type:        "model",
				DisplayName: provider.Name,
				CreatedAt:   provider.UpdatedAt.Format(time.RFC3339),
			})
		}
	}

	c.JSON(http.StatusOK, model.AnthropicModelsResponse{
		Data:    models,
		HasMore: false,
	})
}

// writeSSE 写入 SSE 事件
func (h *AnthropicHandler) writeSSE(c *gin.Context, eventType string, data interface{}) {
	dataBytes, _ := json.Marshal(data)
	c.Writer.Write([]byte("event: " + eventType + "\n"))
	c.Writer.Write([]byte("data: " + string(dataBytes) + "\n\n"))
	c.Writer.Flush()
}

// sendAnthropicSSEError 发送 Anthropic 格式的 SSE 错误
func (h *AnthropicHandler) sendAnthropicSSEError(c *gin.Context, errMsg string) {
	errData, _ := json.Marshal(map[string]interface{}{
		"type":  "error",
		"error": map[string]interface{}{"type": "api_error", "message": errMsg},
	})
	c.Writer.Write([]byte("event: error\n"))
	c.Writer.Write([]byte("data: " + string(errData) + "\n\n"))
	c.Writer.Flush()
}

