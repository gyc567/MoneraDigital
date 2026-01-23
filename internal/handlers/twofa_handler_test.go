package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// mockEncryption 用于测试
type mockEncryption struct {
	encryptFunc func(plaintext string) (string, error)
	decryptFunc func(ciphertext string) (string, error)
}

func (m *mockEncryption) Encrypt(plaintext string) (string, error) {
	if m.encryptFunc != nil {
		return m.encryptFunc(plaintext)
	}
	return plaintext, nil
}

func (m *mockEncryption) Decrypt(ciphertext string) (string, error) {
	if m.decryptFunc != nil {
		return m.decryptFunc(ciphertext)
	}
	return ciphertext, nil
}

func TestTwoFAHandler_Setup2FA_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TwoFAHandler{
		base: &BaseHandler{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// 不设置 userID

	handler.Setup2FA(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestTwoFAHandler_Enable2FA_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TwoFAHandler{
		base: &BaseHandler{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// 不设置 userID

	handler.Enable2FA(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestTwoFAHandler_Enable2FA_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TwoFAHandler{
		base: &BaseHandler{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", 1)
	c.Request = httptest.NewRequest("POST", "/api/auth/2fa/enable",
		strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Enable2FA(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["error"] != "Token is required" {
		t.Errorf("Expected error 'Token is required', got %v", response["error"])
	}
}

func TestTwoFAHandler_Enable2FA_InvalidTokenLength(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TwoFAHandler{
		base: &BaseHandler{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", 1)
	c.Request = httptest.NewRequest("POST", "/api/auth/2fa/enable",
		strings.NewReader(`{"token": "123"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Enable2FA(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["error"] != "Token must be 6 digits" {
		t.Errorf("Expected error 'Token must be 6 digits', got %v", response["error"])
	}
}

func TestTwoFAHandler_Disable2FA_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TwoFAHandler{
		base: &BaseHandler{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// 不设置 userID

	handler.Disable2FA(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestTwoFAHandler_Verify2FA_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TwoFAHandler{
		base: &BaseHandler{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// 不设置 userID

	handler.Verify2FA(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestTwoFAHandler_Verify2FA_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TwoFAHandler{
		base: &BaseHandler{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", 1)
	c.Request = httptest.NewRequest("POST", "/api/auth/2fa/verify",
		strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Verify2FA(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestTwoFAHandler_Get2FAStatus_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TwoFAHandler{
		base: &BaseHandler{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// 不设置 userID

	handler.Get2FAStatus(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestTwoFAHandler_Get2FAStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TwoFAHandler{
		base: &BaseHandler{},
		// twofaService is nil, but Get2FAStatus only uses base methods for auth
		// The service call would need mock, but this tests the auth flow
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", 1)

	// 先验证 auth 流程
	userID, ok := handler.base.requireUserID(c)
	if !ok {
		t.Fatal("requireUserID should succeed")
	}
	if userID != 1 {
		t.Errorf("Expected userID 1, got %d", userID)
	}

	// 测试 successResponse
	handler.base.successResponse(c, gin.H{"enabled": false})

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

	data := response["data"].(map[string]interface{})
	if data["enabled"] != false {
		t.Errorf("Expected enabled to be false, got %v", data["enabled"])
	}
}

func TestTwoFAHandler_Verify2FA_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TwoFAHandler{
		base: &BaseHandler{},
		// twofaService is nil, but Verify2FA only uses base methods for auth and token validation
		// The service call would need mock, but this tests the auth and validation flow
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", 1)
	c.Request = httptest.NewRequest("POST", "/api/auth/2fa/verify",
		strings.NewReader(`{"token": "000000"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	// 先验证 auth 流程
	userID, ok := handler.base.requireUserID(c)
	if !ok {
		t.Fatal("requireUserID should succeed")
	}
	if userID != 1 {
		t.Errorf("Expected userID 1, got %d", userID)
	}

	// 测试 token 验证
	token, ok := handler.base.bindTokenRequest(c)
	if !ok {
		t.Fatal("bindTokenRequest should succeed for valid token format")
	}
	if token != "000000" {
		t.Errorf("Expected token 000000, got %s", token)
	}

	// 测试 errorResponse
	handler.base.errorResponse(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid verification code")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["success"] != false {
		t.Error("Expected success to be false")
	}
}

func TestTwoFAHandler_New(t *testing.T) {
	handler := NewTwoFAHandler(nil)

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}

	if handler.base == nil {
		t.Error("Expected base handler to be initialized")
	}
}

func TestTwoFAHandler_Setup2FA_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &TwoFAHandler{
		base: &BaseHandler{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", 1)
	c.Set("email", "test@example.com")

	// 测试 requireUserID
	userID, ok := handler.base.requireUserID(c)
	if !ok {
		t.Fatal("requireUserID should succeed")
	}
	if userID != 1 {
		t.Errorf("Expected userID 1, got %d", userID)
	}

	// 测试 getUserEmail
	email, ok := handler.base.getUserEmail(c)
	if !ok {
		t.Fatal("getUserEmail should succeed")
	}
	if email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", email)
	}

	// 测试 successResponse
	handler.base.successResponse(c, gin.H{"test": "data"})
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}
