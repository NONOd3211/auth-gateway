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

	targetURL := upstreamURL + req.URL.Path
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