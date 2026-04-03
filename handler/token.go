package handler

import (
	"auth-gateway/database"
	"auth-gateway/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type CreateTokenRequest struct {
	Name           string     `json:"name"`
	ExpiresAt      *time.Time `json:"expires_at"`
	MaxRequests    int        `json:"max_requests"`
	UserID         string     `json:"user_id"`
	Description    string     `json:"description"`
	HourlyLimit    bool       `json:"hourly_limit"`
	WeeklyLimit    bool       `json:"weekly_limit"`
	WeeklyRequests int        `json:"weekly_requests"`
}

type UpdateTokenRequest struct {
	Name         string     `json:"name"`
	ExpiresAt    *time.Time `json:"expires_at"`
	MaxRequests  int        `json:"max_requests"`
	Enabled      *bool      `json:"enabled"`
	Description  string     `json:"description"`
}

func ListTokens(c *gin.Context) {
	var tokens []models.Token
	query := database.DB.Model(&models.Token{})

	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Order("created_at DESC").Find(&tokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

func GetToken(c *gin.Context) {
	id := c.Param("id")
	var token models.Token
	if err := database.DB.First(&token, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}

	var usageCount int64
	database.DB.Model(&models.UsageRecord{}).Where("token_id = ?", token.ID).Count(&usageCount)

	c.JSON(http.StatusOK, gin.H{
		"token":       token,
		"usage_count": usageCount,
	})
}

// LookupToken allows users to look up their token by token value (public endpoint)
func LookupToken(c *gin.Context) {
	tokenValue := c.Query("token")
	if tokenValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token parameter required"})
		return
	}

	var token models.Token
	if err := database.DB.Where("token = ?", tokenValue).First(&token).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}

	// Get usage stats
	var totalRequests int64
	database.DB.Model(&models.UsageRecord{}).Where("token_id = ?", token.ID).Count(&totalRequests)

	var successRequests int64
	database.DB.Model(&models.UsageRecord{}).Where("token_id = ? AND success = ?", token.ID, true).Count(&successRequests)

	c.JSON(http.StatusOK, gin.H{
		"name":            token.Name,
		"max_requests":    token.MaxRequests,
		"used_requests":   token.UsedRequests,
		"total_requests":  totalRequests,
		"success_count":   successRequests,
		"enabled":        token.Enabled,
		"hourly_limit":    token.HourlyLimit,
		"hourly_used":     token.HourlyUsed,
		"weekly_limit":    token.WeeklyLimit,
		"weekly_used":     token.WeeklyUsed,
		"created_at":      token.CreatedAt,
	})
}

func CreateToken(c *gin.Context) {
	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenString, err := generateTokenString()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	token := models.Token{
		ID:              uuid.New().String(),
		Token:           tokenString,
		Name:            req.Name,
		CreatedAt:       time.Now(),
		ExpiresAt:       req.ExpiresAt,
		MaxRequests:     req.MaxRequests,
		UserID:          req.UserID,
		Description:     req.Description,
		HourlyLimit:     req.HourlyLimit,
		HourlyRequests:  0,
		HourlyUsed:      0,
		HourlyResetAt:   time.Now().Add(5 * time.Hour),
		WeeklyLimit:     req.WeeklyLimit,
		WeeklyRequests:  req.WeeklyRequests,
		WeeklyUsed:      0,
		WeeklyResetAt:   getWeeklyResetTime(),
		Enabled:         true,
	}

	if err := database.DB.Create(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "ok",
		"token":  token,
	})
}

func UpdateToken(c *gin.Context) {
	id := c.Param("id")
	var token models.Token
	if err := database.DB.First(&token, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}

	var req UpdateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.ExpiresAt != nil {
		updates["expires_at"] = req.ExpiresAt
	}
	if req.MaxRequests > 0 {
		updates["max_requests"] = req.MaxRequests
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if err := database.DB.Model(&token).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	database.DB.First(&token, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "token": token})
}

func DeleteToken(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Token{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func ResetUsage(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Model(&models.Token{}).Where("id = ?", id).Update("used_requests", 0).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func generateTokenString() (string, error) {
	tokenID := uuid.New().String()[:24]
	hash, err := bcrypt.GenerateFromPassword([]byte(tokenID), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return "sk-" + string(hash)[:32], nil
}

func getWeeklyResetTime() time.Time {
	now := time.Now()
	// Reset at the beginning of next week (Monday)
	daysUntilMonday := (7 - int(now.Weekday())) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	return now.AddDate(0, 0, daysUntilMonday).Truncate(24 * time.Hour)
}
