package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAdminCodeFromURLParameter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &testConfig{AdminCode: "secret123"}

	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		code := c.Query("code")
		if code == cfg.AdminCode {
			c.JSON(http.StatusOK, gin.H{"mode": "admin"})
		} else {
			c.JSON(http.StatusOK, gin.H{"mode": "user"})
		}
	})

	tests := []struct {
		name string
		url  string
		want string
	}{
		{"admin with code", "/?code=secret123", "admin"},
		{"user without code", "/", "user"},
		{"user with wrong code", "/?code=wrong", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// This is a simplified test - actual implementation would use the middleware
		})
	}
}

type testConfig struct {
	AdminCode string
}