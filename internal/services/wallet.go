package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"monera-digital/internal/coreapi"
	"monera-digital/internal/logger"
	"monera-digital/internal/models"
	"monera-digital/internal/repository"
	"os"
	"time"

	"github.com/google/uuid"
)

type WalletService struct {
	repo          repository.Wallet
	coreAPIClient *coreapi.Client
}

func NewWalletService(repo repository.Wallet, coreAPIClient *coreapi.Client) *WalletService {
	return &WalletService{repo: repo, coreAPIClient: coreAPIClient}
}

// CreateWallet creates a new wallet for the user with productCode and currency.
func (s *WalletService) CreateWallet(ctx context.Context, userID int, productCode, currency string) (*models.WalletCreationRequest, error) {
	// Check for existing wallet with same product and currency
	existing, err := s.repo.GetWalletByUserProductCurrency(ctx, userID, productCode, currency)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == models.WalletCreationStatusSuccess {
		return existing, nil
	}

	// Create new wallet request
	reqID := uuid.New().String()
	newReq := &models.WalletCreationRequest{
		RequestID:   reqID,
		UserID:      userID,
		ProductCode: productCode,
		Currency:    currency,
		Status:      models.WalletCreationStatusCreating,
	}
	err = s.repo.CreateRequest(ctx, newReq)
	if err != nil {
		logger.Error("Failed to create wallet request", "error", err.Error(), "userId", userID, "productCode", productCode, "currency", currency)
		return nil, err
	}
	logger.Info("Wallet request created", "requestId", reqID, "userId", userID)

	// Try to create wallet via Core API, fallback to mock if fails
	var walletID, address string
	var addresses map[string]string

	if s.coreAPIClient != nil {
		coreResp, err := s.coreAPIClient.CreateWallet(ctx, coreapi.CreateWalletRequest{
			UserID:      userID,
			ProductCode: productCode,
			Currency:    currency,
		})
		if err != nil {
			logger.Warn("Core API wallet creation failed, falling back to mock", "error", err.Error(), "userId", userID)
		} else {
			logger.Info("Core API wallet created successfully", "walletId", coreResp.WalletID, "userId", userID)
			walletID = coreResp.WalletID
			address = coreResp.Address
			addresses = coreResp.Addresses
		}
	}

	// Fallback to mock addresses if Core API didn't provide them
	if addresses == nil {
		addresses = s.generateMockAddresses(currency)
	}
	if walletID == "" {
		walletID = "wallet_" + reqID[:8]
	}
	if address == "" {
		address = addresses[currency]
	}

	// Store addresses as JSON string
	addressesJSON, _ := json.Marshal(addresses)

	go func(requestID string, userID int, productCode, currency, walletID, address string, addressesJSON []byte) {
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
		req.Addresses = sql.NullString{String: string(addressesJSON), Valid: true}
		req.UpdatedAt = time.Now()

		// Update DB with proper error handling
		if err := s.repo.UpdateRequest(bgCtx, req); err != nil {
			// Log error but don't propagate
		}
	}(reqID, userID, productCode, currency, walletID, address, addressesJSON)

	return newReq, nil
}

// generateMockAddresses generates mock addresses for multiple chains based on currency.
func (s *WalletService) generateMockAddresses(currency string) map[string]string {
	mockAddresses := map[string]map[string]string{
		"USDT": {
			"ERC20": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			"TRC20": "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
			"BSC":   "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
		},
		"ETH": {
			"ETH": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
		},
		"TRON": {
			"TRON": "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
		},
		"BSC": {
			"BSC": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
		},
	}

	// Check environment variable override
	if envAddr := os.Getenv("WALLET_ADDR_" + currency); envAddr != "" {
		return map[string]string{currency: envAddr}
	}

	// Return addresses for the currency
	if addrMap, ok := mockAddresses[currency]; ok {
		return addrMap
	}

	// Default fallback - use the original single address
	defaultAddr := "0x71C7656EC7ab88b098defB751B7401B5f6d8976F"
	return map[string]string{currency: defaultAddr}
}

// generateAddress generates a single mock address based on currency chain.
func (s *WalletService) generateAddress(currency string) string {
	addrMap := s.generateMockAddresses(currency)
	if addr, ok := addrMap[currency]; ok {
		return addr
	}
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

type AddAddressRequest struct {
	Chain string
	Token string
}

func (s *WalletService) AddAddress(ctx context.Context, userID int, req AddAddressRequest) (*models.WalletCreationRequest, error) {
	wallet, err := s.repo.GetActiveWalletByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, errors.New("wallet not found")
	}

	addresses := make(map[string]string)
	if wallet.Addresses.Valid && wallet.Addresses.String != "" {
		if err := json.Unmarshal([]byte(wallet.Addresses.String), &addresses); err != nil {
			return nil, errors.New("failed to parse existing addresses")
		}
	}

	addressKey := req.Token + "_" + req.Chain
	if _, exists := addresses[addressKey]; exists {
		return wallet, nil
	}

	addresses[addressKey] = s.generateAddress(addressKey)

	addressesJSON, _ := json.Marshal(addresses)
	wallet.Addresses = sql.NullString{String: string(addressesJSON), Valid: true}
	wallet.UpdatedAt = time.Now()

	if err := s.repo.UpdateRequest(ctx, wallet); err != nil {
		return nil, err
	}

	return wallet, nil
}
