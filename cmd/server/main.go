package main

import (
	"context"
	"fmt"
	"llm-proxy/internal/config"
	"llm-proxy/internal/handler"
	"llm-proxy/internal/logger"
	"llm-proxy/internal/middleware"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	sqlite "github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 初始化 slog 结构化日志
	if err := logger.Init("llm-proxy.log"); err != nil {
		fmt.Printf("无法创建日志文件: %v\n", err)
		pauseAndExit()
	}

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║           LLM Proxy 中转服务启动中...                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 加载配置
	cfg := config.Load()
	slog.Info("配置加载成功")

	// 连接数据库
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

	// 自动迁移表结构
	fmt.Println("→ 正在初始化数据库表...")
	if err := db.AutoMigrate(&model.ProviderConfig{}, &model.RequestLog{}); err != nil {
		fmt.Println("✗ 数据库初始化失败")
		slog.Error("数据库初始化失败", "error", err)
		pauseAndExit()
	}

	// 删除旧的外键约束（不再使用外键关联）
	if db.Migrator().HasConstraint(&model.RequestLog{}, "fk_request_logs_provider") {
		if err := db.Migrator().DropConstraint(&model.RequestLog{}, "fk_request_logs_provider"); err != nil {
			slog.Warn("删除外键约束失败，可能已不存在", "error", err)
		} else {
			slog.Info("已删除旧的外键约束 fk_request_logs_provider")
		}
	}

	fmt.Println("✓ 数据库表初始化完成")
	slog.Info("数据库表初始化完成")

	// 初始化 Repository
	providerRepo := repository.NewProviderRepository(db)
	requestLogRepo := repository.NewRequestLogRepository(db)

	// 启动时清理旧数据
	fmt.Println("→ 正在清理旧数据...")
	cleanOldData(db)
	fmt.Println("✓ 数据清理完成")

	// 初始化 Service
	providerService := service.NewProviderService(providerRepo)
	proxyService := service.NewProxyService(providerRepo, requestLogRepo)
	statsService := service.NewStatsService(requestLogRepo)

	// 初始化 Handler
	webHandler := handler.NewWebHandler(providerService, statsService)
	proxyHandler := handler.NewProxyHandler(proxyService, requestLogRepo)
	anthropicHandler := handler.NewAnthropicHandler(proxyService, requestLogRepo)
	ollamaHandler := handler.NewOllamaHandler(proxyService, requestLogRepo)

	// 启动 Web 服务
	fmt.Println("→ 正在启动 Web 配置服务 (端口 80)...")
	webServer := startWebServer(cfg.WebPort, webHandler)

	// 启动代理服务
	fmt.Println("→ 正在启动 LLM 代理服务 (端口 8888)...")
	proxyServer := startProxyServer(cfg.ProxyPort, proxyHandler, anthropicHandler, ollamaHandler)

	// 等待一小段时间检查服务是否成功启动
	time.Sleep(500 * time.Millisecond)

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

	// 自动打开浏览器
	webURL := fmt.Sprintf("http://localhost:%s", cfg.WebPort)
	fmt.Println("→ 正在打开浏览器...")
	if err := openBrowser(webURL); err != nil {
		fmt.Printf("无法自动打开浏览器，请手动访问: %s\n", webURL)
		slog.Warn("打开浏览器失败", "error", err)
	} else {
		fmt.Println("✓ 浏览器已打开")
	}

	// 等待中断信号
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

	// 等待用户按键后退出
	fmt.Println()
	fmt.Println("按任意键退出...")
	fmt.Scanln()
}

// pauseAndExit 暂停并退出（用于错误时）
func pauseAndExit() {
	fmt.Println()
	fmt.Println("按任意键退出...")
	fmt.Scanln()
	os.Exit(1)
}

// openBrowser 使用系统默认浏览器打开指定URL
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

