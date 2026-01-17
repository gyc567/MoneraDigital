package services

import (
	"context"
	"database/sql"
	"encoding/json"
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

func (s *WalletService) CreateWallet(ctx context.Context, userID int) (*models.WalletCreationRequest, error) {
	existing, err := s.repo.GetRequestByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// Create new
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

	// Mock Async Completion
	go func() {
		// Simulate delay
		time.Sleep(2 * time.Second)
		
		// Mock Data
		mockAddresses := map[string]string{
			"ETH":  "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			"TRON": "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
            "BSC":  "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
		}
		addrJSON, _ := json.Marshal(mockAddresses)

		// Update DB
		newReq.Status = models.WalletCreationStatusSuccess
		newReq.WalletID = sql.NullString{String: "wallet_" + reqID[:8], Valid: true}
		newReq.Addresses = sql.NullString{String: string(addrJSON), Valid: true}
		newReq.Address = sql.NullString{String: "0x71C7656EC7ab88b098defB751B7401B5f6d8976F", Valid: true}
		newReq.UpdatedAt = time.Now()

        // Create context for background task
        bgCtx := context.Background()
		_ = s.repo.UpdateRequest(bgCtx, newReq)
	}()

	return newReq, nil
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