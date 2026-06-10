package handler

import (
	"encoding/json"
	"fmt"
	"github.com/wanglejiu/llm-proxy/internal/config"
	"github.com/wanglejiu/llm-proxy/internal/converter"
	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/repository"
	"github.com/wanglejiu/llm-proxy/internal/service"
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
func NewOllamaHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository, cfg *config.Config, tracker *ActiveRequestTracker) *OllamaHandler {
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
	sessionID := sessionIDFromContext(c.Request.Context())

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

	resolvedModel := provider.ResolveModel(ollamaReq.Model)

	openAIReq := converter.OllamaToOpenAI(&ollamaReq, resolvedModel)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	respBody, err := h.SendRequest(c.Request.Context(), provider.GetRequestURL(), openAIBody, provider.APIKey)
	if err != nil {
		statusCode, errMsg := ResolveUpstreamError(err)
		slog.Error("ollama request", "requestID", requestID, "provider", provider.Name, "model", resolvedModel, "status", "FAILED", "duration_ms", time.Since(startTime).Milliseconds(), "error", errMsg)
		c.JSON(statusCode, model.OllamaChatResponse{
			Model:      resolvedModel,
			CreatedAt:  time.Now().Format(time.RFC3339),
			Message:    model.OllamaMessage{Role: "assistant", Content: errMsg},
			Done:       true,
			DoneReason: "error",
		})
		return
	}

	ollamaResp := converter.OpenAIToOllama(respBody, resolvedModel)
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
		Model:           resolvedModel,
		RequestBody:     string(openAIBody),
		ResponseBody:    string(respBody),
		ResponseContent: ollamaResp.Message.Content,
		InputTokens:     ollamaResp.PromptEvalCount,
		OutputTokens:    ollamaResp.EvalCount,
		TotalTokens:     ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		Status:          "success",
		Duration:        time.Since(startTime).Milliseconds(),
	}
	h.SaveRequestLog(reqLog, sessionID)
	slog.Info("ollama request", "requestID", requestID, "provider", provider.Name, "model", resolvedModel, "status", "SUCCESS", "duration_ms", time.Since(startTime).Milliseconds())

	// 非流式请求完成后，将响应内容追加到 tracker
	if reqLog.ResponseContent != "" {
		h.tracker.AppendResponse(requestID, reqLog.ResponseContent)
	}

	// 仅在新建会话时注入会话标记
	if isNewSession(c.Request.Context()) && sessionID > 0 {
		ollamaJSON, _ := json.Marshal(ollamaResp)
		modifiedJSON := InjectSessionMarkIntoResponse(ollamaJSON, sessionID, "ollama")
		json.Unmarshal(modifiedJSON, &ollamaResp)
	}

	c.JSON(http.StatusOK, ollamaResp)
}

// handleStreamChat 处理流式聊天请求（支持超时重试）
func (h *OllamaHandler) handleStreamChat(c *gin.Context, body []byte, startTime time.Time) {
	requestID := requestIDFromContext(c.Request.Context())
	sessionID := sessionIDFromContext(c.Request.Context())

	var ollamaReq model.OllamaChatRequest
	json.Unmarshal(body, &ollamaReq)

	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-cache")
	CloseClientConnection(c)

	provider, err := h.GetProviderByModel(ollamaReq.Model)
	if err != nil {
		slog.Error("ollama request", "requestID", requestID, "model", ollamaReq.Model, "status", "FAILED", "error", err.Error())
		h.sendOllamaStreamError(c, err.Error())
		return
	}

	// 更新活跃请求的 Provider 信息
	h.tracker.UpdateProvider(requestID, provider.ID, provider.Name)

	resolvedModel := provider.ResolveModel(ollamaReq.Model)

	openAIReq := converter.OllamaToOpenAI(&ollamaReq, resolvedModel)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	var fullContent strings.Builder
	tracker := h.tracker

	var ollamaSessionInjected bool // 防重复注入
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
					// 在 [DONE] 之前写入独立会话标记行
					WriteStreamSessionMark(c.Writer, c.Request.Context(), requestID, sessionID, &ollamaSessionInjected)
					return true // 收到 [DONE]，停止处理
				}

				var streamResp map[string]interface{}
				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					return false
				}

				ollamaChunk := converter.OpenAIStreamToOllamaChunk(streamResp, resolvedModel)
				fullContent.WriteString(ollamaChunk.Message.Content)
				tracker.AppendResponse(requestID, ollamaChunk.Message.Content)

				// 追踪工具调用
				deltaResult := converter.ExtractDeltaFromChunk(streamResp)
				if len(deltaResult.ToolCallsDelta) > 0 {
					trackToolCallsFromDelta(deltaResult.ToolCallsDelta, requestID, tracker)
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
		slog.Error("ollama request", "requestID", requestID, "provider", provider.Name, "model", resolvedModel, "status", "FAILED", "duration_ms", time.Since(startTime).Milliseconds(), "error", lastErr.Error())
		h.sendOllamaStreamError(c, lastErr.Error())
		// 超时/错误时也必须发送 done:true 的最终消息，否则客户端会一直挂起等待
		finalResp := model.OllamaChatResponse{
			Model:      resolvedModel,
			CreatedAt:  time.Now().Format(time.RFC3339),
			Done:       true,
			DoneReason: "error",
		}
		finalBytes, _ := json.Marshal(finalResp)
		SafeWriteSSE(c, string(finalBytes)+"\n")
		return
	}

	// 在 finalResp 的 message.content 中注入会话标记
	// 仅当流式 processor 中尚未注入时才在 finalResp 注入，防重复
	suffix := ""
	if !ollamaSessionInjected && isNewSession(c.Request.Context()) && sessionID > 0 {
		suffix = service.BuildSessionSuffix(sessionID)
	}

	finalResp := model.OllamaChatResponse{
		Model:           resolvedModel,
		CreatedAt:       time.Now().Format(time.RFC3339),
		Message:         model.OllamaMessage{Role: "assistant", Content: suffix},
		Done:            true,
		DoneReason:      "stop",
		TotalDuration:   time.Since(startTime).Nanoseconds(),
		PromptEvalCount: tokens.InputTokens,
		EvalCount:       tokens.OutputTokens,
	}
	finalBytes, _ := json.Marshal(finalResp)
	SafeWriteSSE(c, string(finalBytes)+"\n")

	ollamaFullContent := fullContent.String()
	if ollamaSessionInjected {
		ollamaFullContent += service.BuildSessionSuffix(sessionID)
	}

	reqLog := &model.RequestLog{
		ProviderID:      provider.ID,
		Model:           resolvedModel,
		RequestBody:     string(openAIBody),
		ResponseBody:    responseBuilder.String(),
		ResponseContent: ollamaFullContent,
		InputTokens:     tokens.InputTokens,
		OutputTokens:    tokens.OutputTokens,
		TotalTokens:     tokens.InputTokens + tokens.OutputTokens,
		Status:          "success",
		Duration:        time.Since(startTime).Milliseconds(),
	}

	h.SaveRequestLog(reqLog, sessionID)

	slog.Info("ollama request", "requestID", requestID, "provider", provider.Name, "model", resolvedModel, "status", "STREAM_END", "duration_ms", time.Since(startTime).Milliseconds())
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
