package main

import (
	"context"
	"encoding/json"
	"fmt"
	"llm-proxy/internal/config"
	"llm-proxy/internal/handler"
	"llm-proxy/internal/logger"
	"llm-proxy/internal/model"
	"llm-proxy/internal/router"
	"llm-proxy/internal/service"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App Wails 绑定层核心结构
type App struct {
	ctx              context.Context
	providerService  *service.ProviderService
	statsService     *service.StatsService
	tracker          *handler.ActiveRequestTracker
	ringBuffer       *logger.RingBuffer
	cleanupSvc       *service.CleanupService
	cfg              *config.Config
	proxyState       proxyState
	cleanupCancel    context.CancelFunc
	proxyHandler     *handler.ProxyHandler
	anthropicHandler *handler.AnthropicHandler
	ollamaHandler    *handler.OllamaHandler
}

// proxyState 代理服务状态
type proxyState struct {
	mu     sync.RWMutex
	status string // "starting" | "running" | "stopped" | "error"
	err    string
	server *http.Server
}

// NewApp 创建 App 实例
func NewApp(
	cfg *config.Config,
	providerService *service.ProviderService,
	statsService *service.StatsService,
	tracker *handler.ActiveRequestTracker,
	ringBuffer *logger.RingBuffer,
	cleanupSvc *service.CleanupService,
	proxyHandler *handler.ProxyHandler,
	anthropicHandler *handler.AnthropicHandler,
	ollamaHandler *handler.OllamaHandler,
) *App {
	return &App{
		cfg:              cfg,
		providerService:  providerService,
		statsService:     statsService,
		tracker:          tracker,
		ringBuffer:       ringBuffer,
		cleanupSvc:       cleanupSvc,
		proxyHandler:     proxyHandler,
		anthropicHandler: anthropicHandler,
		ollamaHandler:    ollamaHandler,
		proxyState:       proxyState{status: "stopped"},
	}
}

// startup Wails 启动回调
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 根据配置决定是否自动启动代理
	if a.cfg.AutoStartProxy {
		resultCh := make(chan error, 1)
		a.startProxyServer(resultCh)
		// 异步等待结果，不阻塞启动
		go func() {
			if err := <-resultCh; err != nil {
				slog.Error("自动启动代理失败", "error", err)
			}
		}()
	}

	// 启动后台任务
	cleanupCtx, cancel := context.WithCancel(ctx)
	a.cleanupCancel = cancel
	go a.cleanupSvc.Start(cleanupCtx)

	// 启动日志事件推送
	go a.startLogEventPusher()
}

// shutdown Wails 关闭回调
func (a *App) shutdown(ctx context.Context) {
	if a.cleanupCancel != nil {
		a.cleanupCancel()
	}
	a.stopProxyServer()
}

// beforeClose 窗口关闭前回调
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	// 暂不实现托盘最小化，直接关闭
	return false
}

// ========== Provider 相关 ==========

// GetProviders 获取所有 Provider（API Key 脱敏）
func (a *App) GetProviders() ([]ProviderVO, error) {
	providers, err := a.providerService.GetAllProviders()
	if err != nil {
		slog.Error("获取所有Provider失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取Provider列表失败")
	}
	return providersToVOs(providers), nil
}

// GetProvider 获取单个 Provider（API Key 脱敏）
func (a *App) GetProvider(id uint) (ProviderVO, error) {
	provider, err := a.providerService.GetProvider(id)
	if err != nil {
		slog.Error("获取Provider失败", "id", id, "error", err)
		return ProviderVO{}, NewAppError("NOT_FOUND", "Provider不存在")
	}
	return providerToVO(provider), nil
}

// CreateProvider 创建 Provider
func (a *App) CreateProvider(data ProviderCreateVO) (ProviderVO, error) {
	provider := createVOToModel(data)
	if err := a.providerService.CreateProvider(provider); err != nil {
		slog.Error("创建Provider失败", "error", err)
		return ProviderVO{}, NewAppError("INTERNAL", "创建Provider失败")
	}
	return providerToVO(provider), nil
}

// UpdateProvider 更新 Provider（含 API Key 保留逻辑）
func (a *App) UpdateProvider(id uint, data ProviderUpdateVO) (ProviderVO, error) {
	// 如果 API Key 是脱敏格式（包含 ****），保留原有密钥
	if strings.Contains(data.APIKey, "****") {
		existing, err := a.providerService.GetProvider(id)
		if err == nil {
			data.APIKey = existing.APIKey
		}
	}

	provider := updateVOToModel(id, data)
	if err := a.providerService.UpdateProvider(provider); err != nil {
		slog.Error("更新Provider失败", "id", id, "error", err)
		return ProviderVO{}, NewAppError("INTERNAL", "更新Provider失败")
	}
	return providerToVO(provider), nil
}

