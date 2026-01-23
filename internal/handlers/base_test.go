package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestBaseHandler_getUserID_Success(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", 42)

	id, ok := handler.getUserID(c)

	if !ok {
		t.Error("Expected ok to be true")
	}
	if id != 42 {
		t.Errorf("Expected id 42, got %d", id)
	}
}

func TestBaseHandler_getUserID_NotFound(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// 不设置 userID

	id, ok := handler.getUserID(c)

	if ok {
		t.Error("Expected ok to be false")
	}
	if id != 0 {
		t.Errorf("Expected id 0, got %d", id)
	}
}

func TestBaseHandler_getUserID_WrongType(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", "not-an-int")

	id, ok := handler.getUserID(c)

	if ok {
		t.Error("Expected ok to be false for wrong type")
	}
	if id != 0 {
		t.Errorf("Expected id 0, got %d", id)
	}
}

func TestBaseHandler_getUserEmail_Success(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("email", "test@example.com")

	email, ok := handler.getUserEmail(c)

	if !ok {
		t.Error("Expected ok to be true")
	}
	if email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", email)
	}
}

func TestBaseHandler_getUserEmail_NotFound(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	email, ok := handler.getUserEmail(c)

	if ok {
		t.Error("Expected ok to be false")
	}
	if email != "" {
		t.Errorf("Expected empty email, got %s", email)
	}
}

func TestBaseHandler_requireUserID_Success(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", 1)

	id, ok := handler.requireUserID(c)

	if !ok {
		t.Error("Expected ok to be true")
	}
	if id != 1 {
		t.Errorf("Expected id 1, got %d", id)
	}
}

func TestBaseHandler_requireUserID_Unauthorized(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// 不设置 userID

	id, ok := handler.requireUserID(c)

	if ok {
		t.Error("Expected ok to be false")
	}
	if id != 0 {
		t.Errorf("Expected id 0, got %d", id)
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["error"] != "Unauthorized" {
		t.Errorf("Expected error message 'Unauthorized', got %v", response["error"])
	}
	if response["code"] != "AUTH_REQUIRED" {
		t.Errorf("Expected code 'AUTH_REQUIRED', got %v", response["code"])
	}
}

func TestBaseHandler_bindTokenRequest_ValidToken(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test",
		strings.NewReader(`{"token": "123456"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	token, ok := handler.bindTokenRequest(c)

	if !ok {
		t.Error("Expected ok to be true")
	}
	if token != "123456" {
		t.Errorf("Expected token 123456, got %s", token)
	}
}

func TestBaseHandler_bindTokenRequest_MissingToken(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test",
		strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	token, ok := handler.bindTokenRequest(c)

	if ok {
		t.Error("Expected ok to be false")
	}
	if token != "" {
		t.Errorf("Expected empty token, got %s", token)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["error"] != "Token is required" {
		t.Errorf("Expected error 'Token is required', got %v", response["error"])
	}
}

func TestBaseHandler_bindTokenRequest_InvalidLength(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test",
		strings.NewReader(`{"token": "123"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	_, ok := handler.bindTokenRequest(c)

	if ok {
		t.Error("Expected ok to be false for invalid token length")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["error"] != "Token must be 6 digits" {
		t.Errorf("Expected error 'Token must be 6 digits', got %v", response["error"])
	}
}

func TestBaseHandler_bindTokenRequest_NotJSON(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test",
		strings.NewReader(`invalid json`))
	c.Request.Header.Set("Content-Type", "application/json")

	_, ok := handler.bindTokenRequest(c)

	if ok {
		t.Error("Expected ok to be false for invalid JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestBaseHandler_successResponse(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.successResponse(c, gin.H{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["success"] != true {
		t.Error("Expected success to be true")
	}
	if response["data"] == nil {
		t.Error("Expected data to be present")
	}
}

func TestBaseHandler_errorResponse(t *testing.T) {
	handler := &BaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.errorResponse(c, http.StatusBadRequest, "TEST_ERROR", "Test error message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["success"] != false {
		t.Error("Expected success to be false")
	}

	errMap, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error to be a map")
	}

	if errMap["code"] != "TEST_ERROR" {
		t.Errorf("Expected code 'TEST_ERROR', got %v", errMap["code"])
	}
	if errMap["message"] != "Test error message" {
		t.Errorf("Expected message 'Test error message', got %v", errMap["message"])
	}
}
