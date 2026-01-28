package services

import (
	"context"
	"database/sql"
	"monera-digital/internal/models"
	"monera-digital/internal/repository"
	"time"

	"github.com/google/uuid"
)

type WalletService struct {
	repo repository.Wallet
}

func NewWalletService(repo repository.Wallet) *WalletService {
	return &WalletService{repo: repo}
}

// CreateWallet creates a new wallet for the user with productCode and currency.
func (s *WalletService) CreateWallet(ctx context.Context, userID int, productCode, currency string) (*models.WalletCreationRequest, error) {
	// Check for existing wallet with same product and currency
	existing, err := s.repo.GetRequestByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == models.WalletCreationStatusSuccess {
		return existing, nil
	}

	// Create new wallet request
	reqID := uuid.New().String()
	newReq := &models.WalletCreationRequest{
		RequestID: reqID,
		UserID:    userID,
		Status:    models.WalletCreationStatusCreating,
	}
	err = s.repo.CreateRequest(ctx, newReq)
	if err != nil {
		return nil, err
	}

	// Generate address based on currency chain
	address := s.generateAddress(currency)
	walletID := "wallet_" + reqID[:8]

	go func(requestID string, userID int, productCode, currency, address, walletID string) {
		// Simulate wallet creation delay
		time.Sleep(500 * time.Millisecond)

		// Create context for background task with timeout
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Fetch fresh request from DB using userID
		req, err := s.repo.GetRequestByUserID(bgCtx, userID)
		if err != nil || req == nil {
			return
		}

		// Update the request with success status
		req.Status = models.WalletCreationStatusSuccess
		req.WalletID = sql.NullString{String: walletID, Valid: true}
		req.Address = sql.NullString{String: address, Valid: true}
		req.UpdatedAt = time.Now()

		// Update DB with proper error handling
		if err := s.repo.UpdateRequest(bgCtx, req); err != nil {
			// Log error but don't propagate
		}
	}(reqID, userID, productCode, currency, address, walletID)

	return newReq, nil
}

// generateAddress generates a mock address based on currency chain.
func (s *WalletService) generateAddress(currency string) string {
	mockAddresses := map[string]string{
		"USDT_ERC20": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
		"USDT_TRC20": "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
		"USDT_BSC":   "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
		"ETH":        "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
		"TRON":       "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
		"BSC":        "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
	}

	if addr, ok := mockAddresses[currency]; ok {
		return addr
	}
	// Default fallback address
	return "0x71C7656EC7ab88b098defB751B7401B5f6d8976F"
}

func (s *WalletService) GetWalletInfo(ctx context.Context, userID int) (*models.WalletCreationRequest, error) {
	// First try to find active/success wallet
	w, err := s.repo.GetActiveWalletByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// If not found, check if there is any request (e.g. creating)
	if w == nil {
		req, err := s.repo.GetRequestByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if req != nil {
			return req, nil
		}
		return nil, nil
	}
	return w, nil
}
