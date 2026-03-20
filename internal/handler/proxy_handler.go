package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// parseStreamResponse 解析 SSE 流式响应，提取完整内容
// 返回格式化的可读 JSON
func parseStreamResponse(sseData string) string {
	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder
	var lastChunk map[string]interface{}

	// tool_calls 拼接：按 index 存储每个 tool_call 的片段
	// toolCalls[index] = {id, type, function: {name, arguments}}
	toolCalls := make(map[int]map[string]interface{})

	lines := strings.Split(sseData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			continue
		}

		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		lastChunk = chunk

		// 提取 choices[0].delta.content
		if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if delta, ok := choice["delta"].(map[string]interface{}); ok {
					// 提取普通内容
					if content, ok := delta["content"].(string); ok {
						contentBuilder.WriteString(content)
					}
					// 提取推理内容
					if reasoning, ok := delta["reasoning_content"].(string); ok && reasoning != "" {
						reasoningBuilder.WriteString(reasoning)
					}
					// 提取 tool_calls
					if toolCallsDelta, ok := delta["tool_calls"].([]interface{}); ok {
						for _, tc := range toolCallsDelta {
							if tcMap, ok := tc.(map[string]interface{}); ok {
								index := 0
								if idx, ok := tcMap["index"].(float64); ok {
									index = int(idx)
								}
								// 初始化该 index 的 tool_call
								if toolCalls[index] == nil {
									toolCalls[index] = map[string]interface{}{
										"id":   "",
										"type": "function",
										"function": map[string]string{
											"name":      "",
											"arguments": "",
										},
									}
								}
								// 拼接 id
								if id, ok := tcMap["id"].(string); ok && id != "" {
									toolCalls[index]["id"] = id
								}
								// 拼接 type
								if t, ok := tcMap["type"].(string); ok && t != "" {
									toolCalls[index]["type"] = t
								}
								// 拼接 function
								if fn, ok := tcMap["function"].(map[string]interface{}); ok {
									fnMap := toolCalls[index]["function"].(map[string]string)
									if name, ok := fn["name"].(string); ok && name != "" {
										fnMap["name"] = name
									}
									if args, ok := fn["arguments"].(string); ok {
										fnMap["arguments"] += args
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 构建可读的 JSON 结果
	result := map[string]interface{}{
		"content": contentBuilder.String(),
	}

	// 如果有推理内容，也加入
	if reasoningBuilder.Len() > 0 {
		result["reasoning_content"] = reasoningBuilder.String()
	}

	// 如果有 tool_calls，按 index 顺序加入
	if len(toolCalls) > 0 {
		// 获取所有 index 并排序
		indices := make([]int, 0, len(toolCalls))
		for idx := range toolCalls {
			indices = append(indices, idx)
		}
		// 简单排序
		for i := 0; i < len(indices)-1; i++ {
			for j := i + 1; j < len(indices); j++ {
				if indices[i] > indices[j] {
					indices[i], indices[j] = indices[j], indices[i]
				}
			}
		}
		// 按顺序构建 tool_calls 数组
		tcArray := make([]map[string]interface{}, len(indices))
		for i, idx := range indices {
			tcArray[i] = toolCalls[idx]
		}
		result["tool_calls"] = tcArray
	}

	// 从最后一个 chunk 提取 usage 信息
	if lastChunk != nil {
		if usage, ok := lastChunk["usage"].(map[string]interface{}); ok {
			result["usage"] = usage
		}
		// 提取 model
		if model, ok := lastChunk["model"].(string); ok {
			result["model"] = model
		}
	}

	// 格式化输出
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return contentBuilder.String()
	}
	return string(jsonBytes)
}

type ProxyHandler struct {
	proxyService   *service.ProxyService
	requestLog     *log.Logger
	requestLogRepo *repository.RequestLogRepository
}

func NewProxyHandler(proxyService *service.ProxyService, requestLogRepo *repository.RequestLogRepository) *ProxyHandler {
	logFile, err := os.OpenFile("proxy-requests.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return &ProxyHandler{
			proxyService:   proxyService,
			requestLog:     log.New(os.Stdout, "[PROXY] ", log.Ldate|log.Ltime|log.Lmicroseconds),
			requestLogRepo: requestLogRepo,
		}
	}

	return &ProxyHandler{
		proxyService:   proxyService,
		requestLog:     log.New(logFile, "", log.Ldate|log.Ltime|log.Lmicroseconds),
		requestLogRepo: requestLogRepo,
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

	// 记录详细信息
	logEntry := fmt.Sprintf("[%s] %s | %s | %s | %s | model=%s | stream=%v | %dms | %s | %s | %s",
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

	h.requestLog.Println(logEntry)
}

// ChatCompletions 中转OpenAI请求
func (h *ProxyHandler) ChatCompletions(c *gin.Context) {
	startTime := time.Now()

	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logRequest(c, body, startTime, "ERROR", "failed to read request body")
		c.JSON(400, gin.H{"error": "failed to read request body"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// 解析请求检查是否流式
	var req model.OpenAIRequest
	json.Unmarshal(body, &req)

	if req.Stream {
		h.handleStreamRequest(c, body, startTime)
	} else {
		h.handleNormalRequest(c, body, startTime)
	}
}

// handleNormalRequest 处理非流式请求
func (h *ProxyHandler) handleNormalRequest(c *gin.Context, body []byte, startTime time.Time) {
	logEntry, err := h.proxyService.ProxyRequest(body)
	if err != nil {
		h.logRequest(c, body, startTime, "FAILED", err.Error())
		c.JSON(500, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "proxy_error",
			},
		})
		return
	}

	// 记录成功日志
	h.logRequest(c, body, startTime, "SUCCESS", "")

	// 返回原始响应
	c.Header("Content-Type", "application/json")
	c.String(200, logEntry.ResponseBody)
}

// handleStreamRequest 处理流式请求
func (h *ProxyHandler) handleStreamRequest(c *gin.Context, body []byte, startTime time.Time) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	provider, err := h.proxyService.GetFirstActiveProvider()
	if err != nil {
		h.logRequest(c, body, startTime, "FAILED", err.Error())
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}

	// 准备请求体
	body = h.proxyService.PrepareRequestBody(body, provider)

	var req model.OpenAIRequest
	json.Unmarshal(body, &req)
	reqLog := &model.RequestLog{
		ProviderID:  provider.ID,
		Model:       provider.Model,
		RequestBody: string(body),
		Status:      "error",
	}

	targetURL := provider.BaseURL
	httpReq, err := http.NewRequest("POST", targetURL, bytes.NewReader(body))
	if err != nil {
		h.logRequest(c, body, startTime, "FAILED", err.Error())
		c.SSEvent("error", gin.H{"error": err.Error()})
		reqLog.ErrorMessage = err.Error()
		h.requestLogRepo.Create(reqLog)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	httpClient := &http.Client{Timeout: 300 * time.Second}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		log.Printf("[ProxyHandler] 发送HTTP请求失败: %v", err)
		h.logRequest(c, body, startTime, "FAILED", err.Error())
		c.SSEvent("error", gin.H{"error": err.Error()})
		reqLog.ErrorMessage = err.Error()
		h.requestLogRepo.Create(reqLog)
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		errMsg := fmt.Sprintf("HTTP %d: %s", httpResp.StatusCode, string(respBody))
		h.logRequest(c, body, startTime, "FAILED", errMsg)
		c.SSEvent("error", gin.H{"error": string(respBody)})
		reqLog.ErrorMessage = errMsg
		h.requestLogRepo.Create(reqLog)
		return
	}

	h.logRequest(c, body, startTime, "STREAM_START", "")

	var responseBuilder strings.Builder
	var inputTokens, outputTokens, totalTokens, cachedTokens int

	scanner := bufio.NewScanner(httpResp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			responseBuilder.WriteString(line + "\n")
			// SSE格式要求每条消息后有两个换行符
			c.Writer.Write([]byte(line + "\n\n"))
			c.Writer.Flush()

			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					continue
				}
				var streamResp map[string]interface{}
				if err := json.Unmarshal([]byte(data), &streamResp); err == nil {
					if usage, ok := streamResp["usage"].(map[string]interface{}); ok {
						if pt, ok := usage["prompt_tokens"].(float64); ok {
							inputTokens = int(pt)
						}
						if ct, ok := usage["completion_tokens"].(float64); ok {
							outputTokens = int(ct)
						}
						if tt, ok := usage["total_tokens"].(float64); ok {
							totalTokens = int(tt)
						}
						// 从 prompt_tokens_details 中提取 cached_tokens
						if ptd, ok := usage["prompt_tokens_details"].(map[string]interface{}); ok {
							if cht, ok := ptd["cached_tokens"].(float64); ok {
								cachedTokens = int(cht)
							}
						}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[ProxyHandler] 读取流式响应失败: %v", err)
	}

	reqLog.ResponseBody = responseBuilder.String()
	reqLog.ResponseContent = parseStreamResponse(responseBuilder.String())
	reqLog.InputTokens = inputTokens
	reqLog.OutputTokens = outputTokens
	reqLog.TotalTokens = totalTokens
	reqLog.CachedTokens = cachedTokens
	reqLog.Duration = time.Since(startTime).Milliseconds()
	reqLog.Status = "success"
	h.requestLogRepo.Create(reqLog)

	h.logRequest(c, body, startTime, "STREAM_END", "")
}

// Models 获取可用模型列表
func (h *ProxyHandler) Models(c *gin.Context) {
	startTime := time.Now()

	providers, err := h.proxyService.GetActiveProviders()
	if err != nil {
		log.Printf("[ProxyHandler] 获取活跃Provider失败: %v", err)
		h.logRequest(c, []byte{}, startTime, "FAILED", err.Error())
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var models []gin.H
	for _, provider := range providers {
		models = append(models, gin.H{
			"id":       provider.Model,
			"object":   "model",
			"provider": provider.Name,
		})
	}

	h.logRequest(c, []byte{}, startTime, "SUCCESS", "")
	c.JSON(200, gin.H{
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

	c.JSON(404, gin.H{
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

	c.JSON(405, gin.H{
		"error": gin.H{
			"message": fmt.Sprintf("Method '%s' is not allowed for endpoint '%s'.", c.Request.Method, c.Request.URL.Path),
			"type":    "method_not_allowed_error",
			"code":    "405",
		},
	})
}
