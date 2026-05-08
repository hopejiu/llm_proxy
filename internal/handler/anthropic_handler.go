package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"llm-proxy/internal/converter"
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
func NewAnthropicHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository, cfg HandlerConfig) *AnthropicHandler {
	return &AnthropicHandler{
		BaseHandler: NewBaseHandler(proxyService, requestLogRepo, cfg),
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

	provider, err := h.GetProviderByModel(anthropicReq.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": err.Error()},
		})
		return
	}

	// 根据 Provider 类型选择处理方式
	if anthropicReq.Stream {
		h.handleStreamMessages(c, &anthropicReq, provider, startTime)
	} else {
		h.handleNonStreamMessages(c, &anthropicReq, provider, startTime)
	}
}

// handleNonStreamMessages 处理非流式请求（协议转换）
func (h *AnthropicHandler) handleNonStreamMessages(c *gin.Context, anthropicReq *model.AnthropicMessagesRequest, provider *model.ProviderConfig, startTime time.Time) {
	openAIReq := converter.AnthropicToOpenAI(anthropicReq, provider.Model)
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

	c.JSON(http.StatusOK, anthropicResp)
}

// handleStreamMessages 处理流式请求（OpenAI 类型 Provider，需要协议转换）
func (h *AnthropicHandler) handleStreamMessages(c *gin.Context, anthropicReq *model.AnthropicMessagesRequest, provider *model.ProviderConfig, startTime time.Time) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	openAIReq := converter.AnthropicToOpenAI(anthropicReq, provider.Model)
	openAIBody, _ := json.Marshal(openAIReq)
	openAIBody = h.PrepareRequestBody(openAIBody, provider)

	// 流式状态
	msgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	var fullContent strings.Builder
	var thinkingContent strings.Builder
	contentBlockIndex := 0
	messageStarted := false
	contentBlockStarted := false
	toolBlockStarted := false
	var currentToolID string
	var currentToolName string
	var toolArgsBuilder strings.Builder
	// OpenAI tool_call index 到 Anthropic content_block index 的映射
	toolCallIndexMap := make(map[int]int)

	// 使用 base_handler 的 ExecuteStreamWithRetry
	responseBuilder, tokens, lastErr := h.ExecuteStreamWithRetry(
		c.Request.Context(),
		provider,
		openAIBody,
		h.DefaultStreamRetryConfig(),
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

			if choices, ok := streamResp["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if delta, ok := choice["delta"].(map[string]interface{}); ok {
						// 提取推理内容（reasoning_content）
						if reasoning, ok := delta["reasoning_content"].(string); ok && reasoning != "" {
							thinkingContent.WriteString(reasoning)
						}

						// 处理文本内容
						if content, ok := delta["content"].(string); ok && content != "" {
							// 首次收到内容，发送 message_start
							if !messageStarted {
								h.writeSSE(c, "message_start", map[string]interface{}{
									"type": "message_start",
									"message": map[string]interface{}{
										"id":      msgID,
										"type":    "message",
										"role":    "assistant",
										"content": []interface{}{},
										"model":   provider.Model,
										"usage": map[string]interface{}{
											"input_tokens":  currentTokens.InputTokens,
											"output_tokens": 0,
										},
									},
								})
								messageStarted = true
							}

							// 如果之前有未关闭的 tool_use 块，先关闭
							if toolBlockStarted {
								h.writeSSE(c, "content_block_stop", map[string]interface{}{
									"type":  "content_block_stop",
									"index": contentBlockIndex,
								})
								contentBlockIndex++
								toolBlockStarted = false
							}

							// 开始文本内容块
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

							fullContent.WriteString(content)
							h.writeSSE(c, "content_block_delta", map[string]interface{}{
								"type":  "content_block_delta",
								"index": contentBlockIndex,
								"delta": map[string]interface{}{
									"type": "text_delta",
									"text": content,
								},
							})
						}

						// 处理 tool_calls
						if toolCallsDelta, ok := delta["tool_calls"].([]interface{}); ok {
							for _, tc := range toolCallsDelta {
								tcMap, ok := tc.(map[string]interface{})
								if !ok {
									continue
								}

								// 获取 OpenAI 的 tool_call index
								openaiToolIdx := 0
								if idx, ok := tcMap["index"].(float64); ok {
									openaiToolIdx = int(idx)
								}

								// 首次收到内容，发送 message_start
								if !messageStarted {
									h.writeSSE(c, "message_start", map[string]interface{}{
										"type": "message_start",
										"message": map[string]interface{}{
											"id":      msgID,
											"type":    "message",
											"role":    "assistant",
											"content": []interface{}{},
											"model":   provider.Model,
											"usage": map[string]interface{}{
												"input_tokens":  currentTokens.InputTokens,
												"output_tokens": 0,
											},
										},
									})
									messageStarted = true
								}

								// 关闭之前的文本内容块
								if contentBlockStarted {
									h.writeSSE(c, "content_block_stop", map[string]interface{}{
										"type":  "content_block_stop",
										"index": contentBlockIndex,
									})
									contentBlockIndex++
									contentBlockStarted = false
								}

								// 新的 tool_call（有 id 和 name）
								if id, ok := tcMap["id"].(string); ok && id != "" {
									// 关闭之前的 tool 块
									if toolBlockStarted {
										h.writeSSE(c, "content_block_stop", map[string]interface{}{
											"type":  "content_block_stop",
											"index": contentBlockIndex,
										})
										contentBlockIndex++
									}

									currentToolID = id
									currentToolName = ""
									toolArgsBuilder.Reset()

									if fn, ok := tcMap["function"].(map[string]interface{}); ok {
										if name, ok := fn["name"].(string); ok {
											currentToolName = name
										}
									}

									// 建立 OpenAI tool_call index 到 Anthropic content_block index 的映射
									toolCallIndexMap[openaiToolIdx] = contentBlockIndex

									h.writeSSE(c, "content_block_start", map[string]interface{}{
										"type":  "content_block_start",
										"index": contentBlockIndex,
										"content_block": map[string]interface{}{
											"type":  "tool_use",
											"id":    currentToolID,
											"name":  currentToolName,
											"input": map[string]interface{}{},
										},
									})
									toolBlockStarted = true
								}

								// tool_call 的 arguments 增量
								if fn, ok := tcMap["function"].(map[string]interface{}); ok {
									if args, ok := fn["arguments"].(string); ok {
										toolArgsBuilder.WriteString(args)
										// 使用映射表获取正确的 Anthropic content_block index
										anthropicIdx, mapped := toolCallIndexMap[openaiToolIdx]
										if !mapped {
											anthropicIdx = contentBlockIndex
										}
										h.writeSSE(c, "content_block_delta", map[string]interface{}{
											"type":  "content_block_delta",
											"index": anthropicIdx,
											"delta": map[string]interface{}{
												"type":         "input_json_delta",
												"partial_json": args,
											},
										})
									}
								}
							}
						}
					}

					// 检查 finish_reason
					if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" && finishReason != "null" {
						// 关闭当前内容块
						if contentBlockStarted {
							h.writeSSE(c, "content_block_stop", map[string]interface{}{
								"type":  "content_block_stop",
								"index": contentBlockIndex,
							})
							contentBlockStarted = false
						}
						if toolBlockStarted {
							h.writeSSE(c, "content_block_stop", map[string]interface{}{
								"type":  "content_block_stop",
								"index": contentBlockIndex,
							})
							toolBlockStarted = false
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

						h.writeSSE(c, "message_stop", map[string]interface{}{
							"type": "message_stop",
						})
						return false
					}
				}
			}
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
	if toolBlockStarted {
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
		ThinkingContent: thinkingContent.String(),
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

