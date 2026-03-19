package handler

import (
	"llm-proxy/internal/model"
	"llm-proxy/internal/service"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type WebHandler struct {
	providerService *service.ProviderService
	statsService    *service.StatsService
}

func NewWebHandler(providerService *service.ProviderService, statsService *service.StatsService) *WebHandler {
	return &WebHandler{
		providerService: providerService,
		statsService:    statsService,
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

// GetProviders 获取所有Provider
func (h *WebHandler) GetProviders(c *gin.Context) {
	providers, err := h.providerService.GetAllProviders()
	if err != nil {
		log.Printf("[WebHandler] 获取所有Provider失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, providers)
}

// GetProvider 获取单个Provider
func (h *WebHandler) GetProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		log.Printf("[WebHandler] 解析Provider ID失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	provider, err := h.providerService.GetProvider(uint(id))
	if err != nil {
		log.Printf("[WebHandler] 获取Provider失败, id=%d: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}
	c.JSON(http.StatusOK, provider)
}

// CreateProvider 创建Provider
func (h *WebHandler) CreateProvider(c *gin.Context) {
	var provider model.ProviderConfig
	if err := c.ShouldBindJSON(&provider); err != nil {
		log.Printf("[WebHandler] 绑定Provider JSON失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.providerService.CreateProvider(&provider); err != nil {
		log.Printf("[WebHandler] 创建Provider失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, provider)
}

// UpdateProvider 更新Provider
func (h *WebHandler) UpdateProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		log.Printf("[WebHandler] 解析Provider ID失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var provider model.ProviderConfig
	if err := c.ShouldBindJSON(&provider); err != nil {
		log.Printf("[WebHandler] 绑定Provider JSON失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider.ID = uint(id)
	if err := h.providerService.UpdateProvider(&provider); err != nil {
		log.Printf("[WebHandler] 更新Provider失败, id=%d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, provider)
}

// DeleteProvider 删除Provider
func (h *WebHandler) DeleteProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		log.Printf("[WebHandler] 解析Provider ID失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.providerService.DeleteProvider(uint(id)); err != nil {
		log.Printf("[WebHandler] 删除Provider失败, id=%d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// ToggleProvider 切换Provider状态
func (h *WebHandler) ToggleProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		log.Printf("[WebHandler] 解析Provider ID失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[WebHandler] 绑定Toggle请求JSON失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.providerService.ToggleProviderStatus(uint(id), req.IsActive); err != nil {
		log.Printf("[WebHandler] 切换Provider状态失败, id=%d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// GetStats 获取统计数据
func (h *WebHandler) GetStats(c *gin.Context) {
	stats, err := h.statsService.GetDashboardStats()
	if err != nil {
		log.Printf("[WebHandler] 获取仪表盘统计失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetDailyStats 获取每日统计
func (h *WebHandler) GetDailyStats(c *gin.Context) {
	stats, err := h.statsService.GetLast30DaysStats()
	if err != nil {
		log.Printf("[WebHandler] 获取30天统计失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
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
		log.Printf("[WebHandler] 获取最近日志失败, limit=%d: %v", limit, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}
