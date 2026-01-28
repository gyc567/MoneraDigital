package migrations

import (
	"database/sql"
	"fmt"
	"monera-digital/internal/migration"
)

// UpdateWalletRequestsTable migration
type UpdateWalletRequestsTable struct{}

func (m *UpdateWalletRequestsTable) Version() string {
	return "007"
}

func (m *UpdateWalletRequestsTable) Description() string {
	return "Update wallet requests table with product code and currency"
}

func (m *UpdateWalletRequestsTable) Up(db *sql.DB) error {
	// Add product_code and currency columns if they don't exist
	_, err := db.Exec(`
		ALTER TABLE wallet_creation_requests 
		ADD COLUMN IF NOT EXISTS product_code VARCHAR(50) DEFAULT '',
		ADD COLUMN IF NOT EXISTS currency VARCHAR(20) DEFAULT '';
	`)
	if err != nil {
		return fmt.Errorf("failed to add columns: %w", err)
	}

	// Create unique index for user_id, product_code, and currency
	// We first need to handle potential duplicates if there's existing data
	// For simplicity in this migration, we'll assume clean data or manual cleanup required if duplicates exist
	// In production, we might want to do a cleanup step first

	_, err = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_requests_user_product_currency 
		ON wallet_creation_requests (user_id, product_code, currency)
		WHERE status = 'SUCCESS';
	`)
	if err != nil {
		return fmt.Errorf("failed to create unique index: %w", err)
	}

	return nil
}

func (m *UpdateWalletRequestsTable) Down(db *sql.DB) error {
	_, err := db.Exec(`
		DROP INDEX IF EXISTS idx_wallet_requests_user_product_currency;
		ALTER TABLE wallet_creation_requests 
		DROP COLUMN IF EXISTS product_code,
		DROP COLUMN IF EXISTS currency;
	`)
	return err
}

// Ensure UpdateWalletRequestsTable implements Migration interface
var _ migration.Migration = (*UpdateWalletRequestsTable)(nil)
