package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTwoFAHandler_Skip2FALogin_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Since Skip2FALogin is on the main Handler struct, we test it there.
	// However, we need to mock AuthService.
	// For this test, we just check binding validation which happens before service call.
	handler := &Handler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/auth/2fa/skip",
		strings.NewReader(`{}`)) // Missing userId
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Skip2FALogin(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["error"] == nil {
		t.Error("Expected error in response")
	}
}
