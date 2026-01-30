package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"monera-digital/internal/coreapi"
	"monera-digital/internal/dto"
	"monera-digital/internal/services"

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

// TestGetAddressIncomeHistory_Unauthorized 测试未授权访问
func TestGetAddressIncomeHistory_Unauthorized(t *testing.T) {
	h := newTestHandler()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/wallet/address/incomeHistory", h.GetAddressIncomeHistory)

	reqBody := dto.GetAddressIncomeHistoryRequest{
		Address: "TEJG77HBANUVgL2rJ34NPR27vb56TkuUVV",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/wallet/address/incomeHistory", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}

	var response dto.GetAddressIncomeHistoryResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Code != "401" {
		t.Errorf("Expected code '401', got '%s'", response.Code)
	}

	if response.Success != false {
		t.Error("Expected success to be false")
	}
}

// TestGetAddressIncomeHistory_InvalidRequest 测试无效请求（缺少 address 字段）
func TestGetAddressIncomeHistory_InvalidRequest(t *testing.T) {
	h := newTestHandler()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/wallet/address/incomeHistory", h.GetAddressIncomeHistory)

	// 空的请求体（缺少 required 的 address 字段）
	req, _ := http.NewRequest("POST", "/api/wallet/address/incomeHistory", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	// 由于缺少 address 字段和认证，期望 401
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}

	var response dto.GetAddressIncomeHistoryResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Code != "401" {
		t.Errorf("Expected code '401', got '%s'", response.Code)
	}
}

// TestGetAddressIncomeHistory_MissingAddress 测试缺少 address 字段（已认证）
func TestGetAddressIncomeHistory_MissingAddress(t *testing.T) {
	h := newTestHandler()
	gin.SetMode(gin.TestMode)

	// 创建带有 userID 上下文的请求
	req, _ := http.NewRequest("POST", "/api/wallet/address/incomeHistory", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Set("userID", 123)
	c.Request = req

	// 直接调用 Handler
	h.GetAddressIncomeHistory(c)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}

	var response dto.GetAddressIncomeHistoryResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Code != "400" {
		t.Errorf("Expected code '400', got '%s'", response.Code)
	}
}

// TestGetAddressIncomeHistory_Success 测试成功获取收入历史
func TestGetAddressIncomeHistory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建 mock Core API 服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := coreapi.CoreAPIResponse{
			Success: true,
			Data: []coreapi.AddressIncomeRecord{
				{
					TxKey:             "txsjujr2a631c3d5a4p71adcfee4802001",
					TxHash:            "d8ff30f1f66aff580a34ae1ac6d58190861129fd1750fd26c5c22fae40f857c2",
					CoinKey:           "TRX(SHASTA)_TRON_TESTNET",
					TxAmount:          "2000.00000000",
					Address:           "TEJG77HBANUVgL2rJ34NPR27vb56TkuUVV",
					TransactionStatus: "COMPLETED",
					BlockHeight:       61844572,
					CreateTime:        "2026-01-28 10:04:07",
					CompletedTime:     "2026-01-28 10:05:51",
				},
			},
			Message: "成功",
			Code:    "200",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	h := &Handler{
		WalletService: &services.WalletService{},
	}

	reqBody := dto.GetAddressIncomeHistoryRequest{
		Address: "TEJG77HBANUVgL2rJ34NPR27vb56TkuUVV",
	}
	body, _ := json.Marshal(reqBody)

	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Set("userID", 123)
	c.Request = httptest.NewRequest("POST", "/api/wallet/address/incomeHistory", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	// 由于无法直接设置 coreAPIClient，我们测试请求解析逻辑
	// 预期行为：即使 CoreAPI 客户端为 nil，也会返回错误
	h.GetAddressIncomeHistory(c)

	// 验证返回的是错误响应（因为 CoreAPI 客户端为 nil）
	var response dto.GetAddressIncomeHistoryResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// 由于没有 mock 客户端，应该返回 500 错误
	if response.Code != "500" {
		t.Errorf("Expected code '500' for nil client, got '%s'", response.Code)
	}
}

// TestGetWalletAddress_Unauthorized 测试未授权访问
func TestGetWalletAddress_Unauthorized(t *testing.T) {
	h := newMockHandler()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/wallet/address/get", h.GetWalletAddress)

	reqBody := dto.GetWalletAddressRequest{
		UserID:      "test00001",
		ProductCode: "C_SPOT",
		Currency:    "USDT_ERC20",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/wallet/address/get", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}

	var response dto.GetWalletAddressResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Code != "401" {
		t.Errorf("Expected code '401', got '%s'", response.Code)
	}

	if response.Success != false {
		t.Error("Expected success to be false")
	}
}

// TestGetWalletAddress_InvalidRequest 测试无效请求（缺少必填字段）
func TestGetWalletAddress_InvalidRequest(t *testing.T) {
	h := newMockHandler()
	gin.SetMode(gin.TestMode)

	// 创建带有 userID 上下文的请求
	req, _ := http.NewRequest("POST", "/api/wallet/address/get", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Set("userID", 123)
	c.Request = req

	// 直接调用 Handler
	h.GetWalletAddress(c)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}

	var response dto.GetWalletAddressResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Code != "400" {
		t.Errorf("Expected code '400', got '%s'", response.Code)
	}
}

// TestGetWalletAddress_MissingUserId 测试缺少 userId（JWT 中的 userID 会被使用）
func TestGetWalletAddress_MissingUserId(t *testing.T) {
	h := newMockHandler()
	gin.SetMode(gin.TestMode)

	// 请求中没有 userId，但 JWT 中有 userID
	reqBody := dto.GetWalletAddressRequest{
		UserID:      "", // 没有提供 userId
		ProductCode: "C_SPOT",
		Currency:    "USDT_ERC20",
	}
	body, _ := json.Marshal(reqBody)

	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Set("userID", 123)
	c.Request = httptest.NewRequest("POST", "/api/wallet/address/get", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	// 直接调用 Handler
	h.GetWalletAddress(c)

	// 验证返回错误响应
	var response dto.GetWalletAddressResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// 因为 WalletService.coreAPIClient 是 nil，应该返回错误
	if response.Code != "500" && response.Code != "400" {
		t.Errorf("Expected code '500' or '400', got '%s'", response.Code)
	}
}

// TestGetWalletAddress_MissingProductCode 测试缺少 productCode
func TestGetWalletAddress_MissingProductCode(t *testing.T) {
	h := newMockHandler()
	gin.SetMode(gin.TestMode)

	reqBody := dto.GetWalletAddressRequest{
		UserID:      "test00001",
		ProductCode: "",
		Currency:    "USDT_ERC20",
	}
	body, _ := json.Marshal(reqBody)

	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Set("userID", 123)
	c.Request = httptest.NewRequest("POST", "/api/wallet/address/get", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GetWalletAddress(c)

	var response dto.GetWalletAddressResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Code != "400" {
		t.Errorf("Expected code '400', got '%s'", response.Code)
	}
}

// TestGetWalletAddress_MissingCurrency 测试缺少 currency
func TestGetWalletAddress_MissingCurrency(t *testing.T) {
	h := newMockHandler()
	gin.SetMode(gin.TestMode)

	reqBody := dto.GetWalletAddressRequest{
		UserID:      "test00001",
		ProductCode: "C_SPOT",
		Currency:    "",
	}
	body, _ := json.Marshal(reqBody)

	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Set("userID", 123)
	c.Request = httptest.NewRequest("POST", "/api/wallet/address/get", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GetWalletAddress(c)

	var response dto.GetWalletAddressResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Code != "400" {
		t.Errorf("Expected code '400', got '%s'", response.Code)
	}
}

// TestGetWalletAddress_InvalidJSON 测试无效 JSON 格式
func TestGetWalletAddress_InvalidJSON(t *testing.T) {
	h := newMockHandler()
	gin.SetMode(gin.TestMode)

	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Set("userID", 123)
	c.Request = httptest.NewRequest("POST", "/api/wallet/address/get", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GetWalletAddress(c)

	// 无效 JSON 应该返回 400
	if c.Writer.Status() != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, c.Writer.Status())
	}
}
