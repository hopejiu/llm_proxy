package main

import (
	"context"
	"fmt"
	"llm-proxy/internal/config"
	"llm-proxy/internal/handler"
	"llm-proxy/internal/middleware"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 设置日志输出到文件和控制台
	logFile, err := os.OpenFile("llm-proxy.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("无法创建日志文件: %v\n", err)
		pauseAndExit()
	}
	defer logFile.Close()

	// 同时输出到控制台和文件
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// 也写入文件
	fileLogger := log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║           LLM Proxy 中转服务启动中...                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 加载配置
	cfg := config.Load()
	fileLogger.Println("配置加载成功")

	// 连接数据库
	fmt.Println("→ 正在连接 MySQL 数据库...")
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		msg := fmt.Sprintf("数据库连接失败: %v\n\n请检查:\n1. MySQL 是否已启动 (3306端口)\n2. 用户名密码是否正确 (root/wang)\n3. 数据库 llm_proxy 是否存在", err)
		fmt.Println("✗ " + msg)
		fileLogger.Printf("数据库连接失败: %v", err)
		pauseAndExit()
	}
	fmt.Println("✓ 数据库连接成功")
	fileLogger.Println("数据库连接成功")

	// 自动迁移表结构
	fmt.Println("→ 正在初始化数据库表...")
	if err := db.AutoMigrate(&model.ProviderConfig{}, &model.RequestLog{}); err != nil {
		msg := fmt.Sprintf("数据库初始化失败: %v", err)
		fmt.Println("✗ " + msg)
		fileLogger.Printf("数据库初始化失败: %v", err)
		pauseAndExit()
	}

	
	fmt.Println("✓ 数据库表初始化完成")
	fileLogger.Println("数据库表初始化完成")

	// 初始化 Repository
	providerRepo := repository.NewProviderRepository(db)
	requestLogRepo := repository.NewRequestLogRepository(db)

	// 启动时清理旧数据
	fmt.Println("→ 正在清理旧数据...")
	cleanOldData(db, fileLogger)
	fmt.Println("✓ 数据清理完成")

	// 初始化 Service
	providerService := service.NewProviderService(providerRepo)
	proxyService := service.NewProxyService(providerRepo, requestLogRepo)
	statsService := service.NewStatsService(requestLogRepo)

	// 初始化 Handler
	webHandler := handler.NewWebHandler(providerService, statsService)
	proxyHandler := handler.NewProxyHandler(proxyService, requestLogRepo)

	// 启动 80 端口 Web 服务
	fmt.Println("→ 正在启动 Web 配置服务 (端口 80)...")
	webServer := startWebServer(cfg.WebPort, webHandler)

	// 启动 8888 端口代理服务
	fmt.Println("→ 正在启动 LLM 代理服务 (端口 8888)...")
	proxyServer := startProxyServer(cfg.ProxyPort, proxyHandler)

	// 等待一小段时间检查服务是否成功启动
	time.Sleep(500 * time.Millisecond)

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  服务启动成功！                            ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  配置页面: http://localhost%-34s║\n", ":"+cfg.WebPort)
	fmt.Printf("║  代理接口: http://localhost%-34s║\n", ":"+cfg.ProxyPort+"/v1/chat/completions")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║  按 Ctrl+C 停止服务                                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fileLogger.Printf("服务启动成功 - Web: %s, Proxy: %s", cfg.WebPort, cfg.ProxyPort)

	// 自动打开浏览器
	webURL := fmt.Sprintf("http://localhost:%s", cfg.WebPort)
	fmt.Println("→ 正在打开浏览器...")
	if err := openBrowser(webURL); err != nil {
		fmt.Printf("无法自动打开浏览器，请手动访问: %s\n", webURL)
		fileLogger.Printf("打开浏览器失败: %v", err)
	} else {
		fmt.Println("✓ 浏览器已打开")
	}

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println()
	fmt.Println("→ 正在关闭服务...")
	fileLogger.Println("正在关闭服务...")

	shutdownServer(webServer, "Web")
	shutdownServer(proxyServer, "Proxy")

	fmt.Println("✓ 服务已停止")
	fileLogger.Println("服务已停止")

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

		// 统计
		api.GET("/stats", h.GetStats)
		api.GET("/stats/daily", h.GetDailyStats)
		api.GET("/logs/recent", h.GetRecentLogs)
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Web server error: %v", err)
		}
	}()

	return server
}

