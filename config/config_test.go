package config

import (
	"os"
	"testing"
)

func TestAllowedOriginsWildcardReturnsEmpty(t *testing.T) {
	// CORS wildcard "*" is a security risk - should return empty string instead
	os.Setenv("ALLOWED_ORIGINS", "*")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	cfg := Load()

	if cfg.AllowedOrigins != "" {
		t.Errorf("AllowedOrigins should be empty when set to '*', got %q", cfg.AllowedOrigins)
	}
}

func TestJWTSecretDefaultPlaceholderReturnsEmpty(t *testing.T) {
	// JWT_SECRET with default placeholder should return empty to force proper configuration
	os.Setenv("JWT_SECRET", "change-this-secret-in-production")
	defer os.Unsetenv("JWT_SECRET")

	cfg := Load()

	if cfg.JWTSecret != "" {
		t.Errorf("JWTSecret should be empty when set to default placeholder, got %q", cfg.JWTSecret)
	}
}

func TestJWTSecretEmptyReturnsEmpty(t *testing.T) {
	// Empty JWT_SECRET should return empty
	os.Setenv("JWT_SECRET", "")
	defer os.Unsetenv("JWT_SECRET")

	cfg := Load()

	if cfg.JWTSecret != "" {
		t.Errorf("JWTSecret should be empty when not set, got %q", cfg.JWTSecret)
	}
}

func TestValidJWTSecretIsPreserved(t *testing.T) {
	// Valid JWT_SECRET should be preserved
	os.Setenv("JWT_SECRET", "my-super-secret-key-12345")
	defer os.Unsetenv("JWT_SECRET")

	cfg := Load()

	if cfg.JWTSecret != "my-super-secret-key-12345" {
		t.Errorf("JWTSecret should be preserved, got %q", cfg.JWTSecret)
	}
}

func TestValidAllowedOriginsIsPreserved(t *testing.T) {
	// Valid AllowedOrigins should be preserved
	os.Setenv("ALLOWED_ORIGINS", "https://example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	cfg := Load()

	if cfg.AllowedOrigins != "https://example.com" {
		t.Errorf("AllowedOrigins should be preserved, got %q", cfg.AllowedOrigins)
	}
}