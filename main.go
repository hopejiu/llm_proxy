package main

import (
	"embed"
	"fmt"
	"llm-proxy/internal/config"
	"llm-proxy/internal/handler"
	"llm-proxy/internal/logger"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsWindows "github.com/wailsapp/wails/v2/pkg/options/windows"

	sqlite "github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"golang.org/x/sys/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// 版本信息，构建时通过 -ldflags 注入
var (
	Version   = "dev"
	BuildTime = "unknown"
)

// fatalMessageBox 在 GUI 程序中弹出错误对话框后退出
func fatalMessageBox(title, message string) {
	slog.Error(message)
	windows.MessageBox(0, windows.StringToUTF16Ptr(message), windows.StringToUTF16Ptr(title), windows.MB_OK|windows.MB_ICONERROR)
	os.Exit(1)
}

func main() {
	// 确定应用数据目录
	dataDir := config.DataDir()

	// 归档旧日志文件（启动前执行，确保 Init 创建新文件）
	logFilePath := filepath.Join(dataDir, "llm-proxy.log")
	logger.ArchiveLogFile(logFilePath)

	// 加载配置
	cfg := config.Load()

	// 清理过期归档日志
	logger.CleanOldArchives(dataDir, cfg.LogCleanupDays)

	// 初始化日志
	if err := logger.Init(logFilePath, slog.LevelInfo); err != nil {
		fmt.Printf("无法创建日志文件: %v\n", err)
		os.Exit(1)
	}

	slog.Info("LLM Proxy 桌面应用启动中...")

	// 根据配置重新设置日志级别
	logger.Init(logFilePath, logger.ParseLevel(cfg.LogLevel))

	slog.Info("配置加载成功", "db_type", cfg.DBType, "proxy_port", cfg.ProxyPort)

	// 初始化数据库
	db, dbFallbackMsg := initDB(cfg)

	// 组装依赖
	providerRepo := repository.NewProviderRepository(db)
	requestLogRepo := repository.NewRequestLogRepository(db, cfg.DBType)
	hourlyStatRepo := repository.NewHourlyStatRepository(db, cfg.DBType)

	// 启动时回填历史汇总数据
	cleanupSvc := service.NewCleanupService(hourlyStatRepo, requestLogRepo, cfg)
	if err := cleanupSvc.BackfillMissingHours(); err != nil {
		slog.Warn("回填历史汇总数据失败", "error", err)
	}

	// 启动时清理旧数据
	cleanOldData(requestLogRepo, cfg.LogCleanupDays, dataDir)

	proxyService := service.NewProxyService(providerRepo, cfg)
	providerService := service.NewProviderService(providerRepo, proxyService)
	statsService := service.NewStatsService(hourlyStatRepo, requestLogRepo, providerService)

	tracker := handler.NewActiveRequestTracker()

	proxyHandler := handler.NewProxyHandler(proxyService, requestLogRepo, cfg, tracker)
	anthropicHandler := handler.NewAnthropicHandler(proxyService, requestLogRepo, cfg, tracker)
	ollamaHandler := handler.NewOllamaHandler(proxyService, requestLogRepo, cfg, tracker)

	// 创建日志读取器
	logReader := logger.NewLogReader(logFilePath)

	// 创建 App
	app := &App{
		cfg:              cfg,
		providerService:  providerService,
		statsService:     statsService,
		tracker:          tracker,
		logReader:        logReader,
		cleanupSvc:       cleanupSvc,
		proxyHandler:     proxyHandler,
		anthropicHandler: anthropicHandler,
		ollamaHandler:    ollamaHandler,
		proxyState:       proxyState{status: "stopped"},
		dbFallbackMsg:    dbFallbackMsg,
	}
	slog.Info("正在启动 Wails 窗口...")

	// Wails 配置
	err := wails.Run(&options.App{
		Title:     "LLM Proxy",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Debug: options.Debug{
			OpenInspectorOnStartup: Version == "dev",
		},
		Windows: &wailsWindows.Options{
			WebviewUserDataPath: dataDir,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		fatalMessageBox("启动失败", "Wails 启动失败: "+err.Error())
	}
}

// initDB 初始化数据库连接和迁移
func initDB(cfg *config.Config) (*gorm.DB, string) {
	db, fallbackMsg := connectDB(cfg)
	migrateDB(db, cfg)
	return db, fallbackMsg
}

// connectDB 连接数据库并配置连接池
// MySQL 连接失败时自动回退到 SQLite，并返回回退提示信息
func connectDB(cfg *config.Config) (*gorm.DB, string) {
	var db *gorm.DB
	var err error

	if cfg.IsSQLite() {
		slog.Info("正在连接 SQLite 数据库...")
		dbPath := cfg.SQLitePath()
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			if err := os.WriteFile(dbPath, []byte{}, 0644); err != nil {
				fatalMessageBox("启动失败", "创建 SQLite 文件失败: "+err.Error())
			}
		}
		db, err = gorm.Open(sqlite.Open(cfg.SQLiteDSN()), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		})
		if err != nil {
			fatalMessageBox("启动失败", "SQLite数据库连接失败: "+err.Error())
		}
		configurePool(db, cfg)
		slog.Info("数据库连接成功")
		return db, ""
	}

	// 尝试连接 MySQL
	slog.Info("正在连接 MySQL 数据库...")
	db, err = gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		slog.Warn("MySQL数据库连接失败，自动回退到 SQLite", "error", err)
		// 回退到 SQLite
		cfg.FallbackToSQLite()
		dbPath := cfg.SQLitePath()
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			if err := os.WriteFile(dbPath, []byte{}, 0644); err != nil {
				fatalMessageBox("启动失败", "创建 SQLite 文件失败: "+err.Error())
			}
		}
		db, err = gorm.Open(sqlite.Open(cfg.SQLiteDSN()), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		})
		if err != nil {
			fatalMessageBox("启动失败", "SQLite数据库连接失败: "+err.Error())
		}
		configurePool(db, cfg)
		slog.Info("已回退到 SQLite 数据库")
		return db, "MySQL 连接失败，已自动回退到 SQLite，请在设置中重新配置数据库"
	}

	configurePool(db, cfg)
	slog.Info("数据库连接成功")
	return db, ""
}

