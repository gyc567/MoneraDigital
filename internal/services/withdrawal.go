package services

import (
	"database/sql"
	"monera-digital/internal/models"
)

type WithdrawalService struct {
	DB *sql.DB
}

func NewWithdrawalService(db *sql.DB) *WithdrawalService {
	return &WithdrawalService{DB: db}
}

func (s *WithdrawalService) GetWithdrawals(userID int, limit, offset int) ([]models.Withdrawal, error) {
	query := `
		SELECT id, user_id, from_address_id, amount, asset, to_address, status, tx_hash, created_at, completed_at, failure_reason
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.DB.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []models.Withdrawal
	for rows.Next() {
		var w models.Withdrawal
		err := rows.Scan(
			&w.ID, &w.UserID, &w.FromAddressID, &w.Amount, &w.Asset, &w.ToAddress,
			&w.Status, &w.TxHash, &w.CreatedAt, &w.CompletedAt, &w.FailureReason,
		)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, w)
	}

	return withdrawals, nil
}

func (s *WithdrawalService) CreateWithdrawal(userID int, req models.CreateWithdrawalRequest) (*models.Withdrawal, error) {
	// TODO: Check if address is verified and belongs to user
	query := `
		INSERT INTO withdrawals (user_id, from_address_id, amount, asset, to_address)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, from_address_id, amount, asset, to_address, status, created_at
	`

	var withdrawal models.Withdrawal
	err := s.DB.QueryRow(query, userID, req.AddressID, req.Amount, req.Asset, req.AddressID).Scan(
		&withdrawal.ID, &withdrawal.UserID, &withdrawal.FromAddressID, &withdrawal.Amount,
		&withdrawal.Asset, &withdrawal.ToAddress, &withdrawal.Status, &withdrawal.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &withdrawal, nil
}

func (s *WithdrawalService) GetWithdrawalByID(userID int, withdrawalID int) (*models.Withdrawal, error) {
	query := `
		SELECT id, user_id, from_address_id, amount, asset, to_address, status, tx_hash, created_at, completed_at, failure_reason
		FROM withdrawals
		WHERE id = $1 AND user_id = $2
	`

	var withdrawal models.Withdrawal
	err := s.DB.QueryRow(query, withdrawalID, userID).Scan(
		&withdrawal.ID, &withdrawal.UserID, &withdrawal.FromAddressID, &withdrawal.Amount,
		&withdrawal.Asset, &withdrawal.ToAddress, &withdrawal.Status, &withdrawal.TxHash,
		&withdrawal.CreatedAt, &withdrawal.CompletedAt, &withdrawal.FailureReason,
	)
	if err != nil {
		return nil, err
	}

	return &withdrawal, nil
}