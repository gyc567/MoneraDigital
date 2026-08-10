package companyfund

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAirwallexIdentityCorrectionPostgresIntegration(t *testing.T) {
	if os.Getenv(airwallexPhase2PostgresGate) != "1" {
		t.Skip("set RUN_COMPANY_FUND_AIRWALLEX_PHASE2_INTEGRATION=1 to run PostgreSQL lifecycle coverage")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when Airwallex lifecycle integration coverage is enabled")
	}
	db := newAirwallexPhase2PostgresFixture(t, databaseURL)
	ctx := context.Background()
	for _, table := range []string{
		"company_fund_account_lifecycle_audits",
		"company_fund_account_lifecycle_commands",
		"company_fund_ledger_tasks",
		"company_fund_transactions",
		"company_fund_provider_transaction_facts",
		"company_fund_provider_events",
		"company_fund_accounts",
	} {
		if _, err := db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}

	const oldKey = "awx-lifecycle-old"
	const newKey = "awx-lifecycle-new"
	var accountID, lifecycleVersion int64
	if err := db.QueryRowContext(ctx, `
INSERT INTO company_fund_accounts (
  channel, provider_account_key, account_name, is_enabled,
  monitoring_started_at, first_enabled_at, airwallex_lifecycle
) VALUES (
  'AIRWALLEX', $1, 'Lifecycle integration', true,
  clock_timestamp(), clock_timestamp(), 'CURRENT'
)
RETURNING id, lifecycle_version`, oldKey).Scan(&accountID, &lifecycleVersion); err != nil {
		t.Fatalf("insert lifecycle account: %v", err)
	}

	repository := NewDBRepository(db)
	movement := validProviderEventWorkerMovement("lifecycle-old-movement-key")
	movement.ProviderAccountKey = oldKey
	movement.ProviderTransactionID = "provider-transaction-stable"
	movement.ProviderMovementID = "provider-movement-stable"
	movement.ToCompanyFundAccountID = &accountID
	first, err := repository.UpsertCompanyFundTransaction(ctx, movement)
	if err != nil || !first.Inserted {
		t.Fatalf("insert pre-correction movement = %#v, %v", first, err)
	}

	providerEvent := validOwnedProviderEvent()
	providerEvent.ProviderEventID = "lifecycle-provider-event"
	providerEvent.ProviderAccountKey = oldKey
	event, err := repository.InsertProviderEvent(ctx, providerEvent)
	if err != nil || !event.Inserted {
		t.Fatalf("insert pre-correction provider event = %#v, %v", event, err)
	}

	if _, err := db.ExecContext(ctx, `
INSERT INTO company_fund_account_lifecycle_commands (
  command_type, target_account_id, requested_provider_account_key,
  requested_by, reason, idempotency_key, expected_target_version
) VALUES (
  'CORRECT_IDENTITY', $1, $2,
  'integration@example.com', 'correct sandbox account identity',
  'lifecycle-correction-integration', $3
)`, accountID, newKey, lifecycleVersion); err != nil {
		t.Fatalf("insert correction command: %v", err)
	}

	registry := NewCompanyFundAccountRegistry(
		NewPostgresAccountRegistryLoader(db),
		time.Minute,
	)
	if err := registry.Load(ctx); err != nil {
		t.Fatalf("load lifecycle registry: %v", err)
	}
	validator := &fakeAirwallexAccountIdentityValidator{
		summary: AirwallexProviderIdentitySummary{
			ProviderAccountID: newKey,
			AccountName:       "Lifecycle integration",
		},
	}
	worker := newAccountLifecycleWorkerForTest(t, repository, validator, registry)
	result, err := worker.ProcessNext(ctx)
	if err != nil || result.Outcome != AccountLifecycleProcessSucceeded {
		t.Fatalf("process correction = %#v, %v", result, err)
	}

	var correctedAccountKey, correctedEventKey, correctedTransactionKey, movementKey string
	var transactionID int64
	if err := db.QueryRowContext(ctx,
		`SELECT provider_account_key FROM company_fund_accounts WHERE id = $1`,
		accountID,
	).Scan(&correctedAccountKey); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT provider_account_key FROM company_fund_provider_events WHERE id = $1`,
		event.ID,
	).Scan(&correctedEventKey); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `
SELECT id, provider_account_key, movement_key
FROM company_fund_transactions
WHERE id = $1`, first.ID).Scan(
		&transactionID,
		&correctedTransactionKey,
		&movementKey,
	); err != nil {
		t.Fatal(err)
	}
	if correctedAccountKey != newKey || correctedEventKey != newKey ||
		correctedTransactionKey != newKey || transactionID != first.ID ||
		movementKey != movement.MovementKey {
		t.Fatalf(
			"correction account=%q event=%q transaction=%q id=%d movement=%q",
			correctedAccountKey,
			correctedEventKey,
			correctedTransactionKey,
			transactionID,
			movementKey,
		)
	}
	if account, found := registry.Snapshot().LookupAirwallex(newKey); !found ||
		account.ID != accountID {
		t.Fatalf("refreshed registry does not expose corrected current account")
	}

	replay := movement
	replay.ProviderAccountKey = newKey
	replay.MovementKey = "lifecycle-new-movement-key"
	replayed, err := repository.UpsertCompanyFundTransaction(ctx, replay)
	if err != nil {
		t.Fatalf("replay corrected movement: %v", err)
	}
	if replayed.Inserted || replayed.ID != first.ID {
		t.Fatalf("replay created duplicate movement: first=%#v replay=%#v", first, replayed)
	}
	var movementCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*)
FROM company_fund_transactions
WHERE channel = 'AIRWALLEX'
  AND provider_transaction_id = $1
  AND provider_movement_id = $2`,
		replay.ProviderTransactionID,
		replay.ProviderMovementID,
	).Scan(&movementCount); err != nil {
		t.Fatal(err)
	}
	if movementCount != 1 {
		t.Fatalf("corrected replay movement count = %d, want 1", movementCount)
	}
}

