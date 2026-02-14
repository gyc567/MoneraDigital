package migrations

import (
	"database/sql"
	"fmt"
	"monera-digital/internal/migration"
)

// AddIsPrimaryToWhitelist migration adds is_primary column to withdrawal_address_whitelist table
type AddIsPrimaryToWhitelist struct{}

func (m *AddIsPrimaryToWhitelist) Version() string {
	return "010"
}

func (m *AddIsPrimaryToWhitelist) Description() string {
	return "Add is_primary column to withdrawal_address_whitelist table"
}

func (m *AddIsPrimaryToWhitelist) Up(db *sql.DB) error {
	// Check if column already exists
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'withdrawal_address_whitelist' 
			AND column_name = 'is_primary'
		)
	`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check column existence: %w", err)
	}

	if !exists {
		_, err = db.Exec(`
			ALTER TABLE withdrawal_address_whitelist 
			ADD COLUMN is_primary BOOLEAN DEFAULT FALSE NOT NULL
		`)
		if err != nil {
			return fmt.Errorf("failed to add is_primary column: %w", err)
		}
	}

	return nil
}

func (m *AddIsPrimaryToWhitelist) Down(db *sql.DB) error {
	_, err := db.Exec(`
		ALTER TABLE withdrawal_address_whitelist 
		DROP COLUMN IF EXISTS is_primary
	`)
	return err
}

// Ensure AddIsPrimaryToWhitelist implements Migration interface
var _ migration.Migration = (*AddIsPrimaryToWhitelist)(nil)