// DeleteProvider 删除 Provider
func (a *App) DeleteProvider(id uint) error {
	if err := a.providerService.DeleteProvider(id); err != nil {
		slog.Error("删除Provider失败", "id", id, "error", err)
		return NewAppError("INTERNAL", "删除Provider失败")
	}
	return nil
}

// ExportProvidersToFile 导出到文件（使用 SaveFileDialog）
func (a *App) ExportProvidersToFile() (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: "providers.json",
		Filters:         []runtime.FileFilter{{DisplayName: "JSON", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return "", err
	}

	providers, err := a.providerService.GetAllProviders()
	if err != nil {
		return "", NewAppError("INTERNAL", "导出失败")
	}

	data, _ := json.MarshalIndent(providers, "", "  ")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", NewAppError("INTERNAL", "写入文件失败")
	}
	return path, nil
}

// ImportProvidersFromFile 从文件导入（使用 OpenFileDialog）
func (a *App) ImportProvidersFromFile() error {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Filters: []runtime.FileFilter{{DisplayName: "JSON", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return NewAppError("INTERNAL", "读取文件失败")
	}

	var providers []model.ProviderConfig
	if err := json.Unmarshal(data, &providers); err != nil {
		return NewAppError("BAD_REQUEST", "文件格式错误")
	}

	// 清空 ID，让数据库自动生成
	for i := range providers {
		providers[i].ID = 0
	}

	if err := a.providerService.ImportAll(providers); err != nil {
		return NewAppError("INTERNAL", "导入Provider失败")
	}
	return nil
}

// SetupCodeBuddy 配置 CodeBuddy models.json（适配动态端口）
func (a *App) SetupCodeBuddy() (CodeBuddyResultVO, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("获取用户目录失败", "error", err)
		return CodeBuddyResultVO{}, NewAppError("INTERNAL", "获取用户目录失败")
	}

	codebuddyDir := filepath.Join(homeDir, ".codebuddy")
	modelsFilePath := filepath.Join(codebuddyDir, "models.json")

	if err := os.MkdirAll(codebuddyDir, 0755); err != nil {
		slog.Error("创建.codebuddy目录失败", "error", err)
		return CodeBuddyResultVO{}, NewAppError("INTERNAL", "创建配置目录失败")
	}

	var config model.CodeBuddyConfig

	if data, err := os.ReadFile(modelsFilePath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			slog.Warn("解析models.json失败", "error", err)
			config = model.CodeBuddyConfig{Models: []model.CodeBuddyModel{}}
		}
	} else {
		config = model.CodeBuddyConfig{Models: []model.CodeBuddyModel{}}
	}

	// 使用实际配置的代理端口
	targetURL := fmt.Sprintf("http://localhost:%s/v1", a.cfg.ProxyPort)
	exists := false
	for _, m := range config.Models {
		if m.URL == targetURL {
			exists = true
			break
		}
	}

	if !exists {
		newModel := model.CodeBuddyModel{
			ID:                "astron-code-latest",
			Name:              "大模型",
			Vendor:            "自定义大模型",
			APIKey:            "miyao",
			URL:               targetURL,
			SupportsToolCall:  true,
			SupportsReasoning: true,
			Temperature:       0.1,
			MaxInputTokens:    128000,
		}
		config.Models = append(config.Models, newModel)
	}

	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return CodeBuddyResultVO{}, NewAppError("INTERNAL", "序列化配置失败")
	}

	if err := os.WriteFile(modelsFilePath, data, 0644); err != nil {
		return CodeBuddyResultVO{}, NewAppError("INTERNAL", "写入配置文件失败")
	}

	message := "配置文件已创建"
	if exists {
		message = "配置已存在，无需添加"
	}

	return CodeBuddyResultVO{
		Message: message,
		Path:    modelsFilePath,
		Exists:  exists,
		Added:   !exists,
		Models:  len(config.Models),
	}, nil
}

// ========== Stats 相关 ==========

// GetStats 获取仪表盘统计
func (a *App) GetStats() (map[string]*model.TokenStats, error) {
	stats, err := a.statsService.GetDashboardStats()
	if err != nil {
		slog.Error("获取仪表盘统计失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取统计数据失败")
	}
	return stats, nil
}

