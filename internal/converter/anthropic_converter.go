package converter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"llm-proxy/internal/model"
)

// AnthropicToOpenAI 将 Anthropic 请求转换为 OpenAI 格式
func AnthropicToOpenAI(req *model.AnthropicMessagesRequest, modelName string) *OpenAIChatRequest {
	messages := make([]OpenAIMessage, 0)

	// 处理 system 字段
	if req.System != nil {
		systemContent := ExtractTextFromAny(req.System)
		if systemContent != "" {
			messages = append(messages, OpenAIMessage{
				Role:    "system",
				Content: systemContent,
			})
		}
	}

	// 转换 messages
	for _, msg := range req.Messages {
		openaiMsg := anthropicMessageToOpenAI(msg)
		messages = append(messages, openaiMsg)
	}

	result := &OpenAIChatRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   req.Stream,
	}

	if req.MaxTokens > 0 {
		result.MaxTokens = req.MaxTokens
	}

	if req.Temperature != nil {
		result.Temperature = req.Temperature
	}

	if req.TopP != nil {
		result.TopP = req.TopP
	}

	if len(req.StopSequences) > 0 {
		result.Stop = req.StopSequences
	}

	// 转换 tools
	if len(req.Tools) > 0 {
		tools := make([]OpenAITool, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, OpenAITool{
				Type: "function",
				Function: OpenAIFuncDef{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.InputSchema,
				},
			})
		}
		result.Tools = tools
	}

	// 转换 tool_choice
	if req.ToolChoice != nil {
		switch v := req.ToolChoice.(type) {
		case string:
			if v == "auto" {
				result.ToolChoice = "auto"
			} else if v == "any" {
				result.ToolChoice = "required"
			} else if v == "none" {
				result.ToolChoice = "none"
			}
		case map[string]interface{}:
			if name, ok := v["name"]; ok {
				result.ToolChoice = map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name": name,
					},
				}
			}
		}
	}

	// 转换 thinking
	if req.Thinking != nil {
		result.Thinking = req.Thinking
	}

	return result
}

// anthropicMessageToOpenAI 将单条 Anthropic 消息转换为 OpenAI 格式
func anthropicMessageToOpenAI(msg model.AnthropicMessage) OpenAIMessage {
	// content 是 string 的情况
	if contentStr, ok := msg.Content.(string); ok {
		return OpenAIMessage{
			Role:    msg.Role,
			Content: contentStr,
		}
	}

	// content 是数组的情况
	contentArr, ok := msg.Content.([]interface{})
	if !ok {
		return OpenAIMessage{
			Role:    msg.Role,
			Content: fmt.Sprintf("%v", msg.Content),
		}
	}

	// 对于 assistant 消息中的 tool_use，需要转换为 OpenAI 的 tool_calls 格式
	if msg.Role == "assistant" {
		var textParts []string
		var toolCalls []OpenAIToolCall

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
				toolCalls = append(toolCalls, OpenAIToolCall{
					ID:   id,
					Type: "function",
					Function: OpenAIFunctionCall{
						Name:      name,
						Arguments: argsStr,
					},
				})
			}
		}

		result := OpenAIMessage{
			Role:      "assistant",
			Content:   strings.Join(textParts, "\n"),
			ToolCalls: toolCalls,
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
				toolUseID, _ := block["tool_use_id"].(string)
				resultContent := ""
				if content, ok := block["content"]; ok {
					resultContent = ExtractTextFromAny(content)
				}
				textParts = append(textParts, fmt.Sprintf("[Tool Result %s]: %s", toolUseID, resultContent))
			}
		}

		return OpenAIMessage{
			Role:    "user",
			Content: strings.Join(textParts, "\n"),
		}
	}

	// 其他角色，简单提取文本
	return OpenAIMessage{
		Role:    msg.Role,
		Content: ExtractTextFromAny(msg.Content),
	}
}

// OpenAIToAnthropic 将 OpenAI 响应转换为 Anthropic 格式
func OpenAIToAnthropic(openAIResp []byte, modelName string) model.AnthropicMessagesResponse {
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

	inputTokens, outputTokens, _, cachedTokens := ExtractUsage(resp)

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

// ExtractTextFromAnthropicContent 从 AnthropicContentBlock 数组提取文本
func ExtractTextFromAnthropicContent(blocks []model.AnthropicContentBlock) string {
	var texts []string
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// ExtractTextFromAny 从 Anthropic content 字段提取文本
func ExtractTextFromAny(content interface{}) string {
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