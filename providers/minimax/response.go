package minimax

import (
	"encoding/json"
)

// ConvertOpenAIToAnthropicResponse converts OpenAI response to Anthropic format
func ConvertOpenAIToAnthropicResponse(body []byte, model string) ([]byte, error) {
	var openaiResp map[string]interface{}
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return body, err
	}

	anthropicResp := convertOpenAIResponseToAnthropic(openaiResp, model)
	return json.Marshal(anthropicResp)
}

func convertOpenAIResponseToAnthropic(openaiResp map[string]interface{}, model string) map[string]interface{} {
	choices, ok := openaiResp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return openaiResp
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return openaiResp
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return openaiResp
	}

	var contentStr string
	if content, ok := message["content"]; ok {
		if str, ok := content.(string); ok {
			contentStr = str
		}
	}

	role := "assistant"
	if r, ok := message["role"].(string); ok {
		role = r
	}

	usage := map[string]interface{}{
		"input_tokens":  0,
		"output_tokens": 0,
	}
	if u, ok := openaiResp["usage"].(map[string]interface{}); ok {
		if promptTokens, ok := u["prompt_tokens"].(float64); ok {
			usage["input_tokens"] = int(promptTokens)
		}
		if completionTokens, ok := u["completion_tokens"].(float64); ok {
			usage["output_tokens"] = int(completionTokens)
		}
	}

	var stopReason interface{} = "end_turn"
	if fr, ok := choice["finish_reason"].(string); ok && fr != "" {
		stopReason = fr
	}

	return map[string]interface{}{
		"id":            openaiResp["id"],
		"type":          "message",
		"role":          role,
		"model":         model,
		"content":       []map[string]interface{}{{"type": "text", "text": contentStr}},
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage":         usage,
	}
}

// IsAnthropicFormatRequest checks if the request body is in Anthropic format
func IsAnthropicFormatRequest(body []byte) bool {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return false
	}

	// Check if messages array has Anthropic-style content blocks
	if messages, hasMessages := req["messages"].([]interface{}); hasMessages && len(messages) > 0 {
		if firstMsg, ok := messages[0].(map[string]interface{}); ok {
			if content, hasContent := firstMsg["content"]; hasContent {
				if contentArr, ok := content.([]interface{}); ok && len(contentArr) > 0 {
					return isAnthropicFormat(contentArr)
				}
			}
		}
	}

	// Check top-level content field
	if content, hasContent := req["content"]; hasContent {
		if contentArr, ok := content.([]interface{}); ok && len(contentArr) > 0 {
			return isAnthropicFormat(contentArr)
		}
	}

	return false
}
