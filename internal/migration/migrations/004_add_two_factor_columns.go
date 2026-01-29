package migrations

import (
	"database/sql"
	"fmt"

	"monera-digital/internal/migration"
)

// AddTwoFactorColumnsMigration adds 2FA columns to the users table
type AddTwoFactorColumnsMigration struct{}

func (m *AddTwoFactorColumnsMigration) Version() string {
	return "004"
}

func (m *AddTwoFactorColumnsMigration) Description() string {
	return "Add two factor columns to users table"
}

func (m *AddTwoFactorColumnsMigration) Up(db *sql.DB) error {
	// Add two_factor_secret column
	_, err := db.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS two_factor_secret TEXT
	`)
	if err != nil {
		return fmt.Errorf("failed to add two_factor_secret column: %w", err)
	}

	// Add two_factor_backup_codes column
	_, err = db.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS two_factor_backup_codes TEXT
	`)
	if err != nil {
		return fmt.Errorf("failed to add two_factor_backup_codes column: %w", err)
	}

	// Add two_factor_enabled column (default false)
	_, err = db.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS two_factor_enabled BOOLEAN DEFAULT FALSE
	`)
	if err != nil {
		return fmt.Errorf("failed to add two_factor_enabled column: %w", err)
	}

	return nil
}

func (m *AddTwoFactorColumnsMigration) Down(db *sql.DB) error {
	// Remove columns in reverse order
	_, err := db.Exec(`
		ALTER TABLE users
		DROP COLUMN IF EXISTS two_factor_enabled
	`)
	if err != nil {
		return fmt.Errorf("failed to drop two_factor_enabled column: %w", err)
	}

	_, err = db.Exec(`
		ALTER TABLE users
		DROP COLUMN IF EXISTS two_factor_backup_codes
	`)
	if err != nil {
		return fmt.Errorf("failed to drop two_factor_backup_codes column: %w", err)
	}

	_, err = db.Exec(`
		ALTER TABLE users
		DROP COLUMN IF EXISTS two_factor_secret
	`)
	if err != nil {
		return fmt.Errorf("failed to drop two_factor_secret column: %w", err)
	}

	return nil
}

// Ensure AddTwoFactorColumnsMigration implements Migration interface
var _ migration.Migration = (*AddTwoFactorColumnsMigration)(nil)
