package logger

import (
	"io"
	"log/slog"
	"os"
)

// Init 初始化全局 Logger，同时输出到 stdout 和指定日志文件
func Init(logFilePath string) error {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	l := slog.New(slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(l)
	return nil
}
