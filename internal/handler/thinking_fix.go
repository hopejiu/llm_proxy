package handler

import (
	"encoding/json"
	"log/slog"

	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/repository"
)

// FixThinkingChain 自动为 messages 中的 assistant 消息补全 reasoning_content
// 当 Provider 开启了 AutoFixThinking 时，从 request_logs 中实时聚合历史 thinking_content，
// 按 content 匹配策略注入到对应的 assistant 消息中。
func (h *BaseHandler) FixThinkingChain(body []byte, sessionID uint, provider model.ProviderConfig) []byte {
	if !provider.AutoFixThinking {
		return body
	}
	if sessionID == 0 {
		slog.Debug("[思维链修复] sessionID=0，跳过", "provider", provider.Name)
		return body
	}

	// 查询思维链历史
	entries := h.requestLogRepo.GetThinkingHistory(sessionID)
	if len(entries) == 0 {
		slog.Debug("[思维链修复] 无历史 thinking_content，跳过", "provider", provider.Name, "sessionID", sessionID)
		return body
	}

	// 解析请求体
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		slog.Warn("[思维链修复] 解析请求体失败", "error", err)
		return body
	}

	messages, ok := data["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		return body
	}

	// 构建消费标记数组
	consumed := make([]bool, len(entries))

	// 游标：每次新请求从 0 开始
	cursor := 0
	fixedCount := 0

	for i, raw := range messages {
		msg, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		if role != "assistant" {
			continue
		}

		content, _ := msg["content"].(string)

		// 尝试匹配并注入 reasoning_content
		matchIdx := h.findThinkingMatch(entries, consumed, content, cursor)
		if matchIdx < 0 {
			// 从 cursor 到末尾没找到，重置游标从头搜索
			matchIdx = h.findThinkingMatch(entries, consumed, content, 0)
		}
		if matchIdx < 0 {
			// 仍然找不到，跳过该 assistant 消息
			continue
		}

		// 注入 reasoning_content
		consumed[matchIdx] = true
		cursor = matchIdx + 1
		msg["reasoning_content"] = entries[matchIdx].ThinkingContent
		messages[i] = msg
		fixedCount++
	}

	if fixedCount == 0 {
		slog.Debug("[思维链修复] 无需修复", "provider", provider.Name, "sessionID", sessionID, "historyCount", len(entries))
		return body
	}

	// 更新 messages 并序列化
	data["messages"] = messages
	newBody, err := json.Marshal(data)
	if err != nil {
		slog.Warn("[思维链修复] 序列化失败", "error", err)
		return body
	}

	slog.Info("[思维链修复] 已注入 reasoning_content",
		"provider", provider.Name,
		"sessionID", sessionID,
		"fixedCount", fixedCount,
		"historyCount", len(entries))

	return newBody
}

// findThinkingMatch 在 entries 中从 startIdx 开始查找匹配的条目
// 匹配策略：
//  1. content 非空时：response_content == content 且未消费
//  2. content 为空时：按顺序取第一个未消费的条目（兜底）
//
// 返回匹配的索引，-1 表示未找到
func (h *BaseHandler) findThinkingMatch(entries []repository.ThinkingEntry, consumed []bool, content string, startIdx int) int {
	if len(entries) == 0 {
		return -1
	}

	// 确保 startIdx 在范围内
	if startIdx >= len(entries) {
		startIdx = 0
	}

	if content != "" {
		// 策略 1：按 content 精确匹配
		for i := startIdx; i < len(entries); i++ {
			if !consumed[i] && entries[i].ResponseContent == content {
				return i
			}
		}
		return -1
	}

	// 策略 2：空 content 兜底 — 按顺序取第一个未消费的
	for i := startIdx; i < len(entries); i++ {
		if !consumed[i] {
			return i
		}
	}
	return -1
}
