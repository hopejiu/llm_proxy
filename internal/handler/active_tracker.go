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

// ActiveTrackerChange 追踪器变更事件，用于回调通知
type ActiveTrackerChange struct {
	Type      string        `json:"type"`      // "add" | "update" | "remove"
	RequestID string        `json:"request_id"`
	Data      *ActiveRequest `json:"data,omitempty"`
}

// ChangeCallback 追踪器变更回调
type ChangeCallback func(change ActiveTrackerChange)

// ActiveRequestTracker 活跃请求追踪器
type ActiveRequestTracker struct {
	mu              sync.RWMutex
	requests        map[string]*ActiveRequest
	onChange        ChangeCallback
	debounceWindow  time.Duration
	debounceTimers  map[string]time.Time // requestID → last notify time
	debounceMu      sync.Mutex           // protects debounceTimers
}

// NewActiveRequestTracker 创建活跃请求追踪器实例
func NewActiveRequestTracker() *ActiveRequestTracker {
	return &ActiveRequestTracker{
		requests:       make(map[string]*ActiveRequest),
		debounceWindow: 200 * time.Millisecond,
		debounceTimers: make(map[string]time.Time),
	}
}

// SetOnChange 设置变更回调（线程安全，建议在启动时调用一次）
func (t *ActiveRequestTracker) SetOnChange(cb ChangeCallback) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onChange = cb
}

// notify 内部通知回调，update 类型受 debounce 限制
func (t *ActiveRequestTracker) notify(change ActiveTrackerChange) {
	if t.onChange == nil {
		return
	}
	// update 类型按 requestID 做时间限频
	if change.Type == "update" && change.Data != nil {
		t.debounceMu.Lock()
		now := time.Now()
		if last, ok := t.debounceTimers[change.RequestID]; ok && now.Sub(last) < t.debounceWindow {
			t.debounceMu.Unlock()
			return
		}
		t.debounceTimers[change.RequestID] = now
		t.debounceMu.Unlock()
	}
	t.onChange(change)
}

// Add 添加一个活跃请求（立即通知）
func (t *ActiveRequestTracker) Add(req *ActiveRequest) {
	t.mu.Lock()
	t.requests[req.RequestID] = req
	cp := *req
	t.mu.Unlock()
	t.notify(ActiveTrackerChange{Type: "add", RequestID: req.RequestID, Data: &cp})
}

// UpdateStatus 更新请求状态（立即通知）
func (t *ActiveRequestTracker) UpdateStatus(requestID string, status string) {
	t.mu.Lock()
	req, ok := t.requests[requestID]
	if ok {
		req.Status = status
	}
	var cp *ActiveRequest
	if ok {
		c := *req
		cp = &c
	}
	t.mu.Unlock()
	if ok {
		t.notify(ActiveTrackerChange{Type: "update", RequestID: requestID, Data: cp})
	}
}

// UpdateProvider 更新请求的 Provider 信息（立即通知）
func (t *ActiveRequestTracker) UpdateProvider(requestID string, providerID uint, providerName string) {
	t.mu.Lock()
	req, ok := t.requests[requestID]
	if ok {
		req.ProviderID = providerID
		req.Provider = providerName
	}
	var cp *ActiveRequest
	if ok {
		c := *req
		cp = &c
	}
	t.mu.Unlock()
	if ok {
		t.notify(ActiveTrackerChange{Type: "update", RequestID: requestID, Data: cp})
	}
}

// AppendResponse 追加响应内容（流式传输中逐字追加，受 debounce 限制）
func (t *ActiveRequestTracker) AppendResponse(requestID string, content string) {
	t.mu.Lock()
	req, ok := t.requests[requestID]
	if ok {
		req.ResponseContent += content
	}
	var cp *ActiveRequest
	if ok {
		c := *req
		cp = &c
	}
	t.mu.Unlock()
	if ok {
		t.notify(ActiveTrackerChange{Type: "update", RequestID: requestID, Data: cp})
	}
}

// AddToolCall 添加一个新的工具调用（立即通知）
func (t *ActiveRequestTracker) AddToolCall(requestID string, id string, name string) {
	t.mu.Lock()
	req, ok := t.requests[requestID]
	if ok {
		req.ToolCalls = append(req.ToolCalls, ActiveToolCall{
			ID:   id,
			Name: name,
		})
	}
	var cp *ActiveRequest
	if ok {
		c := *req
		cp = &c
	}
	t.mu.Unlock()
	if ok {
		t.notify(ActiveTrackerChange{Type: "update", RequestID: requestID, Data: cp})
	}
}

// AppendToolCallArgs 追加工具调用的参数（流式增量，受 debounce 限制）
func (t *ActiveRequestTracker) AppendToolCallArgs(requestID string, toolIndex int, args string) {
	t.mu.Lock()
	req, ok := t.requests[requestID]
	if ok {
		if toolIndex >= 0 && toolIndex < len(req.ToolCalls) {
			req.ToolCalls[toolIndex].Arguments += args
		}
	}
	var cp *ActiveRequest
	if ok {
		c := *req
		cp = &c
	}
	t.mu.Unlock()
	if ok {
		t.notify(ActiveTrackerChange{Type: "update", RequestID: requestID, Data: cp})
	}
}

// Remove 移除一个活跃请求（请求完成时调用，立即通知）
func (t *ActiveRequestTracker) Remove(requestID string) {
	t.mu.Lock()
	delete(t.requests, requestID)
	t.mu.Unlock()
	// 清理 debounce 计时器，防止内存泄漏
	t.debounceMu.Lock()
	delete(t.debounceTimers, requestID)
	t.debounceMu.Unlock()
	t.notify(ActiveTrackerChange{Type: "remove", RequestID: requestID})
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

// GetByID 按 ID 获取单个活跃请求的快照
func (t *ActiveRequestTracker) GetByID(requestID string) *ActiveRequest {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if req, ok := t.requests[requestID]; ok {
		cp := *req
		return &cp
	}
	return nil
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

// DebounceWindow 返回当前 debounce 窗口
func (t *ActiveRequestTracker) DebounceWindow() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.debounceWindow
}
