package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const migration065PostgresIntegrationGate = "RUN_MIGRATION_065_POSTGRES_INTEGRATION"

func TestMigration065PostgresPreservesManualImportIdentityAndLineageContract(t *testing.T) {
	if os.Getenv(migration065PostgresIntegrationGate) != "1" {
		t.Skip("set RUN_MIGRATION_065_POSTGRES_INTEGRATION=1 to run isolated PostgreSQL coverage")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when migration 065 PostgreSQL integration is enabled")
	}
	db, schema := newMigration065PostgresFixture(t, databaseURL)
	if _, err := db.Exec(qualifyCompanyFundIntegrationSQL(migration065SchemaSQL, schema)); err != nil {
		t.Fatalf("apply migration 065 schema: %v", err)
	}

	accountID := insertMigration065Account(t, db, schema)
	manualOne := insertMigration065Transaction(t, db, schema, "MANUAL", "ADJUSTMENT", "INFLOW", nil, accountID, "bank-001")
	manualTwo := insertMigration065Transaction(t, db, schema, "MANUAL", "FEE", "OUTFLOW", accountID, nil, "bank-001")
	if manualOne == manualTwo {
		t.Fatal("fixture transaction IDs unexpectedly collided")
	}
	if _, err := db.Exec(`INSERT INTO "` + schema + `".company_fund_transactions (channel, external_transaction_reference) VALUES ('SAFEHERON', 'bank-001')`); err == nil {
		t.Fatal("provider transaction accepted a manual external reference")
	}

	parent := insertMigration065VoidedBatch(t, db, schema, digest065("parent"), accountID, nil)
	child := insertMigration065Batch(t, db, schema, digest065("parent"), "PROCESSING", &parent)
	if child == parent {
		t.Fatal("fixture batch IDs unexpectedly collided")
	}
	if _, err := db.Exec(migration065BatchInsertSQL(schema), digest065("other"), digest065("request-other"), "PROCESSING", parent, "retry"); err == nil {
		t.Fatal("cross-content predecessor was accepted")
	}
	activeParent := insertMigration065Batch(t, db, schema, digest065("active"), "PROCESSING", nil)
	if _, err := db.Exec(migration065BatchInsertSQL(schema), digest065("active"), digest065("request-active-child"), "PROCESSING", activeParent, "retry"); err == nil {
		t.Fatal("non-voided predecessor was accepted")
	}
	if _, err := db.Exec(migration065BatchInsertSQL(schema), digest065("parent"), digest065("request-second-child"), "PROCESSING", parent, "retry"); err == nil {
		t.Fatal("ambiguous effective predecessor replacement was accepted")
	}
	childPrincipal := insertMigration065Transaction(t, db, schema, "MANUAL", "ADJUSTMENT", "INFLOW", nil, accountID, "parent")
	insertMigration065ImportRow(t, db, schema, child, 2, "parent", childPrincipal, nil, accountID)
	markMigration065BatchVoided(t, db, schema, child, 1, 0)
	if _, err := db.Exec(`UPDATE "`+schema+`".company_fund_transaction_import_batches
SET predecessor_batch_id=$2, reimport_reason='cycle'
WHERE id=$1`, parent, child); err == nil {
		t.Fatal("cyclic predecessor chain was accepted")
	}

	countBatch := insertMigration065Batch(t, db, schema, digest065("counts"), "PROCESSING", nil)
	countPrincipal := insertMigration065Transaction(t, db, schema, "MANUAL", "ADJUSTMENT", "INFLOW", nil, accountID, "count-001")
	countFee := insertMigration065Transaction(t, db, schema, "MANUAL", "FEE", "OUTFLOW", accountID, nil, "count-001")
	insertMigration065ImportRow(t, db, schema, countBatch, 2, "count-001", countPrincipal, countFee, accountID)
	if _, err := db.Exec(`UPDATE "`+schema+`".company_fund_transaction_import_batches
SET status='SUCCEEDED', completed_at=clock_timestamp(), principal_transaction_count=1, fee_transaction_count=0
WHERE id=$1`, countBatch); err == nil {
		t.Fatal("terminal batch accepted counts that do not match its durable rows")
	}
	if _, err := db.Exec(`UPDATE "`+schema+`".company_fund_transaction_import_batches
SET status='SUCCEEDED', completed_at=clock_timestamp(), principal_transaction_count=1, fee_transaction_count=1
WHERE id=$1`, countBatch); err != nil {
		t.Fatalf("complete valid import batch: %v", err)
	}
	if _, err := db.Exec(`UPDATE "`+schema+`".company_fund_transaction_import_batches
SET status='VOIDED', voided_at=clock_timestamp(), voided_by=1, void_reason='test'
WHERE id=$1`, countBatch); err != nil {
		t.Fatalf("void succeeded import batch: %v", err)
	}
	if _, err := db.Exec(`UPDATE "`+schema+`".company_fund_transaction_import_batches
SET status='PROCESSING', completed_at=NULL
WHERE id=$1`, countBatch); err == nil {
		t.Fatal("a terminal import batch returned to processing")
	}
	latePrincipal := insertMigration065Transaction(t, db, schema, "MANUAL", "ADJUSTMENT", "INFLOW", nil, accountID, "count-002")
	if _, err := db.Exec(migration065ImportRowInsertSQL(schema), countBatch, 3, digest065("count-row-3"), "count-002", accountID, latePrincipal, nil); err == nil {
		t.Fatal("a terminal import batch accepted another row")
	}
	if _, err := db.Exec(`DELETE FROM "`+schema+`".company_fund_transaction_import_rows WHERE batch_id=$1`, countBatch); err == nil {
		t.Fatal("a terminal import batch allowed its durable rows to be deleted")
	}

	failedBatch := insertMigration065Batch(t, db, schema, digest065("failed"), "PROCESSING", nil)
	failedPrincipal := insertMigration065Transaction(t, db, schema, "MANUAL", "ADJUSTMENT", "INFLOW", nil, accountID, "failed-001")
	insertMigration065ImportRow(t, db, schema, failedBatch, 2, "failed-001", failedPrincipal, nil, accountID)
	if _, err := db.Exec(`UPDATE "`+schema+`".company_fund_transaction_import_batches
SET status='FAILED', failure_code='IMPORT_COMMIT_FAILED', failure_summary='failed', completed_at=clock_timestamp()
WHERE id=$1`, failedBatch); err == nil {
		t.Fatal("a failed import batch retained durable movement rows")
	}
	if _, err := db.Exec(`DELETE FROM "`+schema+`".company_fund_transaction_import_rows WHERE batch_id=$1`, failedBatch); err != nil {
		t.Fatalf("remove processing import row before failure: %v", err)
	}
	if _, err := db.Exec(`UPDATE "`+schema+`".company_fund_transaction_import_batches
SET status='FAILED', failure_code='IMPORT_COMMIT_FAILED', failure_summary='failed', completed_at=clock_timestamp()
WHERE id=$1`, failedBatch); err != nil {
		t.Fatalf("fail empty processing batch: %v", err)
	}

	directVoidBatch := insertMigration065Batch(t, db, schema, digest065("direct-void"), "PROCESSING", nil)
	if _, err := db.Exec(`UPDATE "`+schema+`".company_fund_transaction_import_batches
SET status='VOIDED', completed_at=clock_timestamp(), voided_at=clock_timestamp(), voided_by=1, void_reason='invalid'
WHERE id=$1`, directVoidBatch); err == nil {
		t.Fatal("a processing batch transitioned directly to voided")
	}

	rowBatch := insertMigration065Batch(t, db, schema, digest065("rows"), "PROCESSING", nil)
	insertMigration065ImportRow(t, db, schema, rowBatch, 2, "bank-001", manualOne, manualTwo, accountID)
	third := insertMigration065Transaction(t, db, schema, "MANUAL", "ADJUSTMENT", "INFLOW", nil, accountID, "bank-001")
	thirdFee := insertMigration065Transaction(t, db, schema, "MANUAL", "FEE", "OUTFLOW", accountID, nil, "bank-001")
	if _, err := db.Exec(migration065ImportRowInsertSQL(schema), rowBatch, 3, digest065("row-3"), "bank-001", accountID, third, manualTwo); err == nil {
		t.Fatal("a movement reused across principal/fee import roles was accepted")
	}
	if _, err := db.Exec(migration065ImportRowInsertSQL(schema), rowBatch, 4, digest065("row-4"), "bank-001", accountID, third, third); err == nil {
		t.Fatal("a row using the same principal and fee movement was accepted")
	}
	if _, err := db.Exec(migration065ImportRowInsertSQL(schema), rowBatch, 5, digest065("row-5"), "bank-001", accountID+99999, third, nil); err == nil {
		t.Fatal("an import row with a missing account FK was accepted")
	}
	provider := insertMigration065Transaction(t, db, schema, "SAFEHERON", "ADJUSTMENT", "INFLOW", nil, accountID, nil)
	if _, err := db.Exec(migration065ImportRowInsertSQL(schema), rowBatch, 6, digest065("row-6"), nil, accountID, provider, thirdFee); err == nil {
		t.Fatal("a provider transaction was accepted as an imported principal")
	}
	nonAdjustment := insertMigration065Transaction(t, db, schema, "MANUAL", "PRINCIPAL", "INFLOW", nil, accountID, "bank-001")
	if _, err := db.Exec(migration065ImportRowInsertSQL(schema), rowBatch, 7, digest065("row-7"), "bank-001", accountID, nonAdjustment, thirdFee); err == nil {
		t.Fatal("a non-adjustment manual transaction was accepted as an imported principal")
	}
	wrongFeeDirection := insertMigration065Transaction(t, db, schema, "MANUAL", "FEE", "INFLOW", nil, accountID, "bank-001")
	if _, err := db.Exec(migration065ImportRowInsertSQL(schema), rowBatch, 8, digest065("row-8"), "bank-001", accountID, third, wrongFeeDirection); err == nil {
		t.Fatal("a non-outflow manual fee was accepted as an imported fee")
	}
	secondAccountID := insertMigration065Account(t, db, schema)
	if _, err := db.Exec(migration065ImportRowInsertSQL(schema), rowBatch, 9, digest065("row-9"), "bank-001", secondAccountID, third, thirdFee); err == nil {
		t.Fatal("an imported transaction linked to a different account was accepted")
	}
	principalWithReference := insertMigration065Transaction(t, db, schema, "MANUAL", "ADJUSTMENT", "INFLOW", nil, accountID, "bank-expected")
	if _, err := db.Exec(migration065ImportRowInsertSQL(schema), rowBatch, 10, digest065("row-10"), "bank-other", accountID, principalWithReference, nil); err == nil {
		t.Fatal("an imported principal with a different external reference was accepted")
	}
	principalForFeeAccount := insertMigration065Transaction(t, db, schema, "MANUAL", "ADJUSTMENT", "INFLOW", nil, accountID, nil)
	feeForDifferentAccount := insertMigration065Transaction(t, db, schema, "MANUAL", "FEE", "OUTFLOW", secondAccountID, nil, nil)
	if _, err := db.Exec(migration065ImportRowInsertSQL(schema), rowBatch, 11, digest065("row-11"), nil, accountID, principalForFeeAccount, feeForDifferentAccount); err == nil {
		t.Fatal("an imported fee linked to a different account was accepted")
	}
	assertConcurrentMigration065MovementOwnership(t, db, schema, accountID)
}

