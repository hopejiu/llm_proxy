package handler

import (
	"encoding/json"
	"llm-proxy/internal/model"
	"llm-proxy/internal/service"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type WebHandler struct {
	providerService *service.ProviderService
	statsService    *service.StatsService
	tracker         *ActiveRequestTracker
}

func NewWebHandler(providerService *service.ProviderService, statsService *service.StatsService, tracker *ActiveRequestTracker) *WebHandler {
	return &WebHandler{
		providerService: providerService,
		statsService:    statsService,
		tracker:         tracker,
	}
}

// Index 配置管理页面
func (h *WebHandler) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

// StatsPage 统计页面
func (h *WebHandler) StatsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "stats.html", gin.H{})
}

// RealtimePage 实时监控页面
func (h *WebHandler) RealtimePage(c *gin.Context) {
	c.HTML(http.StatusOK, "realtime.html", gin.H{})
}

// GetProviders 获取所有Provider
func (h *WebHandler) GetProviders(c *gin.Context) {
	providers, err := h.providerService.GetAllProviders()
	if err != nil {
		slog.Error("获取所有Provider失败", "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "获取Provider列表失败")
		return
	}
	// 脱敏 API Key
	for i := range providers {
		providers[i].APIKey = providers[i].MaskAPIKey()
	}
	respondOK(c, providers)
}

// GetProvider 获取单个Provider
func (h *WebHandler) GetProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		slog.Error("解析Provider ID失败", "error", err)
		respondError(c, http.StatusBadRequest, ErrBadRequest, "无效的Provider ID")
		return
	}

	provider, err := h.providerService.GetProvider(uint(id))
	if err != nil {
		slog.Error("获取Provider失败", "id", id, "error", err)
		respondError(c, http.StatusNotFound, ErrNotFound, "Provider不存在")
		return
	}
	provider.APIKey = provider.MaskAPIKey()
	respondOK(c, provider)
}

// CreateProvider 创建Provider
func (h *WebHandler) CreateProvider(c *gin.Context) {
	var provider model.ProviderConfig
	if err := c.ShouldBindJSON(&provider); err != nil {
		slog.Error("绑定Provider JSON失败", "error", err)
		respondError(c, http.StatusBadRequest, ErrInvalidInput, "请求参数格式错误")
		return
	}

	if err := h.providerService.CreateProvider(&provider); err != nil {
		slog.Error("创建Provider失败", "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "创建Provider失败")
		return
	}

	respondCreated(c, provider)
}

// UpdateProvider 更新Provider
func (h *WebHandler) UpdateProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		slog.Error("解析Provider ID失败", "error", err)
		respondError(c, http.StatusBadRequest, ErrBadRequest, "无效的Provider ID")
		return
	}

	var provider model.ProviderConfig
	if err := c.ShouldBindJSON(&provider); err != nil {
		slog.Error("绑定Provider JSON失败", "error", err)
		respondError(c, http.StatusBadRequest, ErrInvalidInput, "请求参数格式错误")
		return
	}

	provider.ID = uint(id)

	// 如果 API Key 是脱敏格式（包含 ****），保留原有密钥
	if strings.Contains(provider.APIKey, "****") {
		existing, err := h.providerService.GetProvider(provider.ID)
		if err == nil {
			provider.APIKey = existing.APIKey
		}
	}

	if err := h.providerService.UpdateProvider(&provider); err != nil {
		slog.Error("更新Provider失败", "id", id, "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "更新Provider失败")
		return
	}

	provider.APIKey = provider.MaskAPIKey()
	respondOK(c, provider)
}

// DeleteProvider 删除Provider
func (h *WebHandler) DeleteProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		slog.Error("解析Provider ID失败", "error", err)
		respondError(c, http.StatusBadRequest, ErrBadRequest, "无效的Provider ID")
		return
	}

	if err := h.providerService.DeleteProvider(uint(id)); err != nil {
		slog.Error("删除Provider失败", "id", id, "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "删除Provider失败")
		return
	}

	respondOK(c, gin.H{"message": "deleted successfully"})
}

