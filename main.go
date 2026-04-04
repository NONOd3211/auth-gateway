package main

import (
	"auth-gateway/config"
	"auth-gateway/database"
	"auth-gateway/handler"
	"auth-gateway/middleware"
	"auth-gateway/models"
	"auth-gateway/providers"
	"auth-gateway/providers/anthropic"
	"auth-gateway/providers/minimax"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var providerManager *providers.ProviderManager

func main() {
	cfg := config.Load()

	if err := database.Init(cfg.DatabaseURL); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}

	// Cleanup old usage records (keep max 100k)
	if err := handler.CleanupUsageRecords(); err != nil {
		log.Printf("Warning: failed to cleanup usage records: %v", err)
	} else {
		log.Printf("Usage records cleanup completed")
	}

	// Initialize provider manager after database is ready
	initProviderManager(cfg)

	gin.SetMode(gin.ReleaseMode)

	// User panel on PORT (9900)
	go runUserPanel(cfg)

	// Admin panel on ADMIN_PORT (9911)
	go runAdminPanel(cfg)

	// Proxy on PROXY_PORT (9901)
	go runProxy(cfg)

	fmt.Printf("🚀 Auth Gateway\n")
	fmt.Printf("👤 User Panel: http://localhost:%s\n", cfg.Port)
	fmt.Printf("🔐 Admin Panel: http://localhost:%s\n", cfg.AdminPort)
	fmt.Printf("🔑 Proxy: http://localhost:%s\n", cfg.ProxyPort)

	// Wait forever
	select {}
}

func initProviderManager(cfg *config.Config) {
	providerManager = providers.NewProviderManager()
	providerManager.RegisterProvider(minimax.NewExecutor(cfg))
	providerManager.RegisterProvider(anthropic.NewExecutor())

	// Load API keys from MINIMAX_API_KEYS env var
	if cfg.MiniMaxAPIKeys != "" {
		keys := strings.Split(cfg.MiniMaxAPIKeys, ",")
		for _, key := range keys {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			// Check if key already exists
			var existing models.APIKey
			if database.DB.Where("key = ?", key).First(&existing).Error != nil {
				// Create new
				apiKey := models.APIKey{
					ID:        uuid.New().String(),
					Key:       key,
					Name:      "Imported Key",
					Enabled:   true,
					Healthy:   true,
					FailCount: 0,
				}
				database.DB.Create(&apiKey)
			}
		}
	}

	// Load keys into manager
	providerManager.LoadAPIKeys()
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
		admin.GET("/usage/events", handler.GetUsageEvents)

		admin.GET("/keys", handler.ListAPIKeys)
		admin.POST("/keys", handler.CreateAPIKey)
		admin.PUT("/keys/:id", handler.UpdateAPIKey)
		admin.DELETE("/keys/:id", handler.DeleteAPIKey)
		admin.POST("/keys/:id/enable", handler.EnableAPIKey)
		admin.POST("/keys/:id/disable", handler.DisableAPIKey)
	}

	log.Printf("🔐 Admin panel listening on :%s", cfg.AdminPort)
	if err := r.Run(":" + cfg.AdminPort); err != nil {
		log.Fatalf("Admin panel failed: %v", err)
	}
}

func runProxy(cfg *config.Config) {
	handler.SetProviderManager(providerManager)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// Proxy routes (token auth)
	proxy := r.Group("/")
	proxy.Use(middleware.TokenAuth())
	{
		proxy.Any("/v1/*path", handler.ProxyRequest(cfg))
		proxy.Any("/v1beta/*path", handler.ProxyRequest(cfg))
	}

	log.Printf("🔑 Proxy listening on :%s", cfg.ProxyPort)
	if err := r.Run(":" + cfg.ProxyPort); err != nil {
		log.Fatalf("Proxy failed: %v", err)
	}
}