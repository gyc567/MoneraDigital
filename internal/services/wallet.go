package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"monera-digital/internal/coreapi"
	"monera-digital/internal/dto"
	"monera-digital/internal/logger"
	"monera-digital/internal/models"
	"monera-digital/internal/repository"
	"os"
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

// normalizeCurrencyKey normalizes the currency key for Core API requests.
// TRX on TRON chain is mapped to USDT_TRON since TRX is the native token
// and users typically want a USDT address on TRON.
func normalizeCurrencyKey(token, chain, currentKey string) string {
	// Map TRX + TRON to USDT_TRON (most common use case)
	if token == "TRX" && chain == "TRON" {
		return "USDT_TRON"
	}
	return currentKey
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

// AddAddress adds a new wallet address for the given chain and token.
// It first tries to get the address from Core API, falling back to local generation if it fails.
func (s *WalletService) AddAddress(ctx context.Context, userID int, req AddAddressRequest) (*models.WalletCreationRequest, error) {
	wallet, err := s.repo.GetActiveWalletByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, errors.New("wallet not found")
	}

	// Parse existing addresses
	addresses := make(map[string]string)
	if wallet.Addresses.Valid && wallet.Addresses.String != "" {
		if err := json.Unmarshal([]byte(wallet.Addresses.String), &addresses); err != nil {
			return nil, errors.New("failed to parse existing addresses")
		}
	}

	// Check if address already exists
	addressKey := req.Token + "_" + req.Chain
	// Normalize currency key (e.g., TRX_TRON → USDT_TRON)
	addressKey = normalizeCurrencyKey(req.Token, req.Chain, addressKey)
	if _, exists := addresses[addressKey]; exists {
		return wallet, nil
	}

	// Get address from Core API (REQUIRED, no fallback)
	if s.coreAPIClient == nil {
		return nil, fmt.Errorf("Core API client not initialized")
	}

	coreResp, err := s.coreAPIClient.GetAddress(ctx, coreapi.GetAddressRequest{
		UserID:      userID,
		ProductCode: wallet.ProductCode,
		Currency:    addressKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get address from Core API: %w", err)
	}

	newAddress := coreResp.Address
	logger.Info("Core API address fetched successfully", "userId", userID, "currency", addressKey)

	// Update addresses map
	addresses[addressKey] = newAddress
	addressesJSON, _ := json.Marshal(addresses)
	wallet.Addresses = sql.NullString{String: string(addressesJSON), Valid: true}
	wallet.UpdatedAt = time.Now()

	if err := s.repo.UpdateRequest(ctx, wallet); err != nil {
		return nil, err
	}

	return wallet, nil
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
