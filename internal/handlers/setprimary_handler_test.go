package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSetPrimaryAddress_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{}

	r := gin.New()
	r.POST("/api/addresses/:id/primary", handler.SetPrimaryAddress)

	req := httptest.NewRequest("POST", "/api/addresses/10/primary", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSetPrimaryAddress_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Next()
	})
	r.POST("/api/addresses/:id/primary", handler.SetPrimaryAddress)

	req := httptest.NewRequest("POST", "/api/addresses/invalid/primary", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
