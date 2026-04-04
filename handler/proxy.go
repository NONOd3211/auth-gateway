package handler

import (
	"auth-gateway/config"
	"auth-gateway/database"
	"auth-gateway/models"
	"auth-gateway/providers"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ProviderManager instance set by main.go
var providerManager *providers.ProviderManager

// SetProviderManager configures the global ProviderManager instance
func SetProviderManager(pm *providers.ProviderManager) {
	providerManager = pm
}

func ProxyRequest(cfg *config.Config) gin.HandlerFunc {
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

		// Read request body to determine model
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			return
		}

		// Extract model name from request body
		model := extractModel(string(bodyBytes))

		// Get the appropriate provider based on model
		provider := providerManager.GetProviderForModel(model)
		if provider == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no provider available for model: " + model})
			return
		}

		// Restore body for the provider to read
		c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

		// Get API key for this token from ProviderManager
		apiKey, err := providerManager.GetAPIKeyForToken(tokenID)
		if err != nil {
			recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, false, "failed to get API key: "+err.Error())
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no available API keys"})
			return
		}

		// Forward request using the selected provider
		resp, err := provider.Execute(c.Request, apiKey.Key)
		if err != nil {
			recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, false, err.Error())
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream error: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		// Check for quota errors and mark key as failed if needed
		if provider.IsQuotaError(resp) {
			providerManager.MarkKeyFailed(apiKey.ID)
		}

		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, false, "failed to read response body: "+err.Error())
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

		recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, true, "")
	}
}

// extractModel extracts the model name from the request body
func extractModel(body string) string {
	// Try to parse as JSON and extract model field
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal([]byte(body), &req); err == nil && req.Model != "" {
		return req.Model
	}
	return ""
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
