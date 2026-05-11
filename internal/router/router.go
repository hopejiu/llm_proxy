package router

import (
	"llm-proxy/internal/handler"
	"llm-proxy/internal/middleware"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupWeb 注册 Web 管理界面路由，返回配置好的 gin.Engine
func SetupWeb(h *handler.WebHandler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "web/static")

	r.GET("/", h.Index)
	r.GET("/stats", h.StatsPage)
	r.GET("/realtime", h.RealtimePage)

	api := r.Group("/api")
	{
		api.GET("/providers", h.GetProviders)
		api.GET("/providers/:id", h.GetProvider)
		api.POST("/providers", h.CreateProvider)
		api.PUT("/providers/:id", h.UpdateProvider)
		api.DELETE("/providers/:id", h.DeleteProvider)
		api.GET("/providers/export", h.ExportProviders)
		api.POST("/providers/import", h.ImportProviders)

		api.POST("/codebuddy/setup", h.SetupCodeBuddy)

		api.GET("/stats", h.GetStats)
		api.GET("/stats/daily", h.GetDailyStats)
		api.GET("/stats/hourly", h.GetTodayHourlyStats)
		api.GET("/logs/recent", h.GetRecentLogs)
		api.GET("/logs/:id", h.GetLogDetail)
		api.GET("/requests/active", h.GetActiveRequests)
	}

	return r
}

// SetupProxy 注册代理服务路由，返回配置好的 gin.Engine
func SetupProxy(h *handler.ProxyHandler, ah *handler.AnthropicHandler, oh *handler.OllamaHandler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	r.POST("/v1/chat/completions", h.ChatCompletions)
	r.GET("/v1/models", h.Models)

	r.POST("/anthropic/v1/messages", ah.Messages)
	r.GET("/anthropic/v1/models", ah.Models)

	r.POST("/api/chat", oh.Chat)
	r.GET("/api/tags", oh.Tags)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.HandleMethodNotAllowed = true
	r.NoMethod(h.MethodNotAllowed)
	r.NoRoute(h.NotFound)

	return r
}

// StartServer 启动 HTTP 服务
func StartServer(port string, engine *gin.Engine) *http.Server {
	server := &http.Server{
		Addr:    ":" + port,
		Handler: engine,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP服务启动失败", "addr", server.Addr, "error", err)
		}
	}()

	return server
}
