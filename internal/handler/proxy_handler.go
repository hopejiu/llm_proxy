package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)
// ProxyHandler OpenAI API 处理器
type ProxyHandler struct {
	*BaseHandler // 组合基类
	requestLog   *os.File
	logMu        sync.Mutex
}

// NewProxyHandler 创建 ProxyHandler 实例
func NewProxyHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository) *ProxyHandler {
	logFile, err := os.OpenFile("proxy-requests.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, DefaultLogFilePerm)
	if err != nil {
		slog.Warn("无法创建请求日志文件，使用 stdout", "error", err)
		logFile = os.Stdout
	}

	return &ProxyHandler{
		BaseHandler: NewBaseHandler(proxyService, requestLogRepo),
		requestLog:  logFile,
	}
}

// logRequest 记录请求日志
func (h *ProxyHandler) logRequest(c *gin.Context, reqBody []byte, startTime time.Time, status string, errMsg string) {
	duration := time.Since(startTime).Milliseconds()

	// 解析请求体获取模型信息
	var reqInfo struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	json.Unmarshal(reqBody, &reqInfo)

	// 使用 slog 结构化记录
	slog.Info("proxy request",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"clientIP", c.ClientIP(),
		"model", reqInfo.Model,
		"stream", reqInfo.Stream,
		"duration_ms", duration,
		"status", status,
		"error", errMsg,
	)

	// 同时写入请求日志文件（保持原有格式兼容）
	logEntry := fmt.Sprintf("[%s] %s | %s | %s | %s | model=%s | stream=%v | %dms | %s | %s | %s\n",
		time.Now().Format("2006-01-02 15:04:05.000"),
		c.Request.Method,
		c.Request.URL.Path,
		c.ClientIP(),
		c.Request.UserAgent(),
		reqInfo.Model,
		reqInfo.Stream,
		duration,
		status,
		errMsg,
		string(reqBody),
	)
	h.logMu.Lock()
	if _, err := h.requestLog.WriteString(logEntry); err != nil {
		slog.Warn("写入请求日志失败", "error", err)
	}
	h.logMu.Unlock()
}

// ChatCompletions 中转OpenAI请求
func (h *ProxyHandler) ChatCompletions(c *gin.Context) {
	startTime := time.Now()

	// 读取请求体
	body, err := h.ReadBody(c)
	if err != nil {
		h.logRequest(c, nil, startTime, "ERROR", "failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// 解析请求获取 model 和 stream 标记
	var req model.OpenAIRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logRequest(c, body, startTime, "ERROR", "invalid request body")
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
		h.logRequest(c, body, startTime, "FAILED", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "proxy_error",
			},
		})
		return
	}

	// 准备请求体
	body = h.PrepareRequestBody(body, provider)
	reqLog := h.CreateRequestLog(provider, string(body))

	// 发送非流式请求
	respBody, err := h.SendRequest(c.Request.Context(), provider.BaseURL, body, provider.APIKey)
	if err != nil {
		slog.Error("发送HTTP请求失败", "handler", "ProxyHandler", "error", err)
		h.logRequest(c, body, startTime, "FAILED", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "proxy_error",
			},
		})
		reqLog.ErrorMessage = err.Error()
		h.SaveRequestLog(reqLog)
		return
	}

	// 解析响应获取 token 使用情况和内容
	reqLog.ResponseBody = string(respBody)
	reqLog.Duration = time.Since(startTime).Milliseconds()

	var openAIResp model.OpenAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err == nil {
		reqLog.InputTokens = openAIResp.Usage.PromptTokens
		reqLog.OutputTokens = openAIResp.Usage.CompletionTokens
		reqLog.TotalTokens = openAIResp.Usage.TotalTokens
		if openAIResp.Usage.PromptTokensDetails != nil {
			reqLog.CachedTokens = openAIResp.Usage.PromptTokensDetails.CachedTokens
		}

		// 提取 thinking 内容和响应内容
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

	h.logRequest(c, body, startTime, "SUCCESS", "")

	// 返回原始响应
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, string(respBody))
}

// handleStreamRequest 处理流式请求（支持超时重试）
func (h *ProxyHandler) handleStreamRequest(c *gin.Context, body []byte, modelName string, startTime time.Time) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	provider, err := h.GetProviderByModel(modelName)
	if err != nil {
		h.logRequest(c, body, startTime, "FAILED", err.Error())
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}

	// 准备请求体
	body = h.PrepareRequestBody(body, provider)
	reqLog := h.CreateRequestLog(provider, string(body))

	// 使用 base_handler 的 ExecuteStreamWithRetry
	responseBuilder, tokens, lastErr := h.ExecuteStreamWithRetry(
		c.Request.Context(),
		provider,
		body,
		DefaultStreamRetryConfig(),
		func(line string, _ *StreamTokens) bool {
			// SSE格式要求每条消息后有两个换行符
			c.Writer.Write([]byte(line + "\n\n"))
			c.Writer.Flush()
			return false
		},
	)

	if lastErr != nil {
		h.logRequest(c, body, startTime, "FAILED", lastErr.Error())
		c.SSEvent("error", gin.H{"error": lastErr.Error()})
		reqLog.ErrorMessage = lastErr.Error()
		h.SaveRequestLog(reqLog)
		return
	}

	// 成功完成
	reqLog.ResponseBody = responseBuilder.String()
	reqLog.ResponseContent = parseStreamResponse(responseBuilder.String())
	reqLog.InputTokens = tokens.InputTokens
	reqLog.OutputTokens = tokens.OutputTokens
	reqLog.TotalTokens = tokens.TotalTokens
	reqLog.CachedTokens = tokens.CachedTokens
	reqLog.Duration = time.Since(startTime).Milliseconds()
	reqLog.Status = "success"
	h.SaveRequestLog(reqLog)
	h.logRequest(c, body, startTime, "STREAM_END", "")
}

// Models 获取可用模型列表
func (h *ProxyHandler) Models(c *gin.Context) {
	startTime := time.Now()

	providers, err := h.GetAllProviders()
	if err != nil {
		slog.Error("获取Provider列表失败", "handler", "ProxyHandler", "error", err)
		h.logRequest(c, []byte{}, startTime, "FAILED", err.Error())
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

	h.logRequest(c, []byte{}, startTime, "SUCCESS", "")
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

// NotFound 处理未匹配的路由 (404)
func (h *ProxyHandler) NotFound(c *gin.Context) {
	startTime := time.Now()

	// 读取请求体（如果有）
	body, _ := io.ReadAll(c.Request.Body)

	// 记录 404 请求
	h.logRequest(c, body, startTime, "NOT_FOUND", fmt.Sprintf("path not found: %s", c.Request.URL.Path))

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

	// 读取请求体（如果有）
	body, _ := io.ReadAll(c.Request.Body)

	// 记录 405 请求
	h.logRequest(c, body, startTime, "METHOD_NOT_ALLOWED", fmt.Sprintf("method not allowed: %s %s", c.Request.Method, c.Request.URL.Path))

	c.JSON(http.StatusMethodNotAllowed, gin.H{
		"error": gin.H{
			"message": fmt.Sprintf("Method '%s' is not allowed for endpoint '%s'.", c.Request.Method, c.Request.URL.Path),
			"type":    "method_not_allowed_error",
			"code":    "405",
		},
	})
}
