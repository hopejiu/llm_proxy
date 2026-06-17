package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"log/slog"

	sqlite "github.com/glebarez/sqlite"
	"github.com/wanglejiu/llm-proxy/internal/config"
	"github.com/wanglejiu/llm-proxy/internal/handler"
	"github.com/wanglejiu/llm-proxy/internal/logger"
	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/repository"
	"github.com/wanglejiu/llm-proxy/internal/router"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// AppService Wails 绑定服务：代理控制、配置、系统信息
type AppService struct {
	cfg           *config.Config
	dbManager     *repository.DBManager
	proxyHandler  *handler.ProxyHandler
	logReader     *logger.LogReader
	dbFallbackMsg string
	mainWindow    *application.WebviewWindow
	app           *application.App

	proxyState proxyState
	ctx        context.Context
}

type proxyState struct {
	mu     sync.RWMutex
	status string // "starting" | "running" | "stopped" | "error"
	err    string
	server *http.Server
}

func NewAppService(
	cfg *config.Config,
	dbManager *repository.DBManager,
	proxyHandler *handler.ProxyHandler,
	logReader *logger.LogReader,
	dbFallbackMsg string,
) *AppService {
	return &AppService{
		cfg:           cfg,
		dbManager:     dbManager,
		proxyHandler:  proxyHandler,
		logReader:     logReader,
		dbFallbackMsg: dbFallbackMsg,
		proxyState:    proxyState{status: "stopped"},
	}
}

func (s *AppService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx

	// 自动启动代理
	if s.cfg.AutoStartProxy {
		slog.Info("自动启动代理：开始启动代理服务")
		resultCh := make(chan error, 1)
		s.startProxyServer(resultCh)
		go func() {
			if err := <-resultCh; err != nil {
				slog.Error("自动启动代理失败", "error", err)
			}
		}()
	} else {
		slog.Info("自动启动代理：已禁用，跳过自动启动")
	}

	return nil
}

// ========== Proxy 控制 ==========

// GetProxyStatus 获取代理服务状态
func (s *AppService) GetProxyStatus() ProxyStatusVO {
	s.proxyState.mu.RLock()
	defer s.proxyState.mu.RUnlock()
	return ProxyStatusVO{
		Status: s.proxyState.status,
		Port:   s.cfg.GetProxyPort(),
		Error:  s.proxyState.err,
	}
}

// StartProxy 启动代理服务
func (s *AppService) StartProxy() error {
	slog.Info("[StartProxy] 请求启动代理服务")
	s.proxyState.mu.RLock()
	if s.proxyState.status == "running" {
		s.proxyState.mu.RUnlock()
		slog.Warn("[StartProxy] 代理服务已在运行中")
		return ErrProxyRunning
	}
	s.proxyState.mu.RUnlock()

	resultCh := make(chan error, 1)
	s.startProxyServer(resultCh)

	select {
	case err := <-resultCh:
		if err != nil {
			return NewAppError("PROXY_START", err.Error())
		}
		return nil
	case <-time.After(5 * time.Second):
		return nil
	}
}

// StopProxy 停止代理服务
func (s *AppService) StopProxy() error {
	slog.Info("[StopProxy] 请求停止代理服务")
	s.proxyState.mu.RLock()
	if s.proxyState.status == "stopped" {
		s.proxyState.mu.RUnlock()
		slog.Warn("[StopProxy] 代理服务已处于停止状态")
		return ErrProxyStopped
	}
	s.proxyState.mu.RUnlock()

	s.stopProxyServer()
	return nil
}

func (s *AppService) startProxyServer(resultCh chan<- error) {
	s.proxyState.mu.Lock()
	s.proxyState.status = "starting"
	s.proxyState.err = ""
	s.proxyState.mu.Unlock()

	proxyPort := s.cfg.GetProxyPort()

	ln, err := net.Listen("tcp", ":"+proxyPort)
	if err != nil {
		s.proxyState.mu.Lock()
		s.proxyState.status = "error"
		s.proxyState.err = fmt.Sprintf("端口 %s 已被占用", proxyPort)
		s.proxyState.mu.Unlock()
		slog.Error("代理端口被占用", "port", proxyPort, "error", err)
		resultCh <- fmt.Errorf("端口 %s 已被占用: %w", proxyPort, err)
		return
	}
	ln.Close()

	proxyEngine := router.SetupProxy(s.proxyHandler)

	server := &http.Server{
		Addr:    ":" + proxyPort,
		Handler: proxyEngine,
	}

	serveResult := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serveResult <- err
			s.proxyState.mu.Lock()
			s.proxyState.status = "error"
			s.proxyState.err = err.Error()
			s.proxyState.mu.Unlock()
			slog.Error("代理服务运行异常", "error", err)
		}
	}()

	select {
	case err := <-serveResult:
		resultCh <- fmt.Errorf("代理服务启动失败: %w", err)
		return
	case <-time.After(200 * time.Millisecond):
	}

	s.proxyState.mu.Lock()
	s.proxyState.status = "running"
	s.proxyState.server = server
	s.proxyState.mu.Unlock()

	slog.Info("代理服务已启动", "port", proxyPort)
	resultCh <- nil
}

