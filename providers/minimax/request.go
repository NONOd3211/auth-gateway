package minimax

import (
	"bytes"
	"io"
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

	// Convert path from OpenAI format to MiniMax format
	targetPath := convertPath(req.URL.Path)
	targetURL := upstreamURL + targetPath

	proxyReq, err := http.NewRequest(req.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+apiKey)

	// Copy other headers except Host
	for key, values := range req.Header {
		if key == "Host" || key == "Content-Length" {
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