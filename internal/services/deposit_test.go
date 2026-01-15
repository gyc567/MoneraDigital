package services

import (
	"context"
	"testing"
	"time"
	"monera-digital/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDepositService_GetDeposits(t *testing.T) {
	mockRepo := new(MockDepositRepository)
	service := NewDepositService(mockRepo)

	now := time.Now().Format(time.RFC3339)
	mockRepo.On("GetByUserID", mock.Anything, 1, 20, 0).Return([]*repository.DepositModel{
		{ID: 1, UserID: 1, Amount: "100", Asset: "USDT", Status: "CONFIRMED", CreatedAt: now},
	}, int64(1), nil)

	deposits, total, err := service.GetDeposits(context.Background(), 1, 20, 0)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, deposits, 1)
	assert.Equal(t, "100", deposits[0].Amount)
	mockRepo.AssertExpectations(t)
}
