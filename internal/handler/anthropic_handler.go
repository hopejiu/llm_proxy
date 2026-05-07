package handler

import (
	"bufio"
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

	provider, err := h.GetProviderByModel(anthropicReq.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": err.Error()},
		})
		return
	}

	// 根据 Provider 类型选择处理方式
	if provider.IsAnthropic() {
		// Anthropic 类型：直接透传，不转换
		if anthropicReq.Stream {
			h.handleDirectStreamMessages(c, body, provider, startTime)
		} else {
			h.handleDirectNonStreamMessages(c, body, provider, startTime)
		}
	} else {
		// OpenAI 类型：转换协议
		if anthropicReq.Stream {
			h.handleStreamMessages(c, &anthropicReq, provider, startTime)
		} else {
			h.handleNonStreamMessages(c, &anthropicReq, provider, startTime)
		}
	}
}

// handleDirectNonStreamMessages 直接透传非流式请求到 Anthropic API
func (h *AnthropicHandler) handleDirectNonStreamMessages(c *gin.Context, body []byte, provider *model.ProviderConfig, startTime time.Time) {
	// 准备请求体：替换 model 并合并 ExtraParams
	body = h.PrepareRequestBody(body, provider)

	ctx, cancel := context.WithTimeout(c.Request.Context(), DefaultHTTPTimeout)
	defer cancel()

	respBody, err := h.SendAnthropicRequest(ctx, provider.BaseURL, body, provider.APIKey)
	if err != nil {
		slog.Error("Anthropic直接请求失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": err.Error()},
		})
		return
	}

	// 解析响应提取日志信息
	var anthropicResp model.AnthropicMessagesResponse
	reqLog := &model.RequestLog{
		ProviderID:   provider.ID,
		Model:        provider.Model,
		RequestBody:  string(body),
		ResponseBody: string(respBody),
		Status:       "success",
		Duration:     time.Since(startTime).Milliseconds(),
	}

	if err := json.Unmarshal(respBody, &anthropicResp); err == nil {
		reqLog.ResponseContent = extractTextFromAnthropicContent(anthropicResp.Content)
		reqLog.ThinkingContent = extractThinkingFromAnthropicContent(anthropicResp.Content)
		reqLog.InputTokens = anthropicResp.Usage.InputTokens
		reqLog.OutputTokens = anthropicResp.Usage.OutputTokens
		reqLog.TotalTokens = anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens
		reqLog.CachedTokens = anthropicResp.Usage.CacheReadInputTokens
	} else {
		// 尝试解析错误响应
		var errResp map[string]interface{}
		if json.Unmarshal(respBody, &errResp) == nil {
			if errObj, ok := errResp["error"].(map[string]interface{}); ok {
				if msg, ok := errObj["message"].(string); ok {
					reqLog.ErrorMessage = msg
					reqLog.Status = "error"
				}
			}
		}
	}
	h.SaveRequestLog(reqLog)

	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, string(respBody))
}

