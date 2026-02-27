package services

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"monera-digital/internal/coreapi"
	"monera-digital/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestWalletService_AddAddress_Testnet_Success tests adding a testnet address via local generation
func TestWalletService_AddAddress_Testnet_Success(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()
	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		RequestID:   "req-123",
		UserID:      123,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		WalletID:    sql.NullString{String: "wallet-123", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// For testnet, Core API should NOT be called - address is generated locally
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 123).Return(existingWallet, nil)
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 123, "USDT_TRON_TESTNET").Return(nil, nil)
	mockRepo.On("AddUserWalletAddress", mock.Anything, mock.MatchedBy(func(w *models.UserWallet) bool {
		return w.UserID == 123 &&
			w.Currency == "USDT_TRON_TESTNET" &&
			strings.HasPrefix(w.Address, "TTest") &&
			w.Status == models.UserWalletStatusNormal
	})).Return(&models.UserWallet{
		ID:        1,
		UserID:    123,
		WalletID:  "wallet-123",
		Currency:  "USDT_TRON_TESTNET",
		Address:   "TTest00000123FGHIJKLMNOPQRSTUVWXYZ",
		Status:    models.UserWalletStatusNormal,
		IsPrimary: false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	result, err := service.AddAddress(context.Background(), 123, AddAddressRequest{
		Chain: "TRX(SHASTA)_TRON_TESTNET",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "USDT_TRON_TESTNET", result.Currency)
	assert.Contains(t, result.Address, "TTest")
	// Core API should NOT be called for testnet
	mockCoreAPI.AssertNotCalled(t, "GetAddress", mock.Anything, mock.Anything)
}

// TestWalletService_AddAddress_Testnet_USDC_Success tests adding a USDC testnet address via local generation
func TestWalletService_AddAddress_Testnet_USDC_Success(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()
	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		RequestID:   "req-456",
		UserID:      456,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		WalletID:    sql.NullString{String: "wallet-456", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// For testnet, Core API should NOT be called - address is generated locally
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 456).Return(existingWallet, nil)
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 456, "USDC_TRON_TESTNET").Return(nil, nil)
	mockRepo.On("AddUserWalletAddress", mock.Anything, mock.MatchedBy(func(w *models.UserWallet) bool {
		return w.UserID == 456 &&
			w.Currency == "USDC_TRON_TESTNET" &&
			strings.HasPrefix(w.Address, "TTest") &&
			w.Status == models.UserWalletStatusNormal
	})).Return(&models.UserWallet{
		ID:        2,
		UserID:    456,
		WalletID:  "wallet-456",
		Currency:  "USDC_TRON_TESTNET",
		Address:   "TTest00000456FGHIJKLMNOPQRSTUVWXYZ",
		Status:    models.UserWalletStatusNormal,
		IsPrimary: false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	result, err := service.AddAddress(context.Background(), 456, AddAddressRequest{
		Chain: "TRX(SHASTA)_TRON_TESTNET",
		Token: "USDC",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "USDC_TRON_TESTNET", result.Currency)
	assert.Contains(t, result.Address, "TTest")
	// Core API should NOT be called for testnet
	mockCoreAPI.AssertNotCalled(t, "GetAddress", mock.Anything, mock.Anything)
}

// TestWalletService_AddAddress_Mainnet_UsesCoreAPI tests that mainnet still uses Core API
func TestWalletService_AddAddress_Mainnet_UsesCoreAPI(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()
	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		RequestID:   "req-789",
		UserID:      789,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		WalletID:    sql.NullString{String: "wallet-789", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Mock Core API to return mainnet address
	mockCoreAPI.On("GetAddress", mock.Anything, coreapi.GetAddressRequest{
		UserID:      "789",
		ProductCode: "X_FINANCE",
		Currency:    "USDT_TRC20",
	}).Return(&coreapi.AddressInfo{
		Address:     "TJ123456789ABCDEFGHIJKLMNOPQRSTUV",
		AddressType: func() *string { s := "TRC20"; return &s }(),
	}, nil)

	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 789).Return(existingWallet, nil)
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 789, "USDT_TRC20").Return(nil, nil)
	mockRepo.On("AddUserWalletAddress", mock.Anything, mock.MatchedBy(func(w *models.UserWallet) bool {
		return w.UserID == 789 &&
			w.Currency == "USDT_TRC20" &&
			w.Address == "TJ123456789ABCDEFGHIJKLMNOPQRSTUV"
	})).Return(&models.UserWallet{
		ID:        3,
		UserID:    789,
		WalletID:  "wallet-789",
		Currency:  "USDT_TRC20",
		Address:   "TJ123456789ABCDEFGHIJKLMNOPQRSTUV",
		Status:    models.UserWalletStatusNormal,
		IsPrimary: false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil)

	result, err := service.AddAddress(context.Background(), 789, AddAddressRequest{
		Chain: "TRC20",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "USDT_TRC20", result.Currency)
	assert.Equal(t, "TJ123456789ABCDEFGHIJKLMNOPQRSTUV", result.Address)
	mockCoreAPI.AssertCalled(t, "GetAddress", mock.Anything, coreapi.GetAddressRequest{
		UserID:      "789",
		ProductCode: "X_FINANCE",
		Currency:    "USDT_TRC20",
	})
}

// TestWalletService_AddAddress_Testnet_AlreadyExists tests that existing testnet addresses are regenerated locally
func TestWalletService_AddAddress_Testnet_AlreadyExists(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()
	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		RequestID:   "req-999",
		UserID:      999,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		WalletID:    sql.NullString{String: "wallet-999", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	existingAddress := &models.UserWallet{
		ID:        5,
		UserID:    999,
		WalletID:  "wallet-999",
		Currency:  "USDT_TRON_TESTNET",
		Address:   "TExistingAddress123456789012345", // Old address
		Status:    models.UserWalletStatusNormal,
		IsPrimary: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Updated address that will be returned after repository update - generated locally
	updatedAddress := &models.UserWallet{
		ID:        5,
		UserID:    999,
		WalletID:  "wallet-999",
		Currency:  "USDT_TRON_TESTNET",
		Address:   "TTest00000999FGHIJKLMNOPQRSTUVWXYZ", // New address generated locally
		Status:    models.UserWalletStatusNormal,
		IsPrimary: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// For testnet, Core API should NOT be called - address is generated locally
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 999).Return(existingWallet, nil)
	mockRepo.On("GetUserWalletByUserAndCurrency", mock.Anything, 999, "USDT_TRON_TESTNET").Return(existingAddress, nil)
	// AddUserWalletAddress should be called with locally generated address
	mockRepo.On("AddUserWalletAddress", mock.Anything, mock.MatchedBy(func(w *models.UserWallet) bool {
		return w.UserID == 999 &&
			w.Currency == "USDT_TRON_TESTNET" &&
			strings.HasPrefix(w.Address, "TTest")
	})).Return(updatedAddress, nil)

	result, err := service.AddAddress(context.Background(), 999, AddAddressRequest{
		Chain: "TRX(SHASTA)_TRON_TESTNET",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Should return locally generated address
	assert.Contains(t, result.Address, "TTest")
	// Core API should NOT be called for testnet
	mockCoreAPI.AssertNotCalled(t, "GetAddress", mock.Anything, mock.Anything)
}
