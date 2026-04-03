package main

import (
	"auth-gateway/config"
	"auth-gateway/database"
	"auth-gateway/handler"
	"auth-gateway/middleware"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	if err := database.Init(cfg.DatabaseURL); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.AdminCodeAuth(cfg))

	// Static web UI files (no auth required)
	// Serve existing files directly, fallback to index.html for SPA routing
	r.NoRoute(func(c *gin.Context) {
		path := filepath.Join("/webui/dist", c.Request.URL.Path)
		if _, err := os.Stat(path); err == nil {
			c.File(path)
		} else {
			c.File("/webui/dist/index.html")
		}
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public API routes (no auth required)
	api := r.Group("/api")
	{
		api.GET("/lookup", handler.LookupToken)
	}

	// Admin API routes (require ?code=xxx query parameter)
	admin := api.Group("/admin")
	admin.Use(middleware.RequireAdmin())
	admin.Use(middleware.AdminAuth(cfg))
	{
		admin.GET("/tokens", handler.ListTokens)
		admin.POST("/tokens", handler.CreateToken)
		admin.GET("/tokens/:id", handler.GetToken)
		admin.PUT("/tokens/:id", handler.UpdateToken)
		admin.DELETE("/tokens/:id", handler.DeleteToken)
		admin.POST("/tokens/:id/reset", handler.ResetUsage)

		admin.GET("/usage", handler.GetUsageStats)
		admin.GET("/usage/daily", handler.GetUsageByDay)
		admin.GET("/usage/token/:id", handler.GetUsageByToken)
	}

	// Proxy routes (require Bearer token auth)
	proxy := r.Group("/")
	proxy.Use(middleware.TokenAuth())
	{
		proxy.Any("/v1/*path", handler.ProxyRequest(cfg))
		proxy.Any("/v1beta/*path", handler.ProxyRequest(cfg))
	}

	fmt.Printf("🚀 Auth Gateway running on :%s\n", cfg.Port)
	fmt.Printf("📡 Upstream: %s\n", cfg.UpstreamURL)
	fmt.Printf("🔐 Admin Password: %s\n", cfg.AdminPassword)
	fmt.Printf("🔑 Admin Code: %s\n", cfg.AdminCode)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