// GetDailyStats 获取 30 天每日统计
func (a *App) GetDailyStats() ([]model.TokenStats, error) {
	stats, err := a.statsService.GetLast30DaysStats()
	if err != nil {
		slog.Error("获取30天统计失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取每日统计失败")
	}
	return stats, nil
}

// GetHourlyStatsByDate 获取分时统计
func (a *App) GetHourlyStatsByDate(date string) ([]model.HourlyStatsResult, error) {
	if date == "" {
		stats, err := a.statsService.GetTodayHourlyStats()
		if err != nil {
			return nil, NewAppError("INTERNAL", "获取分时统计失败")
		}
		return stats, nil
	}

	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, NewAppError("BAD_REQUEST", "日期格式错误，应为 YYYY-MM-DD")
	}

	stats, err := a.statsService.GetHourlyStatsByDate(parsedDate)
	if err != nil {
		return nil, NewAppError("INTERNAL", "获取分时统计失败")
	}
	return stats, nil
}

// ========== Logs 相关 ==========

// GetRecentLogs 获取最近请求日志
func (a *App) GetRecentLogs(limit int) ([]RequestLogVO, error) {
	if limit <= 0 {
		limit = 20
	}
	logs, err := a.statsService.GetRecentLogs(limit)
	if err != nil {
		slog.Error("获取最近日志失败", "limit", limit, "error", err)
		return nil, NewAppError("INTERNAL", "获取日志列表失败")
	}

	// 批量查询 Provider 名称，避免 N+1
	providerNames := a.buildProviderNameMap(logs)

	result := make([]RequestLogVO, len(logs))
	for i, log := range logs {
		vo := requestLogToVO(&log)
		vo.ProviderName = providerNames[log.ProviderID]
		result[i] = vo
	}
	return result, nil
}

// GetLogDetail 获取单条请求日志详情
func (a *App) GetLogDetail(id uint) (RequestLogDetailVO, error) {
	logDetail, err := a.statsService.GetLogDetail(id)
	if err != nil {
		slog.Error("获取日志详情失败", "id", id, "error", err)
		return RequestLogDetailVO{}, NewAppError("NOT_FOUND", "日志不存在")
	}

	vo := requestLogToDetailVO(logDetail)
	// 填充 Provider 名称
	if logDetail.ProviderID != model.DeletedProviderID {
		if p, err := a.providerService.GetProvider(logDetail.ProviderID); err == nil {
			vo.ProviderName = p.Name
		}
	}
	return vo, nil
}

// ========== Active Requests 相关 ==========

// GetActiveRequests 获取当前活跃请求
func (a *App) GetActiveRequests() []ActiveRequestVO {
	return a.tracker.GetAll()
}

// GetActiveRequest 按 ID 获取单个活跃请求
func (a *App) GetActiveRequest(requestID string) (ActiveRequestVO, error) {
	reqs := a.tracker.GetAll()
	for _, req := range reqs {
		if req.RequestID == requestID {
			return req, nil
		}
	}
	return ActiveRequestVO{}, NewAppError("NOT_FOUND", "请求已完成")
}

// ========== Proxy 相关 ==========

// GetProxyStatus 获取代理服务状态
func (a *App) GetProxyStatus() ProxyStatusVO {
	a.proxyState.mu.RLock()
	defer a.proxyState.mu.RUnlock()
	return ProxyStatusVO{
		Status: a.proxyState.status,
		Port:   a.cfg.ProxyPort,
		Error:  a.proxyState.err,
	}
}

// StartProxy 启动代理服务
func (a *App) StartProxy() error {
	a.proxyState.mu.RLock()
	if a.proxyState.status == "running" {
		a.proxyState.mu.RUnlock()
		return ErrProxyRunning
	}
	a.proxyState.mu.RUnlock()

	resultCh := make(chan error, 1)
	a.startProxyServer(resultCh)

	// 等待启动结果（端口占用等快速错误，或 ListenAndServe 确认成功）
	select {
	case err := <-resultCh:
		if err != nil {
			return NewAppError("PROXY_START", err.Error())
		}
		return nil
	case <-time.After(5 * time.Second):
		// 超时视为成功（ListenAndServe 已在运行但未及时返回）
		return nil
	}
}

// StopProxy 停止代理服务
func (a *App) StopProxy() error {
	a.proxyState.mu.RLock()
	if a.proxyState.status == "stopped" {
		a.proxyState.mu.RUnlock()
		return ErrProxyStopped
	}
	a.proxyState.mu.RUnlock()

	a.stopProxyServer()
	return nil
}

// ========== Log History 相关 ==========

