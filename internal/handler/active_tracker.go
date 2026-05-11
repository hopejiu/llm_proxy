package handler

import (
	"sort"
	"sync"
	"time"
)

// ActiveToolCall 活跃请求中的工具调用信息
type ActiveToolCall struct {
	ID        string `json:"id"`         // tool_call id
	Name      string `json:"name"`       // function name
	Arguments string `json:"arguments"`  // function arguments（逐步追加）
}

// ActiveRequest 正在进行的请求信息
type ActiveRequest struct {
	RequestID       string           `json:"request_id"`
	ProviderID      uint             `json:"provider_id"`
	Provider        string           `json:"provider"`
	Model           string           `json:"model"`
	RequestBody     string           `json:"request_body"`
	ResponseContent string           `json:"response_content"` // 实时响应内容（流式逐字追加）
	ToolCalls       []ActiveToolCall `json:"tool_calls"`       // 实时工具调用列表
	Status          string           `json:"status"`           // "pending" | "streaming" | "error"
	StartTime       time.Time        `json:"start_time"`
	Protocol        string           `json:"protocol"` // "openai" | "anthropic" | "ollama"
	ClientIP        string           `json:"client_ip"`
}

// ActiveRequestTracker 活跃请求追踪器
type ActiveRequestTracker struct {
	mu       sync.RWMutex
	requests map[string]*ActiveRequest
}

// NewActiveRequestTracker 创建活跃请求追踪器实例
func NewActiveRequestTracker() *ActiveRequestTracker {
	return &ActiveRequestTracker{
		requests: make(map[string]*ActiveRequest),
	}
}

// Add 添加一个活跃请求
func (t *ActiveRequestTracker) Add(req *ActiveRequest) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.requests[req.RequestID] = req
}

// UpdateStatus 更新请求状态
func (t *ActiveRequestTracker) UpdateStatus(requestID string, status string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if req, ok := t.requests[requestID]; ok {
		req.Status = status
	}
}

// UpdateProvider 更新请求的 Provider 信息
func (t *ActiveRequestTracker) UpdateProvider(requestID string, providerID uint, providerName string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if req, ok := t.requests[requestID]; ok {
		req.ProviderID = providerID
		req.Provider = providerName
	}
}

// AppendResponse 追加响应内容（流式传输中逐字追加）
func (t *ActiveRequestTracker) AppendResponse(requestID string, content string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if req, ok := t.requests[requestID]; ok {
		req.ResponseContent += content
	}
}

// AddToolCall 添加一个新的工具调用（当流中出现新的 tool_call 时调用）
func (t *ActiveRequestTracker) AddToolCall(requestID string, id string, name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if req, ok := t.requests[requestID]; ok {
		req.ToolCalls = append(req.ToolCalls, ActiveToolCall{
			ID:   id,
			Name: name,
		})
	}
}

// AppendToolCallArgs 追加工具调用的参数（流式增量）
func (t *ActiveRequestTracker) AppendToolCallArgs(requestID string, toolIndex int, args string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if req, ok := t.requests[requestID]; ok {
		if toolIndex >= 0 && toolIndex < len(req.ToolCalls) {
			req.ToolCalls[toolIndex].Arguments += args
		}
	}
}

// Remove 移除一个活跃请求（请求完成时调用）
func (t *ActiveRequestTracker) Remove(requestID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.requests, requestID)
}

// GetAll 获取所有活跃请求的快照（按开始时间降序）
func (t *ActiveRequestTracker) GetAll() []ActiveRequest {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]ActiveRequest, 0, len(t.requests))
	for _, req := range t.requests {
		result = append(result, *req)
	}

	// 按开始时间降序排列
	sortActiveRequests(result)
	return result
}

// Count 获取活跃请求数量
func (t *ActiveRequestTracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.requests)
}

// sortActiveRequests 按开始时间降序排列
func sortActiveRequests(reqs []ActiveRequest) {
	sort.Slice(reqs, func(i, j int) bool {
		return reqs[i].StartTime.After(reqs[j].StartTime)
	})
}

// generateActiveRequestID 生成活跃请求 ID（复用 generateRequestID）
func generateActiveRequestID() string {
	return generateRequestID()
}
