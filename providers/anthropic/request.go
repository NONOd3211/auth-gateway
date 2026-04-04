package anthropic

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MessagesRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Stream      bool      `json:"stream,omitempty"`
	System      string    `json:"system,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

func BuildRequest(req *http.Request, apiKey string, baseURL string) (*http.Request, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body.Close()

	// Anthropic uses /v1/messages endpoint
	targetURL := baseURL + "/v1/messages"

	proxyReq, err := http.NewRequest(req.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Anthropic specific headers
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("x-api-key", apiKey)
	proxyReq.Header.Set("anthropic-version", "2023-06-01")

	// Copy other headers except Host, Content-Length, Authorization (we set our own)
	for key, values := range req.Header {
		if key == "Host" || key == "Content-Length" || key == "x-api-key" || key == "anthropic-version" || key == "Authorization" {
			continue
		}
		proxyReq.Header[key] = values
	}

	return proxyReq, nil
}

// ConvertOpenAIToAnthropic converts OpenAI format request to Anthropic format
func ConvertOpenAIToAnthropic(body []byte) ([]byte, error) {
	var openAIReq struct {
		Model    string                   `json:"model"`
		Messages []map[string]interface{} `json:"messages"`
		Stream   bool                     `json:"stream,omitempty"`
	}

	if err := json.Unmarshal(body, &openAIReq); err != nil {
		return nil, err
	}

	messages := make([]Message, 0, len(openAIReq.Messages))
	var system string

	for _, msg := range openAIReq.Messages {
		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)

		if role == "system" {
			system = content
			continue
		}

		// Convert role names: user -> user, assistant -> assistant
		messages = append(messages, Message{
			Role:    role,
			Content: content,
		})
	}

	anthropicReq := MessagesRequest{
		Model:     openAIReq.Model,
		Messages:  messages,
		MaxTokens: 4096, // Default, should be from request
		Stream:    openAIReq.Stream,
		System:    system,
	}

	return json.Marshal(anthropicReq)
}
