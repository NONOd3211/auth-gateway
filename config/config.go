package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
)

type Config struct {
	UpstreamURL    string
	Port           string
	DatabaseURL    string
	JWTSecret      string
	AdminPassword  string
	AdminCode      string
	UpstreamAPIKey string
	AllowedOrigins string
}

func Load() *Config {
	allowedOrigins := getEnv("ALLOWED_ORIGINS", "")
	// Disallow wildcard "*" for security
	if allowedOrigins == "*" {
		allowedOrigins = ""
	}

	// Generate random JWT secret if not provided
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		jwtSecret = generateRandomSecret(32)
	}

	// Generate random admin code if not provided
	adminCode := getEnv("ADMIN_CODE", "")
	if adminCode == "" {
		adminCode = generateRandomSecret(16)
	}

	return &Config{
		UpstreamURL:    getEnv("UPSTREAM_URL", "http://192.168.1.237:8317"),
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "./data/gateway.db"),
		JWTSecret:      jwtSecret,
		AdminPassword:  getEnv("ADMIN_PASSWORD", "admin123"),
		AdminCode:      adminCode,
		UpstreamAPIKey: getEnv("UPSTREAM_API_KEY", ""),
		AllowedOrigins: allowedOrigins,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func generateRandomSecret(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a default if crypto/rand fails
		return "default-secret-change-me"
	}
	return hex.EncodeToString(bytes)
}
