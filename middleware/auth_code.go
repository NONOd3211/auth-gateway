package middleware

import (
	"auth-gateway/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminCodeAuth checks for ?code= query parameter to determine admin access
func AdminCodeAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		if code != "" && code == cfg.AdminCode {
			c.Set("is_admin", true)
		} else {
			c.Set("is_admin", false)
		}
		c.Next()
	}
}

// RequireAdmin middleware ensures the user has admin access
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("is_admin")
		if !exists || !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}