package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"llm-proxy/internal/config"
	"llm-proxy/internal/converter"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
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
		h.SaveRequestLog(reqLog)
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
	h.SaveRequestLog(reqLog)
	h.LogRequest(c, body, startTime, "SUCCESS", "", provider)

	// 非流式请求完成后，将响应内容追加到 tracker
	requestID := requestIDFromContext(c.Request.Context())
	if reqLog.ResponseContent != "" {
		h.tracker.AppendResponse(requestID, reqLog.ResponseContent)
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
	tracker := h.tracker

	var receivedDone bool
	responseBuilder, tokens, lastErr := h.ExecuteStreamWithRetry(
		c.Request.Context(),
		provider,
		body,
		h.DefaultStreamRetryConfig(),
		func(line string, _ *StreamTokens) bool {
			c.Writer.Write([]byte(line + "\n\n"))
			c.Writer.Flush()

			// 实时提取流式内容追加到 tracker
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					receivedDone = true
					return true // 收到 [DONE]，停止处理
				}
				var chunk map[string]interface{}
				if json.Unmarshal([]byte(data), &chunk) == nil {
					deltaResult := converter.ExtractDeltaFromChunk(chunk)
					if deltaResult.Content != "" {
						tracker.AppendResponse(requestID, deltaResult.Content)
					}
					if deltaResult.ReasoningContent != "" {
						tracker.AppendResponse(requestID, deltaResult.ReasoningContent)
					}
					// 追踪工具调用
					if len(deltaResult.ToolCallsDelta) > 0 {
						trackToolCallsFromDelta(deltaResult.ToolCallsDelta, requestID, tracker)
					}
				}
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
		h.SaveRequestLog(reqLog)
		return
	}

	// 确保发送 [DONE] 标记（某些上游可能不发送，导致客户端一直等待）
	if !receivedDone {
		SafeWriteSSE(c, "data: [DONE]\n\n")
	}

	reqLog.ResponseBody = responseBuilder.String()
	reqLog.ResponseContent = parseStreamResponse(responseBuilder.String())
	reqLog.InputTokens = tokens.InputTokens
	reqLog.OutputTokens = tokens.OutputTokens
	reqLog.TotalTokens = tokens.TotalTokens
	reqLog.CachedTokens = tokens.CachedTokens
	reqLog.Duration = time.Since(startTime).Milliseconds()
	reqLog.Status = "success"

	h.SaveRequestLog(reqLog)

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
