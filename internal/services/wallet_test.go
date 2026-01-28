package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"monera-digital/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWalletService_CreateWallet_New(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	service := NewWalletService(mockRepo)

	// Setup expectations
	mockRepo.On("GetWalletByUserProductCurrency", mock.Anything, 1, "C_SPOT", "USDT_ERC20").Return(nil, nil)
	mockRepo.On("CreateRequest", mock.Anything, mock.AnythingOfType("*models.WalletCreationRequest")).Return(nil)

	req, err := service.CreateWallet(context.Background(), 1, "C_SPOT", "USDT_ERC20")

	assert.NoError(t, err)
	assert.Equal(t, models.WalletCreationStatusCreating, req.Status)
	mockRepo.AssertCalled(t, "CreateRequest", mock.Anything, mock.MatchedBy(func(r *models.WalletCreationRequest) bool {
		return r.UserID == 1
	}))
}

func TestWalletService_CreateWallet_Existing(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	service := NewWalletService(mockRepo)

	now := time.Now()
	existingReq := &models.WalletCreationRequest{
		ID:        1,
		UserID:    1,
		Status:    models.WalletCreationStatusSuccess,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Setup expectations - existing success wallet
	mockRepo.On("GetWalletByUserProductCurrency", mock.Anything, 1, "C_SPOT", "USDT_ERC20").Return(existingReq, nil)

	req, err := service.CreateWallet(context.Background(), 1, "C_SPOT", "USDT_ERC20")

	assert.NoError(t, err)
	assert.Equal(t, existingReq, req)
	mockRepo.AssertNotCalled(t, "CreateRequest")
}

func TestWalletService_CreateWallet_WithProductAndCurrency(t *testing.T) {
	testCases := []struct {
		name        string
		productCode string
		currency    string
	}{
		{"USDT ERC20", "C_SPOT", "USDT_ERC20"},
		{"USDT TRC20", "C_SPOT", "USDT_TRC20"},
		{"USDT BSC", "C_SPOT", "USDT_BSC"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(MockWalletRepository)
			service := NewWalletService(mockRepo)

			mockRepo.On("GetWalletByUserProductCurrency", mock.Anything, 1, tc.productCode, tc.currency).Return(nil, nil)
			mockRepo.On("CreateRequest", mock.Anything, mock.AnythingOfType("*models.WalletCreationRequest")).Return(nil)

			req, err := service.CreateWallet(context.Background(), 1, tc.productCode, tc.currency)

			assert.NoError(t, err)
			assert.Equal(t, models.WalletCreationStatusCreating, req.Status)
		})
	}
}

func TestWalletService_GetWalletInfo_Success(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	service := NewWalletService(mockRepo)

	now := time.Now()
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(&models.WalletCreationRequest{
		ID: 1, UserID: 1, Status: models.WalletCreationStatusSuccess, Address: sql.NullString{String: "0x...", Valid: true}, CreatedAt: now, UpdatedAt: now,
	}, nil)

	info, err := service.GetWalletInfo(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, "0x...", info.Address.String)
}

func TestWalletService_GetWalletInfo_Pending(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	service := NewWalletService(mockRepo)

	now := time.Now()
	pendingReq := &models.WalletCreationRequest{
		ID:        2,
		UserID:    1,
		Status:    models.WalletCreationStatusCreating,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// No active wallet, but has pending request
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(nil, nil)
	mockRepo.On("GetRequestByUserID", mock.Anything, 1).Return(pendingReq, nil)

	info, err := service.GetWalletInfo(context.Background(), 1)
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, models.WalletCreationStatusCreating, info.Status)
}

func TestWalletService_GetWalletInfo_NotFound(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	service := NewWalletService(mockRepo)

	// No active wallet and no pending request
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(nil, nil)
	mockRepo.On("GetRequestByUserID", mock.Anything, 1).Return(nil, nil)

	info, err := service.GetWalletInfo(context.Background(), 1)
	assert.NoError(t, err)
	assert.Nil(t, info)
}
