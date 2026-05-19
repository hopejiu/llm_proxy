package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	logReader        *logger.LogReader
	cleanupSvc       *service.CleanupService
	cfg              *config.Config
	proxyState       proxyState
	cleanupCancel    context.CancelFunc
	proxyHandler     *handler.ProxyHandler
	anthropicHandler *handler.AnthropicHandler
	ollamaHandler    *handler.OllamaHandler
	dbFallbackMsg    string // MySQL 回退到 SQLite 时的提示信息
}

// proxyState 代理服务状态
type proxyState struct {
	mu     sync.RWMutex
	status string // "starting" | "running" | "stopped" | "error"
	err    string
	server *http.Server
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
	preservedKey, err := a.providerService.PreserveAPIKey(id, data.APIKey)
	if err == nil {
		data.APIKey = preservedKey
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
	targetURL := fmt.Sprintf("http://localhost:%s/v1", a.cfg.GetProxyPort())
	result, err := service.SetupCodeBuddy(targetURL)
	if err != nil {
		return CodeBuddyResultVO{}, NewAppError("INTERNAL", err.Error())
	}
	return CodeBuddyResultVO{
		Message: result.Message,
		Path:    result.Path,
		Exists:  result.Exists,
		Added:   result.Added,
		Models:  result.Models,
	}, nil
}

// ========== Provider 辅助功能 ==========

// FetchProviderModels 查询上游 API 的所有可用模型（OpenAI 兼容格式：GET /v1/models）
func (a *App) FetchProviderModels(baseURL, apiKey string) ([]string, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	apiURL := strings.TrimRight(baseURL, "/") + "/v1/models"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误(%d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	var models []string
	if dataArr, ok := result["data"].([]interface{}); ok {
		for _, m := range dataArr {
			if modelObj, ok := m.(map[string]interface{}); ok {
				if id, ok := modelObj["id"].(string); ok {
					models = append(models, id)
				}
			}
		}
	}

	return models, nil
}

// TestProviderConnection 测试 Provider 连接是否可用
func (a *App) TestProviderConnection(baseURL, apiKey, model, urlSuffix string, autoSuffix bool) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// 构建请求 URL
	var requestURL string
	if autoSuffix {
		base := strings.TrimRight(baseURL, "/")
		suffix := urlSuffix
		if suffix != "" && !strings.HasPrefix(suffix, "/") {
			suffix = "/" + suffix
		}
		requestURL = base + suffix
	} else {
		requestURL = baseURL
	}

	if requestURL == "" {
		return "", fmt.Errorf("请求URL为空，请填写Base URL")
	}

	// 最小请求体
	testBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "hi"},
		},
		"max_tokens": 5,
		"stream":     false,
	}
	body, err := json.Marshal(testBody)
	if err != nil {
		return "", fmt.Errorf("构建请求体失败: %w", err)
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("连接失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		return "✅ 连接成功，API 返回正常", nil
	} else {
		// 尝试解析错误响应
		var errResp map[string]interface{}
		errMsg := string(respBody)
		if json.Unmarshal(respBody, &errResp) == nil {
			if msg, ok := errResp["error"].(map[string]interface{}); ok {
				if message, ok := msg["message"].(string); ok {
					errMsg = message
				}
			}
		}
		return fmt.Sprintf("❌ API返回错误(%d): %s", resp.StatusCode, errMsg), nil
	}
}

// ========== Stats 相关 ==========

// GetStats 获取仪表盘统计
func (a *App) GetStats(providerID uint) (map[string]*model.TokenStats, error) {
	stats, err := a.statsService.GetDashboardStats(providerID)
	if err != nil {
		slog.Error("获取仪表盘统计失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取统计数据失败")
	}
	return stats, nil
}

// GetDailyStats 获取 30 天每日统计
func (a *App) GetDailyStats(providerID uint) ([]model.TokenStats, error) {
	stats, err := a.statsService.GetLast30DaysStats(providerID)
	if err != nil {
		slog.Error("获取30天统计失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取每日统计失败")
	}
	return stats, nil
}