func (s *AppService) stopProxyServer() {
	s.proxyState.mu.Lock()
	server := s.proxyState.server
	s.proxyState.mu.Unlock()

	if server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("代理服务关闭失败", "error", err)
	} else {
		slog.Info("代理服务已停止")
	}

	s.proxyState.mu.Lock()
	s.proxyState.status = "stopped"
	s.proxyState.server = nil
	s.proxyState.err = ""
	s.proxyState.mu.Unlock()
}

// ========== 配置管理 ==========

// GetEnvConfig 获取所有环境变量配置项
func (s *AppService) GetEnvConfig() []config.EnvItem {
	return config.GetEnvItems()
}

// SaveEnvConfig 保存环境变量配置到 .env 文件，并热更新内存中的可变配置
func (s *AppService) SaveEnvConfig(items map[string]string) error {
	slog.Info("[SaveEnvConfig] 保存配置", "keys_count", len(items))
	if err := config.SaveEnvItems(items); err != nil {
		slog.Error("[SaveEnvConfig] 写入 .env 文件失败", "error", err)
		return err
	}

	s.cfg.HotUpdate()

	logFilePath := filepath.Join(config.DataDir(), "llm-proxy.log")
	if err := logger.Init(logFilePath, logger.ParseLevel(s.cfg.GetLogLevel())); err != nil {
		slog.Warn("重新初始化日志级别失败", "error", err)
	}

	slog.Info("配置已热更新",
		"log_level", s.cfg.GetLogLevel(),
		"stream_max_retries", s.cfg.GetStreamMaxRetries(),
		"proxy_port", s.cfg.GetProxyPort(),
	)
	return nil
}

// EnvFileExists 检查 .env 文件是否存在
func (s *AppService) EnvFileExists() bool {
	return config.EnvFileExists()
}

// ========== 系统信息 ==========

// GetVersion 获取应用版本号和构建时间
func (s *AppService) GetVersion() map[string]string {
	return map[string]string{
		"version":   Version,
		"buildTime": BuildTime,
	}
}

// GetDBFallbackMsg 获取数据库回退提示信息
func (s *AppService) GetDBFallbackMsg() string {
	return s.dbFallbackMsg
}

// ========== 运行日志 ==========

// GetLogHistory 获取全部运行日志（首次加载）
func (s *AppService) GetLogHistory() []LogEntryVO {
	entries := s.logReader.ReadAllLogs()
	result := make([]LogEntryVO, len(entries))
	for i, e := range entries {
		result[i] = LogEntryVO(e)
	}
	return result
}

// GetNewLogs 获取增量运行日志（轮询）
func (s *AppService) GetNewLogs() []LogEntryVO {
	entries := s.logReader.ReadNewLogs()
	if entries == nil {
		return []LogEntryVO{}
	}
	result := make([]LogEntryVO, len(entries))
	for i, e := range entries {
		result[i] = LogEntryVO(e)
	}
	return result
}

// SetMainWindow 保存窗口引用，供 Show/Hide 使用
func (s *AppService) SetMainWindow(w *application.WebviewWindow) {
	s.mainWindow = w
}

// ========== 窗口控制（供系统托盘调用） ==========

// ShowMainWindow 显示主窗口并置前
func (s *AppService) ShowMainWindow() {
	if s.mainWindow != nil {
		s.mainWindow.Show()
		s.mainWindow.Focus()
	}
}

// HideMainWindow 隐藏主窗口
func (s *AppService) HideMainWindow() {
	if s.mainWindow != nil {
		s.mainWindow.Hide()
	}
}

// SetApp 保存 Wails App 引用，用于 Autostart 等 API
func (s *AppService) SetApp(app *application.App) {
	s.app = app
}

// GetAutostartStatus 获取当前是否已注册开机自启动（以注册表/OS 配置为准）
func (s *AppService) GetAutostartStatus() bool {
	if s.app == nil {
		return false
	}
	enabled, err := s.app.Autostart.IsEnabled()
	if err != nil {
		slog.Warn("[GetAutostartStatus] 查询开机自启动状态失败", "error", err)
		return false
	}
	return enabled
}

// SetAutostart 设置开机自启动（写入 .env + 更新操作系统注册）
func (s *AppService) SetAutostart(enabled bool) error {
	slog.Info("[SetAutostart] 设置开机自启动", "enabled", enabled)

	// 1. 写入 .env 文件持久化
	if err := config.SaveEnvItems(map[string]string{"AUTO_START_APP": fmt.Sprintf("%t", enabled)}); err != nil {
		slog.Error("[SetAutostart] 写入 .env 失败", "error", err)
		return err
	}

	// 2. 通过 Wails API 更新操作系统开机启动注册
	if s.app == nil {
		slog.Warn("[SetAutostart] app 引用为 nil，跳过注册表更新")
		return nil
	}
	if enabled {
		return s.app.Autostart.Enable()
	}
	return s.app.Autostart.Disable()
}

