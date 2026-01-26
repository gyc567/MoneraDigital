package migrations

import (
	"database/sql"
)

// AddTwoFactorTimestamp adds timestamp field for replay attack prevention
func AddTwoFactorTimestamp(tx *sql.Tx) error {
	// Add two_factor_last_used_at column for replay attack prevention
	_, err := tx.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS two_factor_last_used_at BIGINT DEFAULT 0
	`)
	if err != nil {
		return err
	}

	// Create index for performance (optional but recommended)
	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_users_two_factor_last_used_at
		ON users(two_factor_last_used_at)
		WHERE two_factor_enabled = TRUE
	`)
	if err != nil {
		return err
	}

	return nil
}

// RemoveTwoFactorTimestamp removes timestamp field (rollback)
func RemoveTwoFactorTimestamp(tx *sql.Tx) error {
	// Drop index first
	_, err := tx.Exec(`
		DROP INDEX IF EXISTS idx_users_two_factor_last_used_at
	`)
	if err != nil {
		return err
	}

	// Remove column
	_, err = tx.Exec(`
		ALTER TABLE users
		DROP COLUMN IF EXISTS two_factor_last_used_at
	`)
	if err != nil {
		return err
	}

	return nil
}
