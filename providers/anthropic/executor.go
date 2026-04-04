package anthropic

import (
	"io"
	"net/http"
	"time"
)

type Executor struct {
	baseURL string
	timeout time.Duration
}

func NewExecutor() *Executor {
	return &Executor{
		baseURL: "https://api.anthropic.com",
		timeout: 10 * time.Minute,
	}
}

func (e *Executor) Name() string {
	return "anthropic"
}

func (e *Executor) Execute(req *http.Request, apiKey string) (*http.Response, error) {
	proxyReq, err := BuildRequest(req, apiKey, e.baseURL)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: e.timeout}
	return client.Do(proxyReq)
}

func (e *Executor) IsQuotaError(resp *http.Response) bool {
	return IsQuotaError(resp)
}

func (e *Executor) GetQuotaInfo(resp *http.Response) (used, limit int64, err error) {
	return GetQuotaInfo(resp)
}

// PassThroughResponse reads response body and returns it
func PassThroughResponse(resp *http.Response) ([]byte, error) {
	return io.ReadAll(resp.Body)
}
