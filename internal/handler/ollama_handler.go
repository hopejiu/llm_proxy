package handler

import (
	"context"
	"encoding/json"
	"llm-proxy/internal/converter"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
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
func NewOllamaHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository, cfg HandlerConfig) *OllamaHandler {
	return &OllamaHandler{
		BaseHandler: NewBaseHandler(proxyService, requestLogRepo, cfg),
	}
}

// Chat 处理 /api/chat 请求
func (h *OllamaHandler) Chat(c *gin.Context) {
	startTime := time.Now()

	body, err := h.ReadBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.OllamaChatResponse{
			Model:      "",
			CreatedAt:  time.Now().Format(time.RFC3339),
			Message:    model.OllamaMessage{Role: "assistant", Content: ""},
			Done:       true,
			DoneReason: "error",
		})
		return
	}

	var ollamaReq model.OllamaChatRequest
	if err := json.Unmarshal(body, &ollamaReq); err != nil {
		c.JSON(http.StatusBadRequest, model.OllamaChatResponse{
			Model:      "",
			CreatedAt:  time.Now().Format(time.RFC3339),
			Message:    model.OllamaMessage{Role: "assistant", Content: "invalid request"},
			Done:       true,
			DoneReason: "error",
		})
		return
	}

	if ollamaReq.Stream {
		h.handleStreamChat(c, &ollamaReq, startTime)
	} else {
		h.handleNonStreamChat(c, &ollamaReq, startTime)
	}
}

// handleNonStreamChat 处理非流式聊天请求
func (h *OllamaHandler) handleNonStreamChat(c *gin.Context, ollamaReq *model.OllamaChatRequest, startTime time.Time) {
	provider, err := h.GetProviderByModel(ollamaReq.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.OllamaChatResponse{
			Model:      ollamaReq.Model,
			CreatedAt:  time.Now().Format(time.RFC3339),
			Message:    model.OllamaMessage{Role: "assistant", Content: err.Error()},
			Done:       true,
			DoneReason: "error",
		})
		return
	}

	openAIReq := converter.OllamaToOpenAI(ollamaReq, provider.Model)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.config.HTTPTimeout)
	defer cancel()

	respBody, err := h.SendRequest(ctx, provider.GetRequestURL(), openAIBody, provider.APIKey)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errMsg := err.Error()
		if upErr, ok := err.(*UpstreamError); ok {
			statusCode = upErr.StatusCode
			errMsg = upErr.Body
		}
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

	c.JSON(http.StatusOK, ollamaResp)
}

// handleStreamChat 处理流式聊天请求（支持超时重试）
func (h *OllamaHandler) handleStreamChat(c *gin.Context, ollamaReq *model.OllamaChatRequest, startTime time.Time) {
	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	provider, err := h.GetProviderByModel(ollamaReq.Model)
	if err != nil {
		h.sendOllamaStreamError(c, err.Error())
		return
	}

	openAIReq := converter.OllamaToOpenAI(ollamaReq, provider.Model)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	var fullContent strings.Builder

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

				chunkBytes, _ := json.Marshal(ollamaChunk)
				c.Writer.Write(chunkBytes)
				c.Writer.Write([]byte("\n"))
				c.Writer.Flush()
			}
			return false
		},
	)

	if lastErr != nil {
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