// GetLogHistory 获取 RingBuffer 历史日志
func (a *App) GetLogHistory() []LogEntryVO {
	entries := a.ringBuffer.GetAll()
	result := make([]LogEntryVO, len(entries))
	for i, e := range entries {
		result[i] = LogEntryVO(e)
	}
	return result
}

// ========== 内部方法 ==========

func (a *App) startProxyServer(resultCh chan<- error) {
	a.proxyState.mu.Lock()
	a.proxyState.status = "starting"
	a.proxyState.err = ""
	a.proxyState.mu.Unlock()

	// 检查端口可用性
	ln, err := net.Listen("tcp", ":"+a.cfg.ProxyPort)
	if err != nil {
		a.proxyState.mu.Lock()
		a.proxyState.status = "error"
		a.proxyState.err = fmt.Sprintf("端口 %s 已被占用", a.cfg.ProxyPort)
		a.proxyState.mu.Unlock()
		slog.Error("代理端口被占用", "port", a.cfg.ProxyPort, "error", err)
		resultCh <- fmt.Errorf("端口 %s 已被占用: %w", a.cfg.ProxyPort, err)
		return
	}
	ln.Close()

	// 组装代理路由（复用现有 SetupProxy）
	proxyEngine := router.SetupProxy(a.proxyHandler, a.anthropicHandler, a.ollamaHandler)

	server := &http.Server{
		Addr:    ":" + a.cfg.ProxyPort,
		Handler: proxyEngine,
	}

	// 使用 channel 等待 ListenAndServe 的实际结果
	serveResult := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serveResult <- err
			a.proxyState.mu.Lock()
			a.proxyState.status = "error"
			a.proxyState.err = err.Error()
			a.proxyState.mu.Unlock()
			slog.Error("代理服务运行异常", "error", err)
		}
	}()

	// 短暂等待确认服务启动成功（ListenAndServe 成功后会阻塞，不会立即返回）
	// 如果 200ms 内没有错误返回，说明启动成功
	select {
	case err := <-serveResult:
		// 启动立即失败（如端口冲突的极端情况）
		resultCh <- fmt.Errorf("代理服务启动失败: %w", err)
		return
	case <-time.After(200 * time.Millisecond):
		// 没有错误返回，说明服务正在正常监听
	}

	a.proxyState.mu.Lock()
	a.proxyState.status = "running"
	a.proxyState.server = server
	a.proxyState.mu.Unlock()

	slog.Info("代理服务已启动", "port", a.cfg.ProxyPort)
	resultCh <- nil
}

func (a *App) stopProxyServer() {
	a.proxyState.mu.Lock()
	server := a.proxyState.server
	a.proxyState.mu.Unlock()

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

	a.proxyState.mu.Lock()
	a.proxyState.status = "stopped"
	a.proxyState.server = nil
	a.proxyState.err = ""
	a.proxyState.mu.Unlock()
}

func (a *App) startLogEventPusher() {
	ch := a.ringBuffer.Subscribe()
	defer a.ringBuffer.Unsubscribe(ch)

	for {
		select {
		case entry := <-ch:
			runtime.EventsEmit(a.ctx, "logEvent", LogEntryVO{
				Time:    entry.Time,
				Level:   entry.Level,
				Message: entry.Message,
			})
		case <-a.ctx.Done():
			return
		}
	}
}

// buildProviderNameMap 批量查询 Provider 名称，避免 N+1
func (a *App) buildProviderNameMap(logs []model.RequestLog) map[uint]string {
	providerIDs := make(map[uint]bool)
	for _, log := range logs {
		if log.ProviderID != model.DeletedProviderID {
			providerIDs[log.ProviderID] = true
		}
	}

	names := make(map[uint]string)
	for id := range providerIDs {
		if p, err := a.providerService.GetProvider(id); err == nil {
			names[id] = p.Name
		}
	}
	return names
}

// ========== Settings 相关 ==========

// GetEnvConfig 获取所有环境变量配置项
func (a *App) GetEnvConfig() []config.EnvItem {
	return config.GetEnvItems()
}

// SaveEnvConfig 保存环境变量配置到 .env 文件
func (a *App) SaveEnvConfig(items map[string]string) error {
	return config.SaveEnvItems(items)
}

// EnvFileExists 检查 .env 文件是否存在
func (a *App) EnvFileExists() bool {
	return config.EnvFileExists()
}

// GetVersion 获取应用版本号和构建时间
func (a *App) GetVersion() map[string]string {
	return map[string]string{
		"version":   Version,
		"buildTime": BuildTime,
	}
}
