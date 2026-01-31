package migrations

import (
	"database/sql"
	"fmt"
	"monera-digital/internal/migration"
)

// AddUserWalletStatusField migration
type AddUserWalletStatusField struct{}

func (m *AddUserWalletStatusField) Version() string {
	return "009"
}

func (m *AddUserWalletStatusField) Description() string {
	return "Add status field to user_wallets and remove NOT NULL from request_id"
}

func (m *AddUserWalletStatusField) Up(db *sql.DB) error {
	// Remove NOT NULL from request_id
	_, err := db.Exec(`
		ALTER TABLE user_wallets ALTER COLUMN request_id DROP NOT NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to remove NOT NULL from request_id: %w", err)
	}

	// Add status column with default NORMAL
	_, err = db.Exec(`
		ALTER TABLE user_wallets ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'NORMAL'
	`)
	if err != nil {
		return fmt.Errorf("failed to add status column: %w", err)
	}

	// Add check constraint for valid status values
	_, err = db.Exec(`
		ALTER TABLE user_wallets DROP CONSTRAINT IF EXISTS ck_user_wallets_status;
		ALTER TABLE user_wallets ADD CONSTRAINT ck_user_wallets_status
		CHECK (status IN ('NORMAL', 'FROZEN', 'CANCELLED'))
	`)
	if err != nil {
		return fmt.Errorf("failed to add status check constraint: %w", err)
	}

	return nil
}

func (m *AddUserWalletStatusField) Down(db *sql.DB) error {
	// Remove check constraint
	_, err := db.Exec(`
		ALTER TABLE user_wallets DROP CONSTRAINT IF EXISTS ck_user_wallets_status
	`)
	if err != nil {
		return fmt.Errorf("failed to drop check constraint: %w", err)
	}

	// Remove status column
	_, err = db.Exec(`
		ALTER TABLE user_wallets DROP COLUMN IF EXISTS status
	`)
	if err != nil {
		return fmt.Errorf("failed to drop status column: %w", err)
	}

	// Restore NOT NULL on request_id
	_, err = db.Exec(`
		ALTER TABLE user_wallets ALTER COLUMN request_id SET NOT NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to restore NOT NULL on request_id: %w", err)
	}

	return nil
}

// Ensure AddUserWalletStatusField implements Migration interface
var _ migration.Migration = (*AddUserWalletStatusField)(nil)
