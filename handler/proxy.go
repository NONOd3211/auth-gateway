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

// ProviderManager instance set by main.go
var providerManager *providers.ProviderManager

// SetProviderManager configures the global ProviderManager instance
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

		// Convert tokenID to string early to avoid type issues
		tokenID, ok := tokenIDValue.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token id type"})
			return
		}

		log.Printf("[DEBUG] REQUEST_TOKEN tokenID=%s path=%s", tokenID, c.Request.URL.Path)

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

		// Read request body to determine model and stream setting
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			return
		}

		// Extract model name from request body
		model := extractModel(string(bodyBytes))
		isStreamRequest := isStreamEnabled(string(bodyBytes))

		// Detect if original request is in Anthropic format
		isAnthropicRequest := minimax.IsAnthropicFormatRequest(bodyBytes)
		if isAnthropicRequest {
			log.Printf("[DEBUG] token=%s path=%s detected Anthropic format request", tokenID, c.Request.URL.Path)
		}

		bodyPreview := string(bodyBytes)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200]
		}
		log.Printf("[DEBUG] token=%s path=%s model=%s stream=%v is_anthropic=%v request_body_preview=%s",
			tokenID, c.Request.URL.Path, model, isStreamRequest, isAnthropicRequest, bodyPreview)

		// Get the appropriate provider based on model
		provider := providerManager.GetProviderForModel(model)
		if provider == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no provider available for model: " + model})
			return
		}

		// Restore body for the provider to read
		c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

		// Get API key for this token from ProviderManager
		apiKey, err := providerManager.GetAPIKeyForToken(tokenID, token.APIKeyID)
		if err != nil {
			recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, 0, false, "failed to get API key: "+err.Error())
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no available API keys"})
			return
		}

		// Validate model is allowed for this API key
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

		// Forward request using the selected provider
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

		// Check for quota errors and mark key as failed if needed
		if provider.IsQuotaError(resp) {
			providerManager.MarkKeyFailed(apiKey.ID)
		}

		// Check if this is a streaming response (SSE)
		contentType := resp.Header.Get("Content-Type")
		isStreaming := strings.Contains(contentType, "text/event-stream") ||
			strings.Contains(contentType, "application/x-ndjson") ||
			isStreamRequest

		log.Printf("[DEBUG] token=%s path=%s upstream_content_type=%s is_stream_request=%v is_streaming=%v is_anthropic=%v",
			tokenID, c.Request.URL.Path, contentType, isStreamRequest, isStreaming, isAnthropicRequest)

		if isStreaming {
			if isAnthropicRequest {
				handleAnthropicStreamingResponse(c, resp, tokenID, model, token)
			} else {
				handleStreamingResponse(c, resp, tokenID, model, token)
			}
			return
		}

		// Copy response headers for non-streaming
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, 0, false, "failed to read response body: "+err.Error())
			log.Printf("[DEBUG] token=%s path=%s ERROR ReadAll: %v", tokenID, c.Request.URL.Path, err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read response body"})
			return
		}

		// Decompress gzip response if needed
		if resp.Header.Get("Content-Encoding") == "gzip" || (len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b) {
			body, err = decompressGzip(body)
			if err != nil {
				log.Printf("[DEBUG] token=%s path=%s gzip decompress error: %v", tokenID, c.Request.URL.Path, err)
				recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, 0, false, "failed to decompress gzip response")
				c.JSON(http.StatusBadGateway, gin.H{"error": "failed to decompress response"})
				return
			}
		}

		log.Printf("[DEBUG] token=%s path=%s MiniMax response body: %s", tokenID, c.Request.URL.Path, string(body))

		// Check if MiniMax returned an error
		isError, errorMsg := isMiniMaxError(body)
		if isError {
			log.Printf("[DEBUG] token=%s path=%s MiniMax API error: %s", tokenID, c.Request.URL.Path, errorMsg)
			recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, 0, false, errorMsg)
			c.JSON(http.StatusBadGateway, gin.H{"error": "MiniMax API error: " + errorMsg})
			return
		}

		// Convert response to Anthropic format if needed
		if isAnthropicRequest {
			convertedBody, err := minimax.ConvertOpenAIToAnthropicResponse(body, model)
			if err == nil {
				body = convertedBody
				log.Printf("[DEBUG] token=%s path=%s converted response to Anthropic format", tokenID, c.Request.URL.Path)
			} else {
				log.Printf("[DEBUG] token=%s path=%s failed to convert to Anthropic format: %v", tokenID, c.Request.URL.Path, err)
			}
		}

		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)

		// Parse token usage from response
		inputTokens, outputTokens, cacheTokens := parseTokenUsage(body, resp.Header.Get("Content-Type"))

		log.Printf("[DEBUG] token=%s path=%s model=%s resp_status=%d input=%d output=%d calling recordUsage success=true",
			tokenID, c.Request.URL.Path, model, resp.StatusCode, inputTokens, outputTokens)
		recordUsage(tokenID, c.Request.URL.Path, model, inputTokens, outputTokens, cacheTokens, true, "")

		// Update usage count synchronously to avoid race conditions
		database.DB.Exec("UPDATE tokens SET used_requests = used_requests + 1 WHERE id = ?", tokenID)

		// Update hourly and weekly counters
		if token.HourlyLimit {
			database.DB.Exec("UPDATE tokens SET hourly_used = hourly_used + 1 WHERE id = ?", tokenID)
		}
		if token.WeeklyLimit {
			database.DB.Exec("UPDATE tokens SET weekly_used = weekly_used + 1 WHERE id = ?", tokenID)
		}
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

