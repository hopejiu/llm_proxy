package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// 全局日志文件句柄，用于关闭和复用
var globalLogFile *os.File

// Init 初始化全局 Logger，同时输出到 stdout 和指定日志文件
func Init(logFilePath string, level slog.Level) error {
	if globalLogFile != nil {
		globalLogFile.Close()
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	globalLogFile = logFile

	writers := []io.Writer{logFile}
	if os.Stdout != nil {
		writers = append(writers, os.Stdout)
	}
	baseWriter := io.MultiWriter(writers...)

	handler := slog.NewTextHandler(baseWriter, &slog.HandlerOptions{
		Level: level,
	})

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

// Close 关闭日志文件
func Close() {
	if globalLogFile != nil {
		globalLogFile.Close()
		globalLogFile = nil
	}
}

// ArchiveLogFile 归档当前日志文件
// 将 logFilePath 的内容追加到同目录下的 llm-proxy-YYYY-MM-DD.log
// 然后删除原文件，让 Init 创建新的空文件
func ArchiveLogFile(logFilePath string) error {
	info, err := os.Stat(logFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Size() == 0 {
		return nil
	}

	// 读取原文件内容
	data, err := os.ReadFile(logFilePath)
	if err != nil {
		return err
	}

	// 归档文件名：llm-proxy-2026-05-13.log
	dir := filepath.Dir(logFilePath)
	archiveName := "llm-proxy-" + time.Now().Format("2006-01-02") + ".log"
	archivePath := filepath.Join(dir, archiveName)

	// 追加到归档文件（同一天多次启动合并）
	f, err := os.OpenFile(archivePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	f.Write(data)
	f.Close()

	// 删除原文件
	return os.Remove(logFilePath)
}

// CleanOldArchives 清理过期的归档日志文件
func CleanOldArchives(logDir string, keepDays int) error {
	if keepDays <= 0 {
		keepDays = 3
	}

	entries, err := os.ReadDir(logDir)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -keepDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 匹配 llm-proxy-YYYY-MM-DD.log 格式
		if !strings.HasPrefix(name, "llm-proxy-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		// 提取日期部分：llm-proxy-2026-05-13.log → 2026-05-13
		dateStr := strings.TrimPrefix(name, "llm-proxy-")
		dateStr = strings.TrimSuffix(dateStr, ".log")
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			os.Remove(filepath.Join(logDir, name))
		}
	}

	return nil
}
