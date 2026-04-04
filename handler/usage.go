package handler

import (
	"auth-gateway/database"
	"auth-gateway/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type UsageStats struct {
	TotalRequests  int64 `json:"total_requests"`
	SuccessCount   int64 `json:"success_count"`
	FailureCount   int64 `json:"failure_count"`
	TotalTokens    int64 `json:"total_tokens"`
	InputTokens    int64 `json:"input_tokens"`
	OutputTokens   int64 `json:"output_tokens"`
}

func GetUsageStats(c *gin.Context) {
	tokenID := c.Query("token_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	query := database.DB.Model(&models.UsageRecord{})

	if tokenID != "" {
		query = query.Where("token_id = ?", tokenID)
	}
	if startDate != "" {
		t, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format, expected YYYY-MM-DD"})
			return
		}
		query = query.Where("timestamp >= ?", t)
	}
	if endDate != "" {
		t, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format, expected YYYY-MM-DD"})
			return
		}
		query = query.Where("timestamp <= ?", t.Add(24*time.Hour))
	}

	var stats UsageStats
	query.Select("COUNT(*) as total_requests").Scan(&stats.TotalRequests)
	query.Select("SUM(CASE WHEN success THEN 1 ELSE 0 END) as success_count").Scan(&stats.SuccessCount)
	query.Select("SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failure_count").Scan(&stats.FailureCount)
	query.Select("SUM(total_tokens) as total_tokens").Scan(&stats.TotalTokens)
	query.Select("SUM(input_tokens) as input_tokens").Scan(&stats.InputTokens)
	query.Select("SUM(output_tokens) as output_tokens").Scan(&stats.OutputTokens)

	c.JSON(http.StatusOK, stats)
}

func GetUsageByToken(c *gin.Context) {
	tokenID := c.Param("id")

	var records []models.UsageRecord
	if err := database.DB.Where("token_id = ?", tokenID).Order("timestamp DESC").Limit(100).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"records": records})
}

// GetUsageEvents returns usage records with optional filtering
func GetUsageEvents(c *gin.Context) {
	tokenID := c.Query("token_id")
	page := 1
	pageSize := 50

	query := database.DB.Model(&models.UsageRecord{})

	if tokenID != "" {
		query = query.Where("token_id = ?", tokenID)
	}

	var total int64
	query.Count(&total)

	var records []models.UsageRecord
	offset := (page - 1) * pageSize
	if err := query.Order("timestamp DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
		"total":   total,
		"page":    page,
		"page_size": pageSize,
	})
}

func GetUsageByDay(c *gin.Context) {
	tokenID := c.Query("token_id")

	type DayStats struct {
		Date          string `json:"date"`
		Requests      int64  `json:"requests"`
		TotalTokens   int64  `json:"total_tokens"`
		InputTokens   int64  `json:"input_tokens"`
		OutputTokens  int64  `json:"output_tokens"`
	}

	var results []DayStats
	query := database.DB.Model(&models.UsageRecord{}).
		Select("DATE(timestamp) as date, COUNT(*) as requests, SUM(total_tokens) as total_tokens, SUM(input_tokens) as input_tokens, SUM(output_tokens) as output_tokens").
		Group("DATE(timestamp)").
		Order("date DESC").
		Limit(30)

	if tokenID != "" {
		query = query.Where("token_id = ?", tokenID)
	}

	query.Scan(&results)

	c.JSON(http.StatusOK, gin.H{"daily": results})
}

const MaxUsageRecords = 100000

// CleanupUsageRecords removes old records keeping only the latest MaxUsageRecords
func CleanupUsageRecords() error {
	var count int64
	database.DB.Model(&models.UsageRecord{}).Count(&count)

	if count <= MaxUsageRecords {
		return nil
	}

	// Get the ID of the record at position MaxUsageRecords (keep everything after this)
	var cutoffRecord models.UsageRecord
	err := database.DB.Order("timestamp DESC").Offset(MaxUsageRecords - 1).First(&cutoffRecord).Error
	if err != nil {
		return err
	}

	// Delete all records older than or equal to the cutoff timestamp
	// But we need to be more precise - delete records with timestamp <= cutoffRecord.Timestamp
	// that are beyond our limit
	deleteCount := count - int64(MaxUsageRecords)
	if deleteCount <= 0 {
		return nil
	}

	result := database.DB.Where("timestamp <= ?", cutoffRecord.Timestamp).
		Order("timestamp ASC").
		Limit(int(deleteCount)).
		Delete(&models.UsageRecord{})

	return result.Error
}
