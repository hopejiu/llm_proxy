package handler

import (
	"fmt"
	"llm-proxy/internal/config"
	"llm-proxy/internal/model"
	"llm-proxy/internal/service"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type WebHandler struct {
	providerService *service.ProviderService
	statsService    *service.StatsService
	tracker         *ActiveRequestTracker
	cfg             *config.Config
}

func NewWebHandler(providerService *service.ProviderService, statsService *service.StatsService, tracker *ActiveRequestTracker, cfg *config.Config) *WebHandler {
	return &WebHandler{
		providerService: providerService,
		statsService:    statsService,
		tracker:         tracker,
		cfg:             cfg,
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
		respondError(c, http.StatusInternalServerError, CodeInternal, "获取Provider列表失败")
		return
	}
	respondOK(c, providers)
}

// GetProvider 获取单个Provider
func (h *WebHandler) GetProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		slog.Error("解析Provider ID失败", "error", err)
		respondError(c, http.StatusBadRequest, CodeBadRequest, "无效的Provider ID")
		return
	}

	provider, err := h.providerService.GetProvider(uint(id))
	if err != nil {
		slog.Error("获取Provider失败", "id", id, "error", err)
		respondError(c, http.StatusNotFound, CodeNotFound, "Provider不存在")
		return
	}
	respondOK(c, provider)
}

// CreateProvider 创建Provider
func (h *WebHandler) CreateProvider(c *gin.Context) {
	var provider model.ProviderConfig
	if err := c.ShouldBindJSON(&provider); err != nil {
		slog.Error("绑定Provider JSON失败", "error", err)
		respondError(c, http.StatusBadRequest, CodeInvalidInput, "请求参数格式错误")
		return
	}

	if err := h.providerService.CreateProvider(&provider); err != nil {
		slog.Error("创建Provider失败", "error", err)
		respondError(c, http.StatusInternalServerError, CodeInternal, "创建Provider失败")
		return
	}

	respondCreated(c, provider)
}

// UpdateProvider 更新Provider
func (h *WebHandler) UpdateProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		slog.Error("解析Provider ID失败", "error", err)
		respondError(c, http.StatusBadRequest, CodeBadRequest, "无效的Provider ID")
		return
	}

	var provider model.ProviderConfig
	if err := c.ShouldBindJSON(&provider); err != nil {
		slog.Error("绑定Provider JSON失败", "error", err)
		respondError(c, http.StatusBadRequest, CodeInvalidInput, "请求参数格式错误")
		return
	}

	provider.ID = uint(id)

	// 如果 API Key 是脱敏格式（包含 ****），保留原有密钥
	preservedKey, err := h.providerService.PreserveAPIKey(provider.ID, provider.APIKey)
	if err == nil {
		provider.APIKey = preservedKey
	}

	if err := h.providerService.UpdateProvider(&provider); err != nil {
		slog.Error("更新Provider失败", "id", id, "error", err)
		respondError(c, http.StatusInternalServerError, CodeInternal, "更新Provider失败")
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
		respondError(c, http.StatusBadRequest, CodeBadRequest, "无效的Provider ID")
		return
	}

	if err := h.providerService.DeleteProvider(uint(id)); err != nil {
		slog.Error("删除Provider失败", "id", id, "error", err)
		respondError(c, http.StatusInternalServerError, CodeInternal, "删除Provider失败")
		return
	}

	respondOK(c, gin.H{"message": "deleted successfully"})
}

// GetStats 获取统计数据
func (h *WebHandler) GetStats(c *gin.Context) {
	stats, err := h.statsService.GetDashboardStats(0)
	if err != nil {
		slog.Error("获取仪表盘统计失败", "error", err)
		respondError(c, http.StatusInternalServerError, CodeInternal, "获取统计数据失败")
		return
	}
	respondOK(c, stats)
}

// GetDailyStats 获取每日统计
func (h *WebHandler) GetDailyStats(c *gin.Context) {
	stats, err := h.statsService.GetLast30DaysStats(0)
	if err != nil {
		slog.Error("获取30天统计失败", "error", err)
		respondError(c, http.StatusInternalServerError, CodeInternal, "获取每日统计失败")
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
		respondError(c, http.StatusInternalServerError, CodeInternal, "获取日志列表失败")
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
		respondError(c, http.StatusInternalServerError, CodeInternal, "导出Provider失败")
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
		respondError(c, http.StatusBadRequest, CodeInvalidInput, "导入数据格式错误")
		return
	}

	// 清空ID，让数据库自动生成
	for i := range providers {
		providers[i].ID = 0
	}

	if err := h.providerService.ImportAll(providers); err != nil {
		slog.Error("导入Provider失败", "error", err)
		respondError(c, http.StatusInternalServerError, CodeInternal, "导入Provider失败")
		return
	}

	respondOK(c, gin.H{"message": "imported successfully", "count": len(providers)})
}

// GetLogDetail 获取单条请求日志详情
func (h *WebHandler) GetLogDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		slog.Error("解析Log ID失败", "error", err)
		respondError(c, http.StatusBadRequest, CodeBadRequest, "无效的日志ID")
		return
	}

	logDetail, err := h.statsService.GetLogDetail(uint(id))
	if err != nil {
		slog.Error("获取日志详情失败", "id", id, "error", err)
		respondError(c, http.StatusNotFound, CodeNotFound, "日志不存在")
		return
	}

	respondOK(c, logDetail)
}

// GetTodayHourlyStats 获取分时统计（支持 date 查询参数指定日期，默认今日）
func (h *WebHandler) GetTodayHourlyStats(c *gin.Context) {
	dateStr := c.Query("date")

	if dateStr == "" {
		// 默认今日
		stats, err := h.statsService.GetTodayHourlyStats(0)
		if err != nil {
			slog.Error("获取今日分时统计失败", "error", err)
			respondError(c, http.StatusInternalServerError, CodeInternal, "获取分时统计失败")
			return
		}
		respondOK(c, stats)
		return
	}

	// 解析日期参数
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		slog.Error("解析日期参数失败", "date", dateStr, "error", err)
		respondError(c, http.StatusBadRequest, CodeBadRequest, "日期格式错误，应为 YYYY-MM-DD")
		return
	}

	stats, err := h.statsService.GetHourlyStatsByDate(date, 0)
	if err != nil {
		slog.Error("获取指定日期分时统计失败", "date", dateStr, "error", err)
		respondError(c, http.StatusInternalServerError, CodeInternal, "获取分时统计失败")
		return
	}
	respondOK(c, stats)
}

// SetupCodeBuddy 配置 CodeBuddy models.json
func (h *WebHandler) SetupCodeBuddy(c *gin.Context) {
	targetURL := fmt.Sprintf("http://localhost:%s/v1", h.cfg.ProxyPort)
	result, err := service.SetupCodeBuddy(targetURL)
	if err != nil {
		respondError(c, http.StatusInternalServerError, CodeInternal, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": result.Message,
		"path":    result.Path,
		"exists":  result.Exists,
		"added":   result.Added,
		"models":  result.Models,
	})
}
