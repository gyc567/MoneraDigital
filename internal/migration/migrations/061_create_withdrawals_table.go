// internal/migration/migrations/061_create_withdrawals_table.go
package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"monera-digital/internal/migration"
)

// CreateWithdrawalsTable creates the withdrawals table together with its
// prerequisites (withdrawal_status / address_type enums and the
// withdrawal_addresses table).
//
// Background: the withdrawals table was defined in drizzle/schema.ts and
// monera_complete_schema.sql but never had a Go migration. DBs initialized
// via the migration runner (e.g. stage/prod preview) were missing it, so the
// admin member-withdraws route (SELECT ... FROM withdrawals) failed with 500;
// local DBs initialized from the complete-schema dump happened to have it.
//
// All statements are idempotent — DO blocks for enums (duplicate_object no-op)
// and IF NOT EXISTS for tables/indexes — so this is safe to run on a DB that
// already has any of these objects (e.g. local).
type CreateWithdrawalsTable struct{}

func (m *CreateWithdrawalsTable) Version() string { return "061" }

func (m *CreateWithdrawalsTable) Description() string {
	return "Create withdrawals table (with withdrawal_status/address_type enums and withdrawal_addresses)"
}

func (m *CreateWithdrawalsTable) Up(db *sql.DB) error {
	ctx := context.Background()
	stmts := []string{
		`DO $$ BEGIN
			CREATE TYPE address_type AS ENUM ('BTC', 'ETH', 'USDC', 'USDT');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
		`DO $$ BEGIN
			CREATE TYPE withdrawal_status AS ENUM ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
		`CREATE TABLE IF NOT EXISTS withdrawal_addresses (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			address TEXT NOT NULL,
			address_type address_type NOT NULL,
			label TEXT NOT NULL,
			is_verified BOOLEAN DEFAULT FALSE NOT NULL,
			is_primary BOOLEAN DEFAULT FALSE NOT NULL,
			created_at TIMESTAMP DEFAULT NOW() NOT NULL,
			verified_at TIMESTAMP,
			deactivated_at TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_withdrawal_addresses_user_id ON withdrawal_addresses(user_id)`,
		`CREATE TABLE IF NOT EXISTS withdrawals (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			from_address_id INTEGER NOT NULL REFERENCES withdrawal_addresses(id),
			amount NUMERIC(20, 8) NOT NULL,
			asset TEXT NOT NULL,
			to_address TEXT NOT NULL,
			status withdrawal_status DEFAULT 'PENDING' NOT NULL,
			tx_hash TEXT,
			created_at TIMESTAMP DEFAULT NOW() NOT NULL,
			completed_at TIMESTAMP,
			failure_reason TEXT,
			fee_amount NUMERIC(20, 8),
			received_amount NUMERIC(20, 8),
			safeheron_tx_id TEXT,
			chain TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id)`,
	}
	for i, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("061 statement %d failed: %w", i, err)
		}
	}
	return nil
}

func (m *CreateWithdrawalsTable) Down(db *sql.DB) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS withdrawals CASCADE`)
	if err != nil {
		return fmt.Errorf("failed to drop withdrawals table: %w", err)
	}
	return nil
}

var _ migration.Migration = (*CreateWithdrawalsTable)(nil)
