package services

import (
	"context"
	"monera-digital/internal/repository"
	"github.com/stretchr/testify/mock"
)

// MockDepositRepository
type MockDepositRepository struct {
	mock.Mock
}

func (m *MockDepositRepository) Create(ctx context.Context, deposit *repository.DepositModel) error {
	args := m.Called(ctx, deposit)
    if args.Get(0) == nil {
        deposit.ID = 1 
    }
	return args.Error(0)
}

func (m *MockDepositRepository) GetByTxHash(ctx context.Context, txHash string) (*repository.DepositModel, error) {
	args := m.Called(ctx, txHash)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
	return args.Get(0).(*repository.DepositModel), args.Error(1)
}

func (m *MockDepositRepository) GetByUserID(ctx context.Context, userID int, limit, offset int) ([]*repository.DepositModel, int64, error) {
	args := m.Called(ctx, userID, limit, offset)
    if args.Get(0) == nil {
        return nil, 0, args.Error(2)
    }
	return args.Get(0).([]*repository.DepositModel), args.Get(1).(int64), args.Error(2)
}

func (m *MockDepositRepository) UpdateStatus(ctx context.Context, id int, status string, confirmedAt string) error {
	args := m.Called(ctx, id, status, confirmedAt)
	return args.Error(0)
}

// MockWalletRepository
type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) CreateRequest(ctx context.Context, req *repository.WalletCreationRequestModel) error {
	args := m.Called(ctx, req)
    if args.Get(0) == nil {
        req.ID = 1
    }
	return args.Error(0)
}

func (m *MockWalletRepository) GetRequestByUserID(ctx context.Context, userID int) (*repository.WalletCreationRequestModel, error) {
	args := m.Called(ctx, userID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
	return args.Get(0).(*repository.WalletCreationRequestModel), args.Error(1)
}

func (m *MockWalletRepository) UpdateRequest(ctx context.Context, req *repository.WalletCreationRequestModel) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockWalletRepository) GetActiveWalletByUserID(ctx context.Context, userID int) (*repository.WalletCreationRequestModel, error) {
	args := m.Called(ctx, userID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
	return args.Get(0).(*repository.WalletCreationRequestModel), args.Error(1)
}
