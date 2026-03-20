package handler

import (
	"encoding/json"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"llm-proxy/internal/service"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
)

type WebHandler struct {
	providerService *service.ProviderService
	statsService    *service.StatsService
	providerRepo    *repository.ProviderRepository
	requestLogRepo  *repository.RequestLogRepository
}

func NewWebHandler(providerService *service.ProviderService, statsService *service.StatsService, providerRepo *repository.ProviderRepository, requestLogRepo *repository.RequestLogRepository) *WebHandler {
	return &WebHandler{
		providerService: providerService,
		statsService:    statsService,
		providerRepo:    providerRepo,
		requestLogRepo:  requestLogRepo,
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

// ExportProviders 导出所有Provider配置为JSON
func (h *WebHandler) ExportProviders(c *gin.Context) {
	providers, err := h.providerService.GetAllProviders()
	if err != nil {
		log.Printf("[WebHandler] 导出Provider失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=providers.json")
	c.JSON(http.StatusOK, providers)
}

// ImportProviders 导入JSON配置并覆盖现有数据
func (h *WebHandler) ImportProviders(c *gin.Context) {
	var providers []model.ProviderConfig
	if err := c.ShouldBindJSON(&providers); err != nil {
		log.Printf("[WebHandler] 绑定导入JSON失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 清空ID，让数据库自动生成
	for i := range providers {
		providers[i].ID = 0
	}

	if err := h.providerRepo.ImportAll(providers); err != nil {
		log.Printf("[WebHandler] 导入Provider失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "imported successfully", "count": len(providers)})
}

// GetLogDetail 获取单条请求日志详情
func (h *WebHandler) GetLogDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		log.Printf("[WebHandler] 解析Log ID失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	logDetail, err := h.requestLogRepo.GetByID(uint(id))
	if err != nil {
		log.Printf("[WebHandler] 获取日志详情失败, id=%d: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "log not found"})
		return
	}

	c.JSON(http.StatusOK, logDetail)
}

// CodeBuddyModel CodeBuddy models.json 中的模型结构
type CodeBuddyModel struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Vendor            string  `json:"vendor"`
	APIKey            string  `json:"apiKey"`
	URL               string  `json:"url"`
	SupportsToolCall  bool    `json:"supportsToolCall"`
	SupportsReasoning bool    `json:"supportsReasoning"`
	Temperature       float64 `json:"temperature"`
	MaxInputTokens    int     `json:"maxInputTokens"`
}

// CodeBuddyConfig CodeBuddy models.json 配置结构
type CodeBuddyConfig struct {
	Models []CodeBuddyModel `json:"models"`
}

// SetupCodeBuddy 配置 CodeBuddy models.json
func (h *WebHandler) SetupCodeBuddy(c *gin.Context) {
	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[WebHandler] 获取用户目录失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取用户目录"})
		return
	}

	// CodeBuddy 配置目录和文件路径
	codebuddyDir := filepath.Join(homeDir, ".codebuddy")
	modelsFilePath := filepath.Join(codebuddyDir, "models.json")

	// 创建目录（如果不存在）
	if err := os.MkdirAll(codebuddyDir, 0755); err != nil {
		log.Printf("[WebHandler] 创建.codebuddy目录失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建.codebuddy目录"})
		return
	}

	var config CodeBuddyConfig

	// 读取现有配置（如果存在）
	if data, err := os.ReadFile(modelsFilePath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			log.Printf("[WebHandler] 解析models.json失败: %v", err)
			// 解析失败，使用空配置
			config = CodeBuddyConfig{Models: []CodeBuddyModel{}}
		}
	} else {
		// 文件不存在，创建空配置
		config = CodeBuddyConfig{Models: []CodeBuddyModel{}}
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
		newModel := CodeBuddyModel{
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
		log.Printf("[WebHandler] 序列化models.json失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法序列化配置"})
		return
	}

	if err := os.WriteFile(modelsFilePath, data, 0644); err != nil {
		log.Printf("[WebHandler] 写入models.json失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法写入配置文件"})
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
