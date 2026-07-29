package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"monera-digital/internal/migration"
)

// AddAirwallexAccountLifecycle adds the durable account lifecycle and command
// contract shared by the management application and the ingestion service.
type AddAirwallexAccountLifecycle struct {
	LegacyMappingJSON string
}

func (*AddAirwallexAccountLifecycle) Version() string { return "063" }

func (*AddAirwallexAccountLifecycle) Description() string {
	return "Add Airwallex account lifecycle and durable command processing"
}

func (*AddAirwallexAccountLifecycle) RequiredPreexistingVersion() string { return "062" }

func (*AddAirwallexAccountLifecycle) RequiredExpectedCeiling() string { return "063" }

func (*AddAirwallexAccountLifecycle) Up(*sql.DB) error {
	return fmt.Errorf("063 is controlled; run it through Migrator.MigrateWithExpectedCeiling")
}

func (m *AddAirwallexAccountLifecycle) UpTx(tx *sql.Tx) error {
	ctx := context.Background()
	if _, err := tx.ExecContext(ctx, migration063TimeoutsSQL); err != nil {
		return fmt.Errorf("configure migration 063 timeouts: %w", err)
	}

	var violations int64
	var accountIDs, accountDetailsJSON string
	if err := tx.QueryRowContext(ctx, migration063LegacyPreflightSQL).Scan(
		&violations,
		&accountIDs,
		&accountDetailsJSON,
	); err != nil {
		return fmt.Errorf("preflight migration 063 Airwallex lifecycle: %w", err)
	}
	mapping, err := validateMigration063LegacyMapping(
		violations,
		accountDetailsJSON,
		m.LegacyMappingJSON,
	)
	if err != nil {
		return fmt.Errorf(
			"preflight rejected Airwallex lifecycle migration: violations=%d account_ids=%s: %w",
			violations,
			accountIDs,
			err,
		)
	}
	if _, err := tx.ExecContext(ctx, migration063CreateLegacyMappingTableSQL); err != nil {
		return fmt.Errorf("create migration 063 legacy mapping: %w", err)
	}
	for _, accountID := range sortedMigration063MappingIDs(mapping) {
		if _, err := tx.ExecContext(
			ctx,
			migration063InsertLegacyMappingSQL,
			accountID,
			mapping[accountID],
		); err != nil {
			return fmt.Errorf("insert migration 063 legacy mapping: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, migration063SchemaSQL); err != nil {
		return fmt.Errorf("install Airwallex account lifecycle schema: %w", err)
	}
	return nil
}

func (*AddAirwallexAccountLifecycle) Down(*sql.DB) error {
	return fmt.Errorf("063 is forward-only; Airwallex lifecycle must be changed by a new migration")
}

var _ migration.Migration = (*AddAirwallexAccountLifecycle)(nil)
var _ migration.ControlledMigration = (*AddAirwallexAccountLifecycle)(nil)

const migration063TimeoutsSQL = `SET LOCAL search_path = pg_catalog, public; SET LOCAL lock_timeout = '5s'; SET LOCAL statement_timeout = '60s'; SET LOCAL idle_in_transaction_session_timeout = '60s';`

const migration063LegacyPreflightSQL = `
WITH airwallex_accounts AS (
  SELECT id, is_enabled, first_enabled_at
  FROM public.company_fund_accounts
  WHERE channel = 'AIRWALLEX'
),
violating_accounts AS (
  SELECT id
  FROM airwallex_accounts
  WHERE (SELECT count(*) FROM airwallex_accounts) > 1
     OR (NOT is_enabled AND first_enabled_at IS NOT NULL)
)
SELECT count(*), COALESCE(string_agg(id::text, ',' ORDER BY id), '')
  , COALESCE((
      SELECT jsonb_agg(
        jsonb_build_object(
          'id', account.id,
          'isEnabled', account.is_enabled,
          'wasEnabled', account.first_enabled_at IS NOT NULL
        )
        ORDER BY account.id
      )::text
      FROM airwallex_accounts account
    ), '[]')
FROM violating_accounts`

const migration063CreateLegacyMappingTableSQL = `
CREATE TEMP TABLE migration063_airwallex_lifecycle_mapping (
  account_id BIGINT PRIMARY KEY,
  lifecycle VARCHAR(16) NOT NULL
) ON COMMIT DROP`

const migration063InsertLegacyMappingSQL = `
INSERT INTO migration063_airwallex_lifecycle_mapping (account_id, lifecycle)
VALUES ($1, $2)`

const migration063SchemaSQL = `
ALTER TABLE public.company_fund_accounts
  ADD COLUMN airwallex_lifecycle VARCHAR(16),
  ADD COLUMN lifecycle_version BIGINT NOT NULL DEFAULT 1,
  ADD COLUMN airwallex_validated_at TIMESTAMPTZ,
  ADD COLUMN airwallex_provider_identity_summary JSONB,
  ADD COLUMN deleted_at TIMESTAMPTZ,
  ADD COLUMN deleted_by VARCHAR(255),
  ADD COLUMN delete_reason VARCHAR(1000);

UPDATE public.company_fund_accounts
SET airwallex_lifecycle = COALESCE(
  (
    SELECT mapping.lifecycle
    FROM migration063_airwallex_lifecycle_mapping mapping
    WHERE mapping.account_id = company_fund_accounts.id
  ),
  CASE WHEN is_enabled THEN 'CURRENT' ELSE 'CANDIDATE' END
)
WHERE channel = 'AIRWALLEX';

ALTER TABLE public.company_fund_accounts
  ADD CONSTRAINT company_fund_accounts_airwallex_lifecycle_check
  CHECK (
    (
      channel = 'AIRWALLEX'
      AND airwallex_lifecycle IN ('CANDIDATE', 'CURRENT', 'PAUSED', 'RETIRED', 'DELETED')
      AND (
        (airwallex_lifecycle = 'CURRENT' AND is_enabled)
        OR (airwallex_lifecycle <> 'CURRENT' AND NOT is_enabled)
      )
      AND (
        (airwallex_lifecycle = 'DELETED'
          AND deleted_at IS NOT NULL
          AND deleted_by IS NOT NULL
          AND btrim(deleted_by) <> ''
          AND delete_reason IS NOT NULL
          AND btrim(delete_reason) <> '')
        OR (airwallex_lifecycle <> 'DELETED'
          AND deleted_at IS NULL
          AND deleted_by IS NULL
          AND delete_reason IS NULL)
      )
      AND (
        airwallex_provider_identity_summary IS NULL
        OR jsonb_typeof(airwallex_provider_identity_summary) = 'object'
      )
    )
    OR (
      channel <> 'AIRWALLEX'
      AND airwallex_lifecycle IS NULL
      AND airwallex_validated_at IS NULL
      AND airwallex_provider_identity_summary IS NULL
      AND deleted_at IS NULL
      AND deleted_by IS NULL
      AND delete_reason IS NULL
    )
  ) NOT VALID;

ALTER TABLE public.company_fund_accounts
  VALIDATE CONSTRAINT company_fund_accounts_airwallex_lifecycle_check;

DROP INDEX public.idx_company_fund_accounts_airwallex_identity;
CREATE UNIQUE INDEX idx_company_fund_accounts_airwallex_identity
  ON public.company_fund_accounts (channel, provider_account_key)
  WHERE channel = 'AIRWALLEX'
    AND provider_account_key IS NOT NULL
    AND deleted_at IS NULL;

CREATE UNIQUE INDEX uq_company_fund_accounts_airwallex_current
  ON public.company_fund_accounts (channel)
  WHERE channel = 'AIRWALLEX' AND airwallex_lifecycle = 'CURRENT';

CREATE UNIQUE INDEX uq_company_fund_accounts_airwallex_candidate
  ON public.company_fund_accounts (channel)
  WHERE channel = 'AIRWALLEX' AND airwallex_lifecycle = 'CANDIDATE';

CREATE TABLE public.company_fund_account_lifecycle_commands (
  id BIGSERIAL PRIMARY KEY,
  command_type VARCHAR(32) NOT NULL
    CHECK (command_type IN (
      'VALIDATE_CANDIDATE',
      'PAUSE',
      'RESUME',
      'CORRECT_IDENTITY',
      'CUTOVER',
      'DELETE_CANDIDATE'
    )),
  target_account_id BIGINT NOT NULL
    REFERENCES public.company_fund_accounts(id) ON DELETE RESTRICT,
  related_account_id BIGINT
    REFERENCES public.company_fund_accounts(id) ON DELETE RESTRICT,
  requested_provider_account_key VARCHAR(128),
  requested_by VARCHAR(255) NOT NULL CHECK (btrim(requested_by) <> ''),
  reason VARCHAR(1000) NOT NULL CHECK (btrim(reason) <> ''),
  idempotency_key VARCHAR(255) NOT NULL,
  expected_target_version BIGINT NOT NULL,
  expected_related_version BIGINT,
  status VARCHAR(16) NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING', 'PROCESSING', 'SUCCEEDED', 'FAILED')),
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  lease_owner VARCHAR(128),
  lease_expires_at TIMESTAMPTZ,
  result_summary JSONB,
  error_code VARCHAR(64),
  error_message VARCHAR(1000),
  business_applied_at TIMESTAMPTZ,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT company_fund_account_lifecycle_commands_shape_check CHECK (
    (command_type = 'CORRECT_IDENTITY'
      AND requested_provider_account_key IS NOT NULL
      AND btrim(requested_provider_account_key) <> '')
    OR (command_type <> 'CORRECT_IDENTITY' AND requested_provider_account_key IS NULL)
  ),
  CONSTRAINT company_fund_account_lifecycle_commands_cutover_check CHECK (
    (command_type = 'CUTOVER' AND (
      (related_account_id IS NOT NULL AND expected_related_version IS NOT NULL)
      OR (related_account_id IS NULL AND expected_related_version IS NULL)
    ))
    OR (command_type <> 'CUTOVER' AND related_account_id IS NULL
      AND expected_related_version IS NULL)
  ),
  CONSTRAINT company_fund_account_lifecycle_commands_lease_check CHECK (
    (status = 'PROCESSING' AND lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
    OR (status <> 'PROCESSING' AND lease_owner IS NULL AND lease_expires_at IS NULL)
  )
);

CREATE UNIQUE INDEX uq_company_fund_account_lifecycle_commands_idempotency
  ON public.company_fund_account_lifecycle_commands (idempotency_key);

CREATE UNIQUE INDEX uq_company_fund_account_lifecycle_commands_inflight
  ON public.company_fund_account_lifecycle_commands ((1))
  WHERE status IN ('PENDING', 'PROCESSING');

CREATE INDEX idx_company_fund_account_lifecycle_commands_due
  ON public.company_fund_account_lifecycle_commands (next_attempt_at, id)
  WHERE status IN ('PENDING', 'PROCESSING');

CREATE TABLE public.company_fund_account_lifecycle_audits (
  id BIGSERIAL PRIMARY KEY,
  command_id BIGINT NOT NULL
    REFERENCES public.company_fund_account_lifecycle_commands(id) ON DELETE RESTRICT,
  account_id BIGINT NOT NULL
    REFERENCES public.company_fund_accounts(id) ON DELETE RESTRICT,
  command_type VARCHAR(32) NOT NULL,
  old_lifecycle VARCHAR(16),
  new_lifecycle VARCHAR(16),
  old_provider_account_key VARCHAR(128),
  new_provider_account_key VARCHAR(128),
  actor VARCHAR(255) NOT NULL,
  reason VARCHAR(1000) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (command_id, account_id)
);

CREATE TABLE public.company_fund_classification_policy_bindings (
  policy_key VARCHAR(64) PRIMARY KEY
    CHECK (policy_key = 'AIRWALLEX_FEE'),
  channel VARCHAR(16) NOT NULL
    CHECK (channel = 'AIRWALLEX'),
  movement_kind VARCHAR(32) NOT NULL
    CHECK (movement_kind = 'FEE'),
  finance_category_level1_id BIGINT NOT NULL
    REFERENCES public.finance_categories(id) ON DELETE RESTRICT,
  finance_category_level2_id BIGINT NOT NULL
    REFERENCES public.finance_categories(id) ON DELETE RESTRICT,
  policy_version VARCHAR(64) NOT NULL
    CHECK (btrim(policy_version) <> ''),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE OR REPLACE FUNCTION public.company_fund_guard_classification_binding_hierarchy()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  level1_parent BIGINT;
  level1_level INTEGER;
  level1_enabled BOOLEAN;
  level2_parent BIGINT;
  level2_level INTEGER;
  level2_enabled BOOLEAN;
BEGIN
  SELECT parent_id, level, is_enabled
  INTO level1_parent, level1_level, level1_enabled
  FROM public.finance_categories
  WHERE id = NEW.finance_category_level1_id
  FOR KEY SHARE;

  SELECT parent_id, level, is_enabled
  INTO level2_parent, level2_level, level2_enabled
  FROM public.finance_categories
  WHERE id = NEW.finance_category_level2_id
  FOR KEY SHARE;

  IF level1_level IS DISTINCT FROM 1
     OR level1_parent IS NOT NULL
     OR level1_enabled IS NOT TRUE
     OR level2_level IS DISTINCT FROM 2
     OR level2_parent IS DISTINCT FROM NEW.finance_category_level1_id
     OR level2_enabled IS NOT TRUE THEN
    RAISE EXCEPTION 'classification policy binding requires an enabled level-one and child level-two hierarchy';
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_company_fund_classification_binding_hierarchy_guard
BEFORE INSERT OR UPDATE OF finance_category_level1_id,
  finance_category_level2_id, is_active
ON public.company_fund_classification_policy_bindings
FOR EACH ROW
WHEN (NEW.is_active)
EXECUTE FUNCTION public.company_fund_guard_classification_binding_hierarchy();

CREATE OR REPLACE FUNCTION public.company_fund_guard_bound_finance_category()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM public.company_fund_classification_policy_bindings binding
    WHERE binding.is_active
      AND (
        binding.finance_category_level1_id = OLD.id
        OR binding.finance_category_level2_id = OLD.id
      )
  ) AND (
    TG_OP = 'DELETE'
    OR NEW.is_enabled IS FALSE
    OR NEW.parent_id IS DISTINCT FROM OLD.parent_id
  ) THEN
    RAISE EXCEPTION 'finance category is protected by an active system classification policy binding';
  END IF;
  RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$;

CREATE TRIGGER trg_finance_categories_system_binding_guard
BEFORE UPDATE OF is_enabled, parent_id OR DELETE
ON public.finance_categories
FOR EACH ROW EXECUTE FUNCTION public.company_fund_guard_bound_finance_category();

CREATE OR REPLACE FUNCTION public.company_fund_guard_airwallex_projection_write()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  account_exists BOOLEAN;
BEGIN
  IF NEW.channel <> 'AIRWALLEX'
     OR current_setting('monera.airwallex_account_lifecycle_command', true) = 'on' THEN
    RETURN NEW;
  END IF;

  PERFORM pg_advisory_xact_lock_shared(768972734063);

  IF TG_TABLE_NAME = 'company_fund_provider_events' THEN
    SELECT EXISTS (
      SELECT 1
      FROM public.company_fund_accounts account
      WHERE account.channel = 'AIRWALLEX'
        AND account.provider_account_key = NEW.provider_account_key
        AND account.airwallex_lifecycle = 'CURRENT'
        AND account.is_enabled
        AND NOT EXISTS (
          SELECT 1
          FROM public.company_fund_account_lifecycle_commands command
          WHERE command.target_account_id = account.id
            AND command.command_type = 'CORRECT_IDENTITY'
            AND command.status = 'PROCESSING'
        )
    ) INTO account_exists;
  ELSE
    SELECT EXISTS (
      SELECT 1
      FROM public.company_fund_accounts account
      WHERE account.channel = 'AIRWALLEX'
        AND account.provider_account_key = NEW.provider_account_key
        AND account.airwallex_lifecycle <> 'DELETED'
        AND NOT EXISTS (
          SELECT 1
          FROM public.company_fund_account_lifecycle_commands command
          WHERE command.target_account_id = account.id
            AND command.command_type = 'CORRECT_IDENTITY'
            AND command.status = 'PROCESSING'
        )
    ) INTO account_exists;
  END IF;

  IF NOT account_exists THEN
    RAISE EXCEPTION 'Airwallex projection account scope is not eligible';
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_company_fund_provider_events_airwallex_scope_guard
BEFORE INSERT OR UPDATE OF channel, provider_account_key
ON public.company_fund_provider_events
FOR EACH ROW EXECUTE FUNCTION public.company_fund_guard_airwallex_projection_write();

CREATE TRIGGER trg_company_fund_provider_facts_airwallex_scope_guard
BEFORE INSERT OR UPDATE OF channel, provider_account_key
ON public.company_fund_provider_transaction_facts
FOR EACH ROW EXECUTE FUNCTION public.company_fund_guard_airwallex_projection_write();

CREATE TRIGGER trg_company_fund_transactions_airwallex_scope_guard
BEFORE INSERT OR UPDATE OF channel, provider_account_key
ON public.company_fund_transactions
FOR EACH ROW EXECUTE FUNCTION public.company_fund_guard_airwallex_projection_write();

CREATE TRIGGER trg_company_fund_ledger_tasks_airwallex_scope_guard
BEFORE INSERT OR UPDATE OF channel, provider_account_key
ON public.company_fund_ledger_tasks
FOR EACH ROW EXECUTE FUNCTION public.company_fund_guard_airwallex_projection_write();

CREATE OR REPLACE FUNCTION public.company_fund_guard_airwallex_asset_policy_write()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  account_is_airwallex BOOLEAN;
  account_is_eligible BOOLEAN;
BEGIN
  PERFORM pg_advisory_xact_lock_shared(768972734063);
  SELECT account.channel = 'AIRWALLEX',
         account.airwallex_lifecycle <> 'DELETED'
  INTO account_is_airwallex, account_is_eligible
  FROM public.company_fund_accounts account
  WHERE account.id = NEW.company_fund_account_id;

  IF account_is_airwallex AND NOT account_is_eligible THEN
    RAISE EXCEPTION 'deleted Airwallex account cannot receive asset policy references';
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_company_fund_account_asset_policies_airwallex_scope_guard
BEFORE INSERT OR UPDATE OF company_fund_account_id
ON public.company_fund_account_asset_policies
FOR EACH ROW EXECUTE FUNCTION public.company_fund_guard_airwallex_asset_policy_write();

CREATE OR REPLACE FUNCTION public.company_fund_guard_airwallex_account_command_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF OLD.channel = 'AIRWALLEX'
     AND (
       NEW.provider_account_key IS DISTINCT FROM OLD.provider_account_key
       OR NEW.airwallex_lifecycle IS DISTINCT FROM OLD.airwallex_lifecycle
       OR NEW.is_enabled IS DISTINCT FROM OLD.is_enabled
       OR NEW.lifecycle_version IS DISTINCT FROM OLD.lifecycle_version
       OR NEW.airwallex_validated_at IS DISTINCT FROM OLD.airwallex_validated_at
       OR NEW.airwallex_provider_identity_summary IS DISTINCT FROM OLD.airwallex_provider_identity_summary
       OR NEW.deleted_at IS DISTINCT FROM OLD.deleted_at
       OR NEW.deleted_by IS DISTINCT FROM OLD.deleted_by
       OR NEW.delete_reason IS DISTINCT FROM OLD.delete_reason
     )
     AND current_setting('monera.airwallex_account_lifecycle_command', true)
         IS DISTINCT FROM 'on' THEN
    RAISE EXCEPTION 'Airwallex lifecycle and identity fields require a durable lifecycle command';
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_company_fund_accounts_airwallex_command_guard
BEFORE UPDATE OF provider_account_key, airwallex_lifecycle, is_enabled,
  lifecycle_version, airwallex_validated_at, airwallex_provider_identity_summary,
  deleted_at, deleted_by, delete_reason
ON public.company_fund_accounts
FOR EACH ROW EXECUTE FUNCTION public.company_fund_guard_airwallex_account_command_fields();
`

type migration063LegacyAccount struct {
	ID         int64 `json:"id"`
	IsEnabled  bool  `json:"isEnabled"`
	WasEnabled bool  `json:"wasEnabled"`
}

func validateMigration063LegacyMapping(
	violations int64,
	accountDetailsJSON string,
	rawMapping string,
) (map[int64]string, error) {
	var accounts []migration063LegacyAccount
	if err := json.Unmarshal([]byte(accountDetailsJSON), &accounts); err != nil {
		return nil, fmt.Errorf("decode legacy Airwallex account details")
	}
	mapping := make(map[int64]string)
	if strings.TrimSpace(rawMapping) != "" {
		var raw map[string]string
		if err := json.Unmarshal([]byte(rawMapping), &raw); err != nil {
			return nil, fmt.Errorf("AIRWALLEX_LEGACY_LIFECYCLE_MAPPING_JSON must be an object of account IDs to lifecycle values")
		}
		for rawID, lifecycle := range raw {
			id, err := strconv.ParseInt(rawID, 10, 64)
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("legacy mapping contains an invalid account ID")
			}
			switch lifecycle {
			case "CANDIDATE", "CURRENT", "PAUSED", "RETIRED":
			default:
				return nil, fmt.Errorf("legacy mapping contains an unsupported lifecycle")
			}
			mapping[id] = lifecycle
		}
	}
	if violations == 0 {
		if len(mapping) != 0 {
			return nil, fmt.Errorf("legacy mapping must be empty when the account state is unambiguous")
		}
		return mapping, nil
	}
	if len(mapping) != len(accounts) {
		return nil, fmt.Errorf("explicit mapping must cover every listed Airwallex account exactly once")
	}
	currentCount := 0
	candidateCount := 0
	for _, account := range accounts {
		lifecycle, exists := mapping[account.ID]
		if !exists {
			return nil, fmt.Errorf("explicit mapping omits an Airwallex account")
		}
		if account.IsEnabled && lifecycle != "CURRENT" {
			return nil, fmt.Errorf("enabled legacy account must map to CURRENT")
		}
		if !account.IsEnabled && lifecycle == "CURRENT" {
			return nil, fmt.Errorf("disabled legacy account cannot map to CURRENT")
		}
		if account.WasEnabled && lifecycle == "CANDIDATE" {
			return nil, fmt.Errorf("previously enabled legacy account cannot map to CANDIDATE")
		}
		if lifecycle == "CURRENT" {
			currentCount++
		}
		if lifecycle == "CANDIDATE" {
			candidateCount++
		}
	}
	if currentCount > 1 || candidateCount > 1 {
		return nil, fmt.Errorf("legacy mapping may contain at most one CURRENT and one CANDIDATE")
	}
	return mapping, nil
}

func sortedMigration063MappingIDs(mapping map[int64]string) []int64 {
	ids := make([]int64, 0, len(mapping))
	for id := range mapping {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
