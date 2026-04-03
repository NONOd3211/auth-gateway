package handler

import (
	"auth-gateway/config"
	"auth-gateway/database"
	"auth-gateway/models"
	"auth-gateway/proxy"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func ProxyRequest(cfg *config.Config) gin.HandlerFunc {
	client := proxy.NewClient(cfg.UpstreamURL, cfg.UpstreamAPIKey)

	return func(c *gin.Context) {
		tokenIDValue, exists := c.Get("token_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Convert tokenID to string early to avoid type issues
		tokenID, ok := tokenIDValue.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token id type"})
			return
		}

		// Check token limits
		var token models.Token
		if err := database.DB.First(&token, "id = ?", tokenID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token not found"})
			return
		}

		// Check hourly limit (5 hours from creation)
		if token.IsExpired() {
			c.JSON(http.StatusForbidden, gin.H{"error": "token has expired (hourly limit)"})
			return
		}

		// Check weekly limit
		if !token.IsWithinWeeklyLimit() {
			c.JSON(http.StatusForbidden, gin.H{"error": "token has exceeded weekly limit"})
			return
		}

		// Update hourly/weekly counters
		token.CheckAndUpdateLimits()
		database.DB.Save(&token)

		resp, err := client.Forward(c.Request)
		if err != nil {
			recordUsage(tokenID, c.Request.URL.Path, "", 0, 0, false, err.Error())
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream error: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			recordUsage(tokenID, c.Request.URL.Path, "", 0, 0, false, "failed to read response body: "+err.Error())
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read response body"})
			return
		}
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)

		// Update usage count synchronously to avoid race conditions
		database.DB.Exec("UPDATE tokens SET used_requests = used_requests + 1 WHERE id = ?", tokenID)

		// Update hourly and weekly counters
		if token.HourlyLimit {
			database.DB.Exec("UPDATE tokens SET hourly_used = hourly_used + 1 WHERE id = ?", tokenID)
		}
		if token.WeeklyLimit {
			database.DB.Exec("UPDATE tokens SET weekly_used = weekly_used + 1 WHERE id = ?", tokenID)
		}

		recordUsage(tokenID, c.Request.URL.Path, "", 0, 0, true, "")
	}
}

func recordUsage(tokenID, path, model string, inputTokens, outputTokens int, success bool, errMsg string) {
	// Generate unique ID with random suffix to avoid collisions
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback: use timestamp with math random
		randomBytes = []byte(time.Now().Format("150405"))
	}
	// Ensure tokenID is at least 8 characters for ID generation
	tokenIDPrefix := tokenID
	if len(tokenIDPrefix) < 8 {
		tokenIDPrefix = tokenIDPrefix + "xxxxxxxx"[:8-len(tokenIDPrefix)]
	}
	record := models.UsageRecord{
		ID:           time.Now().Format("20060102150405") + "-" + tokenIDPrefix[:8] + "-" + hex.EncodeToString(randomBytes),
		TokenID:      tokenID,
		Timestamp:    time.Now(),
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Success:      success,
		ErrorMessage: errMsg,
		RequestPath:  path,
	}
	database.DB.Create(&record)
}
