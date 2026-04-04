package anthropic

import (
	"net/http"
)

func IsQuotaError(resp *http.Response) bool {
	// 429 Too Many Requests or 429 Rate Limit
	if resp.StatusCode == 429 {
		return true
	}
	return false
}

func GetQuotaInfo(resp *http.Response) (used, limit int64, err error) {
	// Anthropic may include quota info in headers
	// For now, return 0,0 if not available
	return 0, 0, nil
}
