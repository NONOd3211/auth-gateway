package database

import (
	"auth-gateway/models"
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(dbURL string) error {
	dir := filepath.Dir(dbURL)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	if err = DB.AutoMigrate(&models.Token{}, &models.UsageRecord{}, &models.APIKey{}, &models.TokenKeyMapping{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}
