package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AnthropicHandler Anthropic API 适配器，复用 BaseHandler
type AnthropicHandler struct {
	*BaseHandler
}

// NewAnthropicHandler 创建 Anthropic 适配器
func NewAnthropicHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository) *AnthropicHandler {
	return &AnthropicHandler{
		BaseHandler: NewBaseHandler(proxyService, requestLogRepo),
	}
}

// Messages 处理 /anthropic/v1/messages 请求
func (h *AnthropicHandler) Messages(c *gin.Context) {
	startTime := time.Now()

	body, err := h.ReadBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"type":  "error",
			"error": gin.H{"type": "invalid_request_error", "message": err.Error()},
		})
		return
	}

	var anthropicReq model.AnthropicMessagesRequest
	if err := json.Unmarshal(body, &anthropicReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"type":  "error",
			"error": gin.H{"type": "invalid_request_error", "message": "invalid request body"},
		})
		return
	}

	if anthropicReq.Stream {
		h.handleStreamMessages(c, &anthropicReq, startTime)
	} else {
		h.handleNonStreamMessages(c, &anthropicReq, startTime)
	}
}

// handleNonStreamMessages 处理非流式请求
func (h *AnthropicHandler) handleNonStreamMessages(c *gin.Context, anthropicReq *model.AnthropicMessagesRequest, startTime time.Time) {
	provider, err := h.GetProvider()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": err.Error()},
		})
		return
	}

	openAIReq := h.convertAnthropicToOpenAI(anthropicReq, provider)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	ctx, cancel := context.WithTimeout(c.Request.Context(), DefaultHTTPTimeout)
	defer cancel()

	respBody, err := h.SendRequest(ctx, provider.BaseURL, openAIBody, provider.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": err.Error()},
		})
		return
	}

	anthropicResp := h.convertOpenAIToAnthropic(respBody, provider.Model)

	reqLog := &model.RequestLog{
		ProviderID:      provider.ID,
		Model:           provider.Model,
		RequestBody:     string(openAIBody),
		ResponseBody:    string(respBody),
		ResponseContent: extractTextFromAnthropicContent(anthropicResp.Content),
		InputTokens:     anthropicResp.Usage.InputTokens,
		OutputTokens:    anthropicResp.Usage.OutputTokens,
		TotalTokens:     anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		CachedTokens:    anthropicResp.Usage.CacheReadInputTokens,
		Status:          "success",
		Duration:        time.Since(startTime).Milliseconds(),
	}
	h.SaveRequestLog(reqLog)

	c.JSON(http.StatusOK, anthropicResp)
}