// startWebServer 启动配置管理 Web 服务
func startWebServer(port string, h *handler.WebHandler) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// 加载 HTML 模板
	r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "web/static")

	// 页面路由
	r.GET("/", h.Index)
	r.GET("/stats", h.StatsPage)

	// API 路由
	api := r.Group("/api")
	{
		// Provider 管理
		api.GET("/providers", h.GetProviders)
		api.GET("/providers/:id", h.GetProvider)
		api.POST("/providers", h.CreateProvider)
		api.PUT("/providers/:id", h.UpdateProvider)
		api.DELETE("/providers/:id", h.DeleteProvider)
		api.POST("/providers/:id/toggle", h.ToggleProvider)
		api.GET("/providers/export", h.ExportProviders)
		api.POST("/providers/import", h.ImportProviders)

		// CodeBuddy 配置
		api.POST("/codebuddy/setup", h.SetupCodeBuddy)

		// 统计
		api.GET("/stats", h.GetStats)
		api.GET("/stats/daily", h.GetDailyStats)
		api.GET("/stats/hourly", h.GetTodayHourlyStats)
		api.GET("/logs/recent", h.GetRecentLogs)
		api.GET("/logs/:id", h.GetLogDetail)
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Web server error", "error", err)
		}
	}()

	return server
}

// startProxyServer 启动 LLM 代理服务
func startProxyServer(port string, h *handler.ProxyHandler, ah *handler.AnthropicHandler, oh *handler.OllamaHandler) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// OpenAI 兼容 API
	r.POST("/v1/chat/completions", h.ChatCompletions)
	r.GET("/v1/models", h.Models)

	// Anthropic 兼容 API
	r.POST("/anthropic/v1/messages", ah.Messages)
	r.GET("/anthropic/v1/models", ah.Models)

	// Ollama 兼容 API
	r.POST("/api/chat", oh.Chat)
	r.GET("/api/tags", oh.Tags)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 处理不允许的方法
	r.HandleMethodNotAllowed = true
	r.NoMethod(h.MethodNotAllowed)

	// 兜底路由 - 处理 404
	r.NoRoute(h.NotFound)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Proxy server error", "error", err)
		}
	}()

	return server
}

// shutdownServer 优雅关闭服务
func shutdownServer(server *http.Server, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "name", name, "error", err)
	} else {
		slog.Info("server stopped gracefully", "name", name)
	}
}

// cleanOldData 清理旧数据（数据库和日志文件）
func cleanOldData(db *gorm.DB) {
	// 清理两周前的请求体和响应体
	cutoffDate := time.Now().AddDate(0, 0, -14)
	result := db.Exec(`
		UPDATE request_logs 
		SET request_body = '', response_body = '', thinking_content = ''
		WHERE created_at < ? AND (request_body != '' OR response_body != '' OR thinking_content != '')
	`, cutoffDate)

	if result.Error != nil {
		slog.Error("清理数据库旧数据失败", "error", result.Error)
		fmt.Printf("  ✗ 清理数据库失败: %v\n", result.Error)
	} else if result.RowsAffected > 0 {
		slog.Info("已清理两周前的请求/响应体数据", "rowsAffected", result.RowsAffected)
		fmt.Printf("  - 清理数据库: %d 条记录\n", result.RowsAffected)
	}

	// 清理本地日志文件
	logFiles := []string{"proxy-requests.log", "proxy-reqbody.log", "llm-proxy.log"}
	for _, logFile := range logFiles {
		if err := cleanLogFile(logFile, 14); err != nil {
			slog.Error("清理日志文件失败", "file", logFile, "error", err)
		} else {
			fmt.Printf("  - 清理日志文件: %s\n", logFile)
		}
	}
}

// cleanLogFile 清理日志文件（基于文件修改时间判断）
func cleanLogFile(filename string, days int) error {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	cutoffDate := time.Now().AddDate(0, 0, -days)
	// 文件最后修改时间早于截止日期，清空文件
	if info.ModTime().Before(cutoffDate) {
		return os.Truncate(filename, 0)
	}

	return nil
}