// configurePool 配置数据库连接池
func configurePool(db *gorm.DB, cfg *config.Config) {
	sqlDB, _ := db.DB()
	if cfg.IsSQLite() {
		sqlDB.SetMaxOpenConns(1)
	} else {
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
		sqlDB.SetConnMaxIdleTime(time.Minute)
	}
}

// migrateDB 执行数据库迁移和索引创建
func migrateDB(db *gorm.DB, cfg *config.Config) {
	slog.Info("正在初始化数据库表...")

	// SQLite 不支持 longtext，需要先手动建表（用 text 替代 longtext）
	// 这样 AutoMigrate 检测到表已存在时只会添加缺失的列，不会重新创建
	if cfg.IsSQLite() {
		createSQLiteTablesIfNotExist(db)
	}

	if err := db.AutoMigrate(&model.ProviderConfig{}, &model.RequestLog{}, &model.HourlyStat{}); err != nil {
		fatalMessageBox("启动失败", "数据库初始化失败: "+err.Error())
	}

	// MySQL 专有：设置表字符集为 utf8mb4
	if !cfg.IsSQLite() {
		tables := []string{"provider_configs", "request_logs", "hourly_stats"}
		for _, table := range tables {
			sql := fmt.Sprintf("ALTER TABLE %s CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", table)
			if err := db.Exec(sql).Error; err != nil {
				slog.Warn("设置表字符集失败", "table", table, "error", err)
			}
		}
	}

	if db.Migrator().HasConstraint(&model.RequestLog{}, "fk_request_logs_provider") {
		if err := db.Migrator().DropConstraint(&model.RequestLog{}, "fk_request_logs_provider"); err != nil {
			slog.Warn("删除外键约束失败，可能已不存在", "error", err)
		} else {
			slog.Info("已删除旧的外键约束 fk_request_logs_provider")
		}
	}

	// 升级 hourly_stats 表：添加 provider_id 列并更新索引
	migrateHourlyStats(db, cfg)

	createIndexesIfNotExist(db, cfg)
	slog.Info("数据库表初始化完成")
}

