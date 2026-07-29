package companyfund

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestEnqueueCompanyFundLedgerTask_IsIdempotentWithinPolicyVersion(t *testing.T) {
	db, mock := newCompanyFundMockDB(t)
	defer db.Close()
	repository := NewDBRepository(db)
	input := validLedgerTaskInput()
	payload, err := json.Marshal(companyFundLedgerTaskPayload{
		Version:  ledgerTaskPayloadVersion,
		Proposal: input.Proposal,
	})
	if err != nil {
		t.Fatal(err)
	}
	digest := payloadSHA256Hex(payload)

	mock.ExpectQuery(regexp.QuoteMeta(insertCompanyFundLedgerTaskSQL)).
		WithArgs(
			input.Channel,
			input.ProviderAccountKey,
			input.Kind,
			input.ProviderTransactionFactID,
			input.SubjectProviderTransactionID,
			nil,
			input.RelationshipReferenceType,
			input.RelationshipReferenceKey,
			nil,
			input.EvidenceReference,
			string(payload),
			digest,
			input.PolicyVersion,
			input.RelationshipSLA.Microseconds(),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(regexp.QuoteMeta(selectCompanyFundLedgerTaskIdentitySQL)).
		WithArgs(
			input.Channel,
			input.ProviderAccountKey,
			input.Kind,
			input.SubjectProviderTransactionID,
			input.PolicyVersion,
		).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"provider_transaction_fact_id",
			"relationship_reference_type",
			"relationship_reference_key",
			"relationship_group_key",
			"evidence_reference",
			"task_payload_digest",
			"policy_version",
		}).AddRow(
			81,
			input.ProviderTransactionFactID,
			input.RelationshipReferenceType,
			input.RelationshipReferenceKey,
			"",
			input.EvidenceReference,
			digest,
			input.PolicyVersion,
		))

	result, err := repository.EnqueueCompanyFundLedgerTask(context.Background(), input)
	if err != nil || result.Inserted || result.ID != 81 {
		t.Fatalf("EnqueueCompanyFundLedgerTask() = %#v, %v", result, err)
	}
	assertCompanyFundMockExpectations(t, mock)
}

func TestLedgerTaskIdentityIncludesPolicyVersionForSafeRuleUpgrades(t *testing.T) {
	if !regexp.MustCompile(`COALESCE\(policy_version, ''\) = \$5`).MatchString(selectCompanyFundLedgerTaskIdentitySQL) {
		t.Fatal("ledger task readback identity must include policy version")
	}
}

func TestLedgerTaskTerminalWritesFenceStaleLeaseAttempts(t *testing.T) {
	for name, statement := range map[string]string{
		"complete":    completeCompanyFundLedgerTaskSQL,
		"retry":       retryCompanyFundLedgerTaskSQL,
		"dead-letter": deadLetterCompanyFundLedgerTaskSQL,
	} {
		t.Run(name, func(t *testing.T) {
			for _, required := range []string{
				"attempt_count = ",
				"lease_expires_at > clock_timestamp()",
			} {
				if !strings.Contains(statement, required) {
					t.Fatalf("%s terminal write must contain %q", name, required)
				}
			}
		})
	}
}

func TestLedgerTaskNextDueIncludesRetriesSLAsAndExpiredLeases(t *testing.T) {
	for _, required := range []string{
		"LEAST(next_attempt_at, sla_expires_at)",
		"ELSE lease_expires_at",
		"task_state IN ('WAITING', 'LEASED')",
	} {
		if !strings.Contains(nextCompanyFundLedgerTaskDueSQL, required) {
			t.Fatalf("ledger next-due query must contain %q", required)
		}
	}
}

func validLedgerTaskInput() CompanyFundLedgerTaskInput {
	proposal := validProviderEventWorkerMovement("fee-task-movement")
	fromAccountID := int64(7)
	proposal.Channel = ChannelAirwallex
	proposal.ProviderAccountKey = "account-a"
	proposal.ProviderTransactionID = "fee-81"
	proposal.ProviderMovementID = "fee-81"
	proposal.MovementKind = MovementKindFee
	proposal.Direction = DirectionOutflow
	proposal.FromCompanyFundAccountID = &fromAccountID
	proposal.ToCompanyFundAccountID = nil
	return CompanyFundLedgerTaskInput{
		Channel:                      ChannelAirwallex,
		ProviderAccountKey:           "account-a",
		Kind:                         LedgerTaskKindFeeRelationship,
		ProviderTransactionFactID:    71,
		SubjectProviderTransactionID: "fee-81",
		RelationshipReferenceType:    RelationshipReferenceSourceIDExactParent,
		RelationshipReferenceKey:     "payment-81",
		EvidenceReference:            "sandbox-fee-relationship-v1",
		Proposal:                     proposal,
		PolicyVersion:                "fee-policy-v1",
		RelationshipSLA:              24 * time.Hour,
	}
}