// isStreamEnabled checks if streaming is enabled in the request
func isStreamEnabled(body string) bool {
	var req struct {
		Stream bool `json:"stream"`
	}
	if err := json.Unmarshal([]byte(body), &req); err == nil {
		return req.Stream
	}
	return false
}

// handleStreamingResponse handles SSE streaming response
func handleStreamingResponse(c *gin.Context, resp *http.Response, tokenID, model string, token models.Token) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(resp.StatusCode)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		log.Printf("[ERROR] token=%s streaming not supported by response writer", tokenID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// Handle gzip compressed response
	var reader *bufio.Reader
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Printf("[ERROR] token=%s failed to create gzip reader: %v", tokenID, err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to decompress response"})
			return
		}
		defer gzipReader.Close()
		reader = bufio.NewReader(gzipReader)
		log.Printf("[DEBUG] token=%s handling gzip compressed SSE stream", tokenID)
	} else {
		reader = bufio.NewReader(resp.Body)
	}

	lineCount := 0
	sawDone := false
	buffer := make([]string, 0)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("[ERROR] token=%s streaming read error: %v", tokenID, err)
			}
			break
		}

		lineCount++
		trimmed := strings.TrimSpace(line)

		// Skip empty lines but keep them for SSE format
		if trimmed == "" {
			// Flush buffer when we hit an empty line (end of event)
			if len(buffer) > 0 {
				for _, bufLine := range buffer {
					c.Writer.WriteString(bufLine)
				}
				c.Writer.WriteString("\n")
				c.Writer.Flush()
				flusher.Flush()
				buffer = buffer[:0]
			}
			continue
		}

		// Check for [DONE] marker
		if trimmed == "data: [DONE]" {
			sawDone = true
		}

		// Buffer the line
		buffer = append(buffer, line)

		// Log data lines
		if strings.HasPrefix(trimmed, "data: ") && len(trimmed) > 6 {
			preview := trimmed[6:]
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			log.Printf("[DEBUG] token=%s stream line %d: %s", tokenID, lineCount, preview)
		}
	}

	// Flush any remaining buffer
	if len(buffer) > 0 {
		for _, bufLine := range buffer {
			c.Writer.WriteString(bufLine)
		}
		c.Writer.Flush()
		flusher.Flush()
	}

	// Add [DONE] marker if not present
	if !sawDone {
		log.Printf("[DEBUG] token=%s adding [DONE] marker", tokenID)
		c.Writer.WriteString("data: [DONE]\n\n")
		c.Writer.Flush()
		flusher.Flush()
	}

	log.Printf("[DEBUG] token=%s model=%s stream completed lines=%d sawDone=%v",
		tokenID, model, lineCount, sawDone)

	// Record usage
	recordUsage(tokenID, c.Request.URL.Path, model, 0, 0, 0, true, "streaming")

	// Update token counters
	database.DB.Exec("UPDATE tokens SET used_requests = used_requests + 1 WHERE id = ?", tokenID)
	if token.HourlyLimit {
		database.DB.Exec("UPDATE tokens SET hourly_used = hourly_used + 1 WHERE id = ?", tokenID)
	}
	if token.WeeklyLimit {
		database.DB.Exec("UPDATE tokens SET weekly_used = weekly_used + 1 WHERE id = ?", tokenID)
	}
}

