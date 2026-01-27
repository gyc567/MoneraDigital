package postgres

import (
	"context"
	"database/sql"
	"time"

	"monera-digital/internal/models"
	"monera-digital/internal/repository"
)

type WalletRepository struct {
	db *sql.DB
}

func NewWalletRepository(db *sql.DB) repository.Wallet {
	return &WalletRepository{db: db}
}

func (r *WalletRepository) CreateRequest(ctx context.Context, req *models.WalletCreationRequest) error {
	query := `
		INSERT INTO wallet_creation_requests (request_id, user_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	return r.db.QueryRowContext(ctx, query, req.RequestID, req.UserID, req.Status, time.Now(), time.Now()).Scan(&req.ID)
}

func (r *WalletRepository) GetRequestByUserID(ctx context.Context, userID int) (*models.WalletCreationRequest, error) {
	query := `
		SELECT id, request_id, user_id, status, wallet_id, address, addresses, error_message, created_at, updated_at
		FROM wallet_creation_requests WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`

	var w models.WalletCreationRequest
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&w.ID, &w.RequestID, &w.UserID, &w.Status, &w.WalletID, &w.Address, &w.Addresses, &w.ErrorMessage, &w.CreatedAt, &w.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepository) UpdateRequest(ctx context.Context, req *models.WalletCreationRequest) error {
	query := `
		UPDATE wallet_creation_requests
		SET status = $1, wallet_id = $2, address = $3, addresses = $4, error_message = $5, updated_at = $6
		WHERE id = $7`
	_, err := r.db.ExecContext(ctx, query,
		req.Status, req.WalletID, req.Address, req.Addresses, req.ErrorMessage, time.Now(), req.ID,
	)
	return err
}

func (r *WalletRepository) GetActiveWalletByUserID(ctx context.Context, userID int) (*models.WalletCreationRequest, error) {
	query := `
		SELECT id, request_id, user_id, status, wallet_id, address, addresses, error_message, created_at, updated_at
		FROM wallet_creation_requests WHERE user_id = $1 AND status = 'SUCCESS' ORDER BY created_at DESC LIMIT 1`

	var w models.WalletCreationRequest
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&w.ID, &w.RequestID, &w.UserID, &w.Status, &w.WalletID, &w.Address, &w.Addresses, &w.ErrorMessage, &w.CreatedAt, &w.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}