// handleStreamMessages 处理流式请求（支持超时重试）
func (h *AnthropicHandler) handleStreamMessages(c *gin.Context, anthropicReq *model.AnthropicMessagesRequest, startTime time.Time) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	provider, err := h.GetProvider()
	if err != nil {
		h.sendAnthropicSSEError(c, err.Error())
		return
	}

	openAIReq := h.convertAnthropicToOpenAI(anthropicReq, provider)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	// 流式状态
	msgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	var fullContent strings.Builder
	contentBlockIndex := 0
	messageStarted := false
	contentBlockStarted := false

	// 使用 base_handler 的 ExecuteStreamWithRetry
	responseBuilder, tokens, lastErr := h.ExecuteStreamWithRetry(
		c.Request.Context(),
		provider,
		openAIBody,
		DefaultStreamRetryConfig(),
		func(line string, currentTokens *StreamTokens) bool {
			if !strings.HasPrefix(line, "data: ") {
				return false
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return false
			}

			var streamResp map[string]interface{}
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				return false
			}

			// 提取 delta content
			deltaContent := ""
			if choices, ok := streamResp["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if delta, ok := choice["delta"].(map[string]interface{}); ok {
						if content, ok := delta["content"].(string); ok {
							deltaContent = content
						}
					}
					// 检查 finish_reason
					if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" && finishReason != "null" {
						// 结束当前内容块
						if contentBlockStarted {
							h.writeSSE(c, "content_block_stop", map[string]interface{}{
								"type":  "content_block_stop",
								"index": contentBlockIndex,
							})
							contentBlockStarted = false
						}

						// 发送 message_delta
						stopReason := "end_turn"
						if finishReason == "tool_calls" {
							stopReason = "tool_use"
						} else if finishReason == "length" {
							stopReason = "max_tokens"
						}

						h.writeSSE(c, "message_delta", map[string]interface{}{
							"type": "message_delta",
							"delta": map[string]interface{}{
								"stop_reason": stopReason,
							},
							"usage": map[string]interface{}{
								"output_tokens": currentTokens.OutputTokens,
							},
						})

						// 发送 message_stop
						h.writeSSE(c, "message_stop", map[string]interface{}{
							"type": "message_stop",
						})
						return false
					}
				}
			}

			if deltaContent == "" {
				return false
			}

			// 首次收到内容，发送 message_start
			if !messageStarted {
				h.writeSSE(c, "message_start", map[string]interface{}{
					"type": "message_start",
					"message": map[string]interface{}{
						"id":   msgID,
						"type": "message",
						"role": "assistant",
						"content": []interface{}{},
						"model": provider.Model,
						"usage": map[string]interface{}{
							"input_tokens":  currentTokens.InputTokens,
							"output_tokens": 0,
						},
					},
				})
				messageStarted = true
			}

			// 开始第一个内容块
			if !contentBlockStarted {
				h.writeSSE(c, "content_block_start", map[string]interface{}{
					"type":  "content_block_start",
					"index": contentBlockIndex,
					"content_block": map[string]interface{}{
						"type": "text",
						"text": "",
					},
				})
				contentBlockStarted = true
			}

			// 发送内容增量
			fullContent.WriteString(deltaContent)
			h.writeSSE(c, "content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": contentBlockIndex,
				"delta": map[string]interface{}{
					"type": "text_delta",
					"text": deltaContent,
				},
			})
			return false
		},
	)

	if lastErr != nil {
		h.sendAnthropicSSEError(c, lastErr.Error())
		return
	}

	// 如果流结束时没有收到 finish_reason，补发结束事件
	if contentBlockStarted {
		h.writeSSE(c, "content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": contentBlockIndex,
		})
	}
	if messageStarted {
		h.writeSSE(c, "message_delta", map[string]interface{}{
			"type": "message_delta",
			"delta": map[string]interface{}{
				"stop_reason": "end_turn",
			},
			"usage": map[string]interface{}{
				"output_tokens": tokens.OutputTokens,
			},
		})
		h.writeSSE(c, "message_stop", map[string]interface{}{
			"type": "message_stop",
		})
	}

	reqLog := &model.RequestLog{
		ProviderID:      provider.ID,
		Model:           provider.Model,
		RequestBody:     string(openAIBody),
		ResponseBody:    responseBuilder.String(),
		ResponseContent: fullContent.String(),
		InputTokens:     tokens.InputTokens,
		OutputTokens:    tokens.OutputTokens,
		TotalTokens:     tokens.TotalTokens,
		CachedTokens:    tokens.CachedTokens,
		Status:          "success",
		Duration:        time.Since(startTime).Milliseconds(),
	}
	h.SaveRequestLog(reqLog)
}

// Models 处理 /anthropic/v1/models 请求
func (h *AnthropicHandler) Models(c *gin.Context) {
	providers, err := h.GetProviders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": err.Error()},
		})
		return
	}

	var models []model.AnthropicModelInfo
	for _, provider := range providers {
		models = append(models, model.AnthropicModelInfo{
			ID:          provider.Model,
			Type:        "model",
			DisplayName: provider.Name,
			CreatedAt:   provider.UpdatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, model.AnthropicModelsResponse{
		Data:    models,
		HasMore: false,
	})
}

// convertAnthropicToOpenAI 将 Anthropic 请求转换为 OpenAI 格式
func (h *AnthropicHandler) convertAnthropicToOpenAI(req *model.AnthropicMessagesRequest, provider *model.ProviderConfig) map[string]interface{} {
	messages := make([]map[string]interface{}, 0)

	// 处理 system 字段：Anthropic 的 system 是顶层字段，OpenAI 放在 messages 中
	if req.System != nil {
		systemContent := extractTextFromInterface(req.System)
		if systemContent != "" {
			messages = append(messages, map[string]interface{}{
				"role":    "system",
				"content": systemContent,
			})
		}
	}

	// 转换 messages
	for _, msg := range req.Messages {
		content := extractTextFromInterface(msg.Content)
		messages = append(messages, map[string]interface{}{
			"role":    msg.Role,
			"content": content,
		})
	}

	result := map[string]interface{}{
		"model":    provider.Model,
		"messages": messages,
		"stream":   req.Stream,
	}

	if req.MaxTokens > 0 {
		result["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		result["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		result["top_p"] = *req.TopP
	}
	if req.TopK != nil {
		result["top_k"] = *req.TopK
	}
	if len(req.StopSequences) > 0 {
		result["stop"] = req.StopSequences
	}

	// 转换 tools
	if len(req.Tools) > 0 {
		openaiTools := make([]map[string]interface{}, len(req.Tools))
		for i, tool := range req.Tools {
			openaiTools[i] = map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  tool.InputSchema,
				},
			}
		}
		result["tools"] = openaiTools
	}

	return result
}

