# ADR-0004: DBManager 运行时数据库热切换

数据库配置（DB_TYPE / MySQL 连接参数）修改需要重启应用才能生效，影响用户体验。需要实现运行时无缝切换数据库的能力。

## 方案对比

| 方案 | 描述 | 复杂度 | 风险 |
|---|---|---|---|
| Repository 层 `Reconnect` | 每个 Repository 单独实现 Reconnect(newDB) | 高，需改 3 个 Repo + 所有调用方 | 遗漏某个 Repo 导致新旧混合 |
| **全局 DBManager** ✅ | 中心化管理 `*gorm.DB`，Repository 通过 GetDB() 获取 | 低，侵入小 | 旧连接关闭时机需要优雅处理 |
| 全量重建依赖 | 整个 service/repository 层重建 | 极高 | 与 Wails 生命周期耦合复杂 |

## 决策

采用**全局 DBManager 模式**。

- `repository.DBManager` 持有 `*gorm.DB` + `dbType`，用 `sync.RWMutex` 保证线程安全
- 三个 Repository 原字段 `db *gorm.DB` / `dbType string` → 替换为 `dbManager *DBManager`
- `Replace(newDB, newDBType)` 立即切换连接，旧连接在 3 秒延迟后 `Close()`，确保正在执行的请求可以正常完成
- CleanupService 的后台 goroutine 通过 Repository 间接使用 GetDB()，切换后自动使用新连接

## AppService 新增绑定方法

| 绑定方法 | 职责 |
|---|---|
| `TestDBConnection(params)` | 创建临时连接执行 Ping，验证参数可用性（数据库是否存在） |
| `ApplyDBConfig(params)` | 创建新连接 → AutoMigrate 建表 → 保存 .env → Replace 切换 |

## 改动影响

- **Repository 层**：3 个文件的 struct 定义 + 构造器 + 所有 `r.db` → `r.dbManager.GetDB()`
- **main.go**：创建 DBManager 实例注入所有 Repo + AppService
- **AppService**：新增 2 个绑定方法 + DB 连接辅助函数
- **Config**: `HotUpdate()` 新增 DB 字段和 AutoStartProxy 的更新
- **前端**: SettingsPage 重构为分卡片独立操作，数据库卡片有「测试连接 → 应用配置」两段式流程

## 决策日期

2026-06-09
