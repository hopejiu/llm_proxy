package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wanglejiu/llm-proxy/internal/config"
	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/repository"

	sqlite "github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ========== 数据库初始化 ==========

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

	// 带重试的 MySQL 连接
	retryInterval := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second, 7 * time.Second, 9 * time.Second}
	for i, interval := range retryInterval {
		db, err = gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
		if err == nil {
			break
		}
		if i < len(retryInterval)-1 {
			slog.Warn("MySQL数据库连接失败，即将重试", "attempt", i+1, "error", err)
			time.Sleep(interval)
		} else {
			fatalMessageBox("启动失败", fmt.Sprintf("MySQL数据库连接失败，已重试 %d 次: %s", len(retryInterval), err.Error()))
		}
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

// ========== Schema 迁移 ==========

func migrateDB(db *gorm.DB, cfg *config.Config) {
	slog.Info("正在初始化数据库表...")

	if cfg.IsSQLite() {
		createSQLiteTablesIfNotExist(db)
	}

	// 在 AutoMigrate 之前先删除旧索引，避免唯一索引冲突
	dropOldHourlyIndexes(db, cfg)

	if err := db.AutoMigrate(&model.ProviderConfig{}, &model.RequestLog{}, &model.HourlyStat{}, &model.ChatSession{}); err != nil {
		fatalMessageBox("启动失败", "数据库初始化失败: "+err.Error())
	}

	if !cfg.IsSQLite() {
		tables := []string{"provider_configs", "request_logs", "hourly_stats", "chat_sessions"}
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
	migrateHourlyStats(db, cfg)
	migrateProviderConfigs(db, cfg)
	migrateExtraParamsToggle(db, cfg)
	migrateAutoFixThinking(db, cfg)
	createModelIndexes(db)
	slog.Info("数据库表初始化完成")
}

func dropOldHourlyIndexes(db *gorm.DB, cfg *config.Config) {
	if !cfg.IsSQLite() {
		// 直接尝试删除旧索引，不依赖 information_schema 查询（某些 MySQL 版本/权限下可能检测不到）
		if err := db.Exec("ALTER TABLE hourly_stats DROP INDEX idx_hour_provider").Error; err != nil {
			// 错误码 1091 = "Can't DROP INDEX; check that it exists"，属于正常情况
			slog.Debug("删除旧索引 idx_hour_provider（可能不存在）", "error", err)
		} else {
			slog.Info("已删除旧的唯一索引 idx_hour_provider(hour, provider_id)，使用 idx_hour_provider_model(hour, provider_id, model) 替代")
		}
	} else {
		if err := db.Exec("DROP INDEX IF EXISTS idx_hourly_stats_hour").Error; err != nil {
			slog.Debug("删除旧 SQLite 索引失败", "error", err)
		}
		if err := db.Exec("DROP INDEX IF EXISTS idx_hour_provider").Error; err != nil {
			slog.Debug("删除旧 SQLite 索引 idx_hour_provider 失败", "error", err)
		}
	}
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
			enable_extra_params INTEGER DEFAULT 1,
			auto_fix_thinking INTEGER DEFAULT 0,
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
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			models TEXT DEFAULT '',
			request_count INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			total_cost REAL DEFAULT 0,
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

// migrateExtraParamsToggle 为 provider_configs 添加 enable_extra_params 列（默认启用）
func migrateExtraParamsToggle(db *gorm.DB, cfg *config.Config) {
	if db.Migrator().HasColumn(&model.ProviderConfig{}, "enable_extra_params") {
		return
	}

	slog.Info("正在更新 provider_configs 表，添加 enable_extra_params 列...")
	if cfg.IsSQLite() {
		db.Exec("ALTER TABLE provider_configs ADD COLUMN enable_extra_params INTEGER DEFAULT 1")
	} else {
		db.Exec("ALTER TABLE provider_configs ADD COLUMN enable_extra_params TINYINT(1) DEFAULT 1")
	}
	// 确保现有行启用
	db.Exec("UPDATE provider_configs SET enable_extra_params = 1 WHERE enable_extra_params IS NULL OR enable_extra_params = 0")
	slog.Info("enable_extra_params 列添加完成")
}

// migrateAutoFixThinking 为 provider_configs 添加 auto_fix_thinking 列（默认关闭）
func migrateAutoFixThinking(db *gorm.DB, cfg *config.Config) {
	if db.Migrator().HasColumn(&model.ProviderConfig{}, "auto_fix_thinking") {
		return
	}

	slog.Info("正在更新 provider_configs 表，添加 auto_fix_thinking 列...")
	if cfg.IsSQLite() {
		db.Exec("ALTER TABLE provider_configs ADD COLUMN auto_fix_thinking INTEGER DEFAULT 0")
	} else {
		db.Exec("ALTER TABLE provider_configs ADD COLUMN auto_fix_thinking TINYINT(1) DEFAULT 0")
	}
	slog.Info("auto_fix_thinking 列添加完成")
}

func createModelIndexes(db *gorm.DB) {
	migrator := db.Migrator()
	indexNames := []string{
		"idx_created_at_status",
		"idx_created_at",
		"idx_provider_id",
		"idx_session_id",
	}
	for _, name := range indexNames {
		if migrator.HasIndex(&model.RequestLog{}, name) {
			continue
		}
		if err := migrator.CreateIndex(&model.RequestLog{}, name); err != nil {
			slog.Warn("创建索引失败", "index", name, "error", err)
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
func migrateProviderConfigs(db *gorm.DB, cfg *config.Config) {
	if db.Migrator().HasColumn(&model.ProviderConfig{}, "Models") {
		var count int64
		db.Model(&model.ProviderConfig{}).Where("models IS NOT NULL AND models != ''").Count(&count)
		if count > 0 {
			dropOldProviderColumns(db, cfg)
			slog.Debug("provider_configs 已是新格式，跳过迁移", "rows", count)
			return
		}
	}

	hasOldCols := checkOldProviderColumnsExist(db, cfg)
	if !hasOldCols && !db.Migrator().HasColumn(&model.ProviderConfig{}, "Models") {
		return
	}

	if !db.Migrator().HasColumn(&model.ProviderConfig{}, "Models") {
		if err := db.Migrator().AddColumn(&model.ProviderConfig{}, "Models"); err != nil {
			slog.Warn("添加 models 列失败", "error", err)
			return
		}
	}

	if hasOldCols {
		slog.Info("正在迁移 provider_configs 表到新的多模型格式...")

		type oldRow struct {
			ID          uint
			Models      string
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

		dropOldProviderColumns(db, cfg)
		slog.Info("provider_configs 迁移完成")
	}
}

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
	var count int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'provider_configs'
		AND COLUMN_NAME IN ('model','alias','extra_params')`).Scan(&count)
	return count > 0
}

func dropOldProviderColumns(db *gorm.DB, cfg *config.Config) {
	for _, col := range []string{"model", "alias", "extra_params"} {
		var exists bool
		if cfg.IsSQLite() {
			var cols []string
			db.Raw("PRAGMA table_info(provider_configs)").Pluck("name", &cols)
			for _, c := range cols {
				if c == col {
					exists = true
					break
				}
			}
		} else {
			var count int64
			db.Raw(`SELECT COUNT(*) FROM information_schema.COLUMNS
				WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'provider_configs'
				AND COLUMN_NAME = ?`, col).Scan(&count)
			exists = count > 0
		}
		if !exists {
			continue
		}
		if err := db.Exec("ALTER TABLE provider_configs DROP COLUMN " + col).Error; err != nil {
			slog.Warn("删除旧列失败", "column", col, "error", err)
		} else {
			slog.Info("已删除旧列", "column", col)
		}
	}
}

func buildModelEntriesFromOld(modelName, alias, extraParams string) []model.ModelEntry {
	if modelName == "" && alias == "" {
		return nil
	}
	entry := model.ModelEntry{
		Name:        modelName,
		ExtraParams: extraParams,
	}
	if alias != "" {
		var aliases []string
		for _, a := range strings.Split(alias, ",") {
			if trimmed := strings.TrimSpace(a); trimmed != "" {
				aliases = append(aliases, trimmed)
			}
		}
		entry.Aliases = aliases
	}
	if entry.Name == "" && len(entry.Aliases) > 0 {
		entry.Name = entry.Aliases[0]
		entry.Aliases = entry.Aliases[1:]
	}
	if entry.Name == "" {
		return nil
	}
	return []model.ModelEntry{entry}
}

// ========== 启动时清理 ==========

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
