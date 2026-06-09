package repository

import (
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"
)

// DBManager 运行时数据库连接管理器
// 所有 Repository 通过 GetDB() / GetDBType() 获取当前连接，
// 支持运行时 Replace() 热切换数据库。
type DBManager struct {
	mu     sync.RWMutex
	db     *gorm.DB
	dbType string
}

func NewDBManager(db *gorm.DB, dbType string) *DBManager {
	return &DBManager{db: db, dbType: dbType}
}

// GetDB 获取当前数据库连接（线程安全）
func (m *DBManager) GetDB() *gorm.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.db
}

// GetDBType 获取当前数据库类型（线程安全）
func (m *DBManager) GetDBType() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dbType
}

// Replace 热切换数据库连接
// 新连接立即生效，旧连接在 3 秒延迟后关闭，确保正在执行的请求能正常完成
func (m *DBManager) Replace(newDB *gorm.DB, newDBType string) {
	m.mu.Lock()
	oldDB := m.db
	m.db = newDB
	m.dbType = newDBType
	m.mu.Unlock()

	if oldDB != nil {
		go func() {
			time.Sleep(3 * time.Second)
			if sqlDB, err := oldDB.DB(); err == nil && sqlDB != nil {
				if err := sqlDB.Close(); err != nil {
					slog.Warn("关闭旧数据库连接失败", "error", err)
				} else {
					slog.Info("旧数据库连接已关闭")
				}
			}
		}()
	}
}
