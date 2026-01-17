package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"monera-digital/internal/models"
	"monera-digital/internal/repository"
)

// MockSafeheronService
type MockSafeheronService struct {
	mock.Mock
}

func (m *MockSafeheronService) Withdraw(ctx context.Context, req SafeheronWithdrawalRequest) (*SafeheronWithdrawalResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SafeheronWithdrawalResponse), args.Error(1)
}

func TestWithdrawalService_CreateWithdrawal(t *testing.T) {
	// Setup Mocks
	mockAccountRepo := new(MockAccountRepository)
	mockAddressRepo := new(MockAddressRepository)
	mockWithdrawalRepo := new(MockWithdrawalRepository)
	mockSafeheron := new(MockSafeheronService)

	// Combine into Repository
	repo := &repository.Repository{
		Account:    mockAccountRepo,
		Address:    mockAddressRepo,
		Withdrawal: mockWithdrawalRepo,
	}

	service := NewWithdrawalService(nil, repo, mockSafeheron)

	ctx := context.Background()
	userID := 1
	req := models.CreateWithdrawalRequest{
		AddressID: 10,
		Amount:    "100.0",
		Asset:     "USDT",
	}

	// Mock Data
	account := &models.Account{
		UserID:        userID,
		Balance:       200.0,
		FrozenBalance: 0.0,
	}
	address := &models.WithdrawalAddress{
		ID:            10,
		UserID:        userID,
		ChainType:     "TRC20",
		WalletAddress: "Txyz...",
	}
	shResp := &SafeheronWithdrawalResponse{
		TxHash:           "0xtx",
		SafeheronOrderID: "sh-123",
		NetworkFee:       "1.0",
	}
	createdOrder := &models.WithdrawalOrder{ID: 1}

	// Expectations
	mockAccountRepo.On("GetByUserIDAndType", ctx, userID, "WEALTH").Return(account, nil)
	mockAddressRepo.On("GetAddressByID", ctx, 10).Return(address, nil)
	mockAccountRepo.On("UpdateFrozenBalance", ctx, userID, 100.0).Return(nil)
	
	mockSafeheron.On("Withdraw", ctx, mock.MatchedBy(func(r SafeheronWithdrawalRequest) bool {
		return r.Amount == "100.0" && r.ToAddress == "Txyz..."
	})).Return(shResp, nil)

	mockAccountRepo.On("DeductBalance", ctx, userID, 100.0).Return(nil)
	mockWithdrawalRepo.On("CreateOrder", ctx, mock.Anything).Return(createdOrder, nil)

	// Execute
	order, err := service.CreateWithdrawal(ctx, userID, req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 1, order.ID)

	mockAccountRepo.AssertExpectations(t)
	mockAddressRepo.AssertExpectations(t)
	mockSafeheron.AssertExpectations(t)
	mockWithdrawalRepo.AssertExpectations(t)
}

func TestWithdrawalService_CreateWithdrawal_InsufficientBalance(t *testing.T) {
	mockAccountRepo := new(MockAccountRepository)
	repo := &repository.Repository{Account: mockAccountRepo}
	service := NewWithdrawalService(nil, repo, nil)

	ctx := context.Background()
	userID := 1
	req := models.CreateWithdrawalRequest{
		AddressID: 10,
		Amount:    "100.0",
		Asset:     "USDT",
	}

	account := &models.Account{
		UserID:        userID,
		Balance:       50.0,
		FrozenBalance: 0.0,
	}

	mockAccountRepo.On("GetByUserIDAndType", ctx, userID, "WEALTH").Return(account, nil)

	_, err := service.CreateWithdrawal(ctx, userID, req)
	assert.Error(t, err)
	assert.Equal(t, "insufficient balance", err.Error())
}
