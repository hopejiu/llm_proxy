package handler

// trackToolCallsFromDelta 从流式 delta 中提取 tool_calls 信息并更新追踪器
func trackToolCallsFromDelta(delta map[string]interface{}, requestID string, tracker *ActiveRequestTracker) {
	toolCallsDelta, ok := delta["tool_calls"].([]interface{})
	if !ok {
		return
	}
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
		}
		// tool_call 的 arguments 增量
		if fn, ok := tcMap["function"].(map[string]interface{}); ok {
			if args, ok := fn["arguments"].(string); ok && args != "" {
				tracker.AppendToolCallArgs(requestID, toolIdx, args)
			}
		}
	}
}
