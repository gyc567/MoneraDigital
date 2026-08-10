package migrations

import (
	"strings"
	"testing"

	"monera-digital/internal/migration"
)

func TestAddSafeheronRoutingStatusChecksContract(t *testing.T) {
	var controlled migration.ControlledMigration = &AddSafeheronRoutingStatusChecks{}
	if controlled.Version() != "064" || controlled.RequiredPreexistingVersion() != "063" || controlled.RequiredExpectedCeiling() != "064" {
		t.Fatalf("controlled migration contract = %s/%s/%s", controlled.Version(), controlled.RequiredPreexistingVersion(), controlled.RequiredExpectedCeiling())
	}
	if controlled.Description() == "" {
		t.Fatal("migration description must not be empty")
	}
}

func TestMigration064CreatesDurableTxKeyScopedStatusChecks(t *testing.T) {
	for _, fragment := range []string{
		"CREATE TABLE public.safeheron_transaction_routing_status_checks",
		"safeheron_tx_key VARCHAR(128) PRIMARY KEY",
		"attempt_count INTEGER NOT NULL DEFAULT 0",
		"next_check_at TIMESTAMPTZ",
		"last_checked_at TIMESTAMPTZ",
		"last_check_outcome VARCHAR(16)",
		"last_observed_status VARCHAR(64)",
		"last_provider_event_id VARCHAR(128)",
		"last_error_code VARCHAR(64)",
		"lease_owner VARCHAR(128)",
		"lease_expires_at TIMESTAMPTZ",
		"completed_at TIMESTAMPTZ",
		"idx_safeheron_routing_status_checks_claim",
		"'sla:pending:level:'",
		"'sla:open:level:'",
		"|| (payload->>'level')",
		"WHERE alert_type='SLA_ESCALATION'",
	} {
		if !strings.Contains(migration064SchemaSQL, fragment) {
			t.Errorf("migration 064 schema is missing %q", fragment)
		}
	}
}