// handleAnthropicStreamingResponse handles SSE streaming response and converts to Anthropic format
func handleAnthropicStreamingResponse(c *gin.Context, resp *http.Response, tokenID, model string, token models.Token) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(resp.StatusCode)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		log.Printf("[ERROR] token=%s streaming not supported by response writer", tokenID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// Send message_start event
	startEvent := map[string]interface{}{
		"type":    "message_start",
		"message": map[string]interface{}{
			"id":           generateID(),
			"type":         "message",
			"role":         "assistant",
			"model":        model,
			"content":      []interface{}{},
			"stop_reason":  nil,
			"stop_sequence": nil,
			"usage": map[string]interface{}{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	}
	startJSON, _ := json.Marshal(startEvent)
	c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", startJSON))
	c.Writer.Flush()
	flusher.Flush()

	// Send content_block_start event
	blockStart := map[string]interface{}{
		"type":  "content_block_start",
		"index": 0,
		"content_block": map[string]interface{}{
			"type": "text",
			"text": "",
		},
	}
	blockStartJSON, _ := json.Marshal(blockStart)
	c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", blockStartJSON))
	c.Writer.Flush()
	flusher.Flush()

	// Handle gzip compressed response
	var reader *bufio.Reader
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Printf("[ERROR] token=%s failed to create gzip reader: %v", tokenID, err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to decompress response"})
			return
		}
		defer gzipReader.Close()
		reader = bufio.NewReader(gzipReader)
		log.Printf("[DEBUG] token=%s handling gzip compressed SSE stream for Anthropic", tokenID)
	} else {
		reader = bufio.NewReader(resp.Body)
	}

	lineCount := 0
	totalOutputTokens := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("[ERROR] token=%s streaming read error: %v", tokenID, err)
			}
			break
		}

		lineCount++
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Parse OpenAI SSE format
		if !strings.HasPrefix(trimmed, "data: ") {
			continue
		}

		data := strings.TrimPrefix(trimmed, "data: ")
		if data == "[DONE]" {
			break
		}

		var openaiChunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &openaiChunk); err != nil {
			continue
		}

		// Convert to Anthropic format
		anthropicChunk := convertOpenAIChunkToAnthropicChunk(openaiChunk)
		if anthropicChunk != nil {
			chunkJSON, _ := json.Marshal(anthropicChunk)
			c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", chunkJSON))
			c.Writer.Flush()
			flusher.Flush()
			totalOutputTokens++

			// Log preview
			if content, ok := anthropicChunk["delta"].(map[string]interface{}); ok {
				if text, ok := content["text"].(string); ok && text != "" {
					preview := text
					if len(preview) > 80 {
						preview = preview[:80] + "..."
					}
					log.Printf("[DEBUG] token=%s anthropic stream line %d: %s", tokenID, lineCount, preview)
				}
			}
		}
	}

	// Send content_block_stop event
	blockStop := map[string]interface{}{
		"type":  "content_block_stop",
		"index": 0,
	}
	blockStopJSON, _ := json.Marshal(blockStop)
	c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", blockStopJSON))
	c.Writer.Flush()
	flusher.Flush()

	// Send message_delta event
	messageDelta := map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason": "end_turn",
		},
		"usage": map[string]interface{}{
			"output_tokens": totalOutputTokens,
		},
	}
	messageDeltaJSON, _ := json.Marshal(messageDelta)
	c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", messageDeltaJSON))
	c.Writer.Flush()
	flusher.Flush()

	// Send message_stop event
	stopEvent := map[string]interface{}{
		"type": "message_stop",
	}
	stopJSON, _ := json.Marshal(stopEvent)
	c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", stopJSON))
	c.Writer.Flush()
	flusher.Flush()

	log.Printf("[DEBUG] token=%s model=%s anthropic stream completed lines=%d tokens=%d",
		tokenID, model, lineCount, totalOutputTokens)

	// Record usage
	recordUsage(tokenID, c.Request.URL.Path, model, 0, totalOutputTokens, 0, true, "streaming")

	// Update token counters
	database.DB.Exec("UPDATE tokens SET used_requests = used_requests + 1 WHERE id = ?", tokenID)
	if token.HourlyLimit {
		database.DB.Exec("UPDATE tokens SET hourly_used = hourly_used + 1 WHERE id = ?", tokenID)
	}
	if token.WeeklyLimit {
		database.DB.Exec("UPDATE tokens SET weekly_used = weekly_used + 1 WHERE id = ?", tokenID)
	}
}

