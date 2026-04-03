package main

import (
	"auth-gateway/config"
	"auth-gateway/database"
	"auth-gateway/handler"
	"auth-gateway/middleware"
	"fmt"
	"log"

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

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Admin API routes (require ?code=xxx query parameter)
	api := r.Group("/api")
	{
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
	}

	// Token auth middleware for proxy routes
	r.Use(middleware.TokenAuth())
	{
		r.Any("/v1/*path", handler.ProxyRequest(cfg))
		r.Any("/v1beta/*path", handler.ProxyRequest(cfg))
	}

	fmt.Printf("🚀 Auth Gateway running on :%s\n", cfg.Port)
	fmt.Printf("📡 Upstream: %s\n", cfg.UpstreamURL)
	fmt.Printf("🔐 Admin Password: %s\n", cfg.AdminPassword)
	fmt.Printf("🔑 Admin Code: %s\n", cfg.AdminCode)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
