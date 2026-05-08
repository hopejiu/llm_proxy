package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"llm-proxy/internal/converter"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ProxyHandler OpenAI API 处理器
type ProxyHandler struct {
	*BaseHandler // 组合基类
}

// NewProxyHandler 创建 ProxyHandler 实例
func NewProxyHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository, cfg HandlerConfig) *ProxyHandler {
	return &ProxyHandler{
		BaseHandler: NewBaseHandler(proxyService, requestLogRepo, cfg),
	}
}

// logRequest 记录请求汇总日志（唯一日志输出点）
func (h *ProxyHandler) logRequest(c *gin.Context, reqBody []byte, startTime time.Time, status string, errMsg string, provider *model.ProviderConfig) {
	duration := time.Since(startTime).Milliseconds()

	var reqInfo struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	json.Unmarshal(reqBody, &reqInfo)

	requestID := requestIDFromContext(c.Request.Context())

	attrs := []any{
		"requestID", requestID,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"clientIP", c.ClientIP(),
		"model", reqInfo.Model,
		"stream", reqInfo.Stream,
		"duration_ms", duration,
		"status", status,
	}
	if provider != nil {
		attrs = append(attrs, "provider", provider.Name)
	}
	if errMsg != "" {
		attrs = append(attrs, "error", errMsg)
	}

	// 根据状态选择日志级别
	switch status {
	case "ERROR", "FAILED":
		slog.Error("proxy request", attrs...)
	default:
		slog.Info("proxy request", attrs...)
	}
}

// ChatCompletions 中转OpenAI请求
func (h *ProxyHandler) ChatCompletions(c *gin.Context) {
	startTime := time.Now()
	// 注入 requestID 到 context
	ctx := contextWithRequestID(c.Request.Context(), generateRequestID())
	c.Request = c.Request.WithContext(ctx)

	body, err := h.ReadBody(c)
	if err != nil {
		h.logRequest(c, nil, startTime, "ERROR", "failed to read request body", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	var req converter.OpenAISimpleRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logRequest(c, body, startTime, "ERROR", "invalid request body", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Stream {
		h.handleStreamRequest(c, body, req.Model, startTime)
	} else {
		h.handleNormalRequest(c, body, req.Model, startTime)
	}
}

// handleNormalRequest 处理非流式请求
func (h *ProxyHandler) handleNormalRequest(c *gin.Context, body []byte, modelName string, startTime time.Time) {
	provider, err := h.GetProviderByModel(modelName)
	if err != nil {
		h.logRequest(c, body, startTime, "FAILED", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "proxy_error",
			},
		})
		return
	}

	h.handleNormalRequestOpenAI(c, body, provider, startTime)
}

// handleNormalRequestOpenAI 处理 OpenAI 类型 Provider 的非流式请求（直接透传）
func (h *ProxyHandler) handleNormalRequestOpenAI(c *gin.Context, body []byte, provider *model.ProviderConfig, startTime time.Time) {
	body = h.PrepareRequestBody(body, provider)
	reqLog := h.CreateRequestLog(provider, string(body))

	respBody, err := h.SendRequestWithRetry(c.Request.Context(), provider.GetRequestURL(), body, provider.APIKey, h.config.StreamMaxRetries)
	if err != nil {
		h.logRequest(c, body, startTime, "FAILED", err.Error(), provider)
		statusCode := http.StatusInternalServerError
		errBody := err.Error()
		if upErr, ok := err.(*UpstreamError); ok {
			statusCode = upErr.StatusCode
			errBody = upErr.Body
		}
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
	h.logRequest(c, body, startTime, "SUCCESS", "", provider)

	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, string(respBody))
}

// handleStreamRequest 处理流式请求
func (h *ProxyHandler) handleStreamRequest(c *gin.Context, body []byte, modelName string, startTime time.Time) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	provider, err := h.GetProviderByModel(modelName)
	if err != nil {
		h.logRequest(c, body, startTime, "FAILED", err.Error(), nil)
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}

	h.handleStreamRequestOpenAI(c, body, provider, startTime)
}

// handleStreamRequestOpenAI 处理 OpenAI 类型 Provider 的流式请求（直接透传）
func (h *ProxyHandler) handleStreamRequestOpenAI(c *gin.Context, body []byte, provider *model.ProviderConfig, startTime time.Time) {
	body = h.PrepareRequestBody(body, provider)
	reqLog := h.CreateRequestLog(provider, string(body))

	responseBuilder, tokens, lastErr := h.ExecuteStreamWithRetry(
		c.Request.Context(),
		provider,
		body,
		h.DefaultStreamRetryConfig(),
		func(line string, _ *StreamTokens) bool {
			c.Writer.Write([]byte(line + "\n\n"))
			c.Writer.Flush()
			return false
		},
	)

	if lastErr != nil {
		h.logRequest(c, body, startTime, "FAILED", lastErr.Error(), provider)
		c.SSEvent("error", gin.H{"error": lastErr.Error()})
		reqLog.ErrorMessage = lastErr.Error()
		h.SaveRequestLog(reqLog)
		return
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
	h.logRequest(c, body, startTime, "STREAM_END", "", provider)
}

// Models 获取可用模型列表
func (h *ProxyHandler) Models(c *gin.Context) {
	startTime := time.Now()

	providers, err := h.GetAllProviders()
	if err != nil {
		h.logRequest(c, []byte{}, startTime, "FAILED", err.Error(), nil)
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

	h.logRequest(c, []byte{}, startTime, "SUCCESS", "", nil)
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

// NotFound 处理未匹配的路由 (404)
func (h *ProxyHandler) NotFound(c *gin.Context) {
	startTime := time.Now()
	body, _ := io.ReadAll(c.Request.Body)
	h.logRequest(c, body, startTime, "NOT_FOUND", fmt.Sprintf("path not found: %s", c.Request.URL.Path), nil)

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
	h.logRequest(c, body, startTime, "METHOD_NOT_ALLOWED", fmt.Sprintf("method not allowed: %s %s", c.Request.Method, c.Request.URL.Path), nil)

	c.JSON(http.StatusMethodNotAllowed, gin.H{
		"error": gin.H{
			"message": fmt.Sprintf("Method '%s' is not allowed for endpoint '%s'.", c.Request.Method, c.Request.URL.Path),
			"type":    "method_not_allowed_error",
			"code":    "405",
		},
	})
}


