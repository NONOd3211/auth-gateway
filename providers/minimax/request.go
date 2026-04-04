package minimax

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func BuildRequest(req *http.Request, apiKey string, upstreamURL string) (*http.Request, bool, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, false, err
	}
	req.Body.Close()

	var reqMap map[string]interface{}
	if err := json.Unmarshal(body, &reqMap); err != nil {
		reqMap = make(map[string]interface{})
	}

	isAnthropicFormat := isAnthropicFormatRequest(reqMap, req.URL.Path)
	var convertedBody []byte

	if isAnthropicFormat {
		convertedBody = convertAnthropicToOpenAI(reqMap)
	} else {
		convertedBody = body
	}

	targetPath := convertPath(req.URL.Path, isAnthropicFormat)
	targetURL := upstreamURL + targetPath

	proxyReq, err := http.NewRequest(req.Method, targetURL, bytes.NewReader(convertedBody))
	if err != nil {
		return nil, false, err
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+apiKey)

	for key, values := range req.Header {
		if key == "Host" || key == "Content-Length" || key == "Authorization" || key == "Accept-Encoding" {
			continue
		}
		proxyReq.Header[key] = values
	}

	return proxyReq, isAnthropicFormat, nil
}

func isAnthropicFormatRequest(req map[string]interface{}, path string) bool {
	if path == "/v1/messages" {
		return true
	}

	if messages, ok := req["messages"].([]interface{}); ok && len(messages) > 0 {
		if firstMsg, ok := messages[0].(map[string]interface{}); ok {
			if content, ok := firstMsg["content"]; ok {
				if contentArr, ok := content.([]interface{}); ok && len(contentArr) > 0 {
					if block, ok := contentArr[0].(map[string]interface{}); ok {
						if _, hasType := block["type"]; hasType {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

func convertAnthropicToOpenAI(req map[string]interface{}) []byte {
	messages := []map[string]interface{}{}

	if system, ok := req["system"]; ok {
		systemText := extractText(system)
		if systemText != "" {
			messages = append(messages, map[string]interface{}{
				"role":    "system",
				"content": systemText,
			})
		}
	}

	if msgs, ok := req["messages"].([]interface{}); ok {
		for _, msg := range msgs {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				role, _ := msgMap["role"].(string)
				content := extractText(msgMap["content"])
				messages = append(messages, map[string]interface{}{
					"role":    role,
					"content": content,
				})
			}
		}
	}

	model, _ := req["model"].(string)

	result := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	if stream, ok := req["stream"]; ok {
		result["stream"] = stream
		if streamBool, ok := stream.(bool); ok && streamBool {
			result["stream_options"] = map[string]interface{}{
				"include_usage": true,
			}
		}
	}

	if maxTokens, ok := req["max_tokens"]; ok {
		result["max_tokens"] = maxTokens
	}

	if temperature, ok := req["temperature"]; ok {
		result["temperature"] = temperature
	}

	if topP, ok := req["top_p"]; ok {
		result["top_p"] = topP
	}

	if stop, ok := req["stop"]; ok {
		result["stop"] = stop
	}

	converted, _ := json.Marshal(result)
	return converted
}

func extractText(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var sb strings.Builder
		for _, item := range v {
			if str, ok := item.(string); ok {
				sb.WriteString(str)
			} else if block, ok := item.(map[string]interface{}); ok {
				if text, ok := block["text"].(string); ok {
					sb.WriteString(text)
				}
			}
		}
		return sb.String()
	}
	return ""
}

func convertPath(path string, isAnthropic bool) string {
	if isAnthropic && path == "/v1/messages" {
		return "/v1/chat/completions"
	}
	return path
}

func IsAnthropicFormatRequest(body []byte) bool {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return false
	}
	return isAnthropicFormatRequest(req, "")
}
