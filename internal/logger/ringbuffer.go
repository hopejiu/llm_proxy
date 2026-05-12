package logger

import "sync"

// LogEntry 日志条目
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// RingBuffer 固定大小环形缓冲区，线程安全
type RingBuffer struct {
	mu          sync.RWMutex
	entries     []LogEntry
	cap         int
	pos         int // 下一个写入位置
	full        bool
	subscribers []chan LogEntry
	subMu       sync.RWMutex
}

// NewRingBuffer 创建环形缓冲区
func NewRingBuffer(cap int) *RingBuffer {
	if cap <= 0 {
		cap = 1000
	}
	return &RingBuffer{
		entries: make([]LogEntry, cap),
		cap:     cap,
	}
}

// Write 写入一条日志，满时覆盖最旧
func (rb *RingBuffer) Write(entry LogEntry) {
	rb.mu.Lock()
	rb.entries[rb.pos] = entry
	rb.pos = (rb.pos + 1) % rb.cap
	if rb.pos == 0 {
		rb.full = true
	}
	rb.mu.Unlock()

	// 通知订阅者（非阻塞）
	rb.subMu.RLock()
	for _, ch := range rb.subscribers {
		select {
		case ch <- entry:
		default:
			// channel 满则丢弃，不阻塞业务逻辑
		}
	}
	rb.subMu.RUnlock()
}

// GetAll 获取全部历史日志（按时间顺序）
func (rb *RingBuffer) GetAll() []LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if !rb.full && rb.pos == 0 {
		return nil
	}

	var result []LogEntry
	if rb.full {
		result = make([]LogEntry, rb.cap)
		copy(result, rb.entries[rb.pos:])
		copy(result[rb.cap-rb.pos:], rb.entries[:rb.pos])
	} else {
		result = make([]LogEntry, rb.pos)
		copy(result, rb.entries[:rb.pos])
	}
	return result
}

// Subscribe 订阅新日志
func (rb *RingBuffer) Subscribe() <-chan LogEntry {
	ch := make(chan LogEntry, 64)
	rb.subMu.Lock()
	rb.subscribers = append(rb.subscribers, ch)
	rb.subMu.Unlock()
	return ch
}

// Unsubscribe 取消订阅
func (rb *RingBuffer) Unsubscribe(ch <-chan LogEntry) {
	rb.subMu.Lock()
	defer rb.subMu.Unlock()
	for i, sub := range rb.subscribers {
		if sub == ch {
			rb.subscribers = append(rb.subscribers[:i], rb.subscribers[i+1:]...)
			close(sub)
			return
		}
	}
}
