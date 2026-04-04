package handler

import (
	"auth-gateway/config"
	"auth-gateway/database"
	"auth-gateway/models"
	"auth-gateway/providers"
	"auth-gateway/providers/minimax"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var providerManager *providers.ProviderManager

func SetProviderManager(pm *providers.ProviderManager) {
	providerManager = pm
}

func ProxyRequest(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[DEBUG] REQUEST_START method=%s path=%s remote=%s",
			c.Request.Method, c.Request.URL.Path, c.ClientIP())

		tokenIDValue, exists := c.Get("token_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		tokenID, ok := tokenIDValue.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token id type"})
			return
		}

		log.Printf("[DEBUG] REQUEST_TOKEN tokenID=%s path=%s", tokenID, c.Request.URL.Path)

		var token models.Token
		if err := database.DB.First(&token, "id = ?", tokenID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token not found"})
			return
		}

		if token.IsExpired() {
			c.JSON(http.StatusForbidden, gin.H{"error": "token has expired (hourly limit)"})
			return
		}

		if !token.IsWithinWeeklyLimit() {
			c.JSON(http.StatusForbidden, gin.H{"error": "token has exceeded weekly limit"})
			return
		}

		token.CheckAndUpdateLimits()
		database.DB.Save(&token)

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			return
		}

		model := extractModel(string(bodyBytes))
		isStreamRequest := isStreamEnabled(string(bodyBytes))
		isAnthropicRequest := minimax.IsAnthropicFormatRequest(bodyBytes)

		if isAnthropicRequest {
			log.Printf("[DEBUG] token=%s path=%s detected Anthropic format request", tokenID, c.Request.URL.Path)
		}

		log.Printf("[DEBUG] token=%s path=%s model=%s stream=%v is_anthropic=%v",
			tokenID, c.Request.URL.Path, model, isStreamRequest, isAnthropicRequest)

		provider := providerManager.GetProviderForModel(model)
		if provider == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no provider available for model: " + model})
			return
		}

		c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

		apiKey, err := providerManager.GetAPIKeyForToken(tokenID, token.APIKeyID)
		if err != nil {
			recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, 0, false, "failed to get API key: "+err.Error())
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no available API keys"})
			return
		}

		if apiKey.AllowedModels != "" {
			allowedList := strings.Split(apiKey.AllowedModels, ",")
			allowed := false
			for _, m := range allowedList {
				if strings.TrimSpace(m) == model {
					allowed = true
					break
				}
			}
			if !allowed {
				recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, 0, false, "model not allowed for this API key")
				c.JSON(http.StatusForbidden, gin.H{"error": "model '" + model + "' is not allowed for this API key"})
				return
			}
		}

		log.Printf("[DEBUG] token=%s path=%s model=%s provider=%s calling Execute",
			tokenID, c.Request.URL.Path, model, provider.Name())
		resp, err := provider.Execute(c.Request, apiKey.Key)
		if err != nil {
			recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, 0, false, err.Error())
			log.Printf("[DEBUG] token=%s path=%s ERROR Execute: %v", tokenID, c.Request.URL.Path, err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream error: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		if provider.IsQuotaError(resp) {
			providerManager.MarkKeyFailed(apiKey.ID)
		}

		contentType := resp.Header.Get("Content-Type")
		isStreaming := strings.Contains(contentType, "text/event-stream") ||
			strings.Contains(contentType, "application/x-ndjson") ||
			isStreamRequest

		log.Printf("[DEBUG] token=%s path=%s content_type=%s is_streaming=%v is_anthropic=%v",
			tokenID, c.Request.URL.Path, contentType, isStreaming, isAnthropicRequest)

		if isStreaming {
			if isAnthropicRequest {
				handleAnthropicStream(c, resp, tokenID, model, token)
			} else {
				handleOpenAIStream(c, resp, tokenID, model, token)
			}
			return
		}

		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, 0, false, "failed to read response body: "+err.Error())
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read response body"})
			return
		}

		if resp.Header.Get("Content-Encoding") == "gzip" || (len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b) {
			body, err = decompressGzip(body)
			if err != nil {
				recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, 0, false, "failed to decompress gzip response")
				c.JSON(http.StatusBadGateway, gin.H{"error": "failed to decompress response"})
				return
			}
		}

		isError, errorMsg := isAPIError(body)
		if isError {
			recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, 0, false, errorMsg)
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream API error: " + errorMsg})
			return
		}

		if isAnthropicRequest {
			convertedBody, err := minimax.ConvertOpenAIToAnthropicResponse(body, model)
			if err == nil {
				body = convertedBody
				log.Printf("[DEBUG] token=%s path=%s converted response to Anthropic format", tokenID, c.Request.URL.Path)
			}
		}

		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)

		inputTokens, outputTokens, cacheTokens := parseTokenUsage(body, resp.Header.Get("Content-Type"))
		recordUsage(tokenID, c.Request.URL.Path, model, inputTokens, outputTokens, cacheTokens, true, "")

		database.DB.Exec("UPDATE tokens SET used_requests = used_requests + 1 WHERE id = ?", tokenID)
		if token.HourlyLimit {
			database.DB.Exec("UPDATE tokens SET hourly_used = hourly_used + 1 WHERE id = ?", tokenID)
		}
		if token.WeeklyLimit {
			database.DB.Exec("UPDATE tokens SET weekly_used = weekly_used + 1 WHERE id = ?", tokenID)
		}
	}
}

