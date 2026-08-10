package companyfund

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

const ledgerTaskPayloadVersion = "airwallex-ledger-task-v1"
const maxLedgerTaskPayloadBytes = 64 << 10

const insertCompanyFundLedgerTaskSQL = `
INSERT INTO company_fund_ledger_tasks (
	channel,
	provider_account_key,
	task_kind,
	provider_transaction_fact_id,
	subject_provider_transaction_id,
	subject_transaction_id,
	relationship_reference_type,
	relationship_reference_key,
	relationship_group_key,
	evidence_reference,
	task_payload,
	task_payload_digest,
	policy_version,
	next_attempt_at,
	sla_expires_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
	$11, $12, $13, clock_timestamp(),
	clock_timestamp() + ($14::bigint * INTERVAL '1 microsecond')
)
ON CONFLICT DO NOTHING
RETURNING id`

const selectCompanyFundLedgerTaskIdentitySQL = `
SELECT id,
       provider_transaction_fact_id,
       COALESCE(relationship_reference_type, ''),
       COALESCE(relationship_reference_key, ''),
       COALESCE(relationship_group_key, ''),
       evidence_reference,
       task_payload_digest,
       COALESCE(policy_version, '')
FROM company_fund_ledger_tasks
WHERE channel = $1
  AND provider_account_key = $2
  AND task_kind = $3
  AND subject_provider_transaction_id = $4
  AND COALESCE(policy_version, '') = $5`

type companyFundLedgerTaskPayload struct {
	Version  string                 `json:"version"`
	Proposal TransactionUpsertInput `json:"proposal"`
}

func (r *DBRepository) EnqueueCompanyFundLedgerTask(
	ctx context.Context,
	input CompanyFundLedgerTaskInput,
) (CompanyFundLedgerTaskEnqueueResult, error) {
	if err := input.validate(false); err != nil {
		return CompanyFundLedgerTaskEnqueueResult{}, err
	}
	if err := r.requireDB(); err != nil {
		return CompanyFundLedgerTaskEnqueueResult{}, err
	}
	payload, err := json.Marshal(companyFundLedgerTaskPayload{
		Version:  ledgerTaskPayloadVersion,
		Proposal: input.Proposal,
	})
	if err != nil || len(payload) == 0 || len(payload) > maxLedgerTaskPayloadBytes || !json.Valid(payload) {
		return CompanyFundLedgerTaskEnqueueResult{}, fmt.Errorf("marshal bounded company-fund ledger task payload")
	}
	digest := payloadSHA256Hex(payload)
	durationMicros := input.RelationshipSLA.Microseconds()

	var id int64
	err = r.db.QueryRowContext(ctx, insertCompanyFundLedgerTaskSQL,
		input.Channel,
		input.ProviderAccountKey,
		input.Kind,
		input.ProviderTransactionFactID,
		input.SubjectProviderTransactionID,
		nullableInt64(input.SubjectTransactionID),
		nullableRelationshipReferenceType(input.RelationshipReferenceType),
		nullableString(input.RelationshipReferenceKey),
		nullableString(input.RelationshipGroupKey),
		input.EvidenceReference,
		string(payload),
		digest,
		nullableString(input.PolicyVersion),
		durationMicros,
	).Scan(&id)
	if err == nil {
		return CompanyFundLedgerTaskEnqueueResult{ID: id, Inserted: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return CompanyFundLedgerTaskEnqueueResult{}, fmt.Errorf("insert company-fund ledger task: %w", err)
	}

	var (
		existingFactID    int64
		existingType      string
		existingReference string
		existingGroup     string
		existingEvidence  string
		existingDigest    string
		existingPolicy    string
	)
	if err := r.db.QueryRowContext(ctx, selectCompanyFundLedgerTaskIdentitySQL,
		input.Channel,
		input.ProviderAccountKey,
		input.Kind,
		input.SubjectProviderTransactionID,
		input.PolicyVersion,
	).Scan(
		&id,
		&existingFactID,
		&existingType,
		&existingReference,
		&existingGroup,
		&existingEvidence,
		&existingDigest,
		&existingPolicy,
	); err != nil {
		return CompanyFundLedgerTaskEnqueueResult{}, fmt.Errorf("load existing company-fund ledger task: %w", err)
	}
	if existingFactID != input.ProviderTransactionFactID ||
		existingType != string(input.RelationshipReferenceType) ||
		existingReference != input.RelationshipReferenceKey ||
		existingGroup != input.RelationshipGroupKey ||
		existingEvidence != input.EvidenceReference ||
		existingDigest != digest ||
		existingPolicy != input.PolicyVersion {
		return CompanyFundLedgerTaskEnqueueResult{}, fmt.Errorf("company-fund ledger task identity conflicts with existing evidence")
	}
	return CompanyFundLedgerTaskEnqueueResult{ID: id}, nil
}

func decodeCompanyFundLedgerTaskPayload(payload []byte) (TransactionUpsertInput, error) {
	if len(payload) == 0 || len(payload) > maxLedgerTaskPayloadBytes || !json.Valid(payload) {
		return TransactionUpsertInput{}, fmt.Errorf("invalid company-fund ledger task payload")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var decoded companyFundLedgerTaskPayload
	if err := decoder.Decode(&decoded); err != nil {
		return TransactionUpsertInput{}, fmt.Errorf("decode company-fund ledger task payload: %w", err)
	}
	if decoded.Version != ledgerTaskPayloadVersion {
		return TransactionUpsertInput{}, fmt.Errorf("unsupported company-fund ledger task payload version")
	}
	if bytes.Equal(bytes.TrimSpace(decoded.Proposal.ProviderDisplay.Fee.DetailsJSON), []byte("null")) {
		decoded.Proposal.ProviderDisplay.Fee.DetailsJSON = nil
	}
	return decoded.Proposal, nil
}

func nullableRelationshipReferenceType(value RelationshipReferenceType) any {
	if value == "" {
		return nil
	}
	return value
}

var _ interface {
	EnqueueCompanyFundLedgerTask(context.Context, CompanyFundLedgerTaskInput) (CompanyFundLedgerTaskEnqueueResult, error)
} = (*DBRepository)(nil)
