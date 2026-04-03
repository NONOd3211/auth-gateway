package handler

import (
	"auth-gateway/config"
	"auth-gateway/database"
	"auth-gateway/models"
	"auth-gateway/proxy"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func ProxyRequest(cfg *config.Config) gin.HandlerFunc {
	client := proxy.NewClient(cfg.UpstreamURL)

	return func(c *gin.Context) {
		tokenID, exists := c.Get("token_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		resp, err := client.Forward(c.Request)
		if err != nil {
			recordUsage(tokenID.(string), c.Request.URL.Path, "", 0, 0, false, err.Error())
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
			recordUsage(tokenID.(string), c.Request.URL.Path, "", 0, 0, false, "failed to read response body: "+err.Error())
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read response body"})
			return
		}
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)

		// Update usage count synchronously to avoid race conditions
		database.DB.Model(&models.Token{}).Where("id = ?", tokenID).UpdateColumn("used_requests", database.DB.Raw("used_requests + 1"))

		recordUsage(tokenID.(string), c.Request.URL.Path, "", 0, 0, true, "")
	}
}

func recordUsage(tokenID, path, model string, inputTokens, outputTokens int, success bool, errMsg string) {
	record := models.UsageRecord{
		ID:           time.Now().Format("20060102150405") + "-" + tokenID[:8],
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
