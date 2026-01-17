// internal/migration/migrations/003_create_withdrawal_tables.go
package migrations

import (
	"database/sql"
	"fmt"

	"monera-digital/internal/migration"
)

// CreateWithdrawalTables migration
type CreateWithdrawalTables struct{}

func (m *CreateWithdrawalTables) Version() string {
	return "003"
}

func (m *CreateWithdrawalTables) Description() string {
	return "Create withdrawal related tables"
}

func (m *CreateWithdrawalTables) Up(db *sql.DB) error {
	// 1. Create account table (if not exists)
	// This mirrors the 'accounts' table in Drizzle schema but we call it 'account' per PRD SQL.
	// We'll use IF NOT EXISTS to be safe.
	accountQuery := `
	CREATE TABLE IF NOT EXISTS account (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		type VARCHAR(32) NOT NULL, -- 'WEALTH', 'FUND'
		currency VARCHAR(32) NOT NULL,
		balance DECIMAL(32, 16) DEFAULT 0 NOT NULL,
		frozen_balance DECIMAL(32, 16) DEFAULT 0 NOT NULL,
		version BIGINT DEFAULT 1 NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_account_user_id ON account(user_id);
	CREATE INDEX IF NOT EXISTS idx_account_frozen_balance ON account(user_id, frozen_balance);
	`
	if _, err := db.Exec(accountQuery); err != nil {
		return fmt.Errorf("failed to create account table: %w", err)
	}

	// 2. Create withdrawal_address_whitelist
	whitelistQuery := `
	CREATE TABLE IF NOT EXISTS withdrawal_address_whitelist (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		address_alias VARCHAR(255) NOT NULL,
		chain_type VARCHAR(32) NOT NULL,
		wallet_address VARCHAR(255) NOT NULL,
		verified BOOLEAN DEFAULT FALSE,
		verified_at TIMESTAMP,
		verification_method VARCHAR(32),
		is_deleted BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, wallet_address)
	);
	CREATE INDEX IF NOT EXISTS idx_whitelist_user_id ON withdrawal_address_whitelist(user_id);
	`
	if _, err := db.Exec(whitelistQuery); err != nil {
		return fmt.Errorf("failed to create withdrawal_address_whitelist table: %w", err)
	}

	// 3. Create withdrawal_request
	requestQuery := `
	CREATE TABLE IF NOT EXISTS withdrawal_request (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		request_id VARCHAR(64) NOT NULL UNIQUE,
		status VARCHAR(32) DEFAULT 'PROCESSING',
		error_code VARCHAR(64),
		error_message TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_request_request_id ON withdrawal_request(request_id);
	`
	if _, err := db.Exec(requestQuery); err != nil {
		return fmt.Errorf("failed to create withdrawal_request table: %w", err)
	}

	// 4. Create withdrawal_order
	orderQuery := `
	CREATE TABLE IF NOT EXISTS withdrawal_order (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		amount DECIMAL(32, 16) NOT NULL,
		network_fee DECIMAL(32, 16),
		platform_fee DECIMAL(32, 16),
		actual_amount DECIMAL(32, 16),
		chain_type VARCHAR(32) NOT NULL,
		coin_type VARCHAR(32) NOT NULL,
		to_address VARCHAR(255) NOT NULL,
		safeheron_order_id VARCHAR(64),
		transaction_hash VARCHAR(255),
		status VARCHAR(32) DEFAULT 'PENDING',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		sent_at TIMESTAMP,
		confirmed_at TIMESTAMP,
		completed_at TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_order_user_id ON withdrawal_order(user_id);
	CREATE INDEX IF NOT EXISTS idx_order_status ON withdrawal_order(status);
	`
	if _, err := db.Exec(orderQuery); err != nil {
		return fmt.Errorf("failed to create withdrawal_order table: %w", err)
	}

	// 5. Create withdrawal_verification
	verificationQuery := `
	CREATE TABLE IF NOT EXISTS withdrawal_verification (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		withdrawal_order_id INTEGER NOT NULL, -- references withdrawal_order(id), but simplistic here
		verification_method VARCHAR(32) NOT NULL,
		verification_code VARCHAR(255),
		attempts INTEGER DEFAULT 0,
		max_attempts INTEGER DEFAULT 3,
		verified BOOLEAN DEFAULT FALSE,
		verified_at TIMESTAMP,
		expires_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_verification_user_id ON withdrawal_verification(user_id);
	`
	if _, err := db.Exec(verificationQuery); err != nil {
		return fmt.Errorf("failed to create withdrawal_verification table: %w", err)
	}

	// 6. Create withdrawal_freeze_log
	logQuery := `
	CREATE TABLE IF NOT EXISTS withdrawal_freeze_log (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		order_id INTEGER NOT NULL,
		amount DECIMAL(32, 16) NOT NULL,
		frozen_at TIMESTAMP,
		released_at TIMESTAMP,
		reason VARCHAR(64),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_log_user_id ON withdrawal_freeze_log(user_id);
	`
	if _, err := db.Exec(logQuery); err != nil {
		return fmt.Errorf("failed to create withdrawal_freeze_log table: %w", err)
	}

	return nil
}

func (m *CreateWithdrawalTables) Down(db *sql.DB) error {
	queries := []string{
		`DROP TABLE IF EXISTS withdrawal_freeze_log`,
		`DROP TABLE IF EXISTS withdrawal_verification`,
		`DROP TABLE IF EXISTS withdrawal_order`,
		`DROP TABLE IF EXISTS withdrawal_request`,
		`DROP TABLE IF EXISTS withdrawal_address_whitelist`,
		// We do NOT drop 'account' table as it might have been created by other migrations or we don't want to lose data.
		// Or we could drop the column frozen_balance if we wanted to be precise.
		`ALTER TABLE account DROP COLUMN IF EXISTS frozen_balance`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute down migration: %w", err)
		}
	}
	return nil
}

// Ensure CreateWithdrawalTables implements Migration interface
var _ migration.Migration = (*CreateWithdrawalTables)(nil)
