package services

import (
	"context"
	"database/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"monera-digital/internal/models"
	"testing"
	"time"
)

func TestWalletService_CreateWallet_New(t *testing.T) {
	mockRepo := new(MockWalletRepository)
	service := NewWalletService(mockRepo)

	// Setup expectations
	mockRepo.On("GetRequestByUserID", mock.Anything, 1).Return(nil, nil)
	mockRepo.On("CreateRequest", mock.Anything, mock.AnythingOfType("*models.WalletCreationRequest")).Return(nil)

	req, err := service.CreateWallet(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, models.WalletCreationStatusCreating, req.Status)
	mockRepo.AssertCalled(t, "CreateRequest", mock.Anything, mock.MatchedBy(func(r *models.WalletCreationRequest) bool {
		return r.UserID == 1
	}))
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