// GetStats 获取统计数据
func (h *WebHandler) GetStats(c *gin.Context) {
	stats, err := h.statsService.GetDashboardStats()
	if err != nil {
		slog.Error("获取仪表盘统计失败", "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "获取统计数据失败")
		return
	}
	respondOK(c, stats)
}

// GetDailyStats 获取每日统计
func (h *WebHandler) GetDailyStats(c *gin.Context) {
	stats, err := h.statsService.GetLast30DaysStats()
	if err != nil {
		slog.Error("获取30天统计失败", "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "获取每日统计失败")
		return
	}
	respondOK(c, stats)
}

// GetRecentLogs 获取最近日志
func (h *WebHandler) GetRecentLogs(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := h.statsService.GetRecentLogs(limit)
	if err != nil {
		slog.Error("获取最近日志失败", "limit", limit, "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "获取日志列表失败")
		return
	}
	respondOK(c, logs)
}

// GetActiveRequests 获取当前正在进行的请求
func (h *WebHandler) GetActiveRequests(c *gin.Context) {
	tracker := h.tracker
	activeReqs := tracker.GetAll()
	respondOK(c, activeReqs)
}

// ExportProviders 导出所有Provider配置为JSON
func (h *WebHandler) ExportProviders(c *gin.Context) {
	providers, err := h.providerService.GetAllProviders()
	if err != nil {
		slog.Error("导出Provider失败", "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "导出Provider失败")
		return
	}

	c.Header("Content-Disposition", "attachment; filename=providers.json")
	respondOK(c, providers)
}

// ImportProviders 导入JSON配置并覆盖现有数据
func (h *WebHandler) ImportProviders(c *gin.Context) {
	var providers []model.ProviderConfig
	if err := c.ShouldBindJSON(&providers); err != nil {
		slog.Error("绑定导入JSON失败", "error", err)
		respondError(c, http.StatusBadRequest, ErrInvalidInput, "导入数据格式错误")
		return
	}

	// 清空ID，让数据库自动生成
	for i := range providers {
		providers[i].ID = 0
	}

	if err := h.providerService.ImportAll(providers); err != nil {
		slog.Error("导入Provider失败", "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "导入Provider失败")
		return
	}

	respondOK(c, gin.H{"message": "imported successfully", "count": len(providers)})
}

// GetLogDetail 获取单条请求日志详情
func (h *WebHandler) GetLogDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		slog.Error("解析Log ID失败", "error", err)
		respondError(c, http.StatusBadRequest, ErrBadRequest, "无效的日志ID")
		return
	}

	logDetail, err := h.statsService.GetLogDetail(uint(id))
	if err != nil {
		slog.Error("获取日志详情失败", "id", id, "error", err)
		respondError(c, http.StatusNotFound, ErrNotFound, "日志不存在")
		return
	}

	respondOK(c, logDetail)
}

// GetTodayHourlyStats 获取分时统计（支持 date 查询参数指定日期，默认今日）
func (h *WebHandler) GetTodayHourlyStats(c *gin.Context) {
	dateStr := c.Query("date")

	if dateStr == "" {
		// 默认今日
		stats, err := h.statsService.GetTodayHourlyStats()
		if err != nil {
			slog.Error("获取今日分时统计失败", "error", err)
			respondError(c, http.StatusInternalServerError, ErrInternal, "获取分时统计失败")
			return
		}
		respondOK(c, stats)
		return
	}

	// 解析日期参数
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		slog.Error("解析日期参数失败", "date", dateStr, "error", err)
		respondError(c, http.StatusBadRequest, ErrBadRequest, "日期格式错误，应为 YYYY-MM-DD")
		return
	}

	stats, err := h.statsService.GetHourlyStatsByDate(date)
	if err != nil {
		slog.Error("获取指定日期分时统计失败", "date", dateStr, "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "获取分时统计失败")
		return
	}
	respondOK(c, stats)
}

// SetupCodeBuddy 配置 CodeBuddy models.json
func (h *WebHandler) SetupCodeBuddy(c *gin.Context) {
	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("获取用户目录失败", "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "获取用户目录失败")
		return
	}

	// CodeBuddy 配置目录和文件路径
	codebuddyDir := filepath.Join(homeDir, ".codebuddy")
	modelsFilePath := filepath.Join(codebuddyDir, "models.json")

	// 创建目录（如果不存在）
	if err := os.MkdirAll(codebuddyDir, 0755); err != nil {
		slog.Error("创建.codebuddy目录失败", "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "创建配置目录失败")
		return
	}

	var config model.CodeBuddyConfig

	// 读取现有配置（如果存在）
	if data, err := os.ReadFile(modelsFilePath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			slog.Warn("解析models.json失败", "error", err)
			// 解析失败，使用空配置
			config = model.CodeBuddyConfig{Models: []model.CodeBuddyModel{}}
		}
	} else {
		// 文件不存在，创建空配置
		config = model.CodeBuddyConfig{Models: []model.CodeBuddyModel{}}
	}

	// 检查是否已存在 localhost:8888/v1 的配置
	targetURL := "http://localhost:8888/v1"
	exists := false
	for _, m := range config.Models {
		if m.URL == targetURL {
			exists = true
			break
		}
	}

	// 如果不存在，添加新配置
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

	// 写入配置文件
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		slog.Error("序列化models.json失败", "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "序列化配置失败")
		return
	}

	if err := os.WriteFile(modelsFilePath, data, 0644); err != nil {
		slog.Error("写入models.json失败", "error", err)
		respondError(c, http.StatusInternalServerError, ErrInternal, "写入配置文件失败")
		return
	}

	message := "配置文件已创建"
	if exists {
		message = "配置已存在，无需添加"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  message,
		"path":     modelsFilePath,
		"exists":   exists,
		"added":    !exists,
		"models":   len(config.Models),
	})
}
