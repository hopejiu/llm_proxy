package converter

import (
	"encoding/json"
	"fmt"
	"time"

	"llm-proxy/internal/model"
)

// OllamaToOpenAI 将 Ollama 请求转换为 OpenAI 格式
func OllamaToOpenAI(req *model.OllamaChatRequest, modelName string) *OpenAIChatRequest {
	messages := make([]OpenAIMessage, 0)

	for _, msg := range req.Messages {
		openaiMsg := ollamaMessageToOpenAI(msg)
		messages = append(messages, openaiMsg)
	}

	// 处理 tools
	var tools []OpenAITool
	if len(req.Tools) > 0 {
		tools = make([]OpenAITool, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, OpenAITool{
				Type: "function",
				Function: OpenAIFuncDef{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			})
		}
	}

	result := &OpenAIChatRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   req.Stream,
	}
	if len(tools) > 0 {
		result.Tools = tools
	}

	return result
}

// ollamaMessageToOpenAI 将单条 Ollama 消息转换为 OpenAI 格式
func ollamaMessageToOpenAI(msg model.OllamaMessage) OpenAIMessage {
	result := OpenAIMessage{
		Role:    msg.Role,
		Content: msg.Content,
	}

	// 处理 images
	if len(msg.Images) > 0 {
		// 将 images 转换为 OpenAI 的多模态内容格式
		content := make([]interface{}, 0)
		if msg.Content != "" {
			content = append(content, map[string]interface{}{
				"type": "text",
				"text": msg.Content,
			})
		}
		for _, img := range msg.Images {
			content = append(content, map[string]interface{}{
				"type": "image_url",
				"image_url": map[string]interface{}{
					"url": fmt.Sprintf("data:image/png;base64,%s", img),
				},
			})
		}
		result.Content = content
	}

	// 处理 tool_calls
	if len(msg.ToolCalls) > 0 {
		toolCalls := make([]OpenAIToolCall, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			toolCalls = append(toolCalls, OpenAIToolCall{
				ID:   fmt.Sprintf("call_%d", time.Now().UnixNano()),
				Type: "function",
				Function: OpenAIFunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		result.ToolCalls = toolCalls
	}

	return result
}

// OpenAIToOllama 将 OpenAI 响应转换为 Ollama 格式
func OpenAIToOllama(openAIResp []byte, modelName string) model.OllamaChatResponse {
	var resp map[string]interface{}
	if err := json.Unmarshal(openAIResp, &resp); err != nil {
		return model.OllamaChatResponse{
			Model: modelName,
			Message: model.OllamaMessage{
				Role:    "assistant",
				Content: string(openAIResp),
			},
			Done: true,
		}
	}

	content := ""
	var toolCalls []model.OllamaToolCall

	if choices, ok := resp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if c, ok := msg["content"].(string); ok {
					content = c
				}
				// 转换 tool_calls
				if tc, ok := msg["tool_calls"].([]interface{}); ok {
					for _, t := range tc {
						if tcMap, ok := t.(map[string]interface{}); ok {
							if fn, ok := tcMap["function"].(map[string]interface{}); ok {
								name, _ := fn["name"].(string)
								argsStr, _ := fn["arguments"].(string)
								toolCalls = append(toolCalls, model.OllamaToolCall{
									Function: model.OllamaToolCallFunction{
										Name:      name,
										Arguments: argsStr,
									},
								})
							}
						}
					}
				}
			}
		}
	}

	return model.OllamaChatResponse{
		Model: modelName,
		Message: model.OllamaMessage{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
		},
		Done:       true,
		DoneReason: "stop",
	}
}

// OpenAIStreamToOllamaChunk 将 OpenAI 流式响应块转换为 Ollama 格式
func OpenAIStreamToOllamaChunk(streamResp map[string]interface{}, modelName string) model.OllamaChatResponse {
	content := ""

	if choices, ok := streamResp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if delta, ok := choice["delta"].(map[string]interface{}); ok {
				if c, ok := delta["content"].(string); ok {
					content = c
				}
			}
		}
	}

	return model.OllamaChatResponse{
		Model: modelName,
		Message: model.OllamaMessage{
			Role:    "assistant",
			Content: content,
		},
		Done: false,
	}
}
