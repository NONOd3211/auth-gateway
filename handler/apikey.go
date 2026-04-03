package handler

import (
	"auth-gateway/database"
	"auth-gateway/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ListAPIKeys(c *gin.Context) {
	var keys []models.APIKey
	if err := database.DB.Order("created_at DESC").Find(&keys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

func CreateAPIKey(c *gin.Context) {
	var req struct {
		Key  string `json:"key" binding:"required"`
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := models.APIKey{
		ID:        uuid.New().String(),
		Key:       req.Key,
		Name:      req.Name,
		Enabled:   true,
		Healthy:   true,
		FailCount: 0,
	}

	if err := database.DB.Create(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"key": key})
}

func UpdateAPIKey(c *gin.Context) {
	id := c.Param("id")
	var key models.APIKey
	if err := database.DB.First(&key, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	var req struct {
		Name    string `json:"name"`
		Enabled *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		key.Name = req.Name
	}
	if req.Enabled != nil {
		key.Enabled = *req.Enabled
	}

	if err := database.DB.Save(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key})
}

func DeleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.APIKey{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func EnableAPIKey(c *gin.Context) {
	id := c.Param("id")
	var key models.APIKey
	if err := database.DB.First(&key, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	key.Enabled = true
	key.Healthy = true
	if err := database.DB.Save(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key})
}

func DisableAPIKey(c *gin.Context) {
	id := c.Param("id")
	var key models.APIKey
	if err := database.DB.First(&key, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	key.Enabled = false
	if err := database.DB.Save(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key})
}