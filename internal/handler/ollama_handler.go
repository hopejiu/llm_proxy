package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"llm-proxy/internal/converter"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// OllamaHandler Ollama API 适配器，复用 ProxyService
type OllamaHandler struct {
	*BaseHandler // 组合基类
}

// NewOllamaHandler 创建 Ollama 适配器
func NewOllamaHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository, cfg HandlerConfig, tracker *ActiveRequestTracker) *OllamaHandler {
	return &OllamaHandler{
		BaseHandler: NewBaseHandler(proxyService, requestLogRepo, cfg, tracker),
	}
}

// Chat 处理 /api/chat 请求
func (h *OllamaHandler) Chat(c *gin.Context) {
	h.HandleProxyRequest(c, "ollama",
		func(body []byte) (*ProxyRequestInfo, error) {
			var ollamaReq model.OllamaChatRequest
			if err := json.Unmarshal(body, &ollamaReq); err != nil {
				return nil, fmt.Errorf("invalid request body")
			}
			return &ProxyRequestInfo{Model: ollamaReq.Model, Stream: ollamaReq.Stream, Protocol: "ollama"}, nil
		},
		func(c *gin.Context, body []byte, startTime time.Time) {
			h.handleStreamChat(c, body, startTime)
		},
		func(c *gin.Context, body []byte, startTime time.Time) {
			h.handleNonStreamChat(c, body, startTime)
		},
	)
}

// handleNonStreamChat 处理非流式聊天请求
func (h *OllamaHandler) handleNonStreamChat(c *gin.Context, body []byte, startTime time.Time) {
	requestID := requestIDFromContext(c.Request.Context())

	var ollamaReq model.OllamaChatRequest
	json.Unmarshal(body, &ollamaReq)

	provider, err := h.GetProviderByModel(ollamaReq.Model)
	if err != nil {
		slog.Error("ollama request", "requestID", requestID, "model", ollamaReq.Model, "status", "FAILED", "error", err.Error())
		c.JSON(http.StatusInternalServerError, model.OllamaChatResponse{
			Model:      ollamaReq.Model,
			CreatedAt:  time.Now().Format(time.RFC3339),
			Message:    model.OllamaMessage{Role: "assistant", Content: err.Error()},
			Done:       true,
			DoneReason: "error",
		})
		return
	}

	// 更新活跃请求的 Provider 信息
	h.tracker.UpdateProvider(requestID, provider.ID, provider.Name)

	openAIReq := converter.OllamaToOpenAI(&ollamaReq, provider.Model)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.config.HTTPTimeout)
	defer cancel()

	respBody, err := h.SendRequest(ctx, provider.GetRequestURL(), openAIBody, provider.APIKey)
	if err != nil {
		statusCode, errMsg := ResolveUpstreamError(err)
		slog.Error("ollama request", "requestID", requestID, "provider", provider.Name, "model", provider.Model, "status", "FAILED", "duration_ms", time.Since(startTime).Milliseconds(), "error", errMsg)
		c.JSON(statusCode, model.OllamaChatResponse{
			Model:      provider.Model,
			CreatedAt:  time.Now().Format(time.RFC3339),
			Message:    model.OllamaMessage{Role: "assistant", Content: errMsg},
			Done:       true,
			DoneReason: "error",
		})
		return
	}

	ollamaResp := converter.OpenAIToOllama(respBody, provider.Model)
	ollamaResp.CreatedAt = time.Now().Format(time.RFC3339)
	ollamaResp.Done = true
	ollamaResp.DoneReason = "stop"
	ollamaResp.TotalDuration = time.Since(startTime).Nanoseconds()

	// 提取 token 用量
	var rawResp map[string]interface{}
	if json.Unmarshal(respBody, &rawResp) == nil {
		inputTokens, outputTokens, _, _ := h.ExtractUsage(rawResp)
		ollamaResp.PromptEvalCount = inputTokens
		ollamaResp.EvalCount = outputTokens
	}

	reqLog := &model.RequestLog{
		ProviderID:      provider.ID,
		Model:           provider.Model,
		RequestBody:     string(openAIBody),
		ResponseBody:    string(respBody),
		ResponseContent: ollamaResp.Message.Content,
		InputTokens:     ollamaResp.PromptEvalCount,
		OutputTokens:    ollamaResp.EvalCount,
		TotalTokens:     ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		Status:          "success",
		Duration:        time.Since(startTime).Milliseconds(),
	}
	h.SaveRequestLog(reqLog)
	slog.Info("ollama request", "requestID", requestID, "provider", provider.Name, "model", provider.Model, "status", "SUCCESS", "duration_ms", time.Since(startTime).Milliseconds())

	c.JSON(http.StatusOK, ollamaResp)
}

