package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/repository"
)

const (
	SessionMarkPrefix = "[SESSION]"
	SessionMarkSuffix = "[/SESSION]"
)

// sessionMarkRegex 匹配 [SESSION]数字[/SESSION]
var sessionMarkRegex = regexp.MustCompile(`\[SESSION\](\d+)\[/SESSION\]`)

// SessionService 会话管理器，封装会话 ID 的提取、标记注入和清除逻辑
type SessionService struct {
	repo *repository.ChatSessionRepository
}

// NewSessionService 创建 SessionService
func NewSessionService(repo *repository.ChatSessionRepository) *SessionService {
	return &SessionService{repo: repo}
}

// ResolveSession 从请求体中解析会话 ID，返回会话ID、清除标记后的请求体、是否为新会话
// 流程：
//  1. 扫描请求体中的 messages 数组，查找第一条 assistant 消息中的 [SESSION]ID[/SESSION]
//  2. 如果找到 → 提取 ID，从 content 中移除标记
//  3. 如果没找到 → 创建新会话，返回新 ID
func (s *SessionService) ResolveSession(body []byte) (sessionID uint, cleanBody []byte, isNew bool) {
	if !bytes.Contains(body, []byte(SessionMarkPrefix)) {
		return s.createNewSession(body)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return s.createNewSession(body)
	}

	messages, ok := data["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		return s.createNewSession(body)
	}

	sessionID, modifiedMessages := extractSessionFromMessages(messages)
	if sessionID == 0 {
		return s.createNewSession(body)
	}

	// 更新 body 中的 messages
	data["messages"] = modifiedMessages
	cleanBody, _ = json.Marshal(data)

	return sessionID, cleanBody, false
}

// createNewSession 创建新会话
func (s *SessionService) createNewSession(body []byte) (uint, []byte, bool) {
	session := &model.ChatSession{}
	if err := s.repo.Create(session); err != nil {
		return 0, body, true
	}
	return session.ID, body, true
}

// RecalcSessionStats 请求结束后重新聚合会话统计
func (s *SessionService) RecalcSessionStats(sessionID uint) {
	if sessionID == 0 {
		return
	}
	s.repo.RecalcSessionStats(sessionID)
}

// BuildSessionSuffix 构建 [SESSION]ID[/SESSION] 后缀
func BuildSessionSuffix(sessionID uint) string {
	if sessionID == 0 {
		return ""
	}
	return fmt.Sprintf("%s%d%s", SessionMarkPrefix, sessionID, SessionMarkSuffix)
}

// ExtractSessionIDStr 从 content 中提取 [SESSION]数字[/SESSION] 的数字部分
func ExtractSessionIDStr(content string) uint {
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

// StripSessionMark 从 content 中移除 [SESSION]数字[/SESSION]
func StripSessionMark(content string) string {
	return sessionMarkRegex.ReplaceAllString(content, "")
}

// HasSessionMark 检查 content 中是否包含会话标记
func HasSessionMark(content string) bool {
	return sessionMarkRegex.MatchString(content)
}

// StripSessionMarkFromString 从字符串中移除所有会话标记
func StripSessionMarkFromString(content string) string {
	return sessionMarkRegex.ReplaceAllString(content, "")
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

		id := ExtractSessionIDStr(content)
		if id == 0 {
			continue
		}

		// 移除标记
		msg["content"] = StripSessionMark(content)
		modified[i] = msg
		return id, modified
	}

	return 0, messages
}

// BuildStreamMarkChunk 构建流式会话标记 SSE 数据行
func BuildStreamMarkChunk(requestID string, sessionID uint) string {
	if sessionID == 0 {
		return ""
	}
	suffix := BuildSessionSuffix(sessionID)
	return fmt.Sprintf(`{"id":"%s","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"%s"}}]}`,
		requestID, suffix)
}
