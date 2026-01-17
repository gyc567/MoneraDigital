package postgres

import (
	"context"
	"database/sql"
	"time"

	"monera-digital/internal/models"
	"monera-digital/internal/repository"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) repository.Account {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) GetByUserIDAndType(ctx context.Context, userID int, accountType string) (*models.Account, error) {
	var account models.Account
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, type, currency, balance, frozen_balance, version, created_at, updated_at
		 FROM account WHERE user_id = $1 AND type = $2`,
		userID, accountType).Scan(
		&account.ID, &account.UserID, &account.Type, &account.Currency,
		&account.Balance, &account.FrozenBalance, &account.Version,
		&account.CreatedAt, &account.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *AccountRepository) Create(ctx context.Context, account *models.Account) error {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO account (user_id, type, currency, balance, frozen_balance, version, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		account.UserID, account.Type, account.Currency, account.Balance, account.FrozenBalance,
		1, time.Now(), time.Now()).Scan(&account.ID)
	return err
}

func (r *AccountRepository) UpdateFrozenBalance(ctx context.Context, userID int, amount float64) error {
	// Add to frozen (and check balance if needed, but logic usually in Service.
	// PRD says: UPDATE account SET frozen_balance = frozen_balance + amount ...
	result, err := r.db.ExecContext(ctx,
		`UPDATE account
		 SET frozen_balance = frozen_balance + $1, version = version + 1, updated_at = $3
		 WHERE user_id = $2`,
		amount, userID, time.Now())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *AccountRepository) ReleaseFrozenBalance(ctx context.Context, userID int, amount float64) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE account
		 SET frozen_balance = frozen_balance - $1, version = version + 1, updated_at = $3
		 WHERE user_id = $2 AND frozen_balance >= $1`,
		amount, userID, time.Now())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		// Could mean not found OR frozen balance insufficient (should not happen if logic correct)
		return repository.ErrNotFound
	}
	return nil
}

func (r *AccountRepository) DeductBalance(ctx context.Context, userID int, amount float64) error {
	// Deduct from BOTH balance and frozen (since it was frozen first).
	// PRD 5.3: frozen_balance = frozen_balance - amount, balance = balance - amount
	result, err := r.db.ExecContext(ctx,
		`UPDATE account
		 SET frozen_balance = frozen_balance - $1, balance = balance - $1, version = version + 1, updated_at = $3
		 WHERE user_id = $2 AND frozen_balance >= $1`,
		amount, userID, time.Now())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return repository.ErrNotFound
	}
	return nil
}
