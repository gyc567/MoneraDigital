package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"monera-digital/internal/dto"

	"github.com/gin-gonic/gin"
)

func TestCreateWallet_SuccessResponseFormat(t *testing.T) {
	h := newMockHandler()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/wallet/create", h.CreateWallet)

	reqBody := dto.CreateWalletRequest{
		UserID:      "123",
		ProductCode: "BANK_ACCOUNT",
		Currency:    "USD",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/wallet/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify top-level fields
	expectedFields := []string{"code", "message", "data", "success", "timestamp"}
	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Response missing required field: %s", field)
		}
	}

	// Verify specific values
	if response["success"] != true {
		t.Error("Expected success to be true")
	}
	if response["code"] != "200" {
		t.Errorf("Expected code '200', got %v", response["code"])
	}
	if response["message"] != "Success" { // We expect "Success", not "成功"
		t.Errorf("Expected message 'Success', got %v", response["message"])
	}
}

func TestCreateWallet_ErrorResponseFormat(t *testing.T) {
	h := newMockHandler()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/wallet/create", h.CreateWallet)

	reqBody := dto.CreateWalletRequest{
		UserID:      "", // Invalid request
		ProductCode: "BANK_ACCOUNT",
		Currency:    "USD",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/wallet/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify error format matches success format structure
	expectedFields := []string{"code", "message", "success", "timestamp"}
	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Error response missing required field: %s", field)
		}
	}

	if response["success"] != false {
		t.Error("Expected success to be false")
	}
	if response["code"] != "400" {
		t.Errorf("Expected code '400', got %v", response["code"])
	}
}