func assertConcurrentMigration065MovementOwnership(t *testing.T, db *sql.DB, schema string, accountID int64) {
	t.Helper()
	firstBatch := insertMigration065Batch(t, db, schema, digest065("concurrent-a"), "PROCESSING", nil)
	secondBatch := insertMigration065Batch(t, db, schema, digest065("concurrent-b"), "PROCESSING", nil)
	movementID := insertMigration065Transaction(t, db, schema, "MANUAL", "ADJUSTMENT", "INFLOW", nil, accountID, "concurrent-001")
	firstTx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = firstTx.Rollback() })
	secondTx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = secondTx.Rollback() })
	if _, err := firstTx.Exec(migration065ImportRowInsertSQL(schema), firstBatch, 2, digest065("concurrent-row-a"), "concurrent-001", accountID, movementID, nil); err != nil {
		t.Fatalf("insert first concurrent owner: %v", err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := secondTx.Exec(migration065ImportRowInsertSQL(schema), secondBatch, 2, digest065("concurrent-row-b"), "concurrent-001", accountID, movementID, nil)
		result <- err
	}()
	select {
	case err := <-result:
		t.Fatalf("competing movement ownership did not wait for the first transaction: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	if err := firstTx.Commit(); err != nil {
		t.Fatalf("commit first concurrent owner: %v", err)
	}
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("both concurrent movement owners committed")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("competing movement ownership did not resolve after the first commit")
	}
	_ = secondTx.Rollback()
	var owners int
	if err := db.QueryRow(`SELECT count(*) FROM "`+schema+`".company_fund_transaction_import_rows WHERE principal_transaction_id=$1`, movementID).Scan(&owners); err != nil {
		t.Fatal(err)
	}
	if owners != 1 {
		t.Fatalf("concurrent movement owner count = %d", owners)
	}
}

