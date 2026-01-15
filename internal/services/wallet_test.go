package services

import (
	"context"
	"testing"
	"monera-digital/internal/models"
	"monera-digital/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"time"
)

func TestWalletService_CreateWallet_New(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	service := NewWalletService(mockRepo)

	// Setup expectations
	mockRepo.On("GetRequestByUserID", mock.Anything, 1).Return(nil, nil)
	mockRepo.On("CreateRequest", mock.Anything, mock.AnythingOfType("*repository.WalletCreationRequestModel")).Return(nil)
	
	req, err := service.CreateWallet(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, models.WalletCreationStatusCreating, req.Status)
	mockRepo.AssertCalled(t, "CreateRequest", mock.Anything, mock.MatchedBy(func(r *repository.WalletCreationRequestModel) bool {
		return r.UserID == 1
	}))
}

func TestWalletService_GetWalletInfo_Success(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	service := NewWalletService(mockRepo)

	now := time.Now().Format(time.RFC3339)
	mockRepo.On("GetActiveWalletByUserID", mock.Anything, 1).Return(&repository.WalletCreationRequestModel{
		ID: 1, UserID: 1, Status: "SUCCESS", Address: "0x...", CreatedAt: now, UpdatedAt: now,
	}, nil)

	info, err := service.GetWalletInfo(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, "0x...", info.Address.String)
}
