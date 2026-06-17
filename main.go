package main

import (
	"embed"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wanglejiu/llm-proxy/internal/config"
	"github.com/wanglejiu/llm-proxy/internal/handler"
	"github.com/wanglejiu/llm-proxy/internal/logger"
	"github.com/wanglejiu/llm-proxy/internal/repository"
	"github.com/wanglejiu/llm-proxy/internal/service"

	"golang.org/x/sys/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIcon []byte

// 版本信息，构建时通过 -ldflags 注入
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func fatalMessageBox(title, message string) {
	slog.Error(message)
	windows.MessageBox(0, windows.StringToUTF16Ptr(message), windows.StringToUTF16Ptr(title), windows.MB_OK|windows.MB_ICONERROR)
	os.Exit(1)
}

func main() {
	// 1. 初始化日志与配置
	dataDir := config.DataDir()
	logFilePath := filepath.Join(dataDir, "llm-proxy.log")
	logger.ArchiveLogFile(logFilePath)
	cfg := config.Load()
	logger.CleanOldArchives(dataDir, cfg.LogCleanupDays)

	if err := logger.Init(logFilePath, slog.LevelInfo); err != nil {
		slog.Error("无法创建日志文件", "error", err)
		os.Exit(1)
	}
	logger.Init(logFilePath, logger.ParseLevel(cfg.LogLevel))
	slog.Info("LLM Proxy 桌面应用启动中...")
	slog.Info("配置加载成功", "db_type", cfg.DBType, "proxy_port", cfg.ProxyPort, "auto_start_proxy", cfg.AutoStartProxy)

	// 2. 初始化数据库
	db, dbFallbackMsg := initDB(cfg)
	dbManager := repository.NewDBManager(db, cfg.DBType)

	// 3. 组装依赖
	providerRepo := repository.NewProviderRepository(dbManager)
	requestLogRepo := repository.NewRequestLogRepository(dbManager)
	hourlyStatRepo := repository.NewHourlyStatRepository(dbManager)
	sessionRepo := repository.NewChatSessionRepository(dbManager)

	cleanOldData(requestLogRepo, cfg.LogCleanupDays, dataDir)

	providerCache := service.NewProviderCache(providerRepo, cfg)
	proxyService := service.NewProxyService(providerCache)
	providerSvc := service.NewProviderService(providerRepo, providerCache)
	logSvc := service.NewLogService(requestLogRepo)
	statsSvc := service.NewStatsService(hourlyStatRepo, requestLogRepo, providerSvc)
	cleanupSvc := service.NewCleanupService(hourlyStatRepo, requestLogRepo, cfg)
	sessionSvc := service.NewSessionService(sessionRepo)

	if err := cleanupSvc.BackfillMissingHours(); err != nil {
		slog.Warn("回填历史汇总数据失败", "error", err)
	}

	tracker := handler.NewActiveRequestTracker()

	proxyHandler := handler.NewProxyHandler(proxyService, requestLogRepo, cfg, tracker)
	proxyHandler.WithSessionService(sessionSvc)

	logReader := logger.NewLogReader(logFilePath)

	providerBindingService := NewProviderService(providerSvc, cfg, proxyService)
	statsBindingService := NewStatsService(statsSvc, logSvc, providerSvc, tracker)
	statsBindingService.WithSessionRepo(sessionRepo)
	appBindingService := NewAppService(cfg, dbManager, proxyHandler, logReader, dbFallbackMsg)
	cleanupWrapper := NewCleanupServiceWrapper(cleanupSvc)

	slog.Info("正在启动 Wails 窗口...")

	// 4. 创建 Wails 应用 & 窗口
	app := application.New(application.Options{
		Name:        "LLM Proxy",
		Description: "LLM API 代理工具 - 提供 OpenAI 兼容接口",
		Services: []application.Service{
			application.NewService(providerBindingService),
			application.NewService(statsBindingService),
			application.NewService(appBindingService),
			application.NewService(cleanupWrapper),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.teamsun.llm-proxy",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				appBindingService.ShowMainWindow()
			},
		},
	})

	appBindingService.SetApp(app)

	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "LLM Proxy",
		Width:            1200,
		Height:           800,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	mainWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		mainWindow.Hide()
		e.Cancel()
	})

	appBindingService.SetMainWindow(mainWindow)

	tracker.SetOnChange(func(change handler.ActiveTrackerChange) {
		app.Event.Emit("active-request:"+change.Type, change)
	})

	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip("LLM Proxy")
	tray.AttachWindow(mainWindow)

	trayMenu := application.NewMenu()
	trayMenu.Add("显示窗口").OnClick(func(ctx *application.Context) {
		appBindingService.ShowMainWindow()
	})
	trayMenu.Add("隐藏窗口").OnClick(func(ctx *application.Context) {
		appBindingService.HideMainWindow()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("退出").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	tray.SetMenu(trayMenu)

	if err := app.Run(); err != nil {
		fatalMessageBox("启动失败", "应用启动失败: "+err.Error())
	}
}
