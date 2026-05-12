package logger

import (
	"context"
	"log/slog"
)

// multiHandler 自定义 slog Handler，同时写入底层 handler + RingBuffer
type multiHandler struct {
	inner     slog.Handler // 写入日志文件 + stdout 的原始 handler
	ringBuf   *RingBuffer
}

// NewMultiHandler 创建多目标 handler
func NewMultiHandler(inner slog.Handler, ringBuf *RingBuffer) slog.Handler {
	return &multiHandler{
		inner:   inner,
		ringBuf: ringBuf,
	}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	// 先写底层 handler（文件 + stdout）
	if err := h.inner.Handle(ctx, r); err != nil {
		return err
	}

	// 再写 RingBuffer（极轻量）
	if h.ringBuf != nil {
		// 拼接 attrs 到 message，确保键值对信息不丢失
		msg := r.Message
		r.Attrs(func(a slog.Attr) bool {
			msg += " " + a.Key + "=" + a.Value.String()
			return true
		})

		h.ringBuf.Write(LogEntry{
			Time:    r.Time.Format("15:04:05.000"),
			Level:   formatLevel(r.Level),
			Message: msg,
		})
	}

	return nil
}

// formatLevel 将 slog.Level 转为小写简短格式，与前端过滤匹配
func formatLevel(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "error"
	case l >= slog.LevelWarn:
		return "warn"
	case l >= slog.LevelInfo:
		return "info"
	default:
		return "debug"
	}
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &multiHandler{
		inner:   h.inner.WithAttrs(attrs),
		ringBuf: h.ringBuf,
	}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	return &multiHandler{
		inner:   h.inner.WithGroup(name),
		ringBuf: h.ringBuf,
	}
}
