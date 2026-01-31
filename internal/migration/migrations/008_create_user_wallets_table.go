package migrations

import (
	"database/sql"
	"fmt"
	"monera-digital/internal/migration"
)

// CreateUserWalletsTable migration
type CreateUserWalletsTable struct{}

func (m *CreateUserWalletsTable) Version() string {
	return "008"
}

func (m *CreateUserWalletsTable) Description() string {
	return "Create user_wallets table for storing individual wallet addresses"
}

func (m *CreateUserWalletsTable) Up(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_wallets (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			request_id VARCHAR(36) NOT NULL REFERENCES wallet_creation_requests(request_id) ON DELETE SET NULL,
			wallet_id VARCHAR(100) NOT NULL,
			currency VARCHAR(50) NOT NULL,
			address VARCHAR(255) NOT NULL,
			address_type VARCHAR(50),
			derive_path VARCHAR(100),
			is_primary BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create user_wallets table: %w", err)
	}

	// Create indexes for common queries
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_user_wallets_user_id ON user_wallets(user_id)
	`)
	if err != nil {
		return fmt.Errorf("failed to create user_id index: %w", err)
	}

	_, err = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_wallets_user_currency ON user_wallets(user_id, currency)
	`)
	if err != nil {
		return fmt.Errorf("failed to create unique index: %w", err)
	}

	return nil
}

func (m *CreateUserWalletsTable) Down(db *sql.DB) error {
	_, err := db.Exec(`
		DROP INDEX IF EXISTS idx_user_wallets_user_currency;
		DROP INDEX IF EXISTS idx_user_wallets_user_id;
		DROP TABLE IF EXISTS user_wallets;
	`)
	return err
}

// Ensure CreateUserWalletsTable implements Migration interface
var _ migration.Migration = (*CreateUserWalletsTable)(nil)
