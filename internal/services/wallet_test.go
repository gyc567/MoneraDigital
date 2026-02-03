package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"monera-digital/internal/coreapi"
	"monera-digital/internal/logger"
	"monera-digital/internal/models"
)

func init() {
	_ = logger.Init("test")
}

func TestWalletService_AddAddress_Success(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()
	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		RequestID:   "req-123",
		UserID:      1,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		WalletID:    sql.NullString{String: "wallet-123", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Setup expectations - wallet exists, no existing address for this currency
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 1, "USDT_TRC20").Return(nil, nil)
	mockCoreAPI.On("GetAddress", mock.Anything, mock.Anything).Return(&coreapi.AddressInfo{
		Address: "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
	}, nil)
	mockRepo.On("AddUserWalletAddress", mock.Anything, mock.Anything).Return(&models.UserWallet{
		ID:        10,
		UserID:    1,
		WalletID:  "wallet-123",
		Currency:  "USDT_TRC20",
		Address:   "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
		Status:    models.UserWalletStatusNormal,
		IsPrimary: false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW", result.Address)
	assert.Equal(t, "USDT_TRC20", result.Currency)
	mockCoreAPI.AssertCalled(t, "GetAddress", mock.Anything, mock.Anything)
	mockRepo.AssertCalled(t, "AddUserWalletAddress", mock.Anything, mock.Anything)
}

func TestWalletService_AddAddress_AlreadyExists(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	service := NewWalletService(mockRepo, nil)

	now := time.Now()
	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		UserID:      1,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)

	// Address already exists for this currency
	existingUserWallet := &models.UserWallet{
		ID:        5,
		UserID:    1,
		WalletID:  "wallet-123",
		Currency:  "USDT_TRC20",
		Address:   "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
		Status:    models.UserWalletStatusNormal,
		IsPrimary: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 1, "USDT_TRC20").Return(existingUserWallet, nil)

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW", result.Address)
	assert.Equal(t, "USDT_TRC20", result.Currency)
}

func TestWalletService_AddAddress_WalletNotFound(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	service := NewWalletService(mockRepo, nil)

	// No wallet found
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(nil, nil)
	mockRepo.On("GetActiveUserWallet", mock.Anything, 1).Return(nil, nil)

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "wallet not found")
}

