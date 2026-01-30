package coreapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetIncomeHistory_Success 测试成功获取收入历史
func TestGetIncomeHistory_Success(t *testing.T) {
	// 创建模拟服务器
	incomeRecords := []AddressIncomeRecord{
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
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/wallet/address/incomeHistory" {
			t.Errorf("Expected path /api/v1/wallet/address/incomeHistory, got %s", r.URL.Path)
		}

		// 验证请求头
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// 返回成功响应
		response := CoreAPIResponse{
			Success: true,
			Data:    incomeRecords,
			Message: "成功",
			Code:    "200",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(server.URL)

	// 调用 API
	result, err := client.GetIncomeHistory(context.Background(), GetIncomeHistoryRequest{
		Address: "TEJG77HBANUVgL2rJ34NPR27vb56TkuUVV",
	})

	// 验证结果
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 record, got %d", len(result))
	}

	if result[0].TxKey != "txsjujr2a631c3d5a4p71adcfee4802001" {
		t.Errorf("Expected txKey 'txsjujr2a631c3d5a4p71adcfee4802001', got %s", result[0].TxKey)
	}

	if result[0].TxAmount != "2000.00000000" {
		t.Errorf("Expected txAmount '2000.00000000', got %s", result[0].TxAmount)
	}
}

// TestGetIncomeHistory_APIError 测试 API 返回错误
func TestGetIncomeHistory_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回失败响应
		response := CoreAPIResponse{
			Success: false,
			Message: "Address not found",
			Code:    "400",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	_, err := client.GetIncomeHistory(context.Background(), GetIncomeHistoryRequest{
		Address: "INVALID_ADDRESS",
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	expectedMsg := "income history query failed: Address not found"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestGetIncomeHistory_InvalidResponse 测试无效响应格式
func TestGetIncomeHistory_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回无效的 JSON
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)

	_, err := client.GetIncomeHistory(context.Background(), GetIncomeHistoryRequest{
		Address: "TEJG77HBANUVgL2rJ34NPR27vb56TkuUVV",
	})

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

// TestGetIncomeHistory_HTTPError 测试 HTTP 错误
func TestGetIncomeHistory_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	server.Close()

	client := NewClient("http://invalid-server:9999")

	_, err := client.GetIncomeHistory(context.Background(), GetIncomeHistoryRequest{
		Address: "TEJG77HBANUVgL2rJ34NPR27vb56TkuUVV",
	})

	if err == nil {
		t.Error("Expected error for HTTP failure, got nil")
	}
}

// TestGetAddress_Success 测试成功获取钱包地址
func TestGetAddress_Success(t *testing.T) {
	addressType := "TRC20"
	derivePath := "m/44'/195'/0'/0/0"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/wallet/address/get" {
			t.Errorf("Expected path /api/v1/wallet/address/get, got %s", r.URL.Path)
		}

		// 验证请求头
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// 返回成功响应
		response := CoreAPIResponse{
			Success: true,
			Data: AddressInfo{
				Address:     "0x189161Bed2D4c99FDB9D8ebA8B9eFAaBAB123246",
				AddressType: &addressType,
				DerivePath:  &derivePath,
			},
			Message: "成功",
			Code:    "200",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(server.URL)

	// 调用 API
	result, err := client.GetAddress(context.Background(), GetAddressRequest{
		UserID:      123,
		ProductCode: "C_SPOT",
		Currency:    "USDT_ERC20",
	})

	// 验证结果
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Address != "0x189161Bed2D4c99FDB9D8ebA8B9eFAaBAB123246" {
		t.Errorf("Expected address '0x189161Bed2D4c99FDB9D8ebA8B9eFAaBAB123246', got %s", result.Address)
	}

	if result.AddressType == nil || *result.AddressType != "TRC20" {
		t.Error("Expected addressType 'TRC20'")
	}

	if result.DerivePath == nil || *result.DerivePath != "m/44'/195'/0'/0/0" {
		t.Error("Expected derivePath 'm/44'/195'/0'/0/0'")
	}
}

// TestGetAddress_APIError 测试 API 返回错误
func TestGetAddress_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回失败响应
		response := CoreAPIResponse{
			Success: false,
			Message: "User not found",
			Code:    "400",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	_, err := client.GetAddress(context.Background(), GetAddressRequest{
		UserID:      999,
		ProductCode: "C_SPOT",
		Currency:    "USDT_ERC20",
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	expectedMsg := "get address failed: User not found"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestGetAddress_InvalidResponse 测试无效响应格式
func TestGetAddress_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回无效的 JSON
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)

	_, err := client.GetAddress(context.Background(), GetAddressRequest{
		UserID:      123,
		ProductCode: "C_SPOT",
		Currency:    "USDT_ERC20",
	})

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

// TestGetAddress_HTTPError 测试 HTTP 错误
func TestGetAddress_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	server.Close()

	client := NewClient("http://invalid-server:9999")

	_, err := client.GetAddress(context.Background(), GetAddressRequest{
		UserID:      123,
		ProductCode: "C_SPOT",
		Currency:    "USDT_ERC20",
	})

	if err == nil {
		t.Error("Expected error for HTTP failure, got nil")
	}
}
