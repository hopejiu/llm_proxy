package converter

// ExtractUsage 从 OpenAI 格式响应中提取 token 用量
func ExtractUsage(resp map[string]interface{}) (inputTokens, outputTokens, totalTokens, cachedTokens int) {
	if usage, ok := resp["usage"].(map[string]interface{}); ok {
		if pt, ok := usage["prompt_tokens"].(float64); ok {
			inputTokens = int(pt)
		}
		if ct, ok := usage["completion_tokens"].(float64); ok {
			outputTokens = int(ct)
		}
		if tt, ok := usage["total_tokens"].(float64); ok {
			totalTokens = int(tt)
		}
		if cc, ok := usage["prompt_tokens_details"].(map[string]interface{}); ok {
			if cr, ok := cc["cached_tokens"].(float64); ok {
				cachedTokens = int(cr)
			}
		}
	}
	return
}
