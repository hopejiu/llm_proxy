package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// ParseLevel 将字符串解析为 slog.Level
func ParseLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// 全局 RingBuffer 引用，供 App 层获取
var globalRingBuffer *RingBuffer

// 全局日志文件句柄，用于关闭和复用
var globalLogFile *os.File

// GetRingBuffer 获取全局 RingBuffer
func GetRingBuffer() *RingBuffer {
	return globalRingBuffer
}

// Init 初始化全局 Logger，同时输出到 stdout 和指定日志文件
// 如果 ringBuf 不为 nil，日志同时写入 RingBuffer
func Init(logFilePath string, level slog.Level, ringBuf ...*RingBuffer) error {
	// 关闭之前的日志文件（支持重新初始化）
	if globalLogFile != nil {
		globalLogFile.Close()
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	globalLogFile = logFile

	writers := []io.Writer{logFile}
	// GUI 程序中 stdout 可能不可用，仅在有效时添加
	if os.Stdout != nil {
		writers = append(writers, os.Stdout)
	}
	multiWriter := io.MultiWriter(writers...)

	innerHandler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: level,
	})

	var handler slog.Handler = innerHandler

	// 如果传入了 RingBuffer，包装为 multiHandler
	if len(ringBuf) > 0 && ringBuf[0] != nil {
		globalRingBuffer = ringBuf[0]
		handler = NewMultiHandler(innerHandler, globalRingBuffer)
	}

	l := slog.New(handler)
	slog.SetDefault(l)
	return nil
}

// Sync 刷盘日志文件
func Sync() {
	if globalLogFile != nil {
		globalLogFile.Sync()
	}
}
