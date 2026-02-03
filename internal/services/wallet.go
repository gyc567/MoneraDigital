package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"monera-digital/internal/coreapi"
	"monera-digital/internal/currency"
	"monera-digital/internal/dto"
	"monera-digital/internal/logger"
	"monera-digital/internal/models"
	"monera-digital/internal/repository"
	"time"

	"github.com/google/uuid"
)

type WalletService struct {
	repo          repository.Wallet
	coreAPIClient coreapi.CoreAPIClientInterface
}

func NewWalletService(repo repository.Wallet, coreAPIClient coreapi.CoreAPIClientInterface) *WalletService {
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

	if s.coreAPIClient == nil {
		errMsg := "Core API client not initialized"
		logger.Error(errMsg, "userId", userID)
		return nil, errors.New(errMsg)
	}

	coreResp, err := s.coreAPIClient.CreateWallet(ctx, coreapi.CreateWalletRequest{
		UserID:      userID,
		ProductCode: productCode,
		Currency:    currency,
	})
	if err != nil {
		logger.Error("Core API wallet creation failed", "error", err.Error(), "userId", userID, "productCode", productCode, "currency", currency)
		s.repo.UpdateRequest(ctx, &models.WalletCreationRequest{RequestID: reqID, Status: models.WalletCreationStatusFailed})
		return nil, fmt.Errorf("wallet creation failed: %w", err)
	}

	logger.Info("Core API wallet created successfully", "walletId", coreResp.WalletID, "userId", userID)

	newReq.Status = models.WalletCreationStatusSuccess
	newReq.WalletID = sql.NullString{String: coreResp.WalletID, Valid: true}
	newReq.Address = sql.NullString{String: coreResp.Address, Valid: true}
	if coreResp.Addresses != nil {
		addressesJSON, _ := json.Marshal(coreResp.Addresses)
		newReq.Addresses = sql.NullString{String: string(addressesJSON), Valid: true}
	}
	newReq.UpdatedAt = time.Now()

	if err := s.repo.UpdateRequest(ctx, newReq); err != nil {
		logger.Error("Failed to update wallet request", "error", err.Error(), "requestId", reqID)
	}

	// Sync to user_wallets table - store individual wallet addresses
	userWallet := &models.UserWallet{
		UserID:    userID,
		WalletID:  coreResp.WalletID,
		Currency:  currency,
		Address:   coreResp.Address,
		Status:    models.UserWalletStatusNormal,
		IsPrimary: true,
	}
	if reqID != "" {
		userWallet.RequestID = sql.NullString{String: reqID, Valid: true}
	}
	if err := s.repo.CreateUserWallet(ctx, userWallet); err != nil {
		logger.Error("Failed to sync user wallet", "error", err.Error(), "userId", userID, "currency", currency)
		// Don't fail the operation if user_wallet sync fails
	}

	return newReq, nil
}

// buildCurrencyKey builds a valid currency key from token and network.
// Normalizes network aliases (TRON -> TRC20) to standard names.
func buildCurrencyKey(token, network string) string {
	// Normalize network aliases
	network = currency.NormalizeNetwork(network)

	// Build the currency key
	currencyKey := currency.BuildCurrency(token, network)

	// Validate the currency is supported
	if !currency.IsValid(currencyKey) {
		return ""
	}

	return currencyKey
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

	// Merge addresses from user_wallets into wallet_creation_requests
	userWallets, err := s.repo.GetUserWalletsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(userWallets) > 0 {
		// Parse existing addresses
		addresses := make(map[string]string)
		if w.Addresses.Valid && w.Addresses.String != "" {
			if err := json.Unmarshal([]byte(w.Addresses.String), &addresses); err != nil {
				logger.Warn("Failed to parse existing addresses", "error", err.Error())
			}
		}

		// Merge user_wallets addresses
		for _, uw := range userWallets {
			if uw.Address != "" {
				addresses[uw.Currency] = uw.Address
			}
		}

		// Update addresses JSON
		addressesJSON, _ := json.Marshal(addresses)
		w.Addresses = sql.NullString{String: string(addressesJSON), Valid: true}
	}

	return w, nil
}

type AddAddressRequest struct {
	Chain string
	Token string
}

