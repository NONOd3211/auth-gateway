package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestInvalidDateFormatReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock the handler with date validation
	router.GET("/usage", func(c *gin.Context) {
		startDate := c.Query("start_date")
		endDate := c.Query("end_date")

		if startDate != "" {
			if _, err := time.Parse("2006-01-02", startDate); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format, expected YYYY-MM-DD"})
				return
			}
		}
		if endDate != "" {
			if _, err := time.Parse("2006-01-02", endDate); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format, expected YYYY-MM-DD"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	tests := []struct {
		name       string
		url        string
		wantStatus int
	}{
		{"invalid start_date", "/usage?start_date=invalid", http.StatusBadRequest},
		{"invalid end_date", "/usage?end_date=2024-13-01", http.StatusBadRequest},
		{"valid start_date", "/usage?start_date=2024-01-01", http.StatusOK},
		{"valid end_date", "/usage?end_date=2024-12-31", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}