package logger

import (
	"io"
	"log/slog"
	"os"
)

var (
	// L 全局默认 Logger
	L *slog.Logger
	// fileWriter 日志文件的 io.Writer，供多输出复用
	fileWriter io.Writer
)

// Init 初始化全局 Logger，同时输出到 stdout 和指定日志文件
func Init(logFilePath string) error {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	fileWriter = logFile

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	L = slog.New(slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(L)
	return nil
}

// FileWriter 返回日志文件的 io.Writer，供其他需要文件写入的场景复用
func FileWriter() io.Writer {
	return fileWriter
}
