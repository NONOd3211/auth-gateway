package minimax

import (
	"bytes"
	"io"
	"log"
	"net/http"
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

	// Convert path from OpenAI format to MiniMax format
	targetPath := convertPath(req.URL.Path)
	targetURL := upstreamURL + targetPath

	log.Printf("[MiniMax] Target URL: %s", targetURL)

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

// convertPath converts OpenAI-style paths to MiniMax-specific paths
func convertPath(openaiPath string) string {
	// /v1/chat/completions -> /v1/text/chatcompletion_v2
	if openaiPath == "/v1/chat/completions" {
		return "/v1/text/chatcompletion_v2"
	}
	// Other paths pass through as-is
	return openaiPath
}