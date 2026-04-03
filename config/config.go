package config

import (
	"os"
)

type Config struct {
	UpstreamURL    string
	Port           string
	DatabaseURL    string
	JWTSecret      string
	AdminPassword  string
	AllowedOrigins string
}

func Load() *Config {
	allowedOrigins := getEnv("ALLOWED_ORIGINS", "")
	// Disallow wildcard "*" for security
	if allowedOrigins == "*" {
		allowedOrigins = ""
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	// Reject default placeholder secret
	if jwtSecret == "change-this-secret-in-production" || jwtSecret == "" {
		jwtSecret = ""
	}

	return &Config{
		UpstreamURL:    getEnv("UPSTREAM_URL", "http://192.168.1.237:8317"),
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "./data/gateway.db"),
		JWTSecret:      jwtSecret,
		AdminPassword:  getEnv("ADMIN_PASSWORD", "admin123"),
		AllowedOrigins: allowedOrigins,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
