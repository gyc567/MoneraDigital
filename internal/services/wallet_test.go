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
		UserID:      1,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		Addresses:   sql.NullString{String: "{}", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Setup expectations
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)
	mockCoreAPI.On("GetAddress", mock.Anything, mock.Anything).Return(&coreapi.AddressInfo{
		Address: "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
	}, nil)
	mockRepo.On("UpdateRequest", mock.Anything, mock.AnythingOfType("*models.WalletCreationRequest")).Return(nil)

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockCoreAPI.AssertCalled(t, "GetAddress", mock.Anything, mock.Anything)
	mockRepo.AssertCalled(t, "UpdateRequest", mock.Anything, mock.Anything)
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
		Addresses:   sql.NullString{String: `{"USDT_TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW"}`, Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Setup expectations
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)

	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Should not call UpdateRequest when address already exists
	mockRepo.AssertNotCalled(t, "UpdateRequest")
}

func TestWalletService_AddAddress_CoreAPIError(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()
	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		UserID:      1,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		Addresses:   sql.NullString{String: "{}", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Setup expectations - Core API fails, should return error (no fallback)
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)
	mockCoreAPI.On("GetAddress", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Core API unavailable"))

	_, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get address from Core API")
	mockCoreAPI.AssertCalled(t, "GetAddress", mock.Anything, mock.Anything)
	// Should NOT call UpdateRequest when Core API fails
	mockRepo.AssertNotCalled(t, "UpdateRequest")
}

func TestWalletService_AddAddress_NotFound(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	service := NewWalletService(mockRepo, nil)

	// No wallet exists
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(nil, nil)

	_, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wallet not found")
}

func TestWalletService_AddAddress_InvalidJSON(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	service := NewWalletService(mockRepo, nil)

	now := time.Now()
	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		UserID:      1,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		Addresses:   sql.NullString{String: "invalid json", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Setup expectations
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)

	_, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "USDT",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse existing addresses")
}

func TestNormalizeCurrencyKey(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		chain    string
		inputKey string
		expected string
	}{
		{
			name:     "TRX on TRON maps to USDT_TRON",
			token:    "TRX",
			chain:    "TRON",
			inputKey: "TRX_TRON",
			expected: "USDT_TRON",
		},
		{
			name:     "USDT on TRON stays unchanged",
			token:    "USDT",
			chain:    "TRON",
			inputKey: "USDT_TRON",
			expected: "USDT_TRON",
		},
		{
			name:     "ETH on ERC20 stays unchanged",
			token:    "ETH",
			chain:    "ERC20",
			inputKey: "ETH_ERC20",
			expected: "ETH_ERC20",
		},
		{
			name:     "TRX on BSC stays unchanged (unsupported chain)",
			token:    "TRX",
			chain:    "BSC",
			inputKey: "TRX_BSC",
			expected: "TRX_BSC",
		},
		{
			name:     "USDC on TRON stays unchanged",
			token:    "USDC",
			chain:    "TRON",
			inputKey: "USDC_TRON",
			expected: "USDC_TRON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCurrencyKey(tt.token, tt.chain, tt.inputKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWalletService_AddAddress_TRX_TRON_Mapping(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	mockCoreAPI := new(MockCoreAPIClient)
	service := NewWalletService(mockRepo, mockCoreAPI)

	now := time.Now()
	existingWallet := &models.WalletCreationRequest{
		ID:          1,
		UserID:      1,
		ProductCode: "X_FINANCE",
		Status:      models.WalletCreationStatusSuccess,
		Addresses:   sql.NullString{String: "{}", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Setup expectations - TRX_TRON should be mapped to USDT_TRON
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(existingWallet, nil)
	// Core API should be called with USDT_TRON (not TRX_TRON)
	mockCoreAPI.On("GetAddress", mock.Anything, coreapi.GetAddressRequest{
		UserID:      1,
		ProductCode: "X_FINANCE",
		Currency:    "USDT_TRON",
	}).Return(&coreapi.AddressInfo{
		Address: "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
	}, nil)
	mockRepo.On("UpdateRequest", mock.Anything, mock.AnythingOfType("*models.WalletCreationRequest")).Return(nil)

	// Request with TRX + TRON should map to USDT_TRON
	result, err := service.AddAddress(context.Background(), 1, AddAddressRequest{
		Chain: "TRON",
		Token: "TRX",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Verify Core API was called with the normalized currency key
	mockCoreAPI.AssertCalled(t, "GetAddress", mock.Anything, coreapi.GetAddressRequest{
		UserID:      1,
		ProductCode: "X_FINANCE",
		Currency:    "USDT_TRON",
	})
}