// handleDirectStreamMessages 直接透传流式请求到 Anthropic API
func (h *AnthropicHandler) handleDirectStreamMessages(c *gin.Context, body []byte, provider *model.ProviderConfig, startTime time.Time) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// 准备请求体：替换 model 并合并 ExtraParams
	body = h.PrepareRequestBody(body, provider)

	ctx, cancel := context.WithTimeout(c.Request.Context(), DefaultHTTPTimeout)
	defer cancel()

	httpResp, err := h.SendAnthropicStreamRequest(ctx, provider.BaseURL, body, provider.APIKey)
	if err != nil {
		slog.Error("Anthropic流式请求失败", "error", err)
		h.sendAnthropicSSEError(c, err.Error())
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		slog.Error("Anthropic流式请求返回错误", "status", httpResp.StatusCode, "body", string(respBody))
		h.sendAnthropicSSEError(c, fmt.Sprintf("HTTP %d: %s", httpResp.StatusCode, string(respBody)))
		return
	}

	// 直接透传 SSE 流
	var fullContent strings.Builder
	var thinkingContent strings.Builder
	var responseBuilder strings.Builder
	var inputTokens, outputTokens, cachedTokens int

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.sendAnthropicSSEError(c, "streaming not supported")
		return
	}

	scanner := newSSEScanner(httpResp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// 累积原始 SSE 数据用于日志记录
		responseBuilder.WriteString(line + "\n")

		// 直接写入客户端
		c.Writer.Write([]byte(line + "\n"))

		// 解析事件提取统计信息
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var event map[string]interface{}
			if json.Unmarshal([]byte(data), &event) == nil {
				h.extractAnthropicStreamStats(event, &fullContent, &thinkingContent, &inputTokens, &outputTokens, &cachedTokens)
			}
		}

		// 空行表示事件结束，需要 flush
		if line == "" {
			flusher.Flush()
		}
	}

	reqLog := &model.RequestLog{
		ProviderID:      provider.ID,
		Model:           provider.Model,
		RequestBody:     string(body),
		ResponseBody:    responseBuilder.String(),
		ResponseContent: fullContent.String(),
		ThinkingContent: thinkingContent.String(),
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		TotalTokens:     inputTokens + outputTokens,
		CachedTokens:    cachedTokens,
		Status:          "success",
		Duration:        time.Since(startTime).Milliseconds(),
	}
	h.SaveRequestLog(reqLog)
}

// extractAnthropicStreamStats 从 Anthropic 流式事件中提取统计信息
func (h *AnthropicHandler) extractAnthropicStreamStats(event map[string]interface{}, fullContent *strings.Builder, thinkingContent *strings.Builder, inputTokens, outputTokens, cachedTokens *int) {
	eventType, _ := event["type"].(string)

	switch eventType {
	case "message_start":
		if msg, ok := event["message"].(map[string]interface{}); ok {
			if usage, ok := msg["usage"].(map[string]interface{}); ok {
				if it, ok := usage["input_tokens"].(float64); ok {
					*inputTokens = int(it)
				}
				if cr, ok := usage["cache_read_input_tokens"].(float64); ok {
					*cachedTokens = int(cr)
				}
			}
		}
	case "content_block_delta":
		if delta, ok := event["delta"].(map[string]interface{}); ok {
			if deltaType, ok := delta["type"].(string); ok {
				switch deltaType {
				case "text_delta":
					if text, ok := delta["text"].(string); ok {
						fullContent.WriteString(text)
					}
				case "thinking_delta":
					if thinking, ok := delta["thinking"].(string); ok {
						thinkingContent.WriteString(thinking)
					}
				}
			}
		}
	case "message_delta":
		if usage, ok := event["usage"].(map[string]interface{}); ok {
			if it, ok := usage["input_tokens"].(float64); ok {
				if int(it) > 0 {
					*inputTokens = int(it)
				}
			}
			if ot, ok := usage["output_tokens"].(float64); ok {
				if int(ot) > 0 {
					*outputTokens = int(ot)
				}
			}
		}
	}
}

// SendAnthropicRequest 发送请求到 Anthropic API（使用 x-api-key header）
func (h *AnthropicHandler) SendAnthropicRequest(ctx context.Context, url string, body []byte, apiKey string) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

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

// SendAnthropicStreamRequest 发送流式请求到 Anthropic API
func (h *AnthropicHandler) SendAnthropicStreamRequest(ctx context.Context, url string, body []byte, apiKey string) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

	httpResp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return httpResp, nil
}