// ========== 前端日志 ==========

// LogDebug 前端调试日志
func (s *AppService) LogDebug(msg string) {
	slog.Debug("[frontend] " + msg)
}

// LogInfo 前端信息日志
func (s *AppService) LogInfo(msg string) {
	slog.Info("[frontend] " + msg)
}

// LogWarn 前端警告日志
func (s *AppService) LogWarn(msg string) {
	slog.Warn("[frontend] " + msg)
}

// LogError 前端错误日志
func (s *AppService) LogError(msg string) {
	slog.Error("[frontend] " + msg)
}

// ========== 数据库热切换 ==========

// DBTestParams 数据库测试/应用参数
type DBTestParams struct {
	DBType string `json:"db_type"`
	DBHost string `json:"db_host"`
	DBPort string `json:"db_port"`
	DBUser string `json:"db_user"`
	DBPass string `json:"db_pass"`
	DBName string `json:"db_name"`
	DBPath string `json:"db_path"`
}

// DBTestResultVO 数据库测试结果
type DBTestResultVO struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TestDBConnection 测试数据库连接参数
func (s *AppService) TestDBConnection(params DBTestParams) DBTestResultVO {
	slog.Info("[TestDBConnection] 测试数据库连接", "db_type", params.DBType)

	db, err := createDBConnection(params)
	if err != nil {
		slog.Warn("[TestDBConnection] 连接失败", "error", err)
		return DBTestResultVO{Success: false, Message: err.Error()}
	}

	// 关闭临时连接
	if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
		sqlDB.Close()
	}

	slog.Info("[TestDBConnection] 连接成功")
	return DBTestResultVO{Success: true, Message: "数据库连接成功"}
}

// ApplyDBConfig 运行时切换数据库配置
func (s *AppService) ApplyDBConfig(params DBTestParams) error {
	slog.Info("[ApplyDBConfig] 应用数据库配置", "db_type", params.DBType)

	// 1. 创建新数据库连接
	newDB, err := createDBConnection(params)
	if err != nil {
		slog.Error("[ApplyDBConfig] 创建连接失败", "error", err)
		return NewAppError("DB_CONNECT", "数据库连接失败: "+err.Error())
	}

	// 2. 运行 AutoMigrate 确保表结构一致
	if err := newDB.AutoMigrate(&model.ProviderConfig{}, &model.RequestLog{}, &model.HourlyStat{}); err != nil {
		slog.Error("[ApplyDBConfig] 自动迁移失败", "error", err)
		newDBConn, _ := newDB.DB()
		if newDBConn != nil {
			newDBConn.Close()
		}
		return NewAppError("DB_MIGRATE", "自动建表失败: "+err.Error())
	}

	// 配置连接池
	configurePool(newDB, s.cfg)

	// 3. 保存到 .env 文件
	items := map[string]string{
		"DB_TYPE":     params.DBType,
		"DB_HOST":     params.DBHost,
		"DB_PORT":     params.DBPort,
		"DB_USER":     params.DBUser,
		"DB_PASSWORD": params.DBPass,
		"DB_NAME":     params.DBName,
		"DB_PATH":     params.DBPath,
	}
	if err := config.SaveEnvItems(items); err != nil {
		slog.Warn("[ApplyDBConfig] 写入 .env 失败", "error", err)
		// 即使 .env 写入失败，连接已切换成功，不阻止后续流程
	}

	// 4. 热切换数据库连接
	s.dbManager.Replace(newDB, params.DBType)

	// 5. 更新 Config 结构体中的数据库字段
	s.cfg.HotUpdate()

	slog.Info("[ApplyDBConfig] 数据库配置已生效")
	return nil
}

// createDBConnection 根据参数创建数据库连接
func createDBConnection(params DBTestParams) (*gorm.DB, error) {
	if params.DBType == "sqlite" {
		dbPath := params.DBPath
		if dbPath == "" {
			dbPath = "llm_proxy.db"
		}
		if !filepath.IsAbs(dbPath) {
			dbPath = filepath.Join(config.DataDir(), dbPath)
		}
		dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-64000)"
		return gorm.Open(sqlite.Open(dsn), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		})
	}

	host := params.DBHost
	if host == "" {
		host = "localhost"
	}
	port := params.DBPort
	if port == "" {
		port = "3306"
	}
	user := params.DBUser
	if user == "" {
		user = "root"
	}
	dbName := params.DBName
	if dbName == "" {
		dbName = "llm_proxy"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&interpolateParams=true&timeout=5s",
		user, params.DBPass, host, port, dbName)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
