package services

import (
	"database/sql"
	"fmt"
	"time"

	"monera-digital/internal/models"
)

type LendingService struct{
	DB *sql.DB
}

func NewLendingService(db *sql.DB) *LendingService {
	return &LendingService{DB: db}
}

func (s *LendingService) CalculateAPY(asset string, durationDays int) string {
	baseRates := map[string]float64{
		"BTC":  4.5,
		"ETH":  5.2,
		"USDT": 8.5,
		"USDC": 8.2,
		"SOL":  6.8,
	}

	baseRate, exists := baseRates[asset]
	if !exists {
		baseRate = 5.0
	}

	multiplier := 1.0
	if durationDays >= 360 {
		multiplier = 1.5
	} else if durationDays >= 180 {
		multiplier = 1.25
	} else if durationDays >= 90 {
		multiplier = 1.1
	}

	apy := baseRate * multiplier
	return fmt.Sprintf("%.2f", apy)
}

func (s *LendingService) ApplyForLending(userID int, req models.ApplyLendingRequest) (*models.LendingPosition, error) {
	apy := s.CalculateAPY(req.Asset, req.DurationDays)
	startDate := time.Now()
	endDate := startDate.AddDate(0, 0, req.DurationDays)

	query := `
		INSERT INTO lending_positions (user_id, asset, amount, duration_days, apy, status, end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, asset, amount, duration_days, apy, status, accrued_yield, start_date, end_date
	`

	var position models.LendingPosition
	err := s.DB.QueryRow(query, userID, req.Asset, req.Amount, req.DurationDays, apy, "ACTIVE", endDate).Scan(
		&position.ID, &position.UserID, &position.Asset, &position.Amount,
		&position.DurationDays, &position.Apy, &position.Status, &position.AccruedYield,
		&position.StartDate, &position.EndDate,
	)

	if err != nil {
		return nil, err
	}

	return &position, nil
}

func (s *LendingService) GetUserPositions(userID int) ([]models.LendingPosition, error) {
	query := `
		SELECT id, user_id, asset, amount, duration_days, apy, status, accrued_yield, start_date, end_date
		FROM lending_positions
		WHERE user_id = $1
		ORDER BY start_date
	`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []models.LendingPosition
	for rows.Next() {
		var position models.LendingPosition
		err := rows.Scan(
			&position.ID, &position.UserID, &position.Asset, &position.Amount,
			&position.DurationDays, &position.Apy, &position.Status, &position.AccruedYield,
			&position.StartDate, &position.EndDate,
		)
		if err != nil {
			return nil, err
		}
		positions = append(positions, position)
	}

	return positions, nil
}

func (s *LendingService) CalculateEstimatedYield(amount, apy float64, durationDays int) float64 {
	return (amount * (apy / 100) * float64(durationDays)) / 365
}