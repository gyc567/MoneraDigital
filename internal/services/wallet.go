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
		return s.mapToModel(existing), nil
	}

	// Create new
	reqID := uuid.New().String()
	newReq := &repository.WalletCreationRequestModel{
		RequestID: reqID,
		UserID:    userID,
		Status:    string(models.WalletCreationStatusCreating),
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
		updateReq := &repository.WalletCreationRequestModel{
			ID:        newReq.ID,
			Status:    string(models.WalletCreationStatusSuccess),
			WalletID:  "wallet_" + reqID[:8],
			Addresses: string(addrJSON),
            Address:   "0x71C7656EC7ab88b098defB751B7401B5f6d8976F", // Default/Primary
		}
        // Create context for background task
        bgCtx := context.Background()
		_ = s.repo.UpdateRequest(bgCtx, updateReq)
	}()

	return s.mapToModel(newReq), nil
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
            return s.mapToModel(req), nil
        }
		return nil, nil
	}
	return s.mapToModel(w), nil
}

func (s *WalletService) mapToModel(r *repository.WalletCreationRequestModel) *models.WalletCreationRequest {
	t, _ := time.Parse(time.RFC3339, r.CreatedAt)
	u, _ := time.Parse(time.RFC3339, r.UpdatedAt)

	return &models.WalletCreationRequest{
		ID:           r.ID,
		RequestID:    r.RequestID,
		UserID:       r.UserID,
		Status:       models.WalletCreationStatus(r.Status),
		WalletID:     sql.NullString{String: r.WalletID, Valid: r.WalletID != ""},
		Address:      sql.NullString{String: r.Address, Valid: r.Address != ""},
		Addresses:    sql.NullString{String: r.Addresses, Valid: r.Addresses != ""},
		ErrorMessage: sql.NullString{String: r.ErrorMessage, Valid: r.ErrorMessage != ""},
		CreatedAt:    t,
		UpdatedAt:    u,
	}
}
