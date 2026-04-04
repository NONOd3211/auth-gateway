package minimax

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	// Detect format and convert to OpenAI/MiniMax format if needed
	convertedBody, isConverted := detectAndConvertFormat(body)
	if isConverted {
		body = convertedBody
	}

	// Convert path from OpenAI format to MiniMax format
	targetPath := convertPath(req.URL.Path)
	targetURL := upstreamURL + targetPath

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

	// Case 1: Check if messages array has Anthropic-style content blocks
	if messages, hasMessages := req["messages"].([]interface{}); hasMessages && len(messages) > 0 {
		if firstMsg, ok := messages[0].(map[string]interface{}); ok {
			if content, hasContent := firstMsg["content"]; hasContent {
				// Check if content is Anthropic blocks format
				if contentArr, ok := content.([]interface{}); ok && len(contentArr) > 0 {
					if isAnthropicFormat(contentArr) {
						return convertAnthropicMessagesToOpenAI(body, req, messages)
					}
					// Check if content is already OpenAI string format
					if contentStr, ok := content.(string); ok && contentStr != "" {
						return body, false
					}
				}
			}
		}
	}

	// Case 2: Top-level content field (some providers)
	content, hasContent := req["content"]
	if !hasContent {
		return body, false
	}

	// OpenAI format - content is string
	if contentStr, ok := content.(string); ok && contentStr != "" {
		return body, false
	}

	// Anthropic format - content is array of blocks
	if contentArr, ok := content.([]interface{}); ok && len(contentArr) > 0 {
		if isAnthropicFormat(contentArr) {
			return convertAnthropicToOpenAIMessages(body, req, contentArr)
		}
	}

	// Messages array format
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

// convertAnthropicMessagesToOpenAI converts messages with Anthropic content blocks to OpenAI format
func convertAnthropicMessagesToOpenAI(body []byte, req map[string]interface{}, messages []interface{}) ([]byte, bool) {
	convertedMessages := []map[string]interface{}{}

	// Extract system prompt first
	if system, ok := req["system"]; ok {
		systemText := extractTextFromContent(system)
		if systemText != "" {
			convertedMessages = append(convertedMessages, map[string]interface{}{
				"role":    "system",
				"content": systemText,
			})
		}
	}

	for _, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}

		role, _ := msgMap["role"].(string)
		content := msgMap["content"]

		// Extract text from content (could be string or Anthropic blocks)
		contentStr := extractTextFromContent(content)

		convertedMessages = append(convertedMessages, map[string]interface{}{
			"role":    role,
			"content": contentStr,
		})
	}

	model := ""
	if m, ok := req["model"].(string); ok {
		model = m
	}

	converted := map[string]interface{}{
		"model":    model,
		"messages": convertedMessages,
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