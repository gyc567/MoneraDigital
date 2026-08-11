package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const CompanyFundSchemaFingerprintGate = "RUN_COMPANY_FUND_SCHEMA_FINGERPRINT"

type CompanyFundSchemaState string

const (
	CompanyFundSchemaStateA       CompanyFundSchemaState = "A"
	CompanyFundSchemaStateB       CompanyFundSchemaState = "B"
	CompanyFundSchemaStatePartial CompanyFundSchemaState = "PARTIAL"
)

type LiveCompanyFundSchemaReport struct {
	State                CompanyFundSchemaState             `json:"state"`
	Migration052Recorded bool                               `json:"migration_052_recorded"`
	Migration053Recorded bool                               `json:"migration_053_recorded"`
	Digest               string                             `json:"digest"`
	Fingerprint          *FinalCompanyFundSchemaFingerprint `json:"fingerprint,omitempty"`
	Snapshot             FinalCompanyFundSchemaSnapshot     `json:"snapshot"`
}

type companyFundCatalog interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func InspectLiveCompanyFundSchema(ctx context.Context, catalog companyFundCatalog) (LiveCompanyFundSchemaReport, error) {
	snapshot := FinalCompanyFundSchemaSnapshot{
		OccurrenceColumns:           make(map[string]string),
		OccurrenceColumnLengths:     make(map[string]int64),
		OccurrenceColumnsNullable:   make(map[string]bool),
		ConstraintSchemas:           make(map[string]string),
		ConstraintTables:            make(map[string]string),
		ConstraintTypes:             make(map[string]string),
		ConstraintOccurrences:       make(map[string]int64),
		ManualConstraintDefinitions: make(map[string]string),
		ManualConstraintsValidated:  make(map[string]bool),
		ManualImportContract: FinalCompanyFundManualImportContract{
			Columns:     make(map[string]FinalCompanyFundManualImportColumn),
			Indexes:     make(map[string]FinalCompanyFundManualImportIndex),
			Constraints: make(map[string]FinalCompanyFundManualImportConstraint),
			Functions:   make(map[string]FinalCompanyFundManualImportFunction),
			Triggers:    make(map[string]FinalCompanyFundManualImportTrigger),
		},
	}
	if err := inspectOccurrenceColumns(ctx, catalog, &snapshot); err != nil {
		return LiveCompanyFundSchemaReport{}, err
	}
	if err := inspectOccurrenceIndex(ctx, catalog, &snapshot); err != nil {
		return LiveCompanyFundSchemaReport{}, err
	}
	if err := inspectCompanyFundConstraints(ctx, catalog, &snapshot); err != nil {
		return LiveCompanyFundSchemaReport{}, err
	}
	if err := inspectManualProjectionFunction(ctx, catalog, &snapshot); err != nil {
		return LiveCompanyFundSchemaReport{}, fmt.Errorf("inspect MANUAL projection function: %w", err)
	}
	if err := inspectManualProjectionTrigger(ctx, catalog, &snapshot); err != nil {
		return LiveCompanyFundSchemaReport{}, fmt.Errorf("inspect MANUAL projection trigger: %w", err)
	}
	if err := inspectManualImportContract(ctx, catalog, &snapshot); err != nil {
		return LiveCompanyFundSchemaReport{}, fmt.Errorf("inspect migration 065 manual import contract: %w", err)
	}
	var recorded052, recorded053 bool
	if err := catalog.QueryRowContext(ctx, companyFundMigrationProvenanceCatalogSQL).Scan(&recorded052, &recorded053); err != nil {
		return LiveCompanyFundSchemaReport{}, fmt.Errorf("inspect migration 052/053 provenance: %w", err)
	}
	snapshot = normalizeFinalCompanyFundSchemaSnapshot(snapshot)
	report := LiveCompanyFundSchemaReport{Snapshot: snapshot, Migration052Recorded: recorded052, Migration053Recorded: recorded053, Digest: digestCompanyFundSchema(snapshot)}
	if err := validateFinalCompanyFundSchemaA(snapshot); err != nil {
		report.State = CompanyFundSchemaStatePartial
		return report, nil
	}
	if snapshot.SafeheronRequiredConstraintName == "" {
		if recorded053 {
			report.State = CompanyFundSchemaStatePartial
		} else {
			report.State = CompanyFundSchemaStateA
		}
		return report, nil
	}
	fingerprint, err := BuildFinalCompanyFundSchemaFingerprint(snapshot)
	if err != nil {
		report.State = CompanyFundSchemaStatePartial
		return report, nil
	}
	report.State = CompanyFundSchemaStateB
	report.Fingerprint = &fingerprint
	return report, nil
}

func inspectManualImportContract(ctx context.Context, catalog companyFundCatalog, snapshot *FinalCompanyFundSchemaSnapshot) error {
	var raw []byte
	if err := catalog.QueryRowContext(ctx, companyFundManualImportContractCatalogSQL).Scan(&raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		return fmt.Errorf("manual import contract catalog returned no data")
	}
	var contract FinalCompanyFundManualImportContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		return fmt.Errorf("decode manual import contract: %w", err)
	}
	if contract.Columns == nil || contract.Indexes == nil || contract.Constraints == nil || contract.Functions == nil || contract.Triggers == nil {
		return fmt.Errorf("manual import contract is incomplete")
	}
	snapshot.ManualImportContract = contract
	return nil
}

func inspectOccurrenceColumns(ctx context.Context, catalog companyFundCatalog, snapshot *FinalCompanyFundSchemaSnapshot) error {
	rows, err := catalog.QueryContext(ctx, companyFundOccurrenceColumnsCatalogSQL)
	if err != nil {
		return fmt.Errorf("inspect occurrence columns: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var schema, table, name, dataType, nullable string
		var length sql.NullInt64
		if err := rows.Scan(&schema, &table, &name, &dataType, &length, &nullable); err != nil {
			return fmt.Errorf("scan occurrence column: %w", err)
		}
		snapshot.OccurrenceColumns[name] = dataType
		snapshot.OccurrenceColumnLengths[name] = length.Int64
		snapshot.OccurrenceColumnsNullable[name] = nullable == "YES"
		snapshot.OccurrenceSchema = schema
		snapshot.OccurrenceTable = table
	}
	return rows.Err()
}

func inspectOccurrenceIndex(ctx context.Context, catalog companyFundCatalog, snapshot *FinalCompanyFundSchemaSnapshot) error {
	var columns string
	err := catalog.QueryRowContext(ctx, companyFundOccurrenceIndexCatalogSQL).Scan(
		&snapshot.SafeheronOccurrenceIndexSchema, &snapshot.SafeheronOccurrenceIndexTable,
		&snapshot.SafeheronOccurrenceIndexName, &snapshot.SafeheronOccurrenceIndexUnique,
		&snapshot.SafeheronOccurrenceIndexValid, &snapshot.SafeheronOccurrenceIndexReady, &columns,
		&snapshot.SafeheronOccurrenceIndexPredicate, &snapshot.SafeheronOccurrenceIndexDefinition,
	)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect occurrence index: %w", err)
	}
	if columns != "" {
		snapshot.SafeheronOccurrenceIndexColumns = strings.Split(columns, ",")
	}
	return nil
}

func inspectCompanyFundConstraints(ctx context.Context, catalog companyFundCatalog, snapshot *FinalCompanyFundSchemaSnapshot) error {
	rows, err := catalog.QueryContext(ctx, companyFundConstraintsCatalogSQL)
	if err != nil {
		return fmt.Errorf("inspect company-fund constraints: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var schema, table, name, constraintType, definition string
		var validated bool
		if err := rows.Scan(&schema, &table, &name, &constraintType, &validated, &definition); err != nil {
			return fmt.Errorf("scan company-fund constraint: %w", err)
		}
		snapshot.ConstraintSchemas[name] = schema
		snapshot.ConstraintTables[name] = table
		snapshot.ConstraintTypes[name] = constraintType
		snapshot.ConstraintOccurrences[name]++
		if name == migration053ConstraintName {
			snapshot.SafeheronRequiredConstraintName = name
			snapshot.SafeheronRequiredConstraintValidated = validated
			snapshot.SafeheronRequiredConstraintDefinition = definition
			continue
		}
		snapshot.ManualConstraintNames = append(snapshot.ManualConstraintNames, name)
		snapshot.ManualConstraintsValidated[name] = validated
		snapshot.ManualConstraintDefinitions[name] = definition
	}
	return rows.Err()
}

func inspectManualProjectionFunction(ctx context.Context, catalog companyFundCatalog, snapshot *FinalCompanyFundSchemaSnapshot) error {
	err := catalog.QueryRowContext(ctx, companyFundFunctionCatalogSQL).Scan(
		&snapshot.ManualProjectionFunctionSchema, &snapshot.ManualProjectionFunctionName,
		&snapshot.ManualProjectionFunctionArgumentCount, &snapshot.ManualProjectionFunctionResult,
		&snapshot.ManualProjectionFunctionLanguage, &snapshot.ManualProjectionFunctionKind,
		&snapshot.ManualProjectionFunctionOID, &snapshot.ManualProjectionFunctionSource,
	)
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

func inspectManualProjectionTrigger(ctx context.Context, catalog companyFundCatalog, snapshot *FinalCompanyFundSchemaSnapshot) error {
	var columns string
	err := catalog.QueryRowContext(ctx, companyFundTriggerCatalogSQL).Scan(
		&snapshot.ManualProjectionTriggerSchema, &snapshot.ManualProjectionTriggerTable,
		&snapshot.ManualProjectionTriggerFunctionSchema, &snapshot.ManualProjectionTriggerFunctionName,
		&snapshot.ManualProjectionTriggerFunctionOID, &snapshot.ManualProjectionTriggerInternal,
		&snapshot.ManualProjectionTriggerEnabled, &snapshot.ManualProjectionTriggerType, &columns,
	)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if columns != "" {
		snapshot.ManualProjectionTriggerColumns = strings.Split(columns, ",")
	}
	return nil
}

func digestCompanyFundSchema(snapshot FinalCompanyFundSchemaSnapshot) string {
	data, _ := json.Marshal(snapshot)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

const companyFundOccurrenceColumnsCatalogSQL = `
SELECT table_schema, table_name, column_name, data_type, character_maximum_length, is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'company_fund_transactions'
  AND column_name IN ('provider_occurrence_key', 'provider_occurrence_algorithm_version')
ORDER BY column_name`

const companyFundOccurrenceIndexCatalogSQL = `
SELECT namespace.nspname, table_class.relname, index_class.relname,
       idx.indisunique, idx.indisvalid, idx.indisready,
       array_to_string(ARRAY(
         SELECT attribute.attname
         FROM unnest(idx.indkey) WITH ORDINALITY AS indexed_key(attnum, ordinal)
         JOIN pg_attribute AS attribute ON attribute.attrelid = idx.indrelid AND attribute.attnum = indexed_key.attnum
         WHERE indexed_key.ordinal <= idx.indnkeyatts
         ORDER BY indexed_key.ordinal
       ), ','),
       pg_get_expr(idx.indpred, idx.indrelid), pg_get_indexdef(idx.indexrelid)
FROM pg_index AS idx
JOIN pg_class AS index_class ON index_class.oid = idx.indexrelid
JOIN pg_class AS table_class ON table_class.oid = idx.indrelid
JOIN pg_namespace AS namespace ON namespace.oid = table_class.relnamespace
WHERE namespace.nspname = 'public'
  AND table_class.relname = 'company_fund_transactions'
  AND index_class.relname = 'idx_company_fund_transactions_safeheron_occurrence'`

const companyFundConstraintsCatalogSQL = `
SELECT namespace.nspname, table_class.relname, con.conname, con.contype::text,
       con.convalidated, pg_get_constraintdef(con.oid, true)
FROM pg_constraint AS con
JOIN pg_class AS table_class ON table_class.oid = con.conrelid
JOIN pg_namespace AS namespace ON namespace.oid = table_class.relnamespace
WHERE namespace.nspname = 'public'
  AND con.conname IN (
    'company_fund_transactions_safeheron_occurrence_required_check',
    'company_fund_transactions_usd_valuation_source_check',
    'company_fund_valuation_history_source_check',
    'company_fund_valuation_history_manual_metadata_check'
  )
ORDER BY con.conname`

const companyFundFunctionCatalogSQL = `
SELECT namespace.nspname, proc.proname, proc.pronargs,
       pg_get_function_result(proc.oid), language.lanname, proc.prokind::text,
       proc.oid, proc.prosrc
FROM pg_proc AS proc
JOIN pg_namespace AS namespace ON namespace.oid = proc.pronamespace
JOIN pg_language AS language ON language.oid = proc.prolang
WHERE namespace.nspname = 'public'
  AND proc.proname = 'company_fund_enforce_manual_valuation_projection'
  AND proc.pronargs = 0`

const companyFundTriggerCatalogSQL = `
SELECT namespace.nspname, table_class.relname,
       function_namespace.nspname, function_proc.proname, trg.tgfoid,
       trg.tgisinternal, trg.tgenabled::text, trg.tgtype,
       array_to_string(ARRAY(
         SELECT attribute.attname
         FROM unnest(trg.tgattr) WITH ORDINALITY AS protected_key(attnum, ordinal)
         JOIN pg_attribute AS attribute ON attribute.attrelid = trg.tgrelid AND attribute.attnum = protected_key.attnum
         ORDER BY protected_key.ordinal
       ), ',')
FROM pg_trigger AS trg
JOIN pg_class AS table_class ON table_class.oid = trg.tgrelid
JOIN pg_namespace AS namespace ON namespace.oid = table_class.relnamespace
JOIN pg_proc AS function_proc ON function_proc.oid = trg.tgfoid
JOIN pg_namespace AS function_namespace ON function_namespace.oid = function_proc.pronamespace
WHERE namespace.nspname = 'public'
  AND table_class.relname = 'company_fund_transactions'
  AND trg.tgname = 'company_fund_enforce_manual_valuation_projection'
  AND NOT trg.tgisinternal`

const companyFundMigrationProvenanceCatalogSQL = `SELECT
  EXISTS (SELECT 1 FROM public.migrations WHERE version = '052'),
  EXISTS (SELECT 1 FROM public.migrations WHERE version = '053')`

const companyFundManualImportContractCatalogSQL = `
SELECT jsonb_build_object(
  'columns', COALESCE((
    SELECT jsonb_object_agg(
      columns.table_name || '.' || columns.column_name,
      jsonb_build_object(
        'schema', columns.table_schema,
        'table', columns.table_name,
        'data_type', columns.data_type,
        'length', COALESCE(columns.character_maximum_length, 0),
        'nullable', columns.is_nullable = 'YES'
      )
    )
    FROM information_schema.columns AS columns
    WHERE columns.table_schema = 'public'
      AND (columns.table_name, columns.column_name) IN (
        ('company_fund_transactions', 'external_transaction_reference'),
        ('company_fund_transaction_import_batches', 'id'),
        ('company_fund_transaction_import_batches', 'content_digest'),
        ('company_fund_transaction_import_batches', 'request_digest'),
        ('company_fund_transaction_import_batches', 'template_version'),
        ('company_fund_transaction_import_batches', 'original_file_name'),
        ('company_fund_transaction_import_batches', 'status'),
        ('company_fund_transaction_import_batches', 'requested_by'),
        ('company_fund_transaction_import_batches', 'idempotency_key'),
        ('company_fund_transaction_import_batches', 'source_row_count'),
        ('company_fund_transaction_import_batches', 'principal_transaction_count'),
        ('company_fund_transaction_import_batches', 'fee_transaction_count'),
        ('company_fund_transaction_import_batches', 'voided_movement_count'),
        ('company_fund_transaction_import_batches', 'warning_count'),
        ('company_fund_transaction_import_batches', 'duplicate_override_acknowledged'),
        ('company_fund_transaction_import_batches', 'duplicate_override_reason'),
        ('company_fund_transaction_import_batches', 'duplicate_warning_evidence'),
        ('company_fund_transaction_import_batches', 'predecessor_batch_id'),
        ('company_fund_transaction_import_batches', 'reimport_reason'),
        ('company_fund_transaction_import_batches', 'failure_code'),
        ('company_fund_transaction_import_batches', 'failure_summary'),
        ('company_fund_transaction_import_batches', 'completed_at'),
        ('company_fund_transaction_import_batches', 'voided_at'),
        ('company_fund_transaction_import_batches', 'voided_by'),
        ('company_fund_transaction_import_batches', 'void_reason'),
        ('company_fund_transaction_import_batches', 'created_at'),
        ('company_fund_transaction_import_batches', 'updated_at'),
        ('company_fund_transaction_import_rows', 'id'),
        ('company_fund_transaction_import_rows', 'batch_id'),
        ('company_fund_transaction_import_rows', 'source_row_number'),
        ('company_fund_transaction_import_rows', 'row_digest'),
        ('company_fund_transaction_import_rows', 'external_transaction_reference'),
        ('company_fund_transaction_import_rows', 'company_fund_account_id'),
        ('company_fund_transaction_import_rows', 'finance_category_level1_id'),
        ('company_fund_transaction_import_rows', 'finance_category_level2_id'),
        ('company_fund_transaction_import_rows', 'principal_transaction_id'),
        ('company_fund_transaction_import_rows', 'fee_transaction_id'),
        ('company_fund_transaction_import_rows', 'created_at')
      )
  ), '{}'::jsonb),
  'indexes', COALESCE((
    SELECT jsonb_object_agg(index_class.relname, jsonb_build_object(
      'schema', namespace.nspname,
      'table', table_class.relname,
      'unique', idx.indisunique,
      'valid', idx.indisvalid,
      'ready', idx.indisready,
      'columns', COALESCE((
        SELECT jsonb_agg(attribute.attname ORDER BY indexed_key.ordinal)
        FROM unnest(idx.indkey) WITH ORDINALITY AS indexed_key(attnum, ordinal)
        JOIN pg_attribute AS attribute ON attribute.attrelid = idx.indrelid AND attribute.attnum = indexed_key.attnum
        WHERE indexed_key.ordinal <= idx.indnkeyatts
      ), '[]'::jsonb),
      'predicate', COALESCE(pg_get_expr(idx.indpred, idx.indrelid), ''),
      'definition', pg_get_indexdef(idx.indexrelid)
    ))
    FROM pg_index AS idx
    JOIN pg_class AS index_class ON index_class.oid = idx.indexrelid
    JOIN pg_class AS table_class ON table_class.oid = idx.indrelid
    JOIN pg_namespace AS namespace ON namespace.oid = table_class.relnamespace
    WHERE namespace.nspname = 'public'
      AND index_class.relname IN (
        'idx_company_fund_transactions_external_reference',
        'uq_company_fund_transaction_import_batches_effective_content'
      )
  ), '{}'::jsonb),
  'constraints', COALESCE((
    SELECT jsonb_object_agg(con.conname, jsonb_build_object(
      'schema', namespace.nspname,
      'table', table_class.relname,
      'type', con.contype::text,
      'validated', con.convalidated,
      'definition', pg_get_constraintdef(con.oid, true)
    ))
    FROM pg_constraint AS con
    JOIN pg_class AS table_class ON table_class.oid = con.conrelid
    JOIN pg_namespace AS namespace ON namespace.oid = table_class.relnamespace
    WHERE namespace.nspname = 'public'
      AND con.conname IN (
        'company_fund_transactions_external_reference_source_check',
        'company_fund_transaction_import_batches_status_check',
        'company_fund_transaction_import_batches_lifecycle_check',
        'company_fund_transaction_import_batches_predecessor_fk',
        'company_fund_transaction_import_batches_idempotency_unique',
        'company_fund_transaction_import_rows_batch_fk',
        'company_fund_transaction_import_rows_account_fk',
        'company_fund_transaction_import_rows_principal_fk',
        'company_fund_transaction_import_rows_fee_fk',
        'company_fund_transaction_import_rows_principal_unique',
        'company_fund_transaction_import_rows_fee_unique',
        'company_fund_import_rows_principal_fee_distinct_check'
      )
  ), '{}'::jsonb),
  'functions', COALESCE((
    SELECT jsonb_object_agg(proc.proname, jsonb_build_object(
      'schema', namespace.nspname,
      'arguments', proc.pronargs,
      'result', pg_get_function_result(proc.oid),
      'language', language.lanname,
      'kind', proc.prokind::text,
      'source', proc.prosrc
    ))
    FROM pg_proc AS proc
    JOIN pg_namespace AS namespace ON namespace.oid = proc.pronamespace
    JOIN pg_language AS language ON language.oid = proc.prolang
    WHERE namespace.nspname = 'public'
      AND proc.proname IN (
        'company_fund_validate_import_batch_lineage',
        'company_fund_enforce_import_row_transaction_ownership'
      )
  ), '{}'::jsonb),
  'triggers', COALESCE((
    SELECT jsonb_object_agg(trg.tgname, jsonb_build_object(
      'schema', namespace.nspname,
      'table', table_class.relname,
      'function_schema', function_namespace.nspname,
      'function_name', function_proc.proname,
      'constraint', trg.tgconstraint <> 0,
      'internal', trg.tgisinternal,
      'enabled', trg.tgenabled::text,
      'type', trg.tgtype
    ))
    FROM pg_trigger AS trg
    JOIN pg_class AS table_class ON table_class.oid = trg.tgrelid
    JOIN pg_namespace AS namespace ON namespace.oid = table_class.relnamespace
    JOIN pg_proc AS function_proc ON function_proc.oid = trg.tgfoid
    JOIN pg_namespace AS function_namespace ON function_namespace.oid = function_proc.pronamespace
    WHERE namespace.nspname = 'public'
      AND NOT trg.tgisinternal
      AND trg.tgname IN (
        'company_fund_validate_import_batch_lineage_trigger',
        'company_fund_enforce_import_row_transaction_ownership_trigger'
      )
  ), '{}'::jsonb)
)`
