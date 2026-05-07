package handler

import (
	"context"
	"encoding/json"
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
func NewOllamaHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository) *OllamaHandler {
	return &OllamaHandler{
		BaseHandler: NewBaseHandler(proxyService, requestLogRepo),
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

	openAIReq := h.convertOllamaToOpenAI(ollamaReq, provider)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	ctx, cancel := context.WithTimeout(c.Request.Context(), DefaultHTTPTimeout)
	defer cancel()

	respBody, err := h.SendRequest(ctx, provider.GetRequestURL(), openAIBody, provider.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.OllamaChatResponse{
			Model:      provider.Model,
			CreatedAt:  time.Now().Format(time.RFC3339),
			Message:    model.OllamaMessage{Role: "assistant", Content: err.Error()},
			Done:       true,
			DoneReason: "error",
		})
		return
	}

	ollamaResp := h.convertOpenAIToOllama(respBody, provider.Model, startTime)
	ollamaResp.Done = true
	ollamaResp.DoneReason = "stop"

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

	openAIReq := h.convertOllamaToOpenAI(ollamaReq, provider)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	var fullContent strings.Builder

	// 使用 base_handler 的 ExecuteStreamWithRetry
	responseBuilder, tokens, lastErr := h.ExecuteStreamWithRetry(
		c.Request.Context(),
		provider,
		openAIBody,
		DefaultStreamRetryConfig(),
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

				ollamaChunk := h.convertOpenAIStreamToOllama(streamResp, provider.Model)
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

// convertOllamaToOpenAI 将 Ollama 请求转换为 OpenAI 格式
func (h *OllamaHandler) convertOllamaToOpenAI(ollamaReq *model.OllamaChatRequest, provider *model.ProviderConfig) map[string]interface{} {
	messages := make([]map[string]interface{}, len(ollamaReq.Messages))
	for i, msg := range ollamaReq.Messages {
		messages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	return map[string]interface{}{
		"model":    provider.Model,
		"messages": messages,
		"stream":   ollamaReq.Stream,
	}
}

// convertOpenAIToOllama 将 OpenAI 响应转换为 Ollama 格式
func (h *OllamaHandler) convertOpenAIToOllama(openAIResp []byte, modelName string, startTime time.Time) model.OllamaChatResponse {
	var resp map[string]interface{}
	if err := json.Unmarshal(openAIResp, &resp); err != nil {
		return model.OllamaChatResponse{
			Model:      modelName,
			CreatedAt:  time.Now().Format(time.RFC3339),
			Message:    model.OllamaMessage{Role: "assistant", Content: string(openAIResp)},
			Done:       true,
			DoneReason: "stop",
		}
	}

	content := ""
	var toolCalls []model.OllamaToolCall

	if choices, ok := resp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if c, ok := msg["content"].(string); ok {
					content = c
				}
				if tc, ok := msg["tool_calls"].([]interface{}); ok {
					for _, t := range tc {
						if tcMap, ok := t.(map[string]interface{}); ok {
							if fn, ok := tcMap["function"].(map[string]interface{}); ok {
								name, _ := fn["name"].(string)
								args, _ := fn["arguments"].(string)
								if name != "" {
									toolCalls = append(toolCalls, model.OllamaToolCall{
										Function: model.OllamaToolCallFunction{
											Name:      name,
											Arguments: args,
										},
									})
								}
							}
						}
					}
				}
			}
		}
	}

	inputTokens, outputTokens, _, _ := h.ExtractUsage(resp)

	return model.OllamaChatResponse{
		Model:     modelName,
		CreatedAt: time.Now().Format(time.RFC3339),
		Message: model.OllamaMessage{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
		},
		Done:            true,
		DoneReason:      "stop",
		TotalDuration:   time.Since(startTime).Nanoseconds(),
		PromptEvalCount: inputTokens,
		EvalCount:       outputTokens,
	}
}

// convertOpenAIStreamToOllama 将 OpenAI 流式响应转换为 Ollama 格式
func (h *OllamaHandler) convertOpenAIStreamToOllama(streamResp map[string]interface{}, modelName string) model.OllamaChatResponse {
	content := ""

	if choices, ok := streamResp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if delta, ok := choice["delta"].(map[string]interface{}); ok {
				if c, ok := delta["content"].(string); ok {
					content = c
				}
			}
		}
	}

	return model.OllamaChatResponse{
		Model:     modelName,
		CreatedAt: time.Now().Format(time.RFC3339),
		Message: model.OllamaMessage{
			Role:    "assistant",
			Content: content,
		},
		Done: false,
	}
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
