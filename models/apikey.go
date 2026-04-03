package models

import (
	"time"
)

type APIKey struct {
	ID        string     `json:"id" gorm:"primaryKey"`
	Key       string     `json:"key" gorm:"size:200"`        // MiniMax API Key
	Name      string     `json:"name" gorm:"size:100"`       // 名称备注
	Enabled   bool       `json:"enabled" gorm:"default:true"`
	Healthy   bool       `json:"healthy" gorm:"default:true"`
	FailedAt  *time.Time `json:"failed_at"`
	FailCount int        `json:"fail_count" gorm:"default:0"`
	CreatedAt time.Time  `json:"created_at"`
}

type TokenKeyMapping struct {
	TokenID    string    `json:"token_id" gorm:"primaryKey"`
	APIKeyID   string    `json:"api_key_id" gorm:"primaryKey"`
	AssignedAt time.Time `json:"assigned_at"`
}