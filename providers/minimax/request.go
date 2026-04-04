package minimax

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type ChatRequest struct {
	Model    string                   `json:"model"`
	Messages []map[string]interface{} `json:"messages"`
	Stream   bool                     `json:"stream,omitempty"`
}

func BuildRequest(req *http.Request, apiKey string, upstreamURL string) (*http.Request, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body.Close()

	// DEBUG: log the request details
	if len(apiKey) > 10 {
		log.Printf("[MiniMax] API Key: %s...", apiKey[:10])
	} else {
		log.Printf("[MiniMax] API Key: %s", apiKey)
	}
	log.Printf("[MiniMax] Body: %s", string(body))

	// Detect format and convert to OpenAI/MiniMax format if needed
	convertedBody, isConverted := detectAndConvertFormat(body)
	if isConverted {
		log.Printf("[MiniMax] Converted format to OpenAI format")
		body = convertedBody
	} else {
		log.Printf("[MiniMax] No format conversion needed")
	}

	// Convert path from OpenAI format to MiniMax format
	targetPath := convertPath(req.URL.Path)
	targetURL := upstreamURL + targetPath

	log.Printf("[MiniMax] Target URL: %s", targetURL)
	log.Printf("[MiniMax] Original URL path: %s", req.URL.Path)

	proxyReq, err := http.NewRequest(req.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+apiKey)

	// Copy other headers except Host and Authorization (we set our own)
	for key, values := range req.Header {
		if key == "Host" || key == "Content-Length" || key == "Authorization" {
			continue
		}
		proxyReq.Header[key] = values
	}

	return proxyReq, nil
}

// detectAndConvertFormat detects the format and converts to OpenAI/MiniMax format
// Returns converted body and true if conversion was performed
func detectAndConvertFormat(body []byte) ([]byte, bool) {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return body, false
	}

	content, hasContent := req["content"]
	if !hasContent {
		// No content field, already OpenAI format or other
		return body, false
	}

	// Case 1: OpenAI format - content is string
	if contentStr, ok := content.(string); ok && contentStr != "" {
		log.Printf("[MiniMax] Content is string format, no conversion needed")
		return body, false
	}

	// Case 2: Anthropic format - content is array of blocks
	if contentArr, ok := content.([]interface{}); ok && len(contentArr) > 0 {
		// Check if this is Anthropic format (blocks have "type" field)
		if isAnthropicFormat(contentArr) {
			return convertAnthropicToOpenAIMessages(body, req, contentArr)
		}
	}

	// Case 3: content is array of message objects (some providers use this)
	if contentArr, ok := content.([]interface{}); ok && len(contentArr) > 0 {
		if isMessagesArray(contentArr) {
			return convertMessagesArrayToOpenAI(body, req, contentArr)
		}
	}

	return body, false
}

// isAnthropicFormat checks if content blocks are Anthropic format
func isAnthropicFormat(contentArr []interface{}) bool {
	for _, block := range contentArr {
		if blockMap, ok := block.(map[string]interface{}); ok {
			if _, hasType := blockMap["type"]; hasType {
				return true
			}
		}
	}
	return false
}

// isMessagesArray checks if content is an array of message objects
func isMessagesArray(contentArr []interface{}) bool {
	for _, msg := range contentArr {
		if msgMap, ok := msg.(map[string]interface{}); ok {
			if _, hasRole := msgMap["role"]; hasRole {
				return true
			}
		}
	}
	return false
}

// convertAnthropicToOpenAIMessages converts Anthropic format to OpenAI messages format
func convertAnthropicToOpenAIMessages(body []byte, req map[string]interface{}, contentArr []interface{}) ([]byte, bool) {
	messages := []map[string]interface{}{}

	// Extract system prompt
	if system, ok := req["system"]; ok {
		systemText := extractTextFromContent(system)
		if systemText != "" {
			messages = append(messages, map[string]interface{}{
				"role":    "system",
				"content": systemText,
			})
		}
	}

	// Extract text from content blocks
	var textContent strings.Builder
	for _, block := range contentArr {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		if blockType, ok := blockMap["type"].(string); ok && blockType == "text" {
			if text, ok := blockMap["text"].(string); ok {
				textContent.WriteString(text)
			}
		}
	}

	if textContent.Len() > 0 {
		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": textContent.String(),
		})
	}

	// Build converted request
	model := ""
	if m, ok := req["model"].(string); ok {
		model = m
	}

	converted := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	// Copy other fields that might be needed (like stream)
	if stream, ok := req["stream"]; ok {
		converted["stream"] = stream
	}

	result, err := json.Marshal(converted)
	if err != nil {
		return body, false
	}

	return result, true
}

// convertMessagesArrayToOpenAI converts array of messages to OpenAI format
func convertMessagesArrayToOpenAI(body []byte, req map[string]interface{}, contentArr []interface{}) ([]byte, bool) {
	messages := []map[string]interface{}{}

	for _, msg := range contentArr {
		if msgMap, ok := msg.(map[string]interface{}); ok {
			role, _ := msgMap["role"].(string)
			msgContent := msgMap["content"]

			// Handle content that might be array (Anthropic style inside messages)
			contentStr := extractTextFromContent(msgContent)

			messages = append(messages, map[string]interface{}{
				"role":    role,
				"content": contentStr,
			})
		}
	}

	model := ""
	if m, ok := req["model"].(string); ok {
		model = m
	}

	converted := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	if stream, ok := req["stream"]; ok {
		converted["stream"] = stream
	}

	result, err := json.Marshal(converted)
	if err != nil {
		return body, false
	}

	return result, true
}

// extractTextFromContent extracts text from various content formats
func extractTextFromContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	case []interface{}:
		var sb strings.Builder
		for _, item := range v {
			if str, ok := item.(string); ok {
				sb.WriteString(str)
			} else if itemMap, ok := item.(map[string]interface{}); ok {
				if text, ok := itemMap["text"].(string); ok {
					sb.WriteString(text)
				}
			}
		}
		return sb.String()
	}
	return ""
}

// convertPath converts OpenAI-style paths to MiniMax-specific paths
func convertPath(openaiPath string) string {
	// /v1/chat/completions -> /v1/text/chatcompletion_v2
	if openaiPath == "/v1/chat/completions" {
		return "/v1/text/chatcompletion_v2"
	}
	// /v1/messages -> /v1/text/chatcompletion_v2 (Claude Code with ANTHROPIC_BASE_URL)
	if openaiPath == "/v1/messages" {
		return "/v1/text/chatcompletion_v2"
	}
	// Other paths pass through as-is
	return openaiPath
}