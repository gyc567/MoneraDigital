package services

import (
	"context"
	"monera-digital/internal/models"
	"monera-digital/internal/repository"
	"testing"
	"time"
)

// MockWalletRepository implements WalletRepository interface for testing
type MockWalletRepositoryUnique struct {
	// Stores wallets by ID
	wallets map[int]*models.WalletCreationRequest
	// Stores wallets by UserID for GetRequestByUserID
	userWallets map[int]*models.WalletCreationRequest
	// Stores wallets by UserID+ProductCode+Currency
	userProductWallets map[string]*models.WalletCreationRequest
}

func NewMockWalletRepositoryUnique() *MockWalletRepositoryUnique {
	return &MockWalletRepositoryUnique{
		wallets:            make(map[int]*models.WalletCreationRequest),
		userWallets:        make(map[int]*models.WalletCreationRequest),
		userProductWallets: make(map[string]*models.WalletCreationRequest),
	}
}

func (m *MockWalletRepositoryUnique) CreateRequest(ctx context.Context, req *models.WalletCreationRequest) error {
	req.ID = len(m.wallets) + 1
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	// Set status to Success for test simulation if not specified
	if req.Status == "" {
		req.Status = models.WalletCreationStatusSuccess
	}
	m.wallets[req.ID] = req
	m.userWallets[req.UserID] = req

	if req.ProductCode != "" && req.Currency != "" {
		key := getKey(req.UserID, req.ProductCode, req.Currency)
		m.userProductWallets[key] = req
	}
	return nil
}

func (m *MockWalletRepositoryUnique) GetRequestByUserID(ctx context.Context, userID int) (*models.WalletCreationRequest, error) {
	if req, ok := m.userWallets[userID]; ok {
		return req, nil
	}
	return nil, nil
}

func (m *MockWalletRepositoryUnique) GetRequestByID(ctx context.Context, id int) (*models.WalletCreationRequest, error) {
	if req, ok := m.wallets[id]; ok {
		return req, nil
	}
	return nil, nil
}

func (m *MockWalletRepositoryUnique) UpdateRequest(ctx context.Context, req *models.WalletCreationRequest) error {
	req.UpdatedAt = time.Now()
	m.wallets[req.ID] = req
	m.userWallets[req.UserID] = req

	if req.ProductCode != "" && req.Currency != "" {
		key := getKey(req.UserID, req.ProductCode, req.Currency)
		m.userProductWallets[key] = req
	}
	return nil
}

func (m *MockWalletRepositoryUnique) GetWalletByUserProductCurrency(ctx context.Context, userID int, productCode, currency string) (*models.WalletCreationRequest, error) {
	key := getKey(userID, productCode, currency)
	if req, ok := m.userProductWallets[key]; ok {
		return req, nil
	}
	return nil, nil
}

func (m *MockWalletRepositoryUnique) GetActiveWalletByUserID(ctx context.Context, userID int) (*models.WalletCreationRequest, error) {
	// For mock purpose, just return what GetRequestByUserID returns
	return m.GetRequestByUserID(ctx, userID)
}

func getKey(userID int, productCode, currency string) string {
	return string(rune(userID)) + "-" + productCode + "-" + currency
}

// Interface check
var _ repository.Wallet = (*MockWalletRepositoryUnique)(nil)

func TestCreateWallet_UniqueCheck(t *testing.T) {
	repo := NewMockWalletRepositoryUnique()
	service := NewWalletService(repo, nil)

	// Create first wallet
	ctx := context.Background()
	w1, err := service.CreateWallet(ctx, 1, "BANK_ACCOUNT", "USD")
	if err != nil {
		t.Fatalf("Failed to create first wallet: %v", err)
	}
	// Manually set status to Success for mock because async task won't run in unit test environment effectively
	w1.Status = models.WalletCreationStatusSuccess
	repo.UpdateRequest(ctx, w1)

	// Try to create duplicate wallet (same user, product, currency)
	// Should return existing wallet, not create new one
	wallet2, err := service.CreateWallet(ctx, 1, "BANK_ACCOUNT", "USD")
	if err != nil {
		t.Fatalf("Failed to request duplicate wallet: %v", err)
	}

	if wallet2.ID != 1 {
		t.Errorf("Expected wallet ID 1, got %d", wallet2.ID)
	}

	// Try to create different wallet (same user, different currency)
	wallet3, err := service.CreateWallet(ctx, 1, "BANK_ACCOUNT", "EUR")
	if err != nil {
		t.Fatalf("Failed to create different wallet: %v", err)
	}

	if wallet3.ID == 1 {
		t.Errorf("Expected new wallet ID, got 1")
	}
}
