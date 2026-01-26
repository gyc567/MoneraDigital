package migrations

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration("006_create_pending_login_sessions", upCreatePendingLoginSessions, downCreatePendingLoginSessions)
}

func upCreatePendingLoginSessions(ctx context.Context, conn *pgx.Conn) error {
	// Create the pending_login_sessions table for tracking 2FA verification attempts
	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS pending_login_sessions (
			id SERIAL PRIMARY KEY,
			session_id TEXT NOT NULL UNIQUE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT NOW() NOT NULL,
			expires_at TIMESTAMP NOT NULL
		);

		-- Index for fast sessionId lookups
		CREATE INDEX IF NOT EXISTS idx_pending_login_sessions_session_id ON pending_login_sessions(session_id);

		-- Index for cleaning up expired sessions
		CREATE INDEX IF NOT EXISTS idx_pending_login_sessions_expires_at ON pending_login_sessions(expires_at);

		-- Index for user lookups (if needed for cleanup)
		CREATE INDEX IF NOT EXISTS idx_pending_login_sessions_user_id ON pending_login_sessions(user_id);
	`)

	return err
}

func downCreatePendingLoginSessions(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `
		DROP TABLE IF EXISTS pending_login_sessions CASCADE;
	`)
	return err
}
