package config

import (
	"os"
	"testing"
)

func TestAdminCodeFromEnv(t *testing.T) {
	os.Setenv("ADMIN_CODE", "my-secret-code")
	defer os.Unsetenv("ADMIN_CODE")

	cfg := Load()

	if cfg.AdminCode != "my-secret-code" {
		t.Errorf("AdminCode should be 'my-secret-code', got %q", cfg.AdminCode)
	}
}

func TestAdminCodeEmptyGeneratesRandom(t *testing.T) {
	os.Setenv("ADMIN_CODE", "")
	defer os.Unsetenv("ADMIN_CODE")

	cfg := Load()

	// Should generate a random code if empty
	if cfg.AdminCode == "" {
		t.Error("AdminCode should be generated when env is empty")
	}
}

func TestUpstreamAPIKeyFromEnv(t *testing.T) {
	os.Setenv("UPSTREAM_API_KEY", "sk-upstream-key-123")
	defer os.Unsetenv("UPSTREAM_API_KEY")

	cfg := Load()

	if cfg.UpstreamAPIKey != "sk-upstream-key-123" {
		t.Errorf("UpstreamAPIKey should be 'sk-upstream-key-123', got %q", cfg.UpstreamAPIKey)
	}
}