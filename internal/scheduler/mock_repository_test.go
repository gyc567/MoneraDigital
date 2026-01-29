package scheduler

import (
	"context"
	"monera-digital/internal/repository"

	"github.com/stretchr/testify/mock"
)

type MockWealthRepository struct {
	mock.Mock
}

func (m *MockWealthRepository) GetActiveProducts(ctx context.Context) ([]*repository.WealthProductModel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.WealthProductModel), args.Error(1)
}

func (m *MockWealthRepository) GetProductByID(ctx context.Context, id int64) (*repository.WealthProductModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.WealthProductModel), args.Error(1)
}

func (m *MockWealthRepository) CreateOrder(ctx context.Context, order *repository.WealthOrderModel) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockWealthRepository) GetOrdersByUserID(ctx context.Context, userID int64) ([]*repository.WealthOrderModel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.WealthOrderModel), args.Error(1)
}

func (m *MockWealthRepository) GetOrderByID(ctx context.Context, id int64) (*repository.WealthOrderModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.WealthOrderModel), args.Error(1)
}

func (m *MockWealthRepository) UpdateOrder(ctx context.Context, order *repository.WealthOrderModel) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockWealthRepository) UpdateProductSoldQuota(ctx context.Context, id int64, amount string) error {
	args := m.Called(ctx, id, amount)
	return args.Error(0)
}

func (m *MockWealthRepository) GetActiveOrders(ctx context.Context) ([]*repository.WealthOrderModel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.WealthOrderModel), args.Error(1)
}

func (m *MockWealthRepository) GetExpiredOrders(ctx context.Context) ([]*repository.WealthOrderModel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.WealthOrderModel), args.Error(1)
}

func (m *MockWealthRepository) AccrueInterest(ctx context.Context, orderID int64, amount string, date string) error {
	args := m.Called(ctx, orderID, amount, date)
	return args.Error(0)
}

func (m *MockWealthRepository) SettleOrder(ctx context.Context, orderID int64, interestPaid string) error {
	args := m.Called(ctx, orderID, interestPaid)
	return args.Error(0)
}

func (m *MockWealthRepository) RenewOrder(ctx context.Context, order *repository.WealthOrderModel, product *repository.WealthProductModel) (*repository.WealthOrderModel, error) {
	args := m.Called(ctx, order, product)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.WealthOrderModel), args.Error(1)
}

// MockAccountRepositoryV2 implements repository.AccountV2 interface for testing
type MockAccountRepositoryV2 struct {
	mock.Mock
}

func (m *MockAccountRepositoryV2) GetAccountByUserIDAndCurrency(ctx context.Context, userID int64, currency string) (*repository.AccountModel, error) {
	args := m.Called(ctx, userID, currency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.AccountModel), args.Error(1)
}

func (m *MockAccountRepositoryV2) GetAccountsByUserID(ctx context.Context, userID int64) ([]*repository.AccountModel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.AccountModel), args.Error(1)
}

func (m *MockAccountRepositoryV2) FreezeBalance(ctx context.Context, accountID int64, amount string) error {
	args := m.Called(ctx, accountID, amount)
	return args.Error(0)
}

func (m *MockAccountRepositoryV2) UnfreezeBalance(ctx context.Context, accountID int64, amount string) error {
	args := m.Called(ctx, accountID, amount)
	return args.Error(0)
}

func (m *MockAccountRepositoryV2) DeductBalance(ctx context.Context, accountID int64, amount string) error {
	args := m.Called(ctx, accountID, amount)
	return args.Error(0)
}

func (m *MockAccountRepositoryV2) AddBalance(ctx context.Context, accountID int64, amount string) error {
	args := m.Called(ctx, accountID, amount)
	return args.Error(0)
}

type MockJournalRepository struct {
	mock.Mock
}

func (m *MockJournalRepository) CreateJournalRecord(ctx context.Context, record *repository.JournalModel) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}
