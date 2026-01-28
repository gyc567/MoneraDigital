package migrations

import (
	"database/sql"
	"fmt"

	"monera-digital/internal/migration"
)

// AddTwoFactorTimestampMigration adds timestamp field for replay attack prevention
type AddTwoFactorTimestampMigration struct{}

func (m *AddTwoFactorTimestampMigration) Version() string {
	return "005"
}

func (m *AddTwoFactorTimestampMigration) Description() string {
	return "Add two factor timestamp column for replay attack prevention"
}

func (m *AddTwoFactorTimestampMigration) Up(db *sql.DB) error {
	// Add two_factor_last_used_at column for replay attack prevention
	_, err := db.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS two_factor_last_used_at BIGINT DEFAULT 0
	`)
	if err != nil {
		return fmt.Errorf("failed to add two_factor_last_used_at column: %w", err)
	}

	// Create index for performance (optional but recommended)
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_users_two_factor_last_used_at
		ON users(two_factor_last_used_at)
		WHERE two_factor_enabled = TRUE
	`)
	if err != nil {
		return fmt.Errorf("failed to create two_factor_last_used_at index: %w", err)
	}

	return nil
}

func (m *AddTwoFactorTimestampMigration) Down(db *sql.DB) error {
	// Drop index first
	_, err := db.Exec(`
		DROP INDEX IF EXISTS idx_users_two_factor_last_used_at
	`)
	if err != nil {
		return fmt.Errorf("failed to drop two_factor_last_used_at index: %w", err)
	}

	// Remove column
	_, err = db.Exec(`
		ALTER TABLE users
		DROP COLUMN IF EXISTS two_factor_last_used_at
	`)
	if err != nil {
		return fmt.Errorf("failed to drop two_factor_last_used_at column: %w", err)
	}

	return nil
}

// Ensure AddTwoFactorTimestampMigration implements Migration interface
var _ migration.Migration = (*AddTwoFactorTimestampMigration)(nil)
