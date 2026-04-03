package minimax

import (
	"net/http"
)

type QuotaInfo struct {
	Used  int64
	Limit int64
}

func IsQuotaError(resp *http.Response) bool {
	// 429 Too Many Requests
	if resp.StatusCode == 429 {
		return true
	}
	return false
}

func GetQuotaInfo(resp *http.Response) (used, limit int64, err error) {
	// MiniMax API quota info in header or body
	// Return 0,0 if not available
	return 0, 0, nil
}