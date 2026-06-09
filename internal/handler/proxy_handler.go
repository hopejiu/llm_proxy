package handler

import (
	"encoding/json"
	"fmt"
	"io"
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

// ProxyHandler OpenAI API 处理器
type ProxyHandler struct {
	*BaseHandler // 组合基类
}

// NewProxyHandler 创建 ProxyHandler 实例
func NewProxyHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository, cfg *config.Config, tracker *ActiveRequestTracker) *ProxyHandler {
	return &ProxyHandler{
		BaseHandler: NewBaseHandler(proxyService, requestLogRepo, cfg, tracker),
	}
}

// ChatCompletions 中转OpenAI请求
func (h *ProxyHandler) ChatCompletions(c *gin.Context) {
	h.HandleProxyRequest(c, "openai",
		func(body []byte) (*ProxyRequestInfo, error) {
			var req converter.OpenAISimpleRequest
			if err := json.Unmarshal(body, &req); err != nil {
				return nil, fmt.Errorf("invalid request body")
			}
			return &ProxyRequestInfo{Model: req.Model, Stream: req.Stream, Protocol: "openai"}, nil
		},
		func(c *gin.Context, body []byte, startTime time.Time) {
			h.handleStreamRequest(c, body, startTime)
		},
		func(c *gin.Context, body []byte, startTime time.Time) {
			h.handleNormalRequest(c, body, startTime)
		},
	)
}

// handleNormalRequest 处理非流式请求
func (h *ProxyHandler) handleNormalRequest(c *gin.Context, body []byte, startTime time.Time) {
	var reqInfo struct {
		Model string `json:"model"`
	}
	json.Unmarshal(body, &reqInfo)

	provider, err := h.GetProviderByModel(reqInfo.Model)
	if err != nil {
		h.LogRequest(c, body, startTime, "FAILED", err.Error(), model.ProviderConfig{})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "proxy_error",
			},
		})
		return
	}

	// 更新活跃请求的 Provider 信息
	requestID := requestIDFromContext(c.Request.Context())
	h.tracker.UpdateProvider(requestID, provider.ID, provider.Name)

	h.handleNormalRequestOpenAI(c, body, provider, startTime)
}

// handleNormalRequestOpenAI 处理 OpenAI 类型 Provider 的非流式请求（直接透传）
func (h *ProxyHandler) handleNormalRequestOpenAI(c *gin.Context, body []byte, provider model.ProviderConfig, startTime time.Time) {
	body = h.PrepareRequestBody(body, provider)
	reqLog := h.CreateRequestLog(provider, string(body))
	sessionID := sessionIDFromContext(c.Request.Context())

	respBody, err := h.SendRequestWithRetry(c.Request.Context(), provider.GetRequestURL(), body, provider.APIKey, h.cfg.GetStreamMaxRetries())
	if err != nil {
		h.LogRequest(c, body, startTime, "FAILED", err.Error(), provider)
		statusCode, errBody := ResolveUpstreamError(err)
		c.JSON(statusCode, gin.H{
			"error": gin.H{
				"message": errBody,
				"type":    "proxy_error",
			},
		})
		reqLog.ErrorMessage = err.Error()
		h.SaveRequestLog(reqLog, sessionID)
		return
	}

	reqLog.ResponseBody = string(respBody)
	reqLog.Duration = time.Since(startTime).Milliseconds()

	var openAIResp converter.OpenAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err == nil {
		reqLog.InputTokens = openAIResp.Usage.PromptTokens
		reqLog.OutputTokens = openAIResp.Usage.CompletionTokens
		reqLog.TotalTokens = openAIResp.Usage.TotalTokens
		if openAIResp.Usage.PromptTokensDetails != nil {
			reqLog.CachedTokens = openAIResp.Usage.PromptTokensDetails.CachedTokens
		}

		if len(openAIResp.Choices) > 0 {
			reqLog.ResponseContent = openAIResp.Choices[0].Message.Content
			var rawResp map[string]interface{}
			if err := json.Unmarshal(respBody, &rawResp); err == nil {
				if choices, ok := rawResp["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						if msg, ok := choice["message"].(map[string]interface{}); ok {
							if reasoning, ok := msg["reasoning_content"].(string); ok && reasoning != "" {
								reqLog.ThinkingContent = reasoning
							}
						}
					}
				}
			}
		}
	}

	reqLog.Status = "success"
	h.SaveRequestLog(reqLog, sessionID)
	h.LogRequest(c, body, startTime, "SUCCESS", "", provider)

	// 非流式请求完成后，将响应内容追加到 tracker
	requestID := requestIDFromContext(c.Request.Context())
	if reqLog.ResponseContent != "" {
		h.tracker.AppendResponse(requestID, reqLog.ResponseContent)
	}

	// 仅在新建会话时注入会话标记（后续请求已有标记，不重复注入）
	slog.Info("[session] 注入前检查",
		"sessionID", sessionID,
		"isNew", isNewSession(c.Request.Context()),
		"hasSession", strings.Contains(string(respBody), "[SESSION]"))
	if isNewSession(c.Request.Context()) {
		slog.Info("[session] ===> 非流式注入 #1", "sessionID", sessionID)
		respBody = InjectSessionMarkIntoResponse(respBody, sessionID, "openai")
		slog.Info("[session] ===> 非流式注入后响应", "body_truncated", string(respBody)[:min(len(string(respBody)), 300)])
	}

	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, string(respBody))
}



