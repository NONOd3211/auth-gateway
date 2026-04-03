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

	// User panel on PORT (9900)
	go runUserPanel(cfg)

	// Admin panel on ADMIN_PORT (9911)
	go runAdminPanel(cfg)

	fmt.Printf("🚀 Auth Gateway\n")
	fmt.Printf("👤 User Panel: http://localhost:%s\n", cfg.Port)
	fmt.Printf("🔐 Admin Panel: http://localhost:%s\n", cfg.AdminPort)
	fmt.Printf("📡 Upstream: %s\n", cfg.UpstreamURL)

	// Wait forever
	select {}
}

func runUserPanel(cfg *config.Config) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// User panel - static files
	r.NoRoute(func(c *gin.Context) {
		path := filepath.Join("/webui/dist", c.Request.URL.Path)
		if _, err := os.Stat(path); err == nil {
			c.File(path)
		} else {
			// Serve user index.html for SPA
			c.File("/webui/dist/index.html")
		}
	})

	// Public API
	r.GET("/api/lookup", handler.LookupToken)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Printf("👤 User panel listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("User panel failed: %v", err)
	}
}

func runAdminPanel(cfg *config.Config) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// Admin panel - static files (no auth required for web UI)
	r.NoRoute(func(c *gin.Context) {
		path := filepath.Join("/webui/dist", c.Request.URL.Path)
		if _, err := os.Stat(path); err == nil {
			c.File(path)
		} else {
			// Serve admin index.html
			c.File("/webui/dist/index.html")
		}
	})

	// Admin API (auth required)
	admin := r.Group("/api/admin")
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

	// Proxy routes (token auth)
	proxy := r.Group("/")
	proxy.Use(middleware.TokenAuth())
	{
		proxy.Any("/v1/*path", handler.ProxyRequest(cfg))
		proxy.Any("/v1beta/*path", handler.ProxyRequest(cfg))
	}

	log.Printf("🔐 Admin panel listening on :%s", cfg.AdminPort)
	if err := r.Run(":" + cfg.AdminPort); err != nil {
		log.Fatalf("Admin panel failed: %v", err)
	}
}