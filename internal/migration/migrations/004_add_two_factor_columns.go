package migrations

import (
	"database/sql"
)

// AddTwoFactorColumns adds 2FA columns to the users table
func AddTwoFactorColumns(tx *sql.Tx) error {
	// Add two_factor_secret column
	_, err := tx.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS two_factor_secret TEXT
	`)
	if err != nil {
		return err
	}

	// Add two_factor_backup_codes column
	_, err = tx.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS two_factor_backup_codes TEXT
	`)
	if err != nil {
		return err
	}

	// Add two_factor_enabled column (default false)
	_, err = tx.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS two_factor_enabled BOOLEAN DEFAULT FALSE
	`)
	if err != nil {
		return err
	}

	return nil
}

// RemoveTwoFactorColumns removes 2FA columns from the users table (rollback)
func RemoveTwoFactorColumns(tx *sql.Tx) error {
	// Remove columns in reverse order
	_, err := tx.Exec(`
		ALTER TABLE users
		DROP COLUMN IF EXISTS two_factor_enabled
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE users
		DROP COLUMN IF EXISTS two_factor_backup_codes
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		ALTER TABLE users
		DROP COLUMN IF EXISTS two_factor_secret
	`)
	if err != nil {
		return err
	}

	return nil
}
