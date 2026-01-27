package services

import (
	"context"
	"monera-digital/internal/models"
	"monera-digital/internal/repository"
)

type DepositService struct {
	repo repository.Deposit
}

func NewDepositService(repo repository.Deposit) *DepositService {
	return &DepositService{repo: repo}
}

func (s *DepositService) GetDeposits(ctx context.Context, userID int, limit, offset int) ([]*models.Deposit, int64, error) {
	return s.repo.GetByUserID(ctx, userID, limit, offset)
}

func (s *DepositService) HandleWebhook(ctx context.Context, payload map[string]interface{}) error {
	// Implement webhook handling logic
	// 1. Verify signature (skipped for MVP, handled by middleware usually or here)
	// 2. Extract data
	// 3. Update or Insert deposit
	// This requires implementing Upsert logic or Check-then-Insert
	return nil
}
