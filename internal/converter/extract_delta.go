package converter

// DeltaResult 从 SSE chunk 的 delta 中提取的结果
type DeltaResult struct {
	Content          string
	ReasoningContent string
	ToolCallsDelta   []interface{}
}

// ExtractDeltaFromChunk 从 OpenAI 格式的 SSE chunk 中提取 delta 内容
func ExtractDeltaFromChunk(chunk map[string]interface{}) DeltaResult {
	var result DeltaResult
	choices, ok := chunk["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return result
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return result
	}
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return result
	}
	if content, ok := delta["content"].(string); ok {
		result.Content = content
	}
	// reasoning_content: DeepSeek 等使用的字段名
	if reasoning, ok := delta["reasoning_content"].(string); ok && reasoning != "" {
		result.ReasoningContent = reasoning
	}
	// reasoning: OpenAI 推荐的新字段名（vLLM 等已迁移）
	if result.ReasoningContent == "" {
		if reasoning, ok := delta["reasoning"].(string); ok && reasoning != "" {
			result.ReasoningContent = reasoning
		}
	}
	if toolCalls, ok := delta["tool_calls"].([]interface{}); ok {
		result.ToolCallsDelta = toolCalls
	}
	return result
}
