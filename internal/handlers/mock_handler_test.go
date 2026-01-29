package handlers

import (
	"context"
	"monera-digital/internal/models"
	"monera-digital/internal/repository"
	"monera-digital/internal/services"
	"time"
)

// MockWalletRepository for handler tests
type MockWalletRepository struct {
	wallets map[int]*models.WalletCreationRequest
}

func NewMockWalletRepository() *MockWalletRepository {
	return &MockWalletRepository{
		wallets: make(map[int]*models.WalletCreationRequest),
	}
}

func (m *MockWalletRepository) CreateRequest(ctx context.Context, req *models.WalletCreationRequest) error {
	req.ID = len(m.wallets) + 1
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	// Set status to Success by default for tests unless specified otherwise
	if req.Status == "" {
		req.Status = models.WalletCreationStatusSuccess
	}
	m.wallets[req.ID] = req
	return nil
}

func (m *MockWalletRepository) GetRequestByUserID(ctx context.Context, userID int) (*models.WalletCreationRequest, error) {
	return nil, nil
}

func (m *MockWalletRepository) GetRequestByID(ctx context.Context, id int) (*models.WalletCreationRequest, error) {
	return nil, nil
}

func (m *MockWalletRepository) UpdateRequest(ctx context.Context, req *models.WalletCreationRequest) error {
	m.wallets[req.ID] = req
	return nil
}

func (m *MockWalletRepository) GetWalletByUserProductCurrency(ctx context.Context, userID int, productCode, currency string) (*models.WalletCreationRequest, error) {
	// For simplicity in handler tests, always return nil (no existing wallet)
	return nil, nil
}

func (m *MockWalletRepository) GetActiveWalletByUserID(ctx context.Context, userID int) (*models.WalletCreationRequest, error) {
	return nil, nil
}

// Interface check
var _ repository.Wallet = (*MockWalletRepository)(nil)

// Setup test handler with mock service
func newMockHandler() *Handler {
	mockRepo := NewMockWalletRepository()
	walletService := services.NewWalletService(mockRepo)

	h := &Handler{
		WalletService: walletService,
	}
	return h
}