// handleStreamRequest 处理流式请求
func (h *ProxyHandler) handleStreamRequest(c *gin.Context, body []byte, startTime time.Time) {
	var reqInfo struct {
		Model string `json:"model"`
	}
	json.Unmarshal(body, &reqInfo)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	CloseClientConnection(c)

	provider, err := h.GetProviderByModel(reqInfo.Model)
	if err != nil {
		h.LogRequest(c, body, startTime, "FAILED", err.Error(), model.ProviderConfig{})
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}

	// 更新活跃请求的 Provider 信息
	requestID := requestIDFromContext(c.Request.Context())
	h.tracker.UpdateProvider(requestID, provider.ID, provider.Name)

	h.handleStreamRequestOpenAI(c, body, provider, startTime)
}

// handleStreamRequestOpenAI 处理 OpenAI 类型 Provider 的流式请求（直接透传）
func (h *ProxyHandler) handleStreamRequestOpenAI(c *gin.Context, body []byte, provider model.ProviderConfig, startTime time.Time) {
	body = h.PrepareRequestBody(body, provider)
	reqLog := h.CreateRequestLog(provider, string(body))
	requestID := requestIDFromContext(c.Request.Context())
	sessionID := sessionIDFromContext(c.Request.Context())
	tracker := h.tracker

	var receivedDone bool
	var sessionInjected bool // 防重复注入：同一请求只注入一次标记
	responseBuilder, tokens, lastErr := h.ExecuteStreamWithRetry(
		c.Request.Context(),
		provider,
		body,
		h.DefaultStreamRetryConfig(),
		func(line string, _ *StreamTokens) bool {
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					// 在 [DONE] 之前写入独立会话标记行（避免加到 finish_reason chunk 被客户端忽略）
					if !sessionInjected && isNewSession(c.Request.Context()) && sessionID > 0 {
						suffix := fmt.Sprintf("%s%d%s", sessionMarkPrefix, sessionID, sessionMarkSuffix)
						markChunk := fmt.Sprintf(`{"id":"%s","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"%s"}}]}`,
							requestID, suffix)
						c.Writer.Write([]byte("data: " + markChunk + "\n\n"))
						c.Writer.Flush()
						sessionInjected = true
						slog.Info("[session] ===> 流式注入 #1（独立行）", "sessionID", sessionID)
					}

					c.Writer.Write([]byte(line + "\n\n"))
					c.Writer.Flush()
					receivedDone = true
					return true
				}

				// 普通 data 行：透写 + 提取 tracker 内容
				c.Writer.Write([]byte(line + "\n\n"))
				c.Writer.Flush()

				var chunk map[string]interface{}
				if json.Unmarshal([]byte(data), &chunk) == nil {
					deltaResult := converter.ExtractDeltaFromChunk(chunk)
					if deltaResult.Content != "" {
						tracker.AppendResponse(requestID, deltaResult.Content)
					}
					if deltaResult.ReasoningContent != "" {
						tracker.AppendResponse(requestID, deltaResult.ReasoningContent)
					}
					if len(deltaResult.ToolCallsDelta) > 0 {
						trackToolCallsFromDelta(deltaResult.ToolCallsDelta, requestID, tracker)
					}
				}
			} else {
				// 非 data 行（空行分隔符等）直接透写
				c.Writer.Write([]byte(line + "\n\n"))
				c.Writer.Flush()
			}
			return false
		},
	)

	if lastErr != nil {
		h.LogRequest(c, body, startTime, "FAILED", lastErr.Error(), provider)
		SafeWriteSSE(c, "event: error\ndata: {\"error\":\""+lastErr.Error()+"\"}\n\n")
		// 超时/错误时也必须发送 [DONE]，否则客户端会一直挂起等待
		SafeWriteSSE(c, "data: [DONE]\n\n")
		reqLog.ErrorMessage = lastErr.Error()
		h.SaveRequestLog(reqLog, sessionID)
		return
	}

	// 确保发送 [DONE] 标记（某些上游可能不发送，导致客户端一直等待）
	if !receivedDone {
		SafeWriteSSE(c, "data: [DONE]\n\n")
	}

	reqLog.ResponseBody = responseBuilder.String()
	reqLog.ResponseContent = parseStreamResponse(responseBuilder.String())
	// 流式注入的标记在 processor 中已写入客户端，但 responseBuilder 捕获的是原始行
	// 此处补回 sessionContent，使数据库记录与实际客户端接收一致
	if sessionInjected {
		suffix := fmt.Sprintf("%s%d%s", sessionMarkPrefix, sessionID, sessionMarkSuffix)
		reqLog.ResponseContent += suffix
	}
	reqLog.InputTokens = tokens.InputTokens
	reqLog.OutputTokens = tokens.OutputTokens
	reqLog.TotalTokens = tokens.TotalTokens
	reqLog.CachedTokens = tokens.CachedTokens
	reqLog.Duration = time.Since(startTime).Milliseconds()
	reqLog.Status = "success"

	h.SaveRequestLog(reqLog, sessionID)

	h.LogRequest(c, body, startTime, "STREAM_END", "", provider)
}

