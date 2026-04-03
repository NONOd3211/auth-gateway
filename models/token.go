package models

import (
	"time"

	"gorm.io/gorm"
)

type Token struct {
	ID           string         `json:"id" gorm:"primaryKey"`
	Token        string         `json:"token" gorm:"uniqueIndex;size:64"`
	Name         string         `json:"name" gorm:"size:100"`
	CreatedAt    time.Time      `json:"created_at"`
	ExpiresAt    *time.Time     `json:"expires_at"`
	MaxRequests  int            `json:"max_requests" gorm:"default:0"`
	UsedRequests int            `json:"used_requests" gorm:"default:0"`
	Enabled      bool           `json:"enabled" gorm:"default:true"`
	UserID       string         `json:"user_id" gorm:"size:50"`
	Description  string         `json:"description" gorm:"size:255"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
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
