package handler

import (
	"encoding/json"
	"fmt"
	"llm-proxy/internal/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// anthropicStreamState 封装 Anthropic 流式请求的状态
type anthropicStreamState struct {
	handler             *AnthropicHandler
	c                   *gin.Context
	provider            *model.ProviderConfig
	requestID           string
	msgID               string
	fullContent         strings.Builder
	thinkingContent     strings.Builder
	contentBlockIndex   int
	messageStarted      bool
	contentBlockStarted bool
	toolBlockStarted    bool
	currentToolID       string
	currentToolName     string
	toolArgsBuilder     strings.Builder
	toolCallIndexMap    map[int]int // OpenAI tool_call index -> Anthropic content_block index
	tracker             *ActiveRequestTracker
}

// newAnthropicStreamState 创建 Anthropic 流式状态
func newAnthropicStreamState(h *AnthropicHandler, c *gin.Context, provider *model.ProviderConfig, requestID string) *anthropicStreamState {
	return &anthropicStreamState{
		handler:          h,
		c:                c,
		provider:         provider,
		requestID:        requestID,
		msgID:            fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		toolCallIndexMap: make(map[int]int),
		tracker:          h.tracker,
	}
}

// ensureMessageStarted 确保已发送 message_start 事件
func (s *anthropicStreamState) ensureMessageStarted(currentTokens *StreamTokens) {
	if s.messageStarted {
		return
	}
	s.handler.writeSSE(s.c, "message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":      s.msgID,
			"type":    "message",
			"role":    "assistant",
			"content": []interface{}{},
			"model":   s.provider.Model,
			"usage": map[string]interface{}{
				"input_tokens":  currentTokens.InputTokens,
				"output_tokens": 0,
			},
		},
	})
	s.messageStarted = true
}

// closeContentBlock 关闭当前打开的文本内容块
func (s *anthropicStreamState) closeContentBlock() {
	if !s.contentBlockStarted {
		return
	}
	s.handler.writeSSE(s.c, "content_block_stop", map[string]interface{}{
		"type":  "content_block_stop",
		"index": s.contentBlockIndex,
	})
	s.contentBlockIndex++
	s.contentBlockStarted = false
}

// closeToolBlock 关闭当前打开的 tool 块
func (s *anthropicStreamState) closeToolBlock() {
	if !s.toolBlockStarted {
		return
	}
	s.handler.writeSSE(s.c, "content_block_stop", map[string]interface{}{
		"type":  "content_block_stop",
		"index": s.contentBlockIndex,
	})
	s.contentBlockIndex++
	s.toolBlockStarted = false
}

// handleTextDelta 处理文本内容增量
func (s *anthropicStreamState) handleTextDelta(content string, currentTokens *StreamTokens) {
	s.ensureMessageStarted(currentTokens)

	// 如果之前有未关闭的 tool_use 块，先关闭
	s.closeToolBlock()

	// 开始文本内容块
	if !s.contentBlockStarted {
		s.handler.writeSSE(s.c, "content_block_start", map[string]interface{}{
			"type":  "content_block_start",
			"index": s.contentBlockIndex,
			"content_block": map[string]interface{}{
				"type": "text",
				"text": "",
			},
		})
		s.contentBlockStarted = true
	}

	s.fullContent.WriteString(content)
	s.tracker.AppendResponse(s.requestID, content)
	s.handler.writeSSE(s.c, "content_block_delta", map[string]interface{}{
		"type":  "content_block_delta",
		"index": s.contentBlockIndex,
		"delta": map[string]interface{}{
			"type": "text_delta",
			"text": content,
		},
	})
}