// Models 获取可用模型列表
func (h *ProxyHandler) Models(c *gin.Context) {
	startTime := time.Now()

	providers, err := h.GetAllProviders()
	if err != nil {
		h.LogRequest(c, []byte{}, startTime, "FAILED", err.Error(), model.ProviderConfig{})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var models []gin.H
	for _, provider := range providers {
		for _, name := range provider.GetModelNames() {
			models = append(models, gin.H{
				"id":       name,
				"object":   "model",
				"provider": provider.Name,
			})
		}
	}

	h.LogRequest(c, []byte{}, startTime, "SUCCESS", "", model.ProviderConfig{})
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

// NotFound 处理未匹配的路由 (404)
func (h *ProxyHandler) NotFound(c *gin.Context) {
	startTime := time.Now()
	body, _ := io.ReadAll(c.Request.Body)
	h.LogRequest(c, body, startTime, "NOT_FOUND", fmt.Sprintf("path not found: %s", c.Request.URL.Path), model.ProviderConfig{})

	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{
			"message": fmt.Sprintf("The requested endpoint '%s %s' was not found.", c.Request.Method, c.Request.URL.Path),
			"type":    "not_found_error",
			"code":    "404",
		},
	})
}

// MethodNotAllowed 处理不允许的方法 (405)
func (h *ProxyHandler) MethodNotAllowed(c *gin.Context) {
	startTime := time.Now()
	body, _ := io.ReadAll(c.Request.Body)
	h.LogRequest(c, body, startTime, "METHOD_NOT_ALLOWED", fmt.Sprintf("method not allowed: %s %s", c.Request.Method, c.Request.URL.Path), model.ProviderConfig{})

	c.JSON(http.StatusMethodNotAllowed, gin.H{
		"error": gin.H{
			"message": fmt.Sprintf("Method '%s' is not allowed for endpoint '%s'.", c.Request.Method, c.Request.URL.Path),
			"type":    "method_not_allowed_error",
			"code":    "405",
		},
	})
}
