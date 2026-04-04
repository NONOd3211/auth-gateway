package handler

import (
	"auth-gateway/database"
	"auth-gateway/models"
	"net/http"
	"strings"

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
		Key           string `json:"key" binding:"required"`
		Name          string `json:"name"`
		AllowedModels string `json:"allowed_models"` // comma-separated model list
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := models.APIKey{
		ID:             uuid.New().String(),
		Key:            req.Key,
		Name:           req.Name,
		AllowedModels:  req.AllowedModels,
		Enabled:        true,
		Healthy:        true,
		FailCount:      0,
	}

	if err := database.DB.Create(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload keys into provider manager
	if providerManager != nil {
		providerManager.ReloadAPIKeys()
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
		Name          string `json:"name"`
		AllowedModels string `json:"allowed_models"`
		Enabled       *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		key.Name = req.Name
	}
	if req.AllowedModels != "" {
		key.AllowedModels = req.AllowedModels
	}
	if req.Enabled != nil {
		key.Enabled = *req.Enabled
	}

	if err := database.DB.Save(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload keys into provider manager
	if providerManager != nil {
		providerManager.ReloadAPIKeys()
	}

	c.JSON(http.StatusOK, gin.H{"key": key})
}

func DeleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.APIKey{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload keys into provider manager
	if providerManager != nil {
		providerManager.ReloadAPIKeys()
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

	// Reload keys into provider manager
	if providerManager != nil {
		providerManager.ReloadAPIKeys()
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

	// Reload keys into provider manager
	if providerManager != nil {
		providerManager.ReloadAPIKeys()
	}

	c.JSON(http.StatusOK, gin.H{"key": key})
}

// ListModels returns all available models from all enabled API keys
func ListModels(c *gin.Context) {
	var keys []models.APIKey
	if err := database.DB.Where("enabled = ?", true).Find(&keys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Collect unique models from all API keys
	modelSet := make(map[string]bool)
	for _, key := range keys {
		if key.AllowedModels != "" {
			// Split by comma and add each model
			models := strings.Split(key.AllowedModels, ",")
			for _, m := range models {
				m = strings.TrimSpace(m)
				if m != "" {
					modelSet[m] = true
				}
			}
		}
	}

	models := make([]string, 0, len(modelSet))
	for model := range modelSet {
		models = append(models, model)
	}

	c.JSON(http.StatusOK, gin.H{
		"models": models,
	})
}