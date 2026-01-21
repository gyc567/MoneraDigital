package postgres

import (
	"context"
	"database/sql"
	"time"

	"monera-digital/internal/models"
	"monera-digital/internal/repository"
)

type WithdrawalRepository struct {
	db *sql.DB
}

func NewWithdrawalRepository(db *sql.DB) repository.Withdrawal {
	return &WithdrawalRepository{db: db}
}

func (r *WithdrawalRepository) CreateOrder(ctx context.Context, order *models.WithdrawalOrder) (*models.WithdrawalOrder, error) {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO withdrawal_order (
			user_id, amount, network_fee, platform_fee, actual_amount,
			chain_type, coin_type, to_address, safeheron_order_id, transaction_hash,
			status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at`,
		order.UserID, order.Amount, order.NetworkFee, order.PlatformFee, order.ActualAmount,
		order.ChainType, order.CoinType, order.ToAddress, order.SafeheronOrderID, order.TransactionHash,
		order.Status, time.Now(), time.Now(),
	).Scan(&order.ID, &order.CreatedAt)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *WithdrawalRepository) GetOrdersByUserID(ctx context.Context, userID int) ([]*models.WithdrawalOrder, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, amount, network_fee, platform_fee, actual_amount,
			chain_type, coin_type, to_address, safeheron_order_id, transaction_hash,
			status, created_at, sent_at, confirmed_at, completed_at, updated_at
		FROM withdrawal_order WHERE user_id = $1 ORDER BY created_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]*models.WithdrawalOrder, 0, 50)
	for rows.Next() {
		var o models.WithdrawalOrder
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.Amount, &o.NetworkFee, &o.PlatformFee, &o.ActualAmount,
			&o.ChainType, &o.CoinType, &o.ToAddress, &o.SafeheronOrderID, &o.TransactionHash,
			&o.Status, &o.CreatedAt, &o.SentAt, &o.ConfirmedAt, &o.CompletedAt, &o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	return orders, nil
}

func (r *WithdrawalRepository) GetOrderByID(ctx context.Context, id int) (*models.WithdrawalOrder, error) {
	var o models.WithdrawalOrder
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, amount, network_fee, platform_fee, actual_amount,
			chain_type, coin_type, to_address, safeheron_order_id, transaction_hash,
			status, created_at, sent_at, confirmed_at, completed_at, updated_at
		FROM withdrawal_order WHERE id = $1`,
		id).Scan(
		&o.ID, &o.UserID, &o.Amount, &o.NetworkFee, &o.PlatformFee, &o.ActualAmount,
		&o.ChainType, &o.CoinType, &o.ToAddress, &o.SafeheronOrderID, &o.TransactionHash,
		&o.Status, &o.CreatedAt, &o.SentAt, &o.ConfirmedAt, &o.CompletedAt, &o.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *WithdrawalRepository) UpdateOrder(ctx context.Context, order *models.WithdrawalOrder) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE withdrawal_order SET
			status = $1, transaction_hash = $2, sent_at = $3,
			confirmed_at = $4, completed_at = $5, updated_at = $6
		WHERE id = $7`,
		order.Status, order.TransactionHash, order.SentAt,
		order.ConfirmedAt, order.CompletedAt, time.Now(), order.ID)
	return err
}

func (r *WithdrawalRepository) CreateRequest(ctx context.Context, req *models.WithdrawalRequest) error {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO withdrawal_request (user_id, request_id, status, error_code, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		req.UserID, req.RequestID, req.Status, req.ErrorCode, req.ErrorMessage, time.Now(),
	).Scan(&req.ID)
	return err
}

func (r *WithdrawalRepository) GetRequestByID(ctx context.Context, requestID string) (*models.WithdrawalRequest, error) {
	var req models.WithdrawalRequest
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, request_id, status, error_code, error_message, created_at
		FROM withdrawal_request WHERE request_id = $1`,
		requestID).Scan(
		&req.ID, &req.UserID, &req.RequestID, &req.Status, &req.ErrorCode, &req.ErrorMessage, &req.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}