// handleToolCallsDelta 处理工具调用增量
func (s *anthropicStreamState) handleToolCallsDelta(toolCallsDelta []interface{}, currentTokens *StreamTokens) {
	for _, tc := range toolCallsDelta {
		tcMap, ok := tc.(map[string]interface{})
		if !ok {
			continue
		}

		openaiToolIdx := 0
		if idx, ok := tcMap["index"].(float64); ok {
			openaiToolIdx = int(idx)
		}

		s.ensureMessageStarted(currentTokens)
		s.closeContentBlock()

		// 新的 tool_call（有 id 和 name）
		if id, ok := tcMap["id"].(string); ok && id != "" {
			s.closeToolBlock()

			s.currentToolID = id
			s.currentToolName = ""
			s.toolArgsBuilder.Reset()

			if fn, ok := tcMap["function"].(map[string]interface{}); ok {
				if name, ok := fn["name"].(string); ok {
					s.currentToolName = name
				}
			}

			s.tracker.AddToolCall(s.requestID, id, s.currentToolName)
			s.toolCallIndexMap[openaiToolIdx] = s.contentBlockIndex

			s.handler.writeSSE(s.c, "content_block_start", map[string]interface{}{
				"type":  "content_block_start",
				"index": s.contentBlockIndex,
				"content_block": map[string]interface{}{
					"type":  "tool_use",
					"id":    s.currentToolID,
					"name":  s.currentToolName,
					"input": map[string]interface{}{},
				},
			})
			s.toolBlockStarted = true
		}

		// tool_call 的 arguments 增量
		if fn, ok := tcMap["function"].(map[string]interface{}); ok {
			if args, ok := fn["arguments"].(string); ok {
				s.toolArgsBuilder.WriteString(args)
				s.tracker.AppendToolCallArgs(s.requestID, openaiToolIdx, args)
				anthropicIdx, mapped := s.toolCallIndexMap[openaiToolIdx]
				if !mapped {
					anthropicIdx = s.contentBlockIndex
				}
				s.handler.writeSSE(s.c, "content_block_delta", map[string]interface{}{
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

// handleFinishReason 处理结束原因，返回是否应该停止
func (s *anthropicStreamState) handleFinishReason(finishReason string, currentTokens *StreamTokens) bool {
	if finishReason == "" || finishReason == "null" {
		return false
	}

	s.closeContentBlock()
	s.closeToolBlock()

	stopReason := "end_turn"
	if finishReason == "tool_calls" {
		stopReason = "tool_use"
	} else if finishReason == "length" {
		stopReason = "max_tokens"
	}

	s.handler.writeSSE(s.c, "message_delta", map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason": stopReason,
		},
		"usage": map[string]interface{}{
			"output_tokens": currentTokens.OutputTokens,
		},
	})

	s.handler.writeSSE(s.c, "message_stop", map[string]interface{}{
		"type": "message_stop",
	})
	return true
}

// finalize 流结束时补发未关闭的事件
func (s *anthropicStreamState) finalize(tokens *StreamTokens) {
	if s.contentBlockStarted {
		s.handler.writeSSE(s.c, "content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": s.contentBlockIndex,
		})
	}
	if s.toolBlockStarted {
		s.handler.writeSSE(s.c, "content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": s.contentBlockIndex,
		})
	}
	if s.messageStarted {
		s.handler.writeSSE(s.c, "message_delta", map[string]interface{}{
			"type": "message_delta",
			"delta": map[string]interface{}{
				"stop_reason": "end_turn",
			},
			"usage": map[string]interface{}{
				"output_tokens": tokens.OutputTokens,
			},
		})
		s.handler.writeSSE(s.c, "message_stop", map[string]interface{}{
			"type": "message_stop",
		})
	}
}

// processLine 处理单行流式数据，返回是否应该停止
func (s *anthropicStreamState) processLine(line string, currentTokens *StreamTokens) bool {
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

	choices, ok := streamResp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return false
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return false
	}

	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return false
	}

	// 提取推理内容
	if reasoning, ok := delta["reasoning_content"].(string); ok && reasoning != "" {
		s.thinkingContent.WriteString(reasoning)
	}

	// 处理文本内容
	if content, ok := delta["content"].(string); ok && content != "" {
		s.handleTextDelta(content, currentTokens)
	}

	// 处理 tool_calls
	if toolCallsDelta, ok := delta["tool_calls"].([]interface{}); ok {
		s.handleToolCallsDelta(toolCallsDelta, currentTokens)
	}

	// 检查 finish_reason
	if finishReason, ok := choice["finish_reason"].(string); ok {
		return s.handleFinishReason(finishReason, currentTokens)
	}

	return false
}
