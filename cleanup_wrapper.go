package main

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/wanglejiu/llm-proxy/internal/service"
	"log/slog"
)

// CleanupServiceWrapper 包装 internal/service.CleanupService 使其适配 Wails v3 服务生命周期
// 不暴露任何前端绑定方法，仅用于在应用启动/关闭时管理后台 goroutine
type CleanupServiceWrapper struct {
	svc    *service.CleanupService
	cancel context.CancelFunc
}

func NewCleanupServiceWrapper(svc *service.CleanupService) *CleanupServiceWrapper {
	return &CleanupServiceWrapper{svc: svc}
}

func (w *CleanupServiceWrapper) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	slog.Info("启动定时汇总和清理服务...")

	// 启动时回填历史汇总数据
	if err := w.svc.BackfillMissingHours(); err != nil {
		slog.Warn("回填历史汇总数据失败", "error", err)
	}

	// 启动后台 goroutine
	cleanupCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	go w.svc.Start(cleanupCtx)

	return nil
}

func (w *CleanupServiceWrapper) ServiceShutdown() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}