func extractModel(body string) string {
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal([]byte(body), &req); err == nil && req.Model != "" {
		return req.Model
	}
	return ""
}

func isStreamEnabled(body string) bool {
	var req struct {
		Stream bool `json:"stream"`
	}
	if err := json.Unmarshal([]byte(body), &req); err == nil {
		return req.Stream
	}
	return false
}

func handleOpenAIStream(c *gin.Context, resp *http.Response, tokenID, model string, token models.Token) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(resp.StatusCode)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	var reader *bufio.Reader
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to decompress response"})
			return
		}
		defer gzipReader.Close()
		reader = bufio.NewReader(gzipReader)
	} else {
		reader = bufio.NewReader(resp.Body)
	}

	flusher.Flush()

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("[ERROR] token=%s stream read error: %v", tokenID, err)
			}
			break
		}

		c.Writer.WriteString(line)
		c.Writer.Flush()
		flusher.Flush()
	}

	log.Printf("[DEBUG] token=%s model=%s OpenAI stream completed", tokenID, model)

	recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, 0, true, "streaming")
	database.DB.Exec("UPDATE tokens SET used_requests = used_requests + 1 WHERE id = ?", tokenID)
	if token.HourlyLimit {
		database.DB.Exec("UPDATE tokens SET hourly_used = hourly_used + 1 WHERE id = ?", tokenID)
	}
	if token.WeeklyLimit {
		database.DB.Exec("UPDATE tokens SET weekly_used = weekly_used + 1 WHERE id = ?", tokenID)
	}
}

func handleAnthropicStream(c *gin.Context, resp *http.Response, tokenID, model string, token models.Token) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(resp.StatusCode)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	messageID := generateID()

	messageStart := map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":            messageID,
			"type":          "message",
			"role":          "assistant",
			"model":         model,
			"content":       []interface{}{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]interface{}{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	}
	sendAnthropicEvent(c, flusher, "message_start", messageStart)

	var reader *bufio.Reader
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to decompress response"})
			return
		}
		defer gzipReader.Close()
		reader = bufio.NewReader(gzipReader)
	} else {
		reader = bufio.NewReader(resp.Body)
	}

	currentBlockIndex := -1
	currentBlockType := ""
	var inputTokens, outputTokens int

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("[ERROR] token=%s stream read error: %v", tokenID, err)
			}
			break
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !strings.HasPrefix(trimmed, "data: ") {
			continue
		}

		data := strings.TrimPrefix(trimmed, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if usage, ok := chunk["usage"].(map[string]interface{}); ok {
			if pt, ok := usage["prompt_tokens"].(float64); ok {
				inputTokens = int(pt)
			}
			if ot, ok := usage["completion_tokens"].(float64); ok {
				outputTokens = int(ot)
			}
		}

		choices, _ := chunk["choices"].([]interface{})
		if len(choices) == 0 {
			continue
		}

		choice, _ := choices[0].(map[string]interface{})
		delta, _ := choice["delta"].(map[string]interface{})
		finishReason, _ := choice["finish_reason"].(string)

		reasoningContent, _ := delta["reasoning_content"].(string)
		content, _ := delta["content"].(string)

		if reasoningContent != "" {
			if currentBlockType != "thinking" {
				if currentBlockIndex >= 0 {
					sendAnthropicEvent(c, flusher, "content_block_stop", map[string]interface{}{
						"type":  "content_block_stop",
						"index": currentBlockIndex,
					})
				}
				currentBlockIndex++
				currentBlockType = "thinking"
				sendAnthropicEvent(c, flusher, "content_block_start", map[string]interface{}{
					"type":  "content_block_start",
					"index": currentBlockIndex,
					"content_block": map[string]interface{}{
						"type":     "thinking",
						"thinking": "",
					},
				})
			}
			sendAnthropicEvent(c, flusher, "content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": currentBlockIndex,
				"delta": map[string]interface{}{
					"type":     "thinking_delta",
					"thinking": reasoningContent,
				},
			})
		}

		if content != "" {
			if currentBlockType != "text" {
				if currentBlockIndex >= 0 {
					sendAnthropicEvent(c, flusher, "content_block_stop", map[string]interface{}{
						"type":  "content_block_stop",
						"index": currentBlockIndex,
					})
				}
				currentBlockIndex++
				currentBlockType = "text"
				sendAnthropicEvent(c, flusher, "content_block_start", map[string]interface{}{
					"type":  "content_block_start",
					"index": currentBlockIndex,
					"content_block": map[string]interface{}{
						"type": "text",
						"text": "",
					},
				})
			}
			sendAnthropicEvent(c, flusher, "content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": currentBlockIndex,
				"delta": map[string]interface{}{
					"type": "text_delta",
					"text": content,
				},
			})
		}

		if finishReason != "" {
			if currentBlockIndex >= 0 {
				sendAnthropicEvent(c, flusher, "content_block_stop", map[string]interface{}{
					"type":  "content_block_stop",
					"index": currentBlockIndex,
				})
			}

			stopReason := "end_turn"
			if finishReason == "tool_calls" {
				stopReason = "tool_use"
			}

			sendAnthropicEvent(c, flusher, "message_delta", map[string]interface{}{
				"type": "message_delta",
				"delta": map[string]interface{}{
					"stop_reason": stopReason,
				},
				"usage": map[string]interface{}{
					"output_tokens": outputTokens,
				},
			})
		}
	}

	sendAnthropicEvent(c, flusher, "message_stop", map[string]interface{}{
		"type": "message_stop",
	})

	log.Printf("[DEBUG] token=%s model=%s Anthropic stream completed input=%d output=%d",
		tokenID, model, inputTokens, outputTokens)

	recordUsage(tokenID, c.Request.URL.Path, model, inputTokens, outputTokens, 0, true, "streaming")
	database.DB.Exec("UPDATE tokens SET used_requests = used_requests + 1 WHERE id = ?", tokenID)
	if token.HourlyLimit {
		database.DB.Exec("UPDATE tokens SET hourly_used = hourly_used + 1 WHERE id = ?", tokenID)
	}
	if token.WeeklyLimit {
		database.DB.Exec("UPDATE tokens SET weekly_used = weekly_used + 1 WHERE id = ?", tokenID)
	}
}