func newMigration065PostgresFixture(t *testing.T, databaseURL string) (*sql.DB, string) {
	t.Helper()
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	db := stdlib.OpenDB(*config)
	t.Cleanup(func() { _ = db.Close() })
	schema := fmt.Sprintf("migration_065_%d", time.Now().UnixNano())
	if _, err := db.Exec(`CREATE SCHEMA "` + schema + `"`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`); err != nil {
			t.Errorf("drop schema: %v", err)
		}
	})
	fixtureSQL := `
CREATE TABLE "` + schema + `".company_fund_accounts (id BIGSERIAL PRIMARY KEY);
CREATE TABLE "` + schema + `".finance_categories (id BIGSERIAL PRIMARY KEY);
CREATE TABLE "` + schema + `".company_fund_transactions (
  id BIGSERIAL PRIMARY KEY,
  channel VARCHAR(16) NOT NULL,
  movement_kind VARCHAR(16),
  transaction_direction VARCHAR(24),
  from_company_fund_account_id BIGINT,
  to_company_fund_account_id BIGINT
);`
	if _, err := db.ExecContext(context.Background(), fixtureSQL); err != nil {
		t.Fatal(err)
	}
	return db, schema
}

func insertMigration065Transaction(
	t *testing.T,
	db *sql.DB,
	schema, channel, movementKind, direction string,
	fromAccountID, toAccountID, reference any,
) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`INSERT INTO "`+schema+`".company_fund_transactions (
  channel, movement_kind, transaction_direction,
  from_company_fund_account_id, to_company_fund_account_id, external_transaction_reference
) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, channel, movementKind, direction, fromAccountID, toAccountID, reference).Scan(&id); err != nil {
		t.Fatalf("insert %s transaction: %v", channel, err)
	}
	return id
}

func insertMigration065Account(t *testing.T, db *sql.DB, schema string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`INSERT INTO "` + schema + `".company_fund_accounts DEFAULT VALUES RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func insertMigration065Batch(t *testing.T, db *sql.DB, schema, contentDigest, status string, predecessor *int64) int64 {
	t.Helper()
	var id int64
	var completedAt, voidedAt any
	var voidedBy any
	var voidReason any
	if status == "VOIDED" {
		completedAt, voidedAt, voidedBy, voidReason = time.Now(), time.Now(), int64(1), "test"
	}
	var reimportReason any
	if predecessor != nil {
		reimportReason = "retry"
	}
	if err := db.QueryRow(`INSERT INTO "`+schema+`".company_fund_transaction_import_batches (
  content_digest, request_digest, template_version, original_file_name, status,
  requested_by, idempotency_key, source_row_count, predecessor_batch_id,
  reimport_reason, completed_at, voided_at, voided_by, void_reason
) VALUES ($1, $2, 'v2', 'import.xlsx', $3, 1, $4, 1, $5, $6, $7, $8, $9, $10)
RETURNING id`, contentDigest, digest065(fmt.Sprintf("request-%s-%s-%v", contentDigest, status, predecessor)), status, fmt.Sprintf("key-%s-%s-%v", contentDigest[:8], status, predecessor), predecessor, reimportReason, completedAt, voidedAt, voidedBy, voidReason).Scan(&id); err != nil {
		t.Fatalf("insert %s batch: %v", status, err)
	}
	return id
}

func insertMigration065VoidedBatch(
	t *testing.T,
	db *sql.DB,
	schema, contentDigest string,
	accountID int64,
	predecessor *int64,
) int64 {
	t.Helper()
	batchID := insertMigration065Batch(t, db, schema, contentDigest, "PROCESSING", predecessor)
	principalID := insertMigration065Transaction(t, db, schema, "MANUAL", "ADJUSTMENT", "INFLOW", nil, accountID, contentDigest)
	insertMigration065ImportRow(t, db, schema, batchID, 2, contentDigest, principalID, nil, accountID)
	markMigration065BatchVoided(t, db, schema, batchID, 1, 0)
	return batchID
}

func markMigration065BatchVoided(t *testing.T, db *sql.DB, schema string, batchID int64, principalCount, feeCount int) {
	t.Helper()
	if _, err := db.Exec(`UPDATE "`+schema+`".company_fund_transaction_import_batches
SET status='SUCCEEDED', completed_at=clock_timestamp(),
  principal_transaction_count=$2, fee_transaction_count=$3
WHERE id=$1`, batchID, principalCount, feeCount); err != nil {
		t.Fatalf("complete import batch before void: %v", err)
	}
	if _, err := db.Exec(`UPDATE "`+schema+`".company_fund_transaction_import_batches
SET status='VOIDED', voided_at=clock_timestamp(), voided_by=1, void_reason='test'
WHERE id=$1`, batchID); err != nil {
		t.Fatalf("void succeeded import batch: %v", err)
	}
}

func migration065BatchInsertSQL(schema string) string {
	return `INSERT INTO "` + schema + `".company_fund_transaction_import_batches (
  content_digest, request_digest, template_version, original_file_name, status,
  requested_by, idempotency_key, source_row_count, predecessor_batch_id, reimport_reason
) VALUES ($1, $2, 'v2', 'import.xlsx', $3, 1, $2, 1, $4, $5)`
}

func migration065ImportRowInsertSQL(schema string) string {
	return `INSERT INTO "` + schema + `".company_fund_transaction_import_rows (
  batch_id, source_row_number, row_digest, external_transaction_reference, company_fund_account_id,
  principal_transaction_id, fee_transaction_id
) VALUES ($1, $2, $3, $4, $5, $6, $7)`
}

func insertMigration065ImportRow(t *testing.T, db *sql.DB, schema string, batchID int64, sourceRow int, reference any, principal int64, fee any, accountID int64) {
	t.Helper()
	if _, err := db.Exec(migration065ImportRowInsertSQL(schema), batchID, sourceRow, digest065(fmt.Sprintf("row-%d", sourceRow)), reference, accountID, principal, fee); err != nil {
		t.Fatalf("insert valid import row: %v", err)
	}
}

func digest065(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
