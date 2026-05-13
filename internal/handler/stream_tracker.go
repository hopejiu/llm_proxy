package handler

import "fmt"

// trackToolCallsFromDelta 从流式 tool_calls delta 中提取信息并更新追踪器
// 当新增 tool_call 时，将函数名追加到 response_content，确保前端能看到工具调用信息
func trackToolCallsFromDelta(toolCallsDelta []interface{}, requestID string, tracker *ActiveRequestTracker) {
	for _, tc := range toolCallsDelta {
		tcMap, ok := tc.(map[string]interface{})
		if !ok {
			continue
		}
		toolIdx := 0
		if idx, ok := tcMap["index"].(float64); ok {
			toolIdx = int(idx)
		}
		// 新的 tool_call（有 id）
		if id, ok := tcMap["id"].(string); ok && id != "" {
			fnName := ""
			if fn, ok := tcMap["function"].(map[string]interface{}); ok {
				if name, ok := fn["name"].(string); ok {
					fnName = name
				}
			}
			tracker.AddToolCall(requestID, id, fnName)
			// 将工具调用信息追加到 response_content，确保前端能看到
			if fnName != "" {
				tracker.AppendResponse(requestID, fmt.Sprintf("[调用工具: %s]", fnName))
			}
		}
		// tool_call 的 arguments 增量
		if fn, ok := tcMap["function"].(map[string]interface{}); ok {
			if args, ok := fn["arguments"].(string); ok && args != "" {
				tracker.AppendToolCallArgs(requestID, toolIdx, args)
			}
		}
	}
}