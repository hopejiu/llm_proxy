package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/wanglejiu/llm-proxy/internal/config"
	"github.com/wanglejiu/llm-proxy/internal/handler"
	"github.com/wanglejiu/llm-proxy/internal/logger"
	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/repository"
	"github.com/wanglejiu/llm-proxy/internal/service"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	sqlite "github.com/glebarez/sqlite"
	"golang.org/x/sys/windows"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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

	// 启动时清理旧数据
	cleanOldData(requestLogRepo, cfg.LogCleanupDays, dataDir)

	proxyService := service.NewProxyService(providerRepo, cfg)
	providerSvc := service.NewProviderService(providerRepo, proxyService)
	logSvc := service.NewLogService(requestLogRepo)
	statsSvc := service.NewStatsService(hourlyStatRepo, requestLogRepo, providerSvc)
	cleanupSvc := service.NewCleanupService(hourlyStatRepo, requestLogRepo, cfg)

	// 启动时回填历史汇总数据
	if err := cleanupSvc.BackfillMissingHours(); err != nil {
		slog.Warn("回填历史汇总数据失败", "error", err)
	}

	tracker := handler.NewActiveRequestTracker()

	proxyHandler := handler.NewProxyHandler(proxyService, requestLogRepo, cfg, tracker)
	anthropicHandler := handler.NewAnthropicHandler(proxyService, requestLogRepo, cfg, tracker)
	ollamaHandler := handler.NewOllamaHandler(proxyService, requestLogRepo, cfg, tracker)

	// 创建日志读取器
	logReader := logger.NewLogReader(logFilePath)

	// 创建 Wails 3 绑定服务
	providerBindingService := NewProviderService(providerSvc, cfg, proxyService)
	statsBindingService := NewStatsService(statsSvc, logSvc, providerSvc, tracker)
	appBindingService := NewAppService(cfg, proxyHandler, anthropicHandler, ollamaHandler, logReader, dbFallbackMsg)
	cleanupWrapper := NewCleanupServiceWrapper(cleanupSvc)

	slog.Info("正在启动 Wails 窗口...")

	// 创建应用
	app := application.New(application.Options{
		Name:        "LLM Proxy",
		Description: "LLM API 代理工具 - 提供 OpenAI/Anthropic/Ollama 兼容接口",
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
		// 单实例：第二次启动时窗口置前
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.teamsun.llm-proxy",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				appBindingService.ShowMainWindow()
			},
		},
	})

	// 创建窗口
	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "LLM Proxy",
		Width:            1200,
		Height:           800,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	// 关闭按钮 → 隐藏到系统托盘
	mainWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		mainWindow.Hide()
		e.Cancel()
	})

	// 保存窗口引用到绑定服务
	appBindingService.SetMainWindow(mainWindow)

	// 活跃请求变更 → Wails 事件推送（前端通过 Events.On 接收）
	tracker.SetOnChange(func(change handler.ActiveTrackerChange) {
		app.Event.Emit("active-request:"+change.Type, change)
	})

	// 系统托盘
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

	// 运行应用
	err := app.Run()
	if err != nil {
		fatalMessageBox("启动失败", "应用启动失败: "+err.Error())
	}
}

// ========== 数据库 ==========

func initDB(cfg *config.Config) (*gorm.DB, string) {
	db, fallbackMsg := connectDB(cfg)
	migrateDB(db, cfg)
	return db, fallbackMsg
}

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

	slog.Info("正在连接 MySQL 数据库...")
	db, err = gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		slog.Warn("MySQL数据库连接失败，自动回退到 SQLite", "error", err)
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

func migrateDB(db *gorm.DB, cfg *config.Config) {
	slog.Info("正在初始化数据库表...")

	if cfg.IsSQLite() {
		createSQLiteTablesIfNotExist(db)
	}

	if err := db.AutoMigrate(&model.ProviderConfig{}, &model.RequestLog{}, &model.HourlyStat{}); err != nil {
		fatalMessageBox("启动失败", "数据库初始化失败: "+err.Error())
	}

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

	// 删除旧的 (hour, provider_id) 唯一索引，已被 (hour, provider_id, model) 替代
	// 旧索引会导致 provider 合计行(model='') 与 model 明细行冲突
	if !cfg.IsSQLite() {
		var idxCount int64
		db.Raw("SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'hourly_stats' AND INDEX_NAME = 'idx_hour_provider'").Scan(&idxCount)
		if idxCount > 0 {
			if err := db.Exec("ALTER TABLE hourly_stats DROP INDEX idx_hour_provider").Error; err != nil {
				slog.Warn("删除旧索引 idx_hour_provider 失败", "error", err)
			} else {
				slog.Info("已删除旧的唯一索引 idx_hour_provider(hour, provider_id)，使用 idx_hour_provider_model(hour, provider_id, model) 替代")
			}
		}
	} else {
		// SQLite: 旧索引名称可能不同
		if err := db.Exec("DROP INDEX IF EXISTS idx_hourly_stats_hour").Error; err != nil {
			slog.Debug("删除旧 SQLite 索引失败", "error", err)
		}
		if err := db.Exec("DROP INDEX IF EXISTS idx_hour_provider").Error; err != nil {
			slog.Debug("删除旧 SQLite 索引 idx_hour_provider 失败", "error", err)
		}
	}

	migrateHourlyStats(db, cfg)
	migrateProviderConfigs(db, cfg)
	createIndexesIfNotExist(db, cfg)
	slog.Info("数据库表初始化完成")
}

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

func migrateHourlyStats(db *gorm.DB, cfg *config.Config) {
	if !cfg.IsSQLite() && db.Migrator().HasColumn(&model.HourlyStat{}, "provider_id") {
		return
	}
	if cfg.IsSQLite() {
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

	if err := db.Exec("DROP INDEX IF EXISTS idx_hourly_stats_hour").Error; err != nil {
		slog.Warn("删除旧索引 idx_hourly_stats_hour 失败", "error", err)
	}

	slog.Info("hourly_stats 表升级完成")
}

// migrateProviderConfigs 将旧的 model/alias/extra_params 字段迁移到新的 models JSON 字段
// 防护：检查 models 列是否有数据 + 旧列是否存在，双重保障不重复执行
func migrateProviderConfigs(db *gorm.DB, cfg *config.Config) {
	// 1. 如果 models 列已存在且有数据 → 已迁移，直接跳过
	if db.Migrator().HasColumn(&model.ProviderConfig{}, "Models") {
		var count int64
		db.Model(&model.ProviderConfig{}).Where("models IS NOT NULL AND models != ''").Count(&count)
		if count > 0 {
			// 旧列可能还未清理干净（上次 DROP 失败），再试一次
			dropOldProviderColumns(db, cfg)
			slog.Debug("provider_configs 已是新格式，跳过迁移", "rows", count)
			return
		}
	}

	// 2. 检查旧列是否存在（用 raw SQL 避免依赖 struct 字段）
	hasOldCols := checkOldProviderColumnsExist(db, cfg)

	// 如果旧列和 models 列都不存在 → 全新数据库，无事可做
	if !hasOldCols && !db.Migrator().HasColumn(&model.ProviderConfig{}, "Models") {
		return
	}

	// 3. 添加 models 列（如果还没有）
	if !db.Migrator().HasColumn(&model.ProviderConfig{}, "Models") {
		if err := db.Migrator().AddColumn(&model.ProviderConfig{}, "Models"); err != nil {
			slog.Warn("添加 models 列失败", "error", err)
			return
		}
	}

	// 4. 如果旧列存在，迁移数据
	if hasOldCols {
		slog.Info("正在迁移 provider_configs 表到新的多模型格式...")

		// 读取旧数据
		type oldRow struct {
			ID          uint
			Models      string // 仅用于 Scan 占位
			Model       string
			Alias       string
			ExtraParams string
		}
		var rows []oldRow
		rawSQL := "SELECT id, model, alias, extra_params FROM provider_configs WHERE models IS NULL OR models = ''"
		db.Raw(rawSQL).Scan(&rows)

		for _, row := range rows {
			entries := buildModelEntriesFromOld(row.Model, row.Alias, row.ExtraParams)
			jsonBytes, _ := json.Marshal(entries)
			if err := db.Model(&model.ProviderConfig{}).Where("id = ?", row.ID).Update("models", string(jsonBytes)).Error; err != nil {
				slog.Warn("迁移 provider 失败", "id", row.ID, "error", err)
			}
		}

		// 删除旧列
		dropOldProviderColumns(db, cfg)
		slog.Info("provider_configs 迁移完成")
	}
}

// checkOldProviderColumnsExist 用 raw SQL 检查旧列是否存在
func checkOldProviderColumnsExist(db *gorm.DB, cfg *config.Config) bool {
	if cfg.IsSQLite() {
		var cols []string
		db.Raw("PRAGMA table_info(provider_configs)").Pluck("name", &cols)
		for _, c := range cols {
			if c == "model" || c == "alias" || c == "extra_params" {
				return true
			}
		}
		return false
	}
	// MySQL
	var count int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'provider_configs'
		AND COLUMN_NAME IN ('model','alias','extra_params')`).Scan(&count)
	return count > 0
}

// dropOldProviderColumns 删除旧的 model/alias/extra_params 列
func dropOldProviderColumns(db *gorm.DB, cfg *config.Config) {
	for _, col := range []string{"model", "alias", "extra_params"} {
		if cfg.IsSQLite() {
			if err := db.Exec("ALTER TABLE provider_configs DROP COLUMN " + col).Error; err != nil {
				// SQLite 中列可能已不存在
				slog.Debug("删除旧列失败（可能已不存在）", "column", col, "error", err)
			} else {
				slog.Info("已删除旧列", "column", col)
			}
		} else {
			if err := db.Exec("ALTER TABLE provider_configs DROP COLUMN " + col).Error; err != nil {
				slog.Debug("删除旧列失败（可能已不存在）", "column", col, "error", err)
			} else {
				slog.Info("已删除旧列", "column", col)
			}
		}
	}
}

// buildModelEntriesFromOld 从旧的 model/alias/extra_params 构造 ModelEntry 列表
func buildModelEntriesFromOld(modelName, alias, extraParams string) []model.ModelEntry {
	if modelName == "" && alias == "" {
		return nil
	}

	entry := model.ModelEntry{
		Name:        modelName,
		ExtraParams: extraParams,
	}

	// 解析旧格式的逗号分隔别名
	if alias != "" {
		var aliases []string
		for _, a := range strings.Split(alias, ",") {
			if trimmed := strings.TrimSpace(a); trimmed != "" {
				aliases = append(aliases, trimmed)
			}
		}
		entry.Aliases = aliases
	}

	// 如果 model 为空但 alias 不为空，用第一个 alias 作为 name
	if entry.Name == "" && len(entry.Aliases) > 0 {
		entry.Name = entry.Aliases[0]
		entry.Aliases = entry.Aliases[1:]
	}

	// 如果 model 为空且没有别名，跳过
	if entry.Name == "" {
		return nil
	}

	return []model.ModelEntry{entry}
}

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