func TestWalletService_AddAddress_CoreAPIFails(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()
	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		RequestID:   "req-123",
		UserID:      1,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		WalletID:    sql.NullString{String: "wallet-123", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 1, "USDT_TRC20").Return(nil, nil)
	mockCoreAPI.On("GetAddress", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Core API unavailable"))

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get address from Core API")
}

func TestWalletService_AddAddress_UserWalletFallback(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)
	now := time.Now()

	// wallet_creation_requests returns nil (no active request)
	// but user_wallets has an active wallet
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(nil, nil)
	mockRepo.On("GetActiveUserWallet", mock.Anything, 1).Return(&models.UserWallet{
		ID:        1,
		UserID:    1,
		WalletID:  "wallet123",
		Currency:  "USDT_TRC20",
		Address:   "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
		Status:    models.UserWalletStatusNormal,
		IsPrimary: true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	// Request for USDT_ERC20 - should call Core API since it doesn't exist yet
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 1, "USDT_ERC20").Return(nil, nil)
	mockCoreAPI.On("GetAddress", mock.Anything, mock.Anything).Return(&coreapi.AddressInfo{
		Address: "0xTMuA6YqfCeX8EhbfYEg5y7S4DqzSJireY9",
	}, nil)
	mockRepo.On("AddUserWalletAddress", mock.Anything, mock.Anything).Return(&models.UserWallet{
		ID:        10,
		UserID:    1,
		WalletID:  "wallet123",
		Currency:  "USDT_ERC20",
		Address:   "0xTMuA6YqfCeX8EhbfYEg5y7S4DqzSJireY9",
		Status:    models.UserWalletStatusNormal,
		IsPrimary: false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "ERC20",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "0xTMuA6YqfCeX8EhbfYEg5y7S4DqzSJireY9", result.Address)
	mockCoreAPI.AssertCalled(t, "GetAddress", mock.Anything, mock.Anything)
	mockRepo.AssertCalled(t, "AddUserWalletAddress", mock.Anything, mock.Anything)
}

func TestBuildCurrencyKey(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		network  string
		expected string
	}{
		{
			name:     "USDT on ERC20",
			token:    "USDT",
			network:  "ERC20",
			expected: "USDT_ERC20",
		},
		{
			name:     "USDT on TRC20",
			token:    "USDT",
			network:  "TRC20",
			expected: "USDT_TRC20",
		},
		{
			name:     "USDT on BEP20",
			token:    "USDT",
			network:  "BEP20",
			expected: "USDT_BEP20",
		},
		{
			name:     "USDC on ERC20",
			token:    "USDC",
			network:  "ERC20",
			expected: "USDC_ERC20",
		},
		{
			name:     "USDC on TRC20",
			token:    "USDC",
			network:  "TRC20",
			expected: "USDC_TRC20",
		},
		{
			name:     "USDC on BEP20",
			token:    "USDC",
			network:  "BEP20",
			expected: "USDC_BEP20",
		},
		{
			name:     "Invalid token",
			token:    "INVALID",
			network:  "ERC20",
			expected: "",
		},
		{
			name:     "Invalid network",
			token:    "USDT",
			network:  "INVALID",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCurrencyKey(tt.token, tt.network)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWalletService_AddAddress_EmptyProductCode_UsesDefault(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()

	// Simulate wallet with empty ProductCode in database
	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		RequestID:   "req-123",
		UserID:      1,
		ProductCode: "", // Empty in database!
		Status:      models.WalletCreationStatusSuccess,
		WalletID:    sql.NullString{String: "wallet-123", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Setup expectations
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 1, "USDT_TRC20").Return(nil, nil)

	// Capture the actual request to verify ProductCode defaults to X_FINANCE
	var capturedRequest coreapi.GetAddressRequest
	mockCoreAPI.On("GetAddress", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		capturedRequest = args.Get(1).(coreapi.GetAddressRequest)
	}).Return(&coreapi.AddressInfo{
		Address: "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
	}, nil)

	mockRepo.On("AddUserWalletAddress", mock.Anything, mock.Anything).Return(&models.UserWallet{
		ID:        10,
		UserID:    1,
		WalletID:  "wallet-123",
		Currency:  "USDT_TRC20",
		Address:   "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
		Status:    models.UserWalletStatusNormal,
		IsPrimary: false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW", result.Address)

	// Verify that ProductCode defaults to X_FINANCE when empty
	mockCoreAPI.AssertCalled(t, "GetAddress", mock.Anything, mock.Anything)
	assert.Equal(t, "X_FINANCE", capturedRequest.ProductCode, "ProductCode should default to X_FINANCE when wallet.ProductCode is empty")
}

func TestWalletService_AddAddress_UserWalletFallback_UsesDefault(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()

	// First call returns nil (no wallet_creation_requests)
	// Second call returns user_wallet (which doesn't have ProductCode at all)
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(nil, nil)
	mockRepo.On("GetActiveUserWallet", mock.Anything, 1).Return(&models.UserWallet{
		ID:        1,
		UserID:    1,
		WalletID:  "wallet123",
		Currency:  "USDT_TRC20",
		Address:   "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
		Status:    models.UserWalletStatusNormal,
		IsPrimary: true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	// Request for USDT_ERC20
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 1, "USDT_ERC20").Return(nil, nil)

	// Capture the actual request
	var capturedRequest coreapi.GetAddressRequest
	mockCoreAPI.On("GetAddress", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		capturedRequest = args.Get(1).(coreapi.GetAddressRequest)
	}).Return(&coreapi.AddressInfo{
		Address: "0xTMuA6YqfCeX8EhbfYEg5y7S4DqzSJireY9",
	}, nil)

	mockRepo.On("AddUserWalletAddress", mock.Anything, mock.Anything).Return(&models.UserWallet{
		ID:        10,
		UserID:    1,
		WalletID:  "wallet123",
		Currency:  "USDT_ERC20",
		Address:   "0xTMuA6YqfCeX8EhbfYEg5y7S4DqzSJireY9",
		Status:    models.UserWalletStatusNormal,
		IsPrimary: false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "ERC20",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify that ProductCode defaults to X_FINANCE
	mockCoreAPI.AssertCalled(t, "GetAddress", mock.Anything, mock.Anything)
	assert.Equal(t, "X_FINANCE", capturedRequest.ProductCode, "ProductCode should default to X_FINANCE for user_wallets fallback")
}

func TestWalletService_AddAddress_WithOptionalFields(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()
	addressType := "TRC20"
	derivePath := "m/44'/195'/0'/0/0"

	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		RequestID:   "req-123",
		UserID:      1,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		WalletID:    sql.NullString{String: "wallet-123", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 1, "USDT_TRC20").Return(nil, nil)
	mockCoreAPI.On("GetAddress", mock.Anything, mock.Anything).Return(&coreapi.AddressInfo{
		Address:     "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
		AddressType: &addressType,
		DerivePath:  &derivePath,
	}, nil)
	mockRepo.On("AddUserWalletAddress", mock.Anything, mock.Anything).Return(&models.UserWallet{
		ID:          10,
		UserID:      1,
		WalletID:    "wallet-123",
		Currency:    "USDT_TRC20",
		Address:     "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
		AddressType: sql.NullString{String: "TRC20", Valid: true},
		DerivePath:  sql.NullString{String: "m/44'/195'/0'/0/0", Valid: true},
		Status:      models.UserWalletStatusNormal,
		IsPrimary:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil)

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW", result.Address)
	mockRepo.AssertCalled(t, "AddUserWalletAddress", mock.Anything, mock.Anything)
}

func TestWalletService_AddAddress_WithRequestID(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()

	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		RequestID:   "req-123",
		UserID:      1,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		WalletID:    sql.NullString{String: "wallet-123", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 1, "USDT_TRC20").Return(nil, nil)
	mockCoreAPI.On("GetAddress", mock.Anything, mock.Anything).Return(&coreapi.AddressInfo{
		Address: "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
	}, nil)
	mockRepo.On("AddUserWalletAddress", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		wallet := args.Get(1).(*models.UserWallet)
		assert.Equal(t, "req-123", wallet.RequestID.String)
		assert.True(t, wallet.RequestID.Valid)
	}).Return(&models.UserWallet{
		ID:        10,
		UserID:    1,
		WalletID:  "wallet-123",
		Currency:  "USDT_TRC20",
		Address:   "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
		RequestID: sql.NullString{String: "req-123", Valid: true},
		Status:    models.UserWalletStatusNormal,
		IsPrimary: false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW", result.Address)
	mockRepo.AssertCalled(t, "AddUserWalletAddress", mock.Anything, mock.Anything)
}

func TestWalletService_AddAddress_AddUserWalletAddressFails(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()

	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		RequestID:   "req-123",
		UserID:      1,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		WalletID:    sql.NullString{String: "wallet-123", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 1, "USDT_TRC20").Return(nil, nil)
	mockCoreAPI.On("GetAddress", mock.Anything, mock.Anything).Return(&coreapi.AddressInfo{
		Address: "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
	}, nil)
	mockRepo.On("AddUserWalletAddress", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("database error"))

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")
}

func TestWalletService_AddAddress_UserWalletNil(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()

	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		RequestID:   "req-123",
		UserID:      1,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		WalletID:    sql.NullString{String: "wallet-123", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 1, "USDT_TRC20").Return(nil, nil)
	mockCoreAPI.On("GetAddress", mock.Anything, mock.Anything).Return(&coreapi.AddressInfo{
		Address: "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
	}, nil)
	mockRepo.On("AddUserWalletAddress", mock.Anything, mock.Anything).Return(&models.UserWallet{
		ID:        10,
		UserID:    1,
		WalletID:  "wallet-123",
		Currency:  "USDT_TRC20",
		Address:   "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
		Status:    models.UserWalletStatusNormal,
		IsPrimary: false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW", result.Address)
}