func sendAnthropicEvent(c *gin.Context, flusher http.Flusher, eventType string, data map[string]interface{}) {
	dataJSON, _ := json.Marshal(data)
	c.Writer.WriteString(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, dataJSON))
	c.Writer.Flush()
	flusher.Flush()
}

func generateID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return "msg_" + hex.EncodeToString(b)
}

func recordUsage(tokenID, path, model string, inputTokens, outputTokens, cacheTokens int, success bool, errorMsg string) {
	now := time.Now()
	today := now.Format("2006-01-02")
	hour := now.Format("2006-01-02 15:00:00")

	database.DB.Exec(`
		INSERT INTO usage (token_id, date, hour, model, input_tokens, output_tokens, cache_tokens, request_count, success_count, error_count, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)
		ON CONFLICT(token_id, date, hour, model) DO UPDATE SET
			input_tokens = input_tokens + ?,
			output_tokens = output_tokens + ?,
			cache_tokens = cache_tokens + ?,
			request_count = request_count + 1,
			success_count = success_count + ?,
			error_count = error_count + ?,
			error_message = ?
	`, tokenID, today, hour, model, inputTokens, outputTokens, cacheTokens, boolToInt(success), boolToInt(!success), errorMsg,
		inputTokens, outputTokens, cacheTokens, boolToInt(success), boolToInt(!success), errorMsg)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func isAPIError(body []byte) (bool, string) {
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return false, ""
	}

	if errObj, ok := resp["error"].(map[string]interface{}); ok {
		if msg, ok := errObj["message"].(string); ok {
			return true, msg
		}
	}

	if errMsg, ok := resp["error"].(string); ok {
		return true, errMsg
	}

	return false, ""
}

func parseTokenUsage(body []byte, contentType string) (int, int, int) {
	if !strings.Contains(contentType, "application/json") {
		return 0, 0, 0
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, 0
	}

	usage, ok := resp["usage"].(map[string]interface{})
	if !ok {
		return 0, 0, 0
	}

	inputTokens := 0
	outputTokens := 0
	cacheTokens := 0

	if pt, ok := usage["prompt_tokens"].(float64); ok {
		inputTokens = int(pt)
	}
	if ot, ok := usage["completion_tokens"].(float64); ok {
		outputTokens = int(ot)
	}
	if ct, ok := usage["cache_read_input_tokens"].(float64); ok {
		cacheTokens = int(ct)
	}

	return inputTokens, outputTokens, cacheTokens
}
