package main

import (
	"context"
	"fmt"
	"llm-proxy/internal/config"
	"llm-proxy/internal/handler"
	"llm-proxy/internal/logger"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/router"
	"llm-proxy/internal/service"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	sqlite "github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	if err := logger.Init("llm-proxy.log"); err != nil {
		fmt.Printf("无法创建日志文件: %v\n", err)
		pauseAndExit()
	}

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║           LLM Proxy 中转服务启动中...                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	cfg := config.Load()
	slog.Info("配置加载成功")

	db := initDB(cfg)

	providerRepo := repository.NewProviderRepository(db)
	requestLogRepo := repository.NewRequestLogRepository(db, cfg.DBType)

	fmt.Println("→ 正在清理旧数据...")
	cleanOldData(requestLogRepo, cfg.LogCleanupDays)
	fmt.Println("✓ 数据清理完成")

	proxyService := service.NewProxyService(providerRepo, requestLogRepo, cfg.ProviderCacheTTL)
	providerService := service.NewProviderService(providerRepo, proxyService)
	statsService := service.NewStatsService(requestLogRepo)

	webHandler := handler.NewWebHandler(providerService, statsService)

	handlerCfg := handler.HandlerConfig{
		HTTPTimeout:            cfg.HTTPTimeout,
		StreamFirstByteTimeout: cfg.StreamFirstByteTimeout,
		StreamMaxRetries:       cfg.StreamMaxRetries,
		RetryDelayBase:         cfg.RetryDelayBase,
	}
	proxyHandler := handler.NewProxyHandler(proxyService, requestLogRepo, handlerCfg)
	anthropicHandler := handler.NewAnthropicHandler(proxyService, requestLogRepo, handlerCfg)
	ollamaHandler := handler.NewOllamaHandler(proxyService, requestLogRepo, handlerCfg)

	fmt.Println("→ 正在启动 Web 配置服务...")
	webEngine := router.SetupWeb(webHandler)
	webServer := router.StartServer(cfg.WebPort, webEngine)

	fmt.Println("→ 正在启动 LLM 代理服务...")
	proxyEngine := router.SetupProxy(proxyHandler, anthropicHandler, ollamaHandler)
	proxyServer := router.StartServer(cfg.ProxyPort, proxyEngine)

	// 等待端口就绪
	waitForPort(cfg.WebPort, 2*time.Second)

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  服务启动成功！                            ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  配置页面: http://localhost%-34s║\n", ":"+cfg.WebPort)
	fmt.Printf("║  代理接口: http://localhost%-34s║\n", ":"+cfg.ProxyPort+"/v1/chat/completions")
	fmt.Printf("║  Anthropic: http://localhost%-34s║\n", ":"+cfg.ProxyPort+"/anthropic/v1/messages")
	fmt.Printf("║  Ollama:   http://localhost%-34s║\n", ":"+cfg.ProxyPort+"/api/chat")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║  按 Ctrl+C 停止服务                                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	slog.Info("服务启动成功", "webPort", cfg.WebPort, "proxyPort", cfg.ProxyPort)

	webURL := fmt.Sprintf("http://localhost:%s", cfg.WebPort)
	fmt.Println("→ 正在打开浏览器...")
	if err := openBrowser(webURL); err != nil {
		fmt.Printf("无法自动打开浏览器，请手动访问: %s\n", webURL)
		slog.Warn("打开浏览器失败", "error", err)
	} else {
		fmt.Println("✓ 浏览器已打开")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println()
	fmt.Println("→ 正在关闭服务...")
	slog.Info("正在关闭服务...")

	shutdownServer(webServer, "Web")
	shutdownServer(proxyServer, "Proxy")
	proxyService.Close()

	fmt.Println("✓ 服务已停止")
	slog.Info("服务已停止")

	fmt.Println()
	fmt.Println("按任意键退出...")
	fmt.Scanln()
}

// initDB 初始化数据库连接和迁移
func initDB(cfg *config.Config) *gorm.DB {
	var db *gorm.DB
	var err error

	if cfg.IsSQLite() {
		fmt.Println("→ 正在连接 SQLite 数据库...")
		dbPath := cfg.SQLiteDSN()
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Printf("  SQLite 文件不存在，正在创建: %s\n", dbPath)
			if err := os.WriteFile(dbPath, []byte{}, 0644); err != nil {
				fmt.Println("✗ 创建 SQLite 文件失败")
				slog.Error("创建 SQLite 文件失败", "error", err, "path", dbPath)
				pauseAndExit()
			}
			fmt.Println("  ✓ SQLite 文件创建成功")
		}
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			fmt.Println("✗ SQLite数据库连接失败")
			slog.Error("SQLite数据库连接失败", "error", err, "path", dbPath)
			pauseAndExit()
		}
	} else {
		fmt.Println("→ 正在连接 MySQL 数据库...")
		db, err = gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
		if err != nil {
			fmt.Println("✗ 数据库连接失败")
			slog.Error("数据库连接失败", "error", err)
			pauseAndExit()
		}
	}
	fmt.Println("✓ 数据库连接成功")
	slog.Info("数据库连接成功")

	fmt.Println("→ 正在初始化数据库表...")
	if err := db.AutoMigrate(&model.ProviderConfig{}, &model.RequestLog{}); err != nil {
		fmt.Println("✗ 数据库初始化失败")
		slog.Error("数据库初始化失败", "error", err)
		pauseAndExit()
	}

	if db.Migrator().HasConstraint(&model.RequestLog{}, "fk_request_logs_provider") {
		if err := db.Migrator().DropConstraint(&model.RequestLog{}, "fk_request_logs_provider"); err != nil {
			slog.Warn("删除外键约束失败，可能已不存在", "error", err)
		} else {
			slog.Info("已删除旧的外键约束 fk_request_logs_provider")
		}
	}

	fmt.Println("✓ 数据库表初始化完成")
	slog.Info("数据库表初始化完成")

	return db
}

// waitForPort 等待端口就绪（替代 time.Sleep 的硬等待）
func waitForPort(port string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", ":"+port, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func pauseAndExit() {
	fmt.Println()
	fmt.Println("按任意键退出...")
	fmt.Scanln()
	os.Exit(1)
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	return exec.Command(cmd, args...).Start()
}

func shutdownServer(server *http.Server, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "name", name, "error", err)
	} else {
		slog.Info("server stopped gracefully", "name", name)
	}
}

func cleanOldData(requestLogRepo *repository.RequestLogRepository, cleanupDays int) {
	rowsAffected, err := requestLogRepo.CleanOldRequestBodies(cleanupDays)
	if err != nil {
		slog.Error("清理数据库旧数据失败", "error", err)
		fmt.Printf("  ✗ 清理数据库失败: %v\n", err)
	} else if rowsAffected > 0 {
		slog.Info("已清理旧请求/响应体数据", "rowsAffected", rowsAffected, "days", cleanupDays)
		fmt.Printf("  - 清理数据库: %d 条记录\n", rowsAffected)
	}

	logFiles := []string{"proxy-requests.log", "proxy-reqbody.log", "llm-proxy.log"}
	for _, logFile := range logFiles {
		if err := cleanLogFile(logFile, cleanupDays); err != nil {
			slog.Error("清理日志文件失败", "file", logFile, "error", err)
		} else {
			fmt.Printf("  - 清理日志文件: %s\n", logFile)
		}
	}
}

func cleanLogFile(filename string, days int) error {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	cutoffDate := time.Now().AddDate(0, 0, -days)
	if info.ModTime().Before(cutoffDate) {
		return os.Truncate(filename, 0)
	}

	return nil
}
