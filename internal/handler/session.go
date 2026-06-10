package handler

import (
	"context"
	"encoding/json"

	"github.com/wanglejiu/llm-proxy/internal/service"
)

// sessionIDKey 用于在 context 中存储 sessionID
type sessionIDKey struct{}

// sessionNewKey 用于在 context 中标记是否为新创建的会话
type sessionNewKey struct{}

// contextWithSessionID 将 sessionID 存入 context
func contextWithSessionID(ctx context.Context, sessionID uint) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, sessionID)
}

// sessionIDFromContext 从 context 中获取 sessionID
func sessionIDFromContext(ctx context.Context) uint {
	if id, ok := ctx.Value(sessionIDKey{}).(uint); ok {
		return id
	}
	return 0
}

// contextWithSessionNew 标记该请求是否是新创建的会话
func contextWithSessionNew(ctx context.Context, isNew bool) context.Context {
	return context.WithValue(ctx, sessionNewKey{}, isNew)
}

// isNewSession 判断该请求是否是新创建的会话（仅首次注入标记）
func isNewSession(ctx context.Context) bool {
	v, _ := ctx.Value(sessionNewKey{}).(bool)
	return v
}

// InjectSessionMarkIntoResponse 将会话标记注入到非流式响应体中
// 在 choices[0].message.content 或等效字段末尾追加 [SESSION]ID[/SESSION]
func InjectSessionMarkIntoResponse(respBody []byte, sessionID uint, protocol string) []byte {
	if sessionID == 0 {
		return respBody
	}

	suffix := service.BuildSessionSuffix(sessionID)

	var data map[string]interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return respBody
	}

	switch protocol {
	case "openai":
		injectIntoOpenAIResponse(data, suffix)
	case "anthropic":
		injectIntoAnthropicResponse(data, suffix)
	case "ollama":
		injectIntoOllamaResponse(data, suffix)
	}

	newBody, _ := json.Marshal(data)
	return newBody
}

// InjectSessionMarkIntoStreamContent 将会话标记注入到流式的最后一条 content 末尾
// 返回修改后的 content（末尾追加标记）
func InjectSessionMarkIntoStreamContent(content string, sessionID uint) string {
	if sessionID == 0 {
		return content
	}
	return content + service.BuildSessionSuffix(sessionID)
}

// WriteStreamSessionMark 在流式响应 [DONE] 前写入会话标记（统一模式）
// 所有三种协议的流式 handler 复用此方法
func WriteStreamSessionMark(c interface{ Write([]byte) (int, error); Flush() }, ctx context.Context, requestID string, sessionID uint, injected *bool) {
	if *injected || sessionID == 0 || !isNewSession(ctx) {
		return
	}
	markLine := service.BuildStreamMarkChunk(requestID, sessionID)
	if markLine == "" {
		return
	}
	c.Write([]byte("data: " + markLine + "\n\n"))
	c.Flush()
	*injected = true
}

// injectIntoOpenAIResponse 在 OpenAI 响应 choices[0].message.content 末尾追加标记
func injectIntoOpenAIResponse(data map[string]interface{}, suffix string) {
	choices, ok := data["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return
	}
	msg, ok := choice["message"].(map[string]interface{})
	if !ok {
		return
	}
	content, _ := msg["content"].(string)
	if service.HasSessionMark(content) {
		return
	}
	if content != "" {
		msg["content"] = content + suffix
	} else {
		msg["content"] = suffix
	}
}

// injectIntoAnthropicResponse 在 Anthropic 响应 content[0].text 末尾追加标记
func injectIntoAnthropicResponse(data map[string]interface{}, suffix string) {
	contentArr, ok := data["content"].([]interface{})
	if !ok || len(contentArr) == 0 {
		return
	}
	first, ok := contentArr[0].(map[string]interface{})
	if !ok {
		return
	}
	text, _ := first["text"].(string)
	if service.HasSessionMark(text) {
		return
	}
	if text != "" {
		first["text"] = text + suffix
	} else {
		first["text"] = suffix
	}
}

// injectIntoOllamaResponse 在 Ollama 响应 message.content 末尾追加标记
func injectIntoOllamaResponse(data map[string]interface{}, suffix string) {
	msg, ok := data["message"].(map[string]interface{})
	if !ok {
		return
	}
	content, _ := msg["content"].(string)
	if service.HasSessionMark(content) {
		return
	}
	if content != "" {
		msg["content"] = content + suffix
	} else {
		msg["content"] = suffix
	}
}