func TestAirwallexCutoverRejectsUnfinishedPriorAccountEventsPostgresIntegration(t *testing.T) {
	if os.Getenv(airwallexPhase2PostgresGate) != "1" {
		t.Skip("set RUN_COMPANY_FUND_AIRWALLEX_PHASE2_INTEGRATION=1 to run PostgreSQL lifecycle coverage")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when Airwallex lifecycle integration coverage is enabled")
	}
	db := newAirwallexPhase2PostgresFixture(t, databaseURL)
	ctx := context.Background()
	for _, table := range []string{
		"company_fund_account_lifecycle_audits",
		"company_fund_account_lifecycle_commands",
		"company_fund_provider_events",
		"company_fund_accounts",
	} {
		if _, err := db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}

	var currentID, currentVersion, candidateID, candidateVersion int64
	if err := db.QueryRowContext(ctx, `
INSERT INTO company_fund_accounts (
  channel, provider_account_key, account_name, is_enabled,
  monitoring_started_at, first_enabled_at, airwallex_lifecycle
) VALUES (
  'AIRWALLEX', 'awx-cutover-current', 'Current integration account', true,
  clock_timestamp(), clock_timestamp(), 'CURRENT'
)
RETURNING id, lifecycle_version`).Scan(&currentID, &currentVersion); err != nil {
		t.Fatalf("insert current lifecycle account: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
INSERT INTO company_fund_accounts (
  channel, provider_account_key, account_name, is_enabled,
  monitoring_started_at, airwallex_lifecycle
) VALUES (
  'AIRWALLEX', 'awx-cutover-candidate', 'Candidate integration account', false,
  clock_timestamp(), 'CANDIDATE'
)
RETURNING id, lifecycle_version`).Scan(&candidateID, &candidateVersion); err != nil {
		t.Fatalf("insert candidate lifecycle account: %v", err)
	}

	repository := NewDBRepository(db)
	providerEvent := validOwnedProviderEvent()
	providerEvent.ProviderEventID = "cutover-unfinished-provider-event"
	providerEvent.ProviderAccountKey = "awx-cutover-current"
	if result, err := repository.InsertProviderEvent(ctx, providerEvent); err != nil || !result.Inserted {
		t.Fatalf("insert unfinished provider event = %#v, %v", result, err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO company_fund_account_lifecycle_commands (
  command_type, target_account_id, related_account_id,
  requested_by, reason, idempotency_key,
  expected_target_version, expected_related_version
) VALUES (
  'CUTOVER', $1, $2,
  'integration@example.com', 'cut over after queue drain',
  'lifecycle-cutover-unfinished-integration', $3, $4
)`, candidateID, currentID, candidateVersion, currentVersion); err != nil {
		t.Fatalf("insert cutover command: %v", err)
	}

	lease, claimed, err := repository.ClaimAccountLifecycleCommand(ctx, "integration-worker", time.Minute)
	if err != nil || !claimed {
		t.Fatalf("claim cutover command = %#v, %t, %v", lease, claimed, err)
	}
	err = repository.ApplyAccountLifecycleCommand(ctx, AccountLifecycleApplyInput{
		Lease: lease,
		ProviderIdentity: AirwallexProviderIdentitySummary{
			ProviderAccountID: "awx-cutover-candidate",
			AccountName:       "Candidate integration account",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unfinished provider events") {
		t.Fatalf("cutover error = %v, want unfinished provider events rejection", err)
	}

	var currentLifecycle, candidateLifecycle string
	if err := db.QueryRowContext(
		ctx,
		`SELECT airwallex_lifecycle FROM company_fund_accounts WHERE id = $1`,
		currentID,
	).Scan(&currentLifecycle); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(
		ctx,
		`SELECT airwallex_lifecycle FROM company_fund_accounts WHERE id = $1`,
		candidateID,
	).Scan(&candidateLifecycle); err != nil {
		t.Fatal(err)
	}
	if currentLifecycle != "CURRENT" || candidateLifecycle != "CANDIDATE" {
		t.Fatalf("cutover changed lifecycle despite unfinished event: current=%s candidate=%s", currentLifecycle, candidateLifecycle)
	}
}

func TestAirwallexCandidateDeletionIgnoresTerminalLifecycleHistoryPostgresIntegration(t *testing.T) {
	if os.Getenv(airwallexPhase2PostgresGate) != "1" {
		t.Skip("set RUN_COMPANY_FUND_AIRWALLEX_PHASE2_INTEGRATION=1 to run PostgreSQL lifecycle coverage")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when Airwallex lifecycle integration coverage is enabled")
	}
	db := newAirwallexPhase2PostgresFixture(t, databaseURL)
	ctx := context.Background()
	for _, table := range []string{
		"company_fund_account_lifecycle_audits",
		"company_fund_account_lifecycle_commands",
		"company_fund_accounts",
	} {
		if _, err := db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}

	var accountID, lifecycleVersion int64
	if err := db.QueryRowContext(ctx, `
INSERT INTO company_fund_accounts (
  channel, provider_account_key, account_name, is_enabled,
  monitoring_started_at, airwallex_lifecycle
) VALUES (
  'AIRWALLEX', 'awx-delete-failed-candidate', 'Failed candidate', false,
  clock_timestamp(), 'CANDIDATE'
)
RETURNING id, lifecycle_version`).Scan(&accountID, &lifecycleVersion); err != nil {
		t.Fatalf("insert deletion candidate: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO company_fund_account_lifecycle_commands (
  command_type, target_account_id, requested_by, reason,
  idempotency_key, expected_target_version, status,
  error_code, error_message, completed_at
) VALUES (
  'VALIDATE_CANDIDATE', $1, 'integration@example.com',
  'invalid provider identity',
  'lifecycle-delete-terminal-history', $2, 'FAILED',
  'PROVIDER_VALIDATION_FAILED', 'validation failed', clock_timestamp()
)`, accountID, lifecycleVersion); err != nil {
		t.Fatalf("insert terminal validation history: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO company_fund_account_lifecycle_commands (
  command_type, target_account_id, requested_by, reason,
  idempotency_key, expected_target_version
) VALUES (
  'DELETE_CANDIDATE', $1, 'integration@example.com',
  'remove invalid unreferenced candidate',
  'lifecycle-delete-after-failed-validation', $2
)`, accountID, lifecycleVersion); err != nil {
		t.Fatalf("insert deletion command: %v", err)
	}

	worker := newAccountLifecycleWorkerForTest(
		t,
		NewDBRepository(db),
		&fakeAirwallexAccountIdentityValidator{},
		&fakeAccountRegistryRefresher{},
	)
	result, err := worker.ProcessNext(ctx)
	if err != nil || result.Outcome != AccountLifecycleProcessSucceeded {
		t.Fatalf("process candidate deletion = %#v, %v", result, err)
	}

	var lifecycle string
	var deleted bool
	if err := db.QueryRowContext(ctx, `
SELECT airwallex_lifecycle, deleted_at IS NOT NULL
FROM company_fund_accounts
WHERE id = $1`, accountID).Scan(&lifecycle, &deleted); err != nil {
		t.Fatal(err)
	}
	if lifecycle != "DELETED" || !deleted {
		t.Fatalf("candidate lifecycle=%s deleted=%t, want DELETED true", lifecycle, deleted)
	}
}

func TestAirwallexTransientValidationRetryPostgresIntegration(t *testing.T) {
	if os.Getenv(airwallexPhase2PostgresGate) != "1" {
		t.Skip("set RUN_COMPANY_FUND_AIRWALLEX_PHASE2_INTEGRATION=1 to run PostgreSQL lifecycle coverage")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when Airwallex lifecycle integration coverage is enabled")
	}
	db := newAirwallexPhase2PostgresFixture(t, databaseURL)
	ctx := context.Background()
	for _, table := range []string{
		"company_fund_account_lifecycle_audits",
		"company_fund_account_lifecycle_commands",
		"company_fund_accounts",
	} {
		if _, err := db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}

	var accountID, lifecycleVersion int64
	if err := db.QueryRowContext(ctx, `
INSERT INTO company_fund_accounts (
  channel, provider_account_key, account_name, is_enabled,
  monitoring_started_at, airwallex_lifecycle
) VALUES (
  'AIRWALLEX', 'awx-transient-candidate', 'Transient candidate', false,
  clock_timestamp(), 'CANDIDATE'
)
RETURNING id, lifecycle_version`).Scan(&accountID, &lifecycleVersion); err != nil {
		t.Fatalf("insert transient candidate: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO company_fund_account_lifecycle_commands (
  command_type, target_account_id, requested_by, reason,
  idempotency_key, expected_target_version
) VALUES (
  'VALIDATE_CANDIDATE', $1, 'integration@example.com',
  'exercise transient provider retry',
  'lifecycle-transient-retry-integration', $2
)`, accountID, lifecycleVersion); err != nil {
		t.Fatalf("insert validation command: %v", err)
	}

	worker := newAccountLifecycleWorkerForTest(
		t,
		NewDBRepository(db),
		&fakeAirwallexAccountIdentityValidator{err: ErrAirwallexNetwork},
		&fakeAccountRegistryRefresher{},
	)
	result, err := worker.ProcessNext(ctx)
	if err != nil || result.Outcome != AccountLifecycleProcessRetrying {
		t.Fatalf("process transient provider failure = %#v, %v", result, err)
	}

	var status, errorCode string
	var businessApplied bool
	var retryScheduled bool
	if err := db.QueryRowContext(ctx, `
SELECT status, error_code, business_applied_at IS NOT NULL,
       next_attempt_at > clock_timestamp()
FROM company_fund_account_lifecycle_commands
WHERE id = $1`, result.CommandID).Scan(
		&status,
		&errorCode,
		&businessApplied,
		&retryScheduled,
	); err != nil {
		t.Fatal(err)
	}
	if status != "PENDING" || errorCode != "PROVIDER_TEMPORARILY_UNAVAILABLE" ||
		businessApplied || !retryScheduled {
		t.Fatalf(
			"transient retry status=%s error=%s businessApplied=%t retryScheduled=%t",
			status,
			errorCode,
			businessApplied,
			retryScheduled,
		)
	}
}