// GetHourlyStatsByDate 获取分时统计
func (a *App) GetHourlyStatsByDate(date string, providerID uint) ([]model.HourlyStatsResult, error) {
	if date == "" {
		stats, err := a.statsService.GetTodayHourlyStats(providerID)
		if err != nil {
			return nil, NewAppError("INTERNAL", "获取分时统计失败")
		}
		return stats, nil
	}

	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, NewAppError("BAD_REQUEST", "日期格式错误，应为 YYYY-MM-DD")
	}

	stats, err := a.statsService.GetHourlyStatsByDate(parsedDate, providerID)
	if err != nil {
		return nil, NewAppError("INTERNAL", "获取分时统计失败")
	}
	return stats, nil
}

// GetHourlyStatsByDateWithBreakdown 获取按 provider 拆分的分时统计（用于堆叠图）
func (a *App) GetHourlyStatsByDateWithBreakdown(date string) ([]HourlyStatBreakdownVO, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	items, err := a.statsService.GetHourlyStatsByDateWithBreakdown(date)
	if err != nil {
		slog.Error("获取拆分统计失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取拆分统计失败")
	}

	result := make([]HourlyStatBreakdownVO, len(items))
	for i, item := range items {
		result[i] = HourlyStatBreakdownVO{
			Hour:         item.Hour,
			ProviderID:   item.ProviderID,
			ProviderName: item.ProviderName,
			InputTokens:  item.InputTokens,
			OutputTokens: item.OutputTokens,
			TotalTokens:  item.TotalTokens,
		}
	}
	return result, nil
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
		Port:   a.cfg.GetProxyPort(),
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

// GetLogHistory 获取全部运行日志（首次加载）
func (a *App) GetLogHistory() []LogEntryVO {
	entries := a.logReader.ReadAllLogs()
	result := make([]LogEntryVO, len(entries))
	for i, e := range entries {
		result[i] = LogEntryVO(e)
	}
	return result
}

// GetNewLogs 获取增量运行日志（轮询）
func (a *App) GetNewLogs() []LogEntryVO {
	entries := a.logReader.ReadNewLogs()
	if entries == nil {
		return []LogEntryVO{}
	}
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

	proxyPort := a.cfg.GetProxyPort()

	// 检查端口可用性
	ln, err := net.Listen("tcp", ":"+proxyPort)
	if err != nil {
		a.proxyState.mu.Lock()
		a.proxyState.status = "error"
		a.proxyState.err = fmt.Sprintf("端口 %s 已被占用", proxyPort)
		a.proxyState.mu.Unlock()
		slog.Error("代理端口被占用", "port", proxyPort, "error", err)
		resultCh <- fmt.Errorf("端口 %s 已被占用: %w", proxyPort, err)
		return
	}
	ln.Close()

	// 组装代理路由（复用现有 SetupProxy）
	proxyEngine := router.SetupProxy(a.proxyHandler, a.anthropicHandler, a.ollamaHandler)

	server := &http.Server{
		Addr:    ":" + proxyPort,
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

	slog.Info("代理服务已启动", "port", proxyPort)
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

// SaveEnvConfig 保存环境变量配置到 .env 文件，并热更新内存中的可变配置
func (a *App) SaveEnvConfig(items map[string]string) error {
	if err := config.SaveEnvItems(items); err != nil {
		return err
	}

	// 热更新内存中的可变配置（直接从 .env 文件读取，不依赖 os.Getenv）
	a.cfg.HotUpdate()

	// 如果日志级别变更，重新初始化 logger
	logFilePath := filepath.Join(config.DataDir(), "llm-proxy.log")
	if err := logger.Init(logFilePath, logger.ParseLevel(a.cfg.GetLogLevel())); err != nil {
		slog.Warn("重新初始化日志级别失败", "error", err)
	}

	slog.Info("配置已热更新", "log_level", a.cfg.GetLogLevel(), "stream_max_retries", a.cfg.GetStreamMaxRetries(), "proxy_port", a.cfg.GetProxyPort())
	return nil
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

// GetDBFallbackMsg 获取数据库回退提示信息（MySQL 连接失败回退到 SQLite 时有值）
func (a *App) GetDBFallbackMsg() string {
	return a.dbFallbackMsg
}
