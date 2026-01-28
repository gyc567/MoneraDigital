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

func TestCreateWallet_MissingFields(t *testing.T) {
	h := newTestHandler()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/wallet/create", h.CreateWallet)

	tests := []struct {
		name        string
		reqBody     dto.CreateWalletRequest
		expectedMsg string
	}{
		{
			name: "Missing UserID",
			reqBody: dto.CreateWalletRequest{
				UserID:      "",
				ProductCode: "BANK_ACCOUNT",
				Currency:    "USD",
			},
			expectedMsg: "userId, productCode and currency are required",
		},
		{
			name: "Missing ProductCode",
			reqBody: dto.CreateWalletRequest{
				UserID:      "123",
				ProductCode: "",
				Currency:    "USD",
			},
			expectedMsg: "userId, productCode and currency are required",
		},
		{
			name: "Missing Currency",
			reqBody: dto.CreateWalletRequest{
				UserID:      "123",
				ProductCode: "BANK_ACCOUNT",
				Currency:    "",
			},
			expectedMsg: "userId, productCode and currency are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("POST", "/api/v1/wallet/create", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			r.ServeHTTP(resp, req)

			if resp.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.Code)
			}

			var response dto.CreateWalletResponse
			if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if response.Message != tt.expectedMsg {
				t.Errorf("Expected message '%s', got '%s'", tt.expectedMsg, response.Message)
			}
		})
	}
}

func TestCreateWallet_InvalidProductCode(t *testing.T) {
	h := newTestHandler()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/wallet/create", h.CreateWallet)

	reqBody := dto.CreateWalletRequest{
		UserID:      "123",
		ProductCode: "INVALID_PRODUCT",
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

	var response dto.CreateWalletResponse
	json.Unmarshal(resp.Body.Bytes(), &response)

	if response.Message != "Invalid product code" {
		t.Errorf("Expected message 'Invalid product code', got '%s'", response.Message)
	}
}
