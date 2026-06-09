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

	"github.com/wanglejiu/llm-proxy/internal/config"
	"github.com/wanglejiu/llm-proxy/internal/handler"
	"github.com/wanglejiu/llm-proxy/internal/logger"
	"github.com/wanglejiu/llm-proxy/internal/router"
)

// AppService Wails 绑定服务：代理控制、配置、系统信息
type AppService struct {
	cfg              *config.Config
	proxyHandler     *handler.ProxyHandler
	anthropicHandler *handler.AnthropicHandler
	ollamaHandler    *handler.OllamaHandler
	logReader        *logger.LogReader
	dbFallbackMsg    string
	mainWindow       *application.WebviewWindow

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
	proxyHandler *handler.ProxyHandler,
	anthropicHandler *handler.AnthropicHandler,
	ollamaHandler *handler.OllamaHandler,
	logReader *logger.LogReader,
	dbFallbackMsg string,
) *AppService {
	return &AppService{
		cfg:              cfg,
		proxyHandler:     proxyHandler,
		anthropicHandler: anthropicHandler,
		ollamaHandler:    ollamaHandler,
		logReader:        logReader,
		dbFallbackMsg:    dbFallbackMsg,
		proxyState:       proxyState{status: "stopped"},
	}
}

func (s *AppService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx

	// 自动启动代理
	if s.cfg.AutoStartProxy {
		resultCh := make(chan error, 1)
		s.startProxyServer(resultCh)
		go func() {
			if err := <-resultCh; err != nil {
				slog.Error("自动启动代理失败", "error", err)
			}
		}()
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
	s.proxyState.mu.RLock()
	if s.proxyState.status == "running" {
		s.proxyState.mu.RUnlock()
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
	s.proxyState.mu.RLock()
	if s.proxyState.status == "stopped" {
		s.proxyState.mu.RUnlock()
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

	proxyEngine := router.SetupProxy(s.proxyHandler, s.anthropicHandler, s.ollamaHandler)

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
	if err := config.SaveEnvItems(items); err != nil {
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