// handleNonStreamMessages 处理非流式请求（OpenAI 类型 Provider，需要协议转换）
func (h *AnthropicHandler) handleNonStreamMessages(c *gin.Context, anthropicReq *model.AnthropicMessagesRequest, provider *model.ProviderConfig, startTime time.Time) {
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
		ResponseContent: extractTextFromAnthropicContent(anthropicResp.Content),
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

	openAIReq := h.convertAnthropicToOpenAI(anthropicReq, provider)
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

// convertAnthropicToOpenAI 将 Anthropic 请求转换为 OpenAI 格式
func (h *AnthropicHandler) convertAnthropicToOpenAI(req *model.AnthropicMessagesRequest, provider *model.ProviderConfig) map[string]interface{} {
	messages := make([]map[string]interface{}, 0)

	// 处理 system 字段
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
		openaiMsg := h.convertAnthropicMessage(msg)
		messages = append(messages, openaiMsg)
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

	// 转换 tool_choice
	if req.ToolChoice != nil {
		result["tool_choice"] = req.ToolChoice
	}

	return result
}

// convertAnthropicMessage 将单条 Anthropic 消息转换为 OpenAI 格式
func (h *AnthropicHandler) convertAnthropicMessage(msg model.AnthropicMessage) map[string]interface{} {
	// content 是 string 的情况
	if contentStr, ok := msg.Content.(string); ok {
		return map[string]interface{}{
			"role":    msg.Role,
			"content": contentStr,
		}
	}

	// content 是数组的情况
	contentArr, ok := msg.Content.([]interface{})
	if !ok {
		return map[string]interface{}{
			"role":    msg.Role,
			"content": fmt.Sprintf("%v", msg.Content),
		}
	}

	// 对于 assistant 消息中的 tool_use，需要转换为 OpenAI 的 tool_calls 格式
	if msg.Role == "assistant" {
		var textParts []string
		var toolCalls []map[string]interface{}
		toolCallIdx := 0

		for _, item := range contentArr {
			block, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			blockType, _ := block["type"].(string)

			switch blockType {
			case "text":
				if text, ok := block["text"].(string); ok {
					textParts = append(textParts, text)
				}
			case "tool_use":
				id, _ := block["id"].(string)
				name, _ := block["name"].(string)
				input := block["input"]
				argsStr := "{}"
				if input != nil {
					if b, err := json.Marshal(input); err == nil {
						argsStr = string(b)
					}
				}
				toolCalls = append(toolCalls, map[string]interface{}{
					"id":   id,
					"type": "function",
					"function": map[string]interface{}{
						"name":      name,
						"arguments": argsStr,
					},
				})
				toolCallIdx++
			}
		}

		result := map[string]interface{}{
			"role":    "assistant",
			"content": strings.Join(textParts, "\n"),
		}
		if len(toolCalls) > 0 {
			result["tool_calls"] = toolCalls
		}
		return result
	}

	// 对于 user 消息中的 tool_result，转换为 OpenAI 格式
	if msg.Role == "user" {
		var textParts []string

		for _, item := range contentArr {
			block, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			blockType, _ := block["type"].(string)

			switch blockType {
			case "text":
				if text, ok := block["text"].(string); ok {
					textParts = append(textParts, text)
				}
			case "tool_result":
				// tool_result 在 OpenAI 格式中对应 tool role 的消息
				toolUseID, _ := block["tool_use_id"].(string)
				resultContent := ""
				if content, ok := block["content"]; ok {
					resultContent = extractTextFromInterface(content)
				}
				textParts = append(textParts, fmt.Sprintf("[Tool Result %s]: %s", toolUseID, resultContent))
			}
		}

		return map[string]interface{}{
			"role":    "user",
			"content": strings.Join(textParts, "\n"),
		}
	}

	// 其他角色，简单提取文本
	return map[string]interface{}{
		"role":    msg.Role,
		"content": extractTextFromInterface(msg.Content),
	}
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

// extractThinkingFromAnthropicContent 从 AnthropicContentBlock 数组提取 thinking 内容
func extractThinkingFromAnthropicContent(blocks []model.AnthropicContentBlock) string {
	var thoughts []string
	for _, block := range blocks {
		if block.Type == "thinking" && block.Text != "" {
			thoughts = append(thoughts, block.Text)
		}
	}
	return strings.Join(thoughts, "\n")
}

// sseScanner 用于逐行读取 SSE 流（基于 bufio.Scanner 实现真正的流式读取）
type sseScanner struct {
	scanner *bufio.Scanner
}

func newSSEScanner(body io.Reader) *sseScanner {
	return &sseScanner{scanner: bufio.NewScanner(body)}
}

func (s *sseScanner) Scan() bool {
	return s.scanner.Scan()
}

func (s *sseScanner) Text() string {
	return s.scanner.Text()
}