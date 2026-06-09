package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/repository"
)

const (
	sessionMarkPrefix = "[SESSION]"
	sessionMarkSuffix = "[/SESSION]"
)

// sessionMarkRegex 匹配 [SESSION]数字[/SESSION]
var sessionMarkRegex = regexp.MustCompile(`\[SESSION\](\d+)\[/SESSION\]`)

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

// SessionManager 会话管理器，封装会话 ID 的提取、标记注入和清除逻辑
type SessionManager struct {
	repo *repository.ChatSessionRepository
}

// NewSessionManager 创建 SessionManager
func NewSessionManager(repo *repository.ChatSessionRepository) *SessionManager {
	return &SessionManager{repo: repo}
}

// ResolveSession 从请求体中解析会话 ID，返回会话ID、清除标记后的请求体、是否为新会话
// 流程：
//  1. 扫描请求体中的 messages 数组，查找第一条 assistant 消息中的 [SESSION]ID[/SESSION]
//  2. 如果找到 → 提取 ID，从 content 中移除标记
//  3. 如果没找到 → 创建新会话，返回新 ID
func (sm *SessionManager) ResolveSession(body []byte) (sessionID uint, cleanBody []byte, isNew bool) {
	if !bytes.Contains(body, []byte(sessionMarkPrefix)) {
		return sm.createNewSession(body)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return sm.createNewSession(body)
	}

	messages, ok := data["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		return sm.createNewSession(body)
	}

	sessionID, modifiedMessages := extractSessionFromMessages(messages)
	if sessionID == 0 {
		return sm.createNewSession(body)
	}

	// 更新 body 中的 messages
	data["messages"] = modifiedMessages
	cleanBody, _ = json.Marshal(data)

	return sessionID, cleanBody, false
}

// createNewSession 创建新会话
func (sm *SessionManager) createNewSession(body []byte) (uint, []byte, bool) {
	session := &model.ChatSession{}
	if err := sm.repo.Create(session); err != nil {
		return 0, body, true
	}
	return session.ID, body, true
}

// RecalcSessionStats 请求结束后重新聚合会话统计
func (sm *SessionManager) RecalcSessionStats(sessionID uint) {
	if sessionID == 0 {
		return
	}
	sm.repo.RecalcSessionStats(sessionID)
}

// InjectSessionMarkIntoResponse 将会话标记注入到非流式响应体中
// 在 choices[0].message.content 或等效字段末尾追加 [SESSION]ID[/SESSION]
func InjectSessionMarkIntoResponse(respBody []byte, sessionID uint, protocol string) []byte {
	if sessionID == 0 {
		return respBody
	}

	suffix := fmt.Sprintf("%s%d%s", sessionMarkPrefix, sessionID, sessionMarkSuffix)

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
	return content + fmt.Sprintf("%s%d%s", sessionMarkPrefix, sessionID, sessionMarkSuffix)
}

// injectIntoOpenAIResponse 在 OpenAI 响应 choices[0].message.content 末尾追加标记
// 兼容 content 为 null/不存在/空的场景：强行创建 content 字段放入 [SESSION]ID[/SESSION]
// 防重复：如果 content 中已有 [SESSION] 标记，不再追加
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
	slog.Info("[session] injectIntoOpenAIResponse",
		"content_exists", msg["content"] != nil,
		"content_empty", content == "",
		"has_mark", strings.Contains(content, sessionMarkPrefix))
	if strings.Contains(content, sessionMarkPrefix) {
		slog.Info("[session] 注入跳过：content 已有标记", "content", content[:min(len(content), 100)])
		return
	}
	if content != "" {
		msg["content"] = content + suffix
		slog.Info("[session] 注入追加到现有 content", "len", len(content))
	} else {
		msg["content"] = suffix // content 为 null/空时强行创建
		slog.Info("[session] 注入强制创建 content", "suffix", suffix)
	}
}

// injectIntoAnthropicResponse 在 Anthropic 响应 content[0].text 末尾追加标记
// 兼容 text 不存在（如 tool_use 块）的场景：强行创建 text 字段
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
	slog.Info("[session] injectIntoAnthropicResponse",
		"text_empty", text == "", "has_mark", strings.Contains(text, sessionMarkPrefix))
	if strings.Contains(text, sessionMarkPrefix) {
		return
	}
	if text != "" {
		first["text"] = text + suffix
	} else {
		first["text"] = suffix
		slog.Info("[session] 注入强制创建 Anthropic text", "suffix", suffix)
	}
}

// injectIntoOllamaResponse 在 Ollama 响应 message.content 末尾追加标记
// 兼容 content 为 null/空的场景：强行创建 content 字段
func injectIntoOllamaResponse(data map[string]interface{}, suffix string) {
	msg, ok := data["message"].(map[string]interface{})
	if !ok {
		return
	}
	content, _ := msg["content"].(string)
	slog.Info("[session] injectIntoOllamaResponse",
		"content_empty", content == "", "has_mark", strings.Contains(content, sessionMarkPrefix))
	if strings.Contains(content, sessionMarkPrefix) {
		return
	}
	if content != "" {
		msg["content"] = content + suffix
	} else {
		msg["content"] = suffix
		slog.Info("[session] 注入强制创建 Ollama content", "suffix", suffix)
	}
}

// extractSessionFromMessages 从 messages 数组中提取会话 ID
// 规则：扫描所有 messages，找到第一条 role=assistant 且有 [SESSION] 标记的消息，提取 ID 并移除标记
func extractSessionFromMessages(messages []interface{}) (uint, []interface{}) {
	modified := make([]interface{}, len(messages))
	copy(modified, messages)

	for i, raw := range modified {
		msg, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		if role != "assistant" {
			continue
		}
		content, _ := msg["content"].(string)
		if content == "" {
			continue
		}

		id := extractSessionIDStr(content)
		if id == 0 {
			continue
		}

		// 移除标记
		msg["content"] = stripSessionMark(content)
		modified[i] = msg
		return id, modified
	}

	return 0, messages
}

// extractSessionIDStr 从 content 中提取 [SESSION]数字[/SESSION] 的数字部分
func extractSessionIDStr(content string) uint {
	matches := sessionMarkRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return 0
	}
	id, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

// stripSessionMark 从 content 中移除 [SESSION]数字[/SESSION]
func stripSessionMark(content string) string {
	return sessionMarkRegex.ReplaceAllString(content, "")
}

// HasSessionMark 检查 content 中是否包含会话标记
func HasSessionMark(content string) bool {
	return sessionMarkRegex.MatchString(content)
}

// StripSessionMarkFromString 从字符串中移除所有会话标记（供外部包使用）
func StripSessionMarkFromString(content string) string {
	return sessionMarkRegex.ReplaceAllString(content, "")
}