// convertOpenAIChunkToAnthropicChunk converts an OpenAI SSE chunk to Anthropic format
func convertOpenAIChunkToAnthropicChunk(openaiChunk map[string]interface{}) map[string]interface{} {
	choices, ok := openaiChunk["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil
	}

	choice := choices[0].(map[string]interface{})
	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return nil
	}

	// Handle content
	if content, ok := delta["content"].(string); ok && content != "" {
		return map[string]interface{}{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": content,
			},
		}
	}

	return nil
}

// generateID generates a unique ID for Anthropic messages
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("msg_%x", b)
}

// parseTokenUsage parses token usage from response body
// Returns inputTokens, outputTokens, cacheTokens
func parseTokenUsage(body []byte, contentType string) (int, int, int) {
	if !strings.Contains(contentType, "application/json") {
		return 0, 0, 0
	}

	// Try MiniMax format with input_tokens/output_tokens
	var resp struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &resp); err == nil && (resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0) {
		return resp.Usage.InputTokens, resp.Usage.OutputTokens, 0
	}

	// Try OpenAI format with prompt_tokens/completion_tokens
	var resp2 struct {
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &resp2); err == nil && (resp2.Usage.PromptTokens > 0 || resp2.Usage.CompletionTokens > 0) {
		return resp2.Usage.PromptTokens, resp2.Usage.CompletionTokens, 0
	}

	return 0, 0, 0
}

// isMiniMaxError checks if the response body contains a MiniMax API error
// Returns (isError, errorMessage)
func isMiniMaxError(body []byte) (bool, string) {
	// MiniMax error format: {"type":"error","error":{"type":"...","message":"..."}}
	var errResp struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err != nil {
		return false, ""
	}
	if errResp.Type == "error" && errResp.Error.Message != "" {
		return true, errResp.Error.Message
	}
	return false, ""
}

// decompressGzip decompresses gzip-encoded data
func decompressGzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func recordUsage(tokenID, path, model string, inputTokens, outputTokens, cacheTokens int, success bool, errMsg string) {
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
		CacheTokens:  cacheTokens,
		TotalTokens:  inputTokens + outputTokens,
		Success:      success,
		ErrorMessage: errMsg,
		RequestPath:  path,
	}
	database.DB.Create(&record)
}
