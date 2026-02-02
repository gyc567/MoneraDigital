package postgres

import (
	"context"
	"database/sql"
	"fmt"
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
		INSERT INTO wallet_creation_requests (request_id, user_id, product_code, currency, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`
	return r.db.QueryRowContext(ctx, query, req.RequestID, req.UserID, req.ProductCode, req.Currency, req.Status, time.Now(), time.Now()).Scan(&req.ID)
}

func (r *WalletRepository) GetRequestByUserID(ctx context.Context, userID int) (*models.WalletCreationRequest, error) {
	query := `
		SELECT id, request_id, user_id, product_code, currency, status, wallet_id, address, addresses, error_message, created_at, updated_at
		FROM wallet_creation_requests WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`

	var w models.WalletCreationRequest
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&w.ID, &w.RequestID, &w.UserID, &w.ProductCode, &w.Currency, &w.Status, &w.WalletID, &w.Address, &w.Addresses, &w.ErrorMessage, &w.CreatedAt, &w.UpdatedAt,
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

func (r *WalletRepository) GetWalletByUserProductCurrency(ctx context.Context, userID int, productCode, currency string) (*models.WalletCreationRequest, error) {
	query := `
		SELECT id, request_id, user_id, product_code, currency, status, wallet_id, address, addresses, error_message, created_at, updated_at
		FROM wallet_creation_requests 
		WHERE user_id = $1 AND product_code = $2 AND currency = $3 
		ORDER BY created_at DESC LIMIT 1`

	var w models.WalletCreationRequest
	err := r.db.QueryRowContext(ctx, query, userID, productCode, currency).Scan(
		&w.ID, &w.RequestID, &w.UserID, &w.ProductCode, &w.Currency, &w.Status, &w.WalletID, &w.Address, &w.Addresses, &w.ErrorMessage, &w.CreatedAt, &w.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepository) GetActiveWalletByUserID(ctx context.Context, userID int) (*models.WalletCreationRequest, error) {
	query := `
		SELECT id, request_id, user_id, product_code, currency, status, wallet_id, address, addresses, error_message, created_at, updated_at
		FROM wallet_creation_requests WHERE user_id = $1 AND status = 'SUCCESS' ORDER BY created_at DESC LIMIT 1`

	var w models.WalletCreationRequest
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&w.ID, &w.RequestID, &w.UserID, &w.ProductCode, &w.Currency, &w.Status, &w.WalletID, &w.Address, &w.Addresses, &w.ErrorMessage, &w.CreatedAt, &w.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// CreateUserWallet inserts a new user wallet record
func (r *WalletRepository) CreateUserWallet(ctx context.Context, wallet *models.UserWallet) error {
	query := `
		INSERT INTO user_wallets (user_id, request_id, wallet_id, currency, address, address_type, derive_path, is_primary, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`
	return r.db.QueryRowContext(ctx, query,
		wallet.UserID, wallet.RequestID, wallet.WalletID, wallet.Currency,
		wallet.Address, wallet.AddressType, wallet.DerivePath, wallet.IsPrimary,
		time.Now(), time.Now(),
	).Scan(&wallet.ID)
}

// GetUserWalletsByUserID retrieves all wallets for a user
func (r *WalletRepository) GetUserWalletsByUserID(ctx context.Context, userID int) ([]*models.UserWallet, error) {
	query := `
		SELECT id, user_id, request_id, wallet_id, currency, address, address_type, derive_path, status, is_primary, created_at, updated_at
		FROM user_wallets WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []*models.UserWallet
	for rows.Next() {
		var w models.UserWallet
		err := rows.Scan(
			&w.ID, &w.UserID, &w.RequestID, &w.WalletID, &w.Currency,
			&w.Address, &w.AddressType, &w.DerivePath, &w.Status, &w.IsPrimary,
			&w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, &w)
	}
	return wallets, rows.Err()
}

// GetUserWalletByCurrency retrieves a specific wallet by currency
func (r *WalletRepository) GetUserWalletByCurrency(ctx context.Context, userID int, currency string) (*models.UserWallet, error) {
	query := `
		SELECT id, user_id, request_id, wallet_id, currency, address, address_type, derive_path, status, is_primary, created_at, updated_at
		FROM user_wallets WHERE user_id = $1 AND currency = $2 LIMIT 1`

	var w models.UserWallet
	err := r.db.QueryRowContext(ctx, query, userID, currency).Scan(
		&w.ID, &w.UserID, &w.RequestID, &w.WalletID, &w.Currency,
		&w.Address, &w.AddressType, &w.DerivePath, &w.Status, &w.IsPrimary,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// UpdateUserWalletStatus updates the status of a user wallet
func (r *WalletRepository) UpdateUserWalletStatus(ctx context.Context, id int, status models.UserWalletStatus) error {
	query := `UPDATE user_wallets SET status = $1, updated_at = NOW() WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("user wallet not found: %d", id)
	}
	return nil
}

// GetActiveUserWallet retrieves the first active (non-cancelled) wallet for a user from user_wallets
func (r *WalletRepository) GetActiveUserWallet(ctx context.Context, userID int) (*models.UserWallet, error) {
	query := `
		SELECT id, user_id, request_id, wallet_id, currency, address, address_type, derive_path, status, is_primary, created_at, updated_at
		FROM user_wallets WHERE user_id = $1 AND status != 'CANCELLED' ORDER BY created_at DESC LIMIT 1`

	var w models.UserWallet
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&w.ID, &w.UserID, &w.RequestID, &w.WalletID, &w.Currency,
		&w.Address, &w.AddressType, &w.DerivePath, &w.Status, &w.IsPrimary,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// AddUserWalletAddress adds a new address for the user, checking if it already exists.
// Returns the existing wallet if found, or creates a new one.
func (r *WalletRepository) AddUserWalletAddress(ctx context.Context, wallet *models.UserWallet) (*models.UserWallet, error) {
	// Check if wallet already exists for this user and currency
	existing, err := r.GetUserWalletByCurrency(ctx, wallet.UserID, wallet.Currency)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// Create new wallet address
	wallet.Status = models.UserWalletStatusNormal
	wallet.IsPrimary = false
	err = r.CreateUserWallet(ctx, wallet)
	if err != nil {
		return nil, err
	}
	return wallet, nil
}

// GetUserWalletByUserAndCurrency gets wallet by user and currency
func (r *WalletRepository) GetUserWalletByUserAndCurrency(ctx context.Context, userID int, currency string) (*models.UserWallet, error) {
	return r.GetUserWalletByCurrency(ctx, userID, currency)
}

// Ensure WalletRepository implements repository.Wallet
var _ repository.Wallet = (*WalletRepository)(nil)
