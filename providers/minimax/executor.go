package minimax

import (
	"auth-gateway/config"
	"bytes"
	"io"
	"log"
	"net/http"
	"time"
)

type Executor struct {
	baseURL string
	timeout time.Duration
}

func NewExecutor(cfg *config.Config) *Executor {
	return &Executor{
		baseURL: getMiniMaxBaseURL(cfg),
		timeout: 10 * time.Minute,
	}
}

func getMiniMaxBaseURL(cfg *config.Config) string {
	// MiniMax API base URL
	return "https://api.minimaxi.com"
}

func (e *Executor) Name() string {
	return "minimax"
}

func (e *Executor) Execute(req *http.Request, apiKey string) (*http.Response, error) {
	proxyReq, _, err := BuildRequest(req, apiKey, e.baseURL)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: e.timeout}
	resp, err := client.Do(proxyReq)
	if err != nil {
		return nil, err
	}

	// Check for errors in response
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("[MiniMax] Error response: status=%d body=%s", resp.StatusCode, string(body))
		// Restore body for error response
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}

	return resp, nil
}

func (e *Executor) IsQuotaError(resp *http.Response) bool {
	// MiniMax quota errors typically return 429 status code
	if resp.StatusCode == 429 {
		return true
	}
	return false
}

func (e *Executor) GetQuotaInfo(resp *http.Response) (used, limit int64, err error) {
	// MiniMax doesn't provide quota info in response headers
	// Return 0, 0 to indicate no info available
	return 0, 0, nil
}

// PassThroughResponse reads response body and returns it
func PassThroughResponse(resp *http.Response) ([]byte, error) {
	return io.ReadAll(resp.Body)
}