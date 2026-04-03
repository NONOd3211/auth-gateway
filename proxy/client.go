package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL     string
	upstreamKey string
	httpClient  *http.Client
}

func NewClient(baseURL string, upstreamKey string) *Client {
	return &Client{
		baseURL:     baseURL,
		upstreamKey: upstreamKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *Client) Forward(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body.Close()

	targetURL, err := url.Parse(c.baseURL + req.URL.Path)
	if err != nil {
		return nil, err
	}
	targetURL.RawQuery = req.URL.RawQuery

	proxyReq, err := http.NewRequest(req.Method, targetURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Add upstream API key if configured
	if c.upstreamKey != "" {
		proxyReq.Header.Set("Authorization", "Bearer "+c.upstreamKey)
	}

	for key, values := range req.Header {
		if key == "Host" {
			continue
		}
		// Use Set to replace existing headers instead of Add (which appends)
		if len(values) > 0 {
			proxyReq.Header.Set(key, values[0])
			for _, value := range values[1:] {
				proxyReq.Header.Add(key, value)
			}
		}
	}

	return c.httpClient.Do(proxyReq)
}

func (c *Client) ForwardStream(req *http.Request) (*http.Response, error) {
	return c.Forward(req)
}