// startProxyServer 启动 LLM 代理服务
func startProxyServer(port string, h *handler.ProxyHandler) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// OpenAI 兼容 API
	r.POST("/v1/chat/completions", h.ChatCompletions)
	r.GET("/v1/models", h.Models)

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
			log.Printf("Proxy server error: %v", err)
		}
	}()

	return server
}

// shutdownServer 优雅关闭服务
func shutdownServer(server *http.Server, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("%s server forced to shutdown: %v", name, err)
	} else {
		log.Printf("%s server stopped gracefully", name)
	}
}

// cleanOldData 清理旧数据（数据库和日志文件）
func cleanOldData(db *gorm.DB, logger *log.Logger) {
	// 1. 清理两周前的请求体和响应体
	cutoffDate := time.Now().AddDate(0, 0, -14)
	result := db.Exec(`
		UPDATE request_logs 
		SET request_body = '', response_body = '', thinking_content = ''
		WHERE created_at < ? AND (request_body != '' OR response_body != '' OR thinking_content != '')
	`, cutoffDate)

	if result.Error != nil {
		logger.Printf("清理数据库旧数据失败: %v", result.Error)
		fmt.Printf("  ✗ 清理数据库失败: %v\n", result.Error)
	} else if result.RowsAffected > 0 {
		logger.Printf("已清理 %d 条两周前的请求/响应体数据", result.RowsAffected)
		fmt.Printf("  - 清理数据库: %d 条记录\n", result.RowsAffected)
	}

	// 2. 清理本地日志文件
	logFiles := []string{"proxy-requests.log", "proxy-reqbody.log", "llm-proxy.log"}
	for _, logFile := range logFiles {
		if err := cleanLogFile(logFile, 14); err != nil {
			logger.Printf("清理日志文件 %s 失败: %v", logFile, err)
		} else {
			fmt.Printf("  - 清理日志文件: %s\n", logFile)
		}
	}
}

// cleanLogFile 清理日志文件（删除超过指定天数的行）
func cleanLogFile(filename string, days int) error {
	// 如果文件不存在，跳过
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil
	}

	// 读取文件内容
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	cutoffDate := time.Now().AddDate(0, 0, -days)
	var keptLines []string

	for _, line := range lines {
		if line == "" {
			continue
		}
		// 尝试解析日期（格式：2026/03/17 或 2026-03-17）
		if len(line) >= 10 {
			dateStr := line[:10]
			lineDate, err := time.Parse("2006/01/02", dateStr)
			if err != nil {
				lineDate, err = time.Parse("2006-01-02", dateStr)
			}
			if err == nil && lineDate.After(cutoffDate) {
				keptLines = append(keptLines, line)
			}
		} else {
			keptLines = append(keptLines, line)
		}
	}

	// 写回文件
	newContent := strings.Join(keptLines, "\n")
	return os.WriteFile(filename, []byte(newContent), 0666)
}

// migrateModelsToModel 迁移旧字段 models 到新字段 model，并删除 priority 字段
func migrateModelsToModel(db *gorm.DB) error {
	// 检查旧列是否存在
	var hasOldColumn int
	db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'provider_configs' AND column_name = 'models'").Scan(&hasOldColumn)
	
	// 如果旧列存在，则迁移数据
	if hasOldColumn > 0 {
		// 将 models 的第一个模型（逗号分隔）迁移到 model
		result := db.Exec(`
			UPDATE provider_configs 
			SET model = TRIM(SUBSTRING_INDEX(models, ',', 1))
			WHERE model IS NULL OR model = ''
		`)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			log.Printf("已迁移 %d 条记录的 models 字段到 model 字段", result.RowsAffected)
		}
		
		// 删除旧列
		db.Exec("ALTER TABLE provider_configs DROP COLUMN models")
	}
	
	// 删除 priority 字段
	var hasPriorityColumn int
	db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'provider_configs' AND column_name = 'priority'").Scan(&hasPriorityColumn)
	if hasPriorityColumn > 0 {
		db.Exec("ALTER TABLE provider_configs DROP COLUMN priority")
		log.Println("已删除 priority 字段")
	}
	
	return nil
}
