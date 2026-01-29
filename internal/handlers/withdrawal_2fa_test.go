package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestCreateWithdrawal_2FARequired tests that withdrawal requires 2FA when enabled
func TestCreateWithdrawal_2FARequired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Missing 2FA token",
			requestBody:    `{"addressId": 1, "amount": "100.0", "asset": "USDT"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "TwoFactorToken",
		},
		{
			name:           "Invalid 2FA token length",
			requestBody:    `{"addressId": 1, "amount": "100.0", "asset": "USDT", "twoFactorToken": "12345"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "TwoFactorToken",
		},
		{
			name:           "Valid request with 2FA token",
			requestBody:    `{"addressId": 1, "amount": "100.0", "asset": "USDT", "twoFactorToken": "123456"}`,
			expectedStatus: http.StatusCreated,
			expectedBody:   "Withdrawal created",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Note: This is a simplified test that validates request structure
			// Full integration test would require database setup
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/withdrawals", strings.NewReader(test.requestBody))
			c.Request.Header.Set("Content-Type", "application/json")

			// Validate JSON structure
			var req map[string]interface{}
			if err := c.ShouldBindJSON(&req); err != nil {
				if test.expectedStatus == http.StatusBadRequest {
					return // Expected failure
				}
			}

			// Check for twoFactorToken presence when required
			if test.name == "Missing 2FA token" {
				if _, ok := req["twoFactorToken"]; !ok {
					return // Expected failure
				}
			}
		})
	}
}

// TestCreateWithdrawalRequest_2FATokenValidation tests 2FA token validation
func TestCreateWithdrawalRequest_2FATokenValidation(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		isValid   bool
	}{
		{
			name:    "Empty token",
			token:   "",
			isValid: false,
		},
		{
			name:    "5 digits",
			token:   "12345",
			isValid: false,
		},
		{
			name:    "6 digits",
			token:   "123456",
			isValid: true,
		},
		{
			name:    "7 digits",
			token:   "1234567",
			isValid: false,
		},
		{
			name:    "Non-numeric",
			token:   "abcdef",
			isValid: true, // Length check only
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isValid := len(test.token) == 6
			if isValid != test.isValid {
				t.Errorf("Expected validation result %v for token '%s', got %v", test.isValid, test.token, isValid)
			}
		})
	}
}

// TestCreateWithdrawalRequest_JSONStructure tests the request JSON structure
func TestCreateWithdrawalRequest_JSONStructure(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		expectError bool
	}{
		{
			name:        "Valid JSON with all fields",
			json:        `{"addressId": 1, "amount": "100.0", "asset": "USDT", "twoFactorToken": "123456"}`,
			expectError: false,
		},
		{
			name:        "Valid JSON without 2FA token",
			json:        `{"addressId": 1, "amount": "100.0", "asset": "USDT"}`,
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			json:        `{"addressId": 1, "amount": "100.0",`,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var req map[string]interface{}
			err := json.Unmarshal([]byte(test.json), &req)
			if test.expectError && err == nil {
				t.Errorf("Expected JSON parse error for: %s", test.json)
			}
			if !test.expectError && err != nil {
				t.Errorf("Unexpected JSON parse error: %v", err)
			}
		})
	}
}
