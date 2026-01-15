package services

import (
	"context"
	"database/sql"
	"monera-digital/internal/models"
	"monera-digital/internal/repository"
	"time"
)

type DepositService struct {
	repo repository.Deposit
}

func NewDepositService(repo repository.Deposit) *DepositService {
	return &DepositService{repo: repo}
}

func (s *DepositService) GetDeposits(ctx context.Context, userID int, limit, offset int) ([]*models.Deposit, int64, error) {
	repoDeps, total, err := s.repo.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var deposits []*models.Deposit
	for _, d := range repoDeps {
		t, _ := time.Parse(time.RFC3339, d.CreatedAt)
        
        var confirmedAt sql.NullTime
        if d.ConfirmedAt != "" {
            ct, err := time.Parse(time.RFC3339, d.ConfirmedAt)
            if err == nil {
                confirmedAt = sql.NullTime{Time: ct, Valid: true}
            }
        }

		deposits = append(deposits, &models.Deposit{
			ID:          d.ID,
			UserID:      d.UserID,
			TxHash:      d.TxHash,
			Amount:      d.Amount,
			Asset:       d.Asset,
			Chain:       d.Chain,
			Status:      models.DepositStatus(d.Status),
			FromAddress: sql.NullString{String: d.FromAddress, Valid: d.FromAddress != ""},
			ToAddress:   sql.NullString{String: d.ToAddress, Valid: d.ToAddress != ""},
			CreatedAt:   t,
			ConfirmedAt: confirmedAt,
		})
	}
	return deposits, total, nil
}

func (s *DepositService) HandleWebhook(ctx context.Context, payload map[string]interface{}) error {
	// Implement webhook handling logic
    // 1. Verify signature (skipped for MVP, handled by middleware usually or here)
    // 2. Extract data
    // 3. Update or Insert deposit
    // This requires implementing Upsert logic or Check-then-Insert
    return nil
}
