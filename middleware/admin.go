package middleware

import (
	"auth-gateway/config"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AdminAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
			c.Abort()
			return
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		tokenType := strings.ToLower(parts[0])
		token := parts[1]

		if token == cfg.AdminPassword {
			c.Next()
			return
		}

		if tokenType == "bearer" {
			claims, err := ValidateJWT(token, cfg.JWTSecret)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
				c.Abort()
				return
			}
			c.Set("admin_user", claims.Subject)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
			c.Abort()
			return
		}

		c.Next()
	}
}