// handleStreamChat 处理流式聊天请求（支持超时重试）
func (h *OllamaHandler) handleStreamChat(c *gin.Context, body []byte, startTime time.Time) {
	requestID := requestIDFromContext(c.Request.Context())

	var ollamaReq model.OllamaChatRequest
	json.Unmarshal(body, &ollamaReq)

	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	provider, err := h.GetProviderByModel(ollamaReq.Model)
	if err != nil {
		slog.Error("ollama request", "requestID", requestID, "model", ollamaReq.Model, "status", "FAILED", "error", err.Error())
		h.sendOllamaStreamError(c, err.Error())
		return
	}

	// 更新活跃请求的 Provider 信息
	h.tracker.UpdateProvider(requestID, provider.ID, provider.Name)

	openAIReq := converter.OllamaToOpenAI(&ollamaReq, provider.Model)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	var fullContent strings.Builder
	tracker := h.tracker

	// 使用 base_handler 的 ExecuteStreamWithRetry
	responseBuilder, tokens, lastErr := h.ExecuteStreamWithRetry(
		c.Request.Context(),
		provider,
		openAIBody,
		h.DefaultStreamRetryConfig(),
		func(line string, _ *StreamTokens) bool {
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					return false
				}

				var streamResp map[string]interface{}
				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					return false
				}

				ollamaChunk := converter.OpenAIStreamToOllamaChunk(streamResp, provider.Model)
				fullContent.WriteString(ollamaChunk.Message.Content)
				tracker.AppendResponse(requestID, ollamaChunk.Message.Content)

				// 追踪工具调用
				if choices, ok := streamResp["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						if delta, ok := choice["delta"].(map[string]interface{}); ok {
							trackToolCallsFromDelta(delta, requestID, tracker)
						}
					}
				}

				chunkBytes, _ := json.Marshal(ollamaChunk)
				c.Writer.Write(chunkBytes)
				c.Writer.Write([]byte("\n"))
				c.Writer.Flush()
			}
			return false
		},
	)

	if lastErr != nil {
		slog.Error("ollama request", "requestID", requestID, "provider", provider.Name, "model", provider.Model, "status", "FAILED", "duration_ms", time.Since(startTime).Milliseconds(), "error", lastErr.Error())
		h.sendOllamaStreamError(c, lastErr.Error())
		return
	}

	finalResp := model.OllamaChatResponse{
		Model:           provider.Model,
		CreatedAt:       time.Now().Format(time.RFC3339),
		Message:         model.OllamaMessage{Role: "assistant", Content: ""},
		Done:            true,
		DoneReason:      "stop",
		TotalDuration:   time.Since(startTime).Nanoseconds(),
		PromptEvalCount: tokens.InputTokens,
		EvalCount:       tokens.OutputTokens,
	}
	finalBytes, _ := json.Marshal(finalResp)
	c.Writer.Write(finalBytes)
	c.Writer.Write([]byte("\n"))
	c.Writer.Flush()

	reqLog := &model.RequestLog{
		ProviderID:      provider.ID,
		Model:           provider.Model,
		RequestBody:     string(openAIBody),
		ResponseBody:    responseBuilder.String(),
		ResponseContent: fullContent.String(),
		InputTokens:     tokens.InputTokens,
		OutputTokens:    tokens.OutputTokens,
		TotalTokens:     tokens.InputTokens + tokens.OutputTokens,
		Status:          "success",
		Duration:        time.Since(startTime).Milliseconds(),
	}
	h.SaveRequestLog(reqLog)
	slog.Info("ollama request", "requestID", requestID, "provider", provider.Name, "model", provider.Model, "status", "STREAM_END", "duration_ms", time.Since(startTime).Milliseconds())
}

// Tags 处理 /api/tags 请求
func (h *OllamaHandler) Tags(c *gin.Context) {
	providers, err := h.GetAllProviders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var models []model.OllamaModelInfo
	for _, provider := range providers {
		for _, name := range provider.GetModelNames() {
			models = append(models, model.OllamaModelInfo{
				Name:       name,
				Model:      name,
				ModifiedAt: provider.UpdatedAt.Format(time.RFC3339),
				Size:       0,
				Digest:     "",
				Details: model.OllamaModelDetails{
					Format: "api",
					Family: "llm-proxy",
				},
			})
		}
	}

	c.JSON(http.StatusOK, model.OllamaTagsResponse{Models: models})
}

// sendOllamaStreamError 发送流式错误响应
func (h *OllamaHandler) sendOllamaStreamError(c *gin.Context, errMsg string) {
	errResp := model.OllamaChatResponse{
		Model:      "",
		CreatedAt:  time.Now().Format(time.RFC3339),
		Message:    model.OllamaMessage{Role: "assistant", Content: errMsg},
		Done:       true,
		DoneReason: "error",
	}
	errBytes, _ := json.Marshal(errResp)
	c.Writer.Write(errBytes)
	c.Writer.Write([]byte("\n"))
	c.Writer.Flush()
}