// createSQLiteTablesIfNotExist 为 SQLite 手动创建表（将 longtext 映射为 text）
func createSQLiteTablesIfNotExist(db *gorm.DB) {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS provider_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			auto_suffix INTEGER DEFAULT 0,
			url_suffix TEXT DEFAULT '',
			base_url TEXT NOT NULL,
			api_key TEXT NOT NULL,
			model TEXT,
			alias TEXT,
			extra_params TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS request_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER,
			model TEXT,
			input_tokens INTEGER,
			output_tokens INTEGER,
			total_tokens INTEGER,
			cached_tokens INTEGER DEFAULT 0,
			request_body TEXT,
			response_body TEXT,
			response_content TEXT,
			thinking_content TEXT,
			status TEXT,
			error_message TEXT,
			duration INTEGER,
			aggregated INTEGER DEFAULT 0,
			created_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS hourly_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hour DATETIME NOT NULL,
			provider_id INTEGER NOT NULL DEFAULT 0,
			input_tokens INTEGER,
			output_tokens INTEGER,
			total_tokens INTEGER,
			cached_tokens INTEGER,
			request_count INTEGER,
			total_duration INTEGER,
			created_at DATETIME,
			updated_at DATETIME
		)`,
	}
	for _, sql := range tables {
		if err := db.Exec(sql).Error; err != nil {
			slog.Warn("SQLite手动建表失败（可能已存在）", "error", err)
		}
	}
}

// createIndexesIfNotExist 为已有数据库创建索引
func createIndexesIfNotExist(db *gorm.DB, cfg *config.Config) {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_request_logs_created_at_status ON request_logs(created_at, status)",
		"CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_request_logs_provider_id ON request_logs(provider_id)",
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			slog.Warn("创建索引失败", "sql", idx, "error", err)
		}
	}
}

// migrateHourlyStats 升级 hourly_stats 表，添加 provider_id 列并重建唯一索引
func migrateHourlyStats(db *gorm.DB, cfg *config.Config) {
	// 检查 provider_id 列是否已存在
	if !cfg.IsSQLite() && db.Migrator().HasColumn(&model.HourlyStat{}, "provider_id") {
		return
	}
	if cfg.IsSQLite() {
		// SQLite 中检查列是否存在的方法
		var cols []string
		db.Raw("PRAGMA table_info(hourly_stats)").Pluck("name", &cols)
		for _, c := range cols {
			if c == "provider_id" {
				return
			}
		}
	}

	slog.Info("正在升级 hourly_stats 表，添加 provider_id 列...")

	if err := db.Exec("ALTER TABLE hourly_stats ADD COLUMN provider_id INTEGER NOT NULL DEFAULT 0").Error; err != nil {
		slog.Warn("添加 provider_id 列失败（可能已存在）", "error", err)
	}

	// 尝试删除旧的唯一索引
	if err := db.Exec("DROP INDEX IF EXISTS idx_hourly_stats_hour").Error; err != nil {
		slog.Warn("删除旧索引 idx_hourly_stats_hour 失败", "error", err)
	}

	slog.Info("hourly_stats 表升级完成")
}

// cleanOldData 清理旧数据
func cleanOldData(requestLogRepo *repository.RequestLogRepository, cleanupDays int, baseDir string) {
	rowsAffected, err := requestLogRepo.DeleteOldRequestLogs(cleanupDays)
	if err != nil {
		slog.Error("清理数据库旧数据失败", "error", err)
	} else if rowsAffected > 0 {
		slog.Info("已删除旧请求数据", "rowsAffected", rowsAffected, "days", cleanupDays)
	}

	logFiles := []string{"proxy-requests.log", "proxy-reqbody.log", "llm-proxy.log"}
	for _, logFile := range logFiles {
		fullPath := filepath.Join(baseDir, logFile)
		if err := cleanLogFile(fullPath, cleanupDays); err != nil {
			slog.Error("清理日志文件失败", "file", fullPath, "error", err)
		}
	}
}

// cleanLogFile 清理过期日志文件
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
