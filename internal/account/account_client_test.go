package account

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"monera-digital/internal/dto"
	"monera-digital/internal/logger"
)

func setupTestClient(handler http.Handler) *Client {
	server := httptest.NewServer(handler)
	// The test client should point to the mock server
	client := NewClient(server.URL)
	return client
}

func TestMain(m *testing.M) {
	// Suppress logging during tests
	logger.Init("test")
	m.Run()
}

func TestGetUserAccounts_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/accounts", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "123", r.URL.Query().Get("userId"))

		resp := dto.GetUserAccountsResponse{
			Code:    "0",
			Message: "Success",
			Data: []dto.Account{
				{AccountID: "acc1"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	client := setupTestClient(handler)
	req := dto.GetUserAccountsRequest{UserID: "123"}
	resp, err := client.GetUserAccounts(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "0", resp.Code)
	assert.Len(t, resp.Data, 1)
}

func TestGetUserAccounts_Error(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	})

	client := setupTestClient(handler)
	req := dto.GetUserAccountsRequest{UserID: "123"}
	_, err := client.GetUserAccounts(context.Background(), req)

	require.Error(t, err)
	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
}

func TestCreateAccount_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/accounts", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var req dto.CreateAccountRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "123", req.UserID)

		resp := dto.CreateAccountResponse{
			Code:    "0",
			Message: "Success",
			Data:    dto.Account{AccountID: "acc2"},
		}
		json.NewEncoder(w).Encode(resp)
	})

	client := setupTestClient(handler)
	req := dto.CreateAccountRequest{UserID: "123", AccountType: "WEALTH", Currency: "USD"}
	resp, err := client.CreateAccount(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "acc2", resp.Data.AccountID)
}

// Add more tests for other methods...
func TestFreezeBalance_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/accounts/freeze", r.URL.Path)
		var req dto.FreezeBalanceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "acc1", req.AccountID)

		resp := dto.FreezeBalanceResponse{Code: "0", Data: struct {
			Success bool `json:"success"`
		}{Success: true}}
		json.NewEncoder(w).Encode(resp)
	})

	client := setupTestClient(handler)
	req := dto.FreezeBalanceRequest{AccountID: "acc1", UserID: "123", Amount: "100"}
	resp, err := client.FreezeBalance(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Data.Success)
}

func TestUnfreezeBalance_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/accounts/unfreeze", r.URL.Path)
		resp := dto.UnfreezeBalanceResponse{Code: "0", Data: struct {
			Success bool `json:"success"`
		}{Success: true}}
		json.NewEncoder(w).Encode(resp)
	})

	client := setupTestClient(handler)
	req := dto.UnfreezeBalanceRequest{AccountID: "acc1", UserID: "123", Amount: "100"}
	resp, err := client.UnfreezeBalance(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Data.Success)
}

func TestTransfer_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/accounts/transfer", r.URL.Path)
		resp := dto.TransferResponse{Code: "0", Data: struct {
			Success bool `json:"success"`
		}{Success: true}}
		json.NewEncoder(w).Encode(resp)
	})

	client := setupTestClient(handler)
	req := dto.TransferRequest{FromAccountID: "acc1", ToAccountID: "acc2", Amount: "50"}
	resp, err := client.Transfer(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Data.Success)
}
