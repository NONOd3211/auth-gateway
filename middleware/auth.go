package middleware

import (
	"auth-gateway/database"
	"auth-gateway/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func TokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c.GetHeader("Authorization"))
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		var tokenRecord models.Token
		if err := database.DB.Where("token = ?", token).First(&tokenRecord).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		if !tokenRecord.Enabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "token disabled"})
			c.Abort()
			return
		}

		if tokenRecord.ExpiresAt != nil && time.Now().After(*tokenRecord.ExpiresAt) {
			c.JSON(http.StatusForbidden, gin.H{"error": "token expired"})
			c.Abort()
			return
		}

		if tokenRecord.MaxRequests > 0 && tokenRecord.UsedRequests >= tokenRecord.MaxRequests {
			c.JSON(http.StatusForbidden, gin.H{"error": "request limit exceeded"})
			c.Abort()
			return
		}

		c.Set("token_id", tokenRecord.ID)
		c.Set("token_record", &tokenRecord)
		c.Next()
	}
}

func extractToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
