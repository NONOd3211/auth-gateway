package models

import (
	"time"

	"gorm.io/gorm"
)

type Token struct {
	ID              string         `json:"id" gorm:"primaryKey"`
	Token           string         `json:"token" gorm:"uniqueIndex;size:64"`
	Name            string         `json:"name" gorm:"size:100"`
	CreatedAt       time.Time      `json:"created_at"`
	ExpiresAt       *time.Time     `json:"expires_at"`
	MaxRequests     int            `json:"max_requests" gorm:"default:0"`
	UsedRequests    int            `json:"used_requests" gorm:"default:0"`
	HourlyLimit     bool           `json:"hourly_limit" gorm:"default:false"`
	HourlyRequests  int            `json:"hourly_requests" gorm:"default:0"`
	HourlyUsed      int            `json:"hourly_used" gorm:"default:0"`
	HourlyResetAt   time.Time      `json:"hourly_reset_at"`
	WeeklyLimit     bool           `json:"weekly_limit" gorm:"default:false"`
	WeeklyRequests  int            `json:"weekly_requests" gorm:"default:0"`
	WeeklyUsed      int            `json:"weekly_used" gorm:"default:0"`
	WeeklyResetAt   time.Time      `json:"weekly_reset_at"`
	Enabled         bool           `json:"enabled" gorm:"default:true"`
	UserID          string         `json:"user_id" gorm:"size:50"`
	Description     string         `json:"description" gorm:"size:255"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// IsExpired checks if the token has expired based on hourly limit (5 hours)
func (t *Token) IsExpired() bool {
	if t.HourlyLimit && time.Since(t.CreatedAt) > 5*time.Hour {
		return true
	}
	return false
}

// IsWithinWeeklyLimit checks if the token is within weekly usage limit
func (t *Token) IsWithinWeeklyLimit() bool {
	if !t.WeeklyLimit {
		return true
	}
	if t.WeeklyUsed >= t.WeeklyRequests {
		return false
	}
	return true
}

// CheckAndUpdateLimits checks and resets hourly/weekly counters if needed
func (t *Token) CheckAndUpdateLimits() {
	now := time.Now()

	// Reset hourly counter if 5 hours have passed
	if t.HourlyLimit && now.After(t.HourlyResetAt) {
		t.HourlyUsed = 0
		t.HourlyResetAt = now.Add(5 * time.Hour)
	}

	// Reset weekly counter if 7 days have passed
	if t.WeeklyLimit && now.After(t.WeeklyResetAt) {
		t.WeeklyUsed = 0
		t.WeeklyResetAt = now.Add(7 * 24 * time.Hour)
	}
}

type UsageRecord struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	TokenID         string    `json:"token_id" gorm:"index"`
	Timestamp       time.Time `json:"timestamp"`
	Model           string    `json:"model" gorm:"size:100"`
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	TotalTokens     int       `json:"total_tokens"`
	UpstreamProvider string   `json:"upstream_provider" gorm:"size:50"`
	Success         bool      `json:"success"`
	ErrorMessage    string    `json:"error_message" gorm:"size:500"`
	RequestPath     string    `json:"request_path" gorm:"size:200"`
}
