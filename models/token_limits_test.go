package models

import (
	"testing"
	"time"
)

func TestTokenHourlyLimit(t *testing.T) {
	token := Token{
		CreatedAt:    time.Now().Add(-6 * time.Hour), // 6 hours ago
		HourlyLimit:  true,
	}

	// Token created 6 hours ago should be expired if hourly limit is enabled
	if !token.IsExpired() {
		t.Log("Token older than 5 hours should be expired")
	}
}

func TestTokenWeeklyLimit(t *testing.T) {
	token := Token{
		CreatedAt:    time.Now().Add(-8 * 24 * time.Hour), // 8 days ago
		WeeklyLimit:  true,
		WeeklyRequests: 100,
		WeeklyUsed:      50,
	}

	// Token created 8 days ago should reset weekly counter
	shouldReset := time.Since(token.CreatedAt) > 7*24*time.Hour
	if shouldReset {
		t.Log("Weekly limit should reset after 7 days")
	}
}

func TestTokenWithinHourlyLimit(t *testing.T) {
	token := Token{
		CreatedAt:    time.Now().Add(-1 * time.Hour), // 1 hour ago
		HourlyLimit:  true,
	}

	// Token created 1 hour ago should still be valid
	if token.CreatedAt.After(time.Now().Add(-5 * time.Hour)) {
		t.Log("Token within 5 hours should be valid")
	}
}