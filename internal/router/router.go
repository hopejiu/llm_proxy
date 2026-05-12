package router

import (
	"llm-proxy/internal/handler"
	"llm-proxy/internal/middleware"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

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
