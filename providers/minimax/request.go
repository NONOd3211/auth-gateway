package minimax

import (
	"bytes"
	"encoding/json"
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

	// Convert Anthropic format to OpenAI format if needed
	convertedBody, isConverted := convertAnthropicToOpenAI(body)
	if isConverted {
		log.Printf("[MiniMax] Converted Anthropic format to OpenAI format")
		body = convertedBody
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

// convertAnthropicToOpenAI converts Anthropic SDK format to OpenAI/MiniMax format
// Returns converted body and true if conversion was performed
func convertAnthropicToOpenAI(body []byte) ([]byte, bool) {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return body, false
	}

	// Check if this is Anthropic format (has content as array of blocks)
	content, hasContent := req["content"]
	if !hasContent {
		return body, false
	}

	contentArr, isArray := content.([]interface{})
	if !isArray || len(contentArr) == 0 {
		return body, false
	}

	// Check if content blocks have "type" and "text" fields (Anthropic format)
	firstBlock, ok := contentArr[0].(map[string]interface{})
	if !ok {
		return body, false
	}

	if _, hasType := firstBlock["type"]; !hasType {
		return body, false
	}

	// This is Anthropic format, convert it
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

// extractTextFromContent extracts text from various content formats
func extractTextFromContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
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
	// Other paths pass through as-is
	return openaiPath
}