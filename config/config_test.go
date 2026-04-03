package config

import (
	"os"
	"testing"
)

func TestAllowedOriginsWildcardReturnsEmpty(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "*")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	cfg := Load()

	if cfg.AllowedOrigins != "" {
		t.Errorf("AllowedOrigins should be empty when set to '*', got %q", cfg.AllowedOrigins)
	}
}

func TestJWTSecretFromEnv(t *testing.T) {
	os.Setenv("JWT_SECRET", "my-super-secret-key-12345")
	defer os.Unsetenv("JWT_SECRET")

	cfg := Load()

	if cfg.JWTSecret != "my-super-secret-key-12345" {
		t.Errorf("JWTSecret should use env value, got %q", cfg.JWTSecret)
	}
}

func TestJWTSecretGeneratedWhenEmpty(t *testing.T) {
	os.Setenv("JWT_SECRET", "")
	defer os.Unsetenv("JWT_SECRET")

	cfg := Load()

	// Should generate a random secret
	if cfg.JWTSecret == "" {
		t.Error("JWTSecret should be generated when env is empty")
	}

	// Should be a hex string (32 bytes = 64 hex chars)
	if len(cfg.JWTSecret) != 64 {
		t.Errorf("JWTSecret should be 64 hex chars (32 bytes), got %d chars", len(cfg.JWTSecret))
	}
}

func TestValidAllowedOriginsIsPreserved(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "https://example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	cfg := Load()

	if cfg.AllowedOrigins != "https://example.com" {
		t.Errorf("AllowedOrigins should be preserved, got %q", cfg.AllowedOrigins)
	}
}