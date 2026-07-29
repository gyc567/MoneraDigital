package companyfund

import (
	"context"
	"testing"
	"time"
)

func TestProviderEventNormalizationResult_AcceptsDeferredFeeBoundToDurableFact(t *testing.T) {
	fact := validProviderEventWorkerFact(
		"fee-fact",
		ProviderValueScopeDirectItem,
		ProviderFactAllocationStateNotApplicable,
	)
	proposal := validProviderEventWorkerMovement("fee-movement")
	fromAccountID := int64(7)
	proposal.ProviderTransactionID = "fee-71"
	proposal.MovementKind = MovementKindFee
	proposal.Direction = DirectionOutflow
	proposal.FromCompanyFundAccountID = &fromAccountID
	proposal.ParentMovementKey = ""

	result := ProviderEventNormalizationResult{
		Facts: []ProviderEventNormalizedFact{{
			Reference: "fee-fact-reference",
			Input:     fact,
		}},
		DeferredMovements: []ProviderEventDeferredMovement{{
			FactReference: "fee-fact-reference",
			Task: CompanyFundLedgerTaskInput{
				Channel:                      ChannelAirwallex,
				ProviderAccountKey:           "account-a",
				Kind:                         LedgerTaskKindFeeRelationship,
				SubjectProviderTransactionID: "fee-71",
				RelationshipReferenceType:    RelationshipReferenceSourceIDExactParent,
				RelationshipReferenceKey:     "payment-71",
				EvidenceReference:            "production-fee-fixture-71",
				Proposal:                     proposal,
				RelationshipSLA:              7 * 24 * time.Hour,
			},
		}},
	}

	if err := result.validate(); err != nil {
		t.Fatalf("deferred provider event result must be valid: %v", err)
	}
}

func TestProviderEventNormalizationResult_RejectsDeferredMovementWithoutKnownFact(t *testing.T) {
	proposal := validProviderEventWorkerMovement("fee-movement")
	proposal.MovementKind = MovementKindFee
	proposal.Direction = DirectionOutflow
	result := ProviderEventNormalizationResult{
		Facts: []ProviderEventNormalizedFact{{
			Reference: "known-fact",
			Input: validProviderEventWorkerFact(
				"fee-fact",
				ProviderValueScopeDirectItem,
				ProviderFactAllocationStateNotApplicable,
			),
		}},
		DeferredMovements: []ProviderEventDeferredMovement{{
			FactReference: "missing-fact",
			Task: CompanyFundLedgerTaskInput{
				Channel:                      ChannelAirwallex,
				ProviderAccountKey:           "account-a",
				Kind:                         LedgerTaskKindFeeRelationship,
				SubjectProviderTransactionID: "fee-71",
				RelationshipReferenceType:    RelationshipReferenceSourceIDExactParent,
				RelationshipReferenceKey:     "payment-71",
				EvidenceReference:            "production-fee-fixture-71",
				Proposal:                     proposal,
				RelationshipSLA:              7 * 24 * time.Hour,
			},
		}},
	}

	if err := result.validate(); err == nil {
		t.Fatal("deferred movement with unknown fact reference must fail")
	}
}

func TestProviderEventWorker_PersistsDeferredTaskWithoutHalfMovement(t *testing.T) {
	lease := validProviderEventWorkerLease()
	lease.Channel = ChannelAirwallex
	lease.ProviderAccountKey = "account-a"
	repository := &providerEventWorkerRepositoryStub{lease: &lease}
	proposal := validProviderEventWorkerMovement("fee-movement")
	fromAccountID := int64(7)
	proposal.Channel = ChannelAirwallex
	proposal.ProviderAccountKey = "account-a"
	proposal.ProviderTransactionID = "fee-71"
	proposal.ProviderMovementID = "fee-71"
	proposal.MovementKind = MovementKindFee
	proposal.Direction = DirectionOutflow
	proposal.FromCompanyFundAccountID = &fromAccountID
	proposal.ToCompanyFundAccountID = nil
	proposal.ParentMovementKey = ""

	worker := newProviderEventWorkerForTest(
		t,
		repository,
		&providerEventPayloadReaderStub{payload: []byte(`{"id":"fee-71"}`)},
		map[TransactionSource]ProviderEventNormalizer{
			ChannelAirwallex: &providerEventNormalizerStub{result: ProviderEventNormalizationResult{
				Facts: []ProviderEventNormalizedFact{{
					Reference: "fee-fact-reference",
					Input: func() ProviderTransactionFactInput {
						fact := validProviderEventWorkerFact(
							"fee-fact",
							ProviderValueScopeDirectItem,
							ProviderFactAllocationStateNotApplicable,
						)
						fact.ProviderTransactionID = "fee-71"
						return fact
					}(),
				}},
				DeferredMovements: []ProviderEventDeferredMovement{{
					FactReference: "fee-fact-reference",
					Task: CompanyFundLedgerTaskInput{
						Channel:                      ChannelAirwallex,
						ProviderAccountKey:           "account-a",
						Kind:                         LedgerTaskKindFeeRelationship,
						SubjectProviderTransactionID: "fee-71",
						RelationshipReferenceType:    RelationshipReferenceSourceIDExactParent,
						RelationshipReferenceKey:     "payment-71",
						EvidenceReference:            "production-fee-fixture-71",
						Proposal:                     proposal,
						RelationshipSLA:              7 * 24 * time.Hour,
					},
				}},
			}},
		},
		time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC),
	)

	result, err := worker.ProcessNext(context.Background())
	if err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}
	if result.Outcome != ProviderEventFinalizeProcessed || result.FactCount != 1 || result.TaskCount != 1 || result.MovementCount != 0 {
		t.Fatalf("ProcessNext() = %#v, want durable fact and task without movement", result)
	}
	if len(repository.tasks) != 1 || repository.tasks[0].ProviderTransactionFactID != 1 {
		t.Fatalf("persisted tasks = %#v, want fact-bound task", repository.tasks)
	}
	if len(repository.upserts) != 0 {
		t.Fatalf("deferred relationship must not create a half movement: %#v", repository.upserts)
	}
}