// AddAddress adds a new wallet address for the given chain and token.
// It gets the address from Core API and stores it in user_wallets table.
func (s *WalletService) AddAddress(ctx context.Context, userID int, req AddAddressRequest) (*models.UserWallet, error) {
	// Get user's wallet info to get wallet_id and productCode
	wallet, err := s.repo.GetActiveWalletByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Fall back to user_wallets if no wallet_creation_requests
	if wallet == nil {
		userWallet, err := s.repo.GetActiveUserWallet(ctx, userID)
		if err != nil {
			return nil, err
		}
		if userWallet != nil {
			wallet = convertUserWalletToRequest(userWallet)
		}
	}

	if wallet == nil {
		return nil, errors.New("wallet not found")
	}

	// Calculate currency key for the address
	addressKey := buildCurrencyKey(req.Token, req.Chain)
	if addressKey == "" {
		return nil, fmt.Errorf("invalid currency: %s_%s", req.Token, req.Chain)
	}

	// Check if address already exists in user_wallets
	existingWallet, err := s.repo.GetUserWalletByUserAndCurrency(ctx, userID, addressKey)
	if err != nil {
		return nil, err
	}
	if existingWallet != nil {
		logger.Info("Address already exists", "userId", userID, "currency", addressKey)
		return existingWallet, nil
	}

	// Get address from Core API (REQUIRED, no fallback)
	if s.coreAPIClient == nil {
		return nil, fmt.Errorf("Core API client not initialized")
	}

	// Use wallet's ProductCode or default to X_FINANCE
	productCode := wallet.ProductCode
	if productCode == "" {
		productCode = "X_FINANCE"
	}

	coreResp, err := s.coreAPIClient.GetAddress(ctx, coreapi.GetAddressRequest{
		UserID:      userID,
		ProductCode: productCode,
		Currency:    addressKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get address from Core API: %w", err)
	}

	logger.Info("Core API address fetched successfully", "userId", userID, "currency", addressKey)

	// Create new UserWallet record
	newWallet := &models.UserWallet{
		UserID:    userID,
		WalletID:  wallet.WalletID.String,
		Currency:  addressKey,
		Address:   coreResp.Address,
		Status:    models.UserWalletStatusNormal,
		IsPrimary: false,
	}

	// Set optional fields if available
	if coreResp.AddressType != nil {
		newWallet.AddressType = sql.NullString{String: *coreResp.AddressType, Valid: true}
	}
	if coreResp.DerivePath != nil {
		newWallet.DerivePath = sql.NullString{String: *coreResp.DerivePath, Valid: true}
	}
	if wallet.RequestID != "" {
		newWallet.RequestID = sql.NullString{String: wallet.RequestID, Valid: true}
	}

	// Store in user_wallets table
	result, err := s.repo.AddUserWalletAddress(ctx, newWallet)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetAddressIncomeHistory 获取地址链上收款记录
func (s *WalletService) GetAddressIncomeHistory(ctx context.Context, userID int, address string) ([]coreapi.AddressIncomeRecord, error) {
	if s.coreAPIClient == nil {
		return nil, fmt.Errorf("Core API client not initialized")
	}

	return s.coreAPIClient.GetIncomeHistory(ctx, coreapi.GetIncomeHistoryRequest{
		Address: address,
	})
}

// GetWalletAddress 获取钱包地址
// 优先从 Core API 获取，如果失败则从本地数据库获取
func (s *WalletService) GetWalletAddress(ctx context.Context, userID int, req dto.GetWalletAddressRequest) (*dto.WalletAddress, error) {
	// 优先从 Core API 获取
	if s.coreAPIClient != nil {
		addressInfo, err := s.coreAPIClient.GetAddress(ctx, coreapi.GetAddressRequest{
			UserID:      userID,
			ProductCode: req.ProductCode,
			Currency:    req.Currency,
		})
		if err == nil {
			return &dto.WalletAddress{
				Address:     addressInfo.Address,
				AddressType: addressInfo.AddressType,
				DerivePath:  addressInfo.DerivePath,
			}, nil
		}
		// 如果 Core API 返回错误，继续尝试从本地数据库获取
		logger.Info("Core API GetAddress failed, falling back to local database", "error", err.Error())
	}

	// 从本地数据库获取钱包信息作为降级方案
	wallet, err := s.GetWalletInfo(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet address: %v", err)
	}
	if wallet == nil {
		return nil, fmt.Errorf("wallet not found")
	}

	// 解析 addresses JSON 并获取对应 currency 的地址
	var address string
	if wallet.Addresses.Valid && wallet.Addresses.String != "" {
		addresses := make(map[string]string)
		if err := json.Unmarshal([]byte(wallet.Addresses.String), &addresses); err != nil {
			logger.Info("Failed to parse addresses JSON", "error", err.Error())
		} else {
			// 优先查找对应 currency 的地址
			address = addresses[req.Currency]
			// 如果找不到，尝试查找任一地址
			if address == "" {
				for _, v := range addresses {
					address = v
					break
				}
			}
		}
	}

	// 如果没有找到地址，尝试使用单一的 address 字段
	if address == "" && wallet.Address.Valid && wallet.Address.String != "" {
		address = wallet.Address.String
	}

	if address == "" {
		return nil, fmt.Errorf("wallet address not found")
	}

	return &dto.WalletAddress{
		Address: address,
	}, nil
}

// Ensure dto types are not optimized away by linker
// This prevents "undefined type" errors in some build environments
var _ = dto.GetWalletAddressRequest{}
var _ = dto.WalletAddress{}

// convertUserWalletToRequest converts a UserWallet to WalletCreationRequest format
// for use in AddAddress flow. This enables looking up wallets from user_wallets
// table when wallet_creation_requests doesn't have a matching record.
func convertUserWalletToRequest(uw *models.UserWallet) *models.WalletCreationRequest {
	addressesJSON, _ := json.Marshal(map[string]string{uw.Currency: uw.Address})
	return &models.WalletCreationRequest{
		UserID:      uw.UserID,
		ProductCode: "X_FINANCE",
		Currency:    uw.Currency,
		Status:      models.WalletCreationStatusSuccess,
		WalletID:    sql.NullString{String: uw.WalletID, Valid: uw.WalletID != ""},
		Address:     sql.NullString{String: uw.Address, Valid: uw.Address != ""},
		Addresses:   sql.NullString{String: string(addressesJSON), Valid: true},
	}
}
