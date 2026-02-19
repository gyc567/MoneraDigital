package services

import (
	"context"
	"fmt"
	"monera-digital/internal/coreapi"
	"monera-digital/internal/models"
	"monera-digital/internal/repository"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
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

	existing := m.wallets[req.ID]
	if existing != nil {
		m.wallets[req.ID] = req
		m.userWallets[existing.UserID] = req

		if req.ProductCode == "" {
			req.ProductCode = existing.ProductCode
		}
		if req.Currency == "" {
			req.Currency = existing.Currency
		}
		if req.UserID == 0 {
			req.UserID = existing.UserID
		}
	} else {
		m.wallets[req.ID] = req
		m.userWallets[req.UserID] = req
	}

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

func (m *MockWalletRepositoryUnique) CreateUserWallet(ctx context.Context, wallet *models.UserWallet) error {
	wallet.ID = len(m.wallets) + 1
	wallet.CreatedAt = time.Now()
	wallet.UpdatedAt = time.Now()
	return nil
}

func (m *MockWalletRepositoryUnique) GetUserWalletsByUserID(ctx context.Context, userID int) ([]*models.UserWallet, error) {
	return nil, nil
}

func (m *MockWalletRepositoryUnique) GetUserWalletByCurrency(ctx context.Context, userID int, currency string) (*models.UserWallet, error) {
	return nil, nil
}

func (m *MockWalletRepositoryUnique) UpdateUserWalletStatus(ctx context.Context, id int, status models.UserWalletStatus) error {
	return nil
}

func (m *MockWalletRepositoryUnique) GetActiveUserWallet(ctx context.Context, userID int) (*models.UserWallet, error) {
	return nil, nil
}

func (m *MockWalletRepositoryUnique) AddUserWalletAddress(ctx context.Context, wallet *models.UserWallet) (*models.UserWallet, error) {
	return wallet, nil
}

func (m *MockWalletRepositoryUnique) GetUserWalletByUserAndCurrency(ctx context.Context, userID int, currency string) (*models.UserWallet, error) {
	return nil, nil
}

func getKey(userID int, productCode, currency string) string {
	return string(rune(userID)) + "-" + productCode + "-" + currency
}

// Interface check
var _ repository.Wallet = (*MockWalletRepositoryUnique)(nil)

func TestCreateWallet_UniqueCheck(t *testing.T) {
	repo := NewMockWalletRepositoryUnique()

	// Create mock Core API client
	mockCoreAPI := new(MockCoreAPIClient)
	mockCoreAPI.On("CreateWallet", mock.Anything, mock.Anything).Return(&coreapi.CreateWalletResponse{
		WalletID:  "wallet_test_123",
		Address:   "0xTestAddress",
		Addresses: map[string]string{"USD": "0xTestAddress"},
		Status:    "SUCCESS",
	}, nil)
	mockCoreAPI.On("GetAddress", mock.Anything, mock.Anything).Return(&coreapi.AddressInfo{
		Address: "0xTestAddressFromGetAddress",
	}, nil)

	service := NewWalletService(repo, mockCoreAPI)

	// Create first wallet
	ctx := context.Background()
	w1, err := service.CreateWallet(ctx, 1, "BANK_ACCOUNT", "USD")
	if err != nil {
		t.Fatalf("Failed to create first wallet: %v", err)
	}

	// Try to create duplicate wallet (same user, product, currency)
	// Should return existing wallet, not create new one
	wallet2, err := service.CreateWallet(ctx, 1, "BANK_ACCOUNT", "USD")
	if err != nil {
		t.Fatalf("Failed to request duplicate wallet: %v", err)
	}

	if wallet2.ID != w1.ID {
		t.Errorf("Expected wallet ID %d, got %d", w1.ID, wallet2.ID)
	}

	// Try to create different wallet (same user, different currency)
	wallet3, err := service.CreateWallet(ctx, 1, "BANK_ACCOUNT", "EUR")
	if err != nil {
		t.Fatalf("Failed to create different wallet: %v", err)
	}

	if wallet3.ID == w1.ID {
		t.Errorf("Expected new wallet ID, got %d", w1.ID)
	}
}

// TestCreateWallet_CoreAPIFailure_UpdatesStatusToFailed tests that when Core API fails,
// the wallet status is properly updated to FAILED in the database.
// This test reproduces the bug where status stays CREATING forever.
func TestCreateWallet_CoreAPIFailure_UpdatesStatusToFailed(t *testing.T) {
	repo := NewMockWalletRepositoryUnique()

	// Create mock Core API client that fails
	mockCoreAPI := new(MockCoreAPIClient)
	mockCoreAPI.On("CreateWallet", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("core api error"))

	service := NewWalletService(repo, mockCoreAPI)

	// Attempt to create wallet - should fail
	ctx := context.Background()
	_, err := service.CreateWallet(ctx, 1, "X_FINANCE", "TRON")

	// Should return error
	if err == nil {
		t.Fatal("Expected error from Core API failure, got nil")
	}

	// Check that the wallet request in repo has status FAILED
	// This is the key assertion - without the fix, status would be CREATING
	req := repo.userWallets[1]
	if req == nil {
		t.Fatal("Wallet request should exist in repository")
	}

	if req.Status != models.WalletCreationStatusFailed {
		t.Errorf("Expected status FAILED after Core API error, got %s. "+
			"This indicates the UpdateRequest was not called with correct ID.", req.Status)
	}
}