// convertOpenAIToAnthropic 将 OpenAI 响应转换为 Anthropic 格式
func (h *AnthropicHandler) convertOpenAIToAnthropic(openAIResp []byte, modelName string) model.AnthropicMessagesResponse {
	var resp map[string]interface{}
	if err := json.Unmarshal(openAIResp, &resp); err != nil {
		return model.AnthropicMessagesResponse{
			ID:   fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			Type: "message",
			Role: "assistant",
			Content: []model.AnthropicContentBlock{
				{Type: "text", Text: string(openAIResp)},
			},
			Model:      modelName,
			StopReason: "end_turn",
			Usage:      model.AnthropicUsage{},
		}
	}

	content := ""
	var toolCalls []model.AnthropicContentBlock
	stopReason := "end_turn"

	if choices, ok := resp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if c, ok := msg["content"].(string); ok {
					content = c
				}
				// 转换 tool_calls
				if tc, ok := msg["tool_calls"].([]interface{}); ok {
					for idx, t := range tc {
						if tcMap, ok := t.(map[string]interface{}); ok {
							if fn, ok := tcMap["function"].(map[string]interface{}); ok {
								name, _ := fn["name"].(string)
								argsStr, _ := fn["arguments"].(string)
								var args interface{}
								json.Unmarshal([]byte(argsStr), &args)
								id, _ := tcMap["id"].(string)
								if id == "" {
									id = fmt.Sprintf("toolu_%d", idx)
								}
								toolCalls = append(toolCalls, model.AnthropicContentBlock{
									Type:  "tool_use",
									ID:    id,
									Name:  name,
									Input: args,
								})
							}
						}
					}
					stopReason = "tool_use"
				}
			}
			// finish_reason
			if fr, ok := choice["finish_reason"].(string); ok {
				if fr == "length" {
					stopReason = "max_tokens"
				}
			}
		}
	}

	inputTokens, outputTokens, _, cachedTokens := h.ExtractUsage(resp)

	// 构建内容块
	var contentBlocks []model.AnthropicContentBlock
	if content != "" {
		contentBlocks = append(contentBlocks, model.AnthropicContentBlock{
			Type: "text",
			Text: content,
		})
	}
	contentBlocks = append(contentBlocks, toolCalls...)

	if len(contentBlocks) == 0 {
		contentBlocks = append(contentBlocks, model.AnthropicContentBlock{
			Type: "text",
			Text: "",
		})
	}

	return model.AnthropicMessagesResponse{
		ID:         fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Type:       "message",
		Role:       "assistant",
		Content:    contentBlocks,
		Model:      modelName,
		StopReason: stopReason,
		Usage: model.AnthropicUsage{
			InputTokens:          inputTokens,
			OutputTokens:         outputTokens,
			CacheReadInputTokens: cachedTokens,
		},
	}
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

// extractTextFromInterface 从 Anthropic content 字段提取文本
// content 可能是 string 或 []ContentBlock
func extractTextFromInterface(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var texts []string
		for _, item := range v {
			if block, ok := item.(map[string]interface{}); ok {
				if text, ok := block["text"].(string); ok {
					texts = append(texts, text)
				}
			}
		}
		return strings.Join(texts, "\n")
	default:
		return fmt.Sprintf("%v", content)
	}
}

// extractTextFromAnthropicContent 从 AnthropicContentBlock 数组提取文本
func extractTextFromAnthropicContent(blocks []model.AnthropicContentBlock) string {
	var texts []string
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}
	return strings.Join(texts, "\n")
}


