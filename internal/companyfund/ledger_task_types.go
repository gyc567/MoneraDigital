package companyfund

import (
	"fmt"
	"strings"
	"time"
)

const (
	maxLedgerTaskEvidenceReferenceBytes = 512
	maxLedgerTaskErrorCodeBytes         = 128
)

// LedgerTaskKind identifies a durable company-fund maintenance action. These
// tasks contain only allowlisted normalized data; retained provider payloads
// remain encrypted in company_fund_provider_events.
type LedgerTaskKind string

const (
	LedgerTaskKindFeeRelationship      LedgerTaskKind = "FEE_RELATIONSHIP"
	LedgerTaskKindConversionPair       LedgerTaskKind = "CONVERSION_PAIR"
	LedgerTaskKindReversalRelationship LedgerTaskKind = "REVERSAL_RELATIONSHIP"
	LedgerTaskKindFeeClassification    LedgerTaskKind = "FEE_CLASSIFICATION"
	LedgerTaskKindReversalInheritance  LedgerTaskKind = "REVERSAL_INHERITANCE"
)

func (kind LedgerTaskKind) Valid() bool {
	switch kind {
	case LedgerTaskKindFeeRelationship,
		LedgerTaskKindConversionPair,
		LedgerTaskKindReversalRelationship,
		LedgerTaskKindFeeClassification,
		LedgerTaskKindReversalInheritance:
		return true
	default:
		return false
	}
}

type LedgerTaskState string

const (
	LedgerTaskStateWaiting    LedgerTaskState = "WAITING"
	LedgerTaskStateLeased     LedgerTaskState = "LEASED"
	LedgerTaskStateCompleted  LedgerTaskState = "COMPLETED"
	LedgerTaskStateDeadLetter LedgerTaskState = "DEAD_LETTER"
)

func (state LedgerTaskState) Valid() bool {
	switch state {
	case LedgerTaskStateWaiting, LedgerTaskStateLeased, LedgerTaskStateCompleted, LedgerTaskStateDeadLetter:
		return true
	default:
		return false
	}
}

// RelationshipReferenceType records the provider contract that made a link
// eligible for resolution. The raw source reference remains independent from
// movement_key and therefore cannot change ledger identity.
type RelationshipReferenceType string

const (
	RelationshipReferenceSourceIDExactParent    RelationshipReferenceType = "SOURCE_ID_EXACT_PARENT"
	RelationshipReferenceSourceIDReversalTarget RelationshipReferenceType = "SOURCE_ID_REVERSAL_TARGET"
	RelationshipReferenceSourceIDConversion     RelationshipReferenceType = "SOURCE_ID_CONVERSION_GROUP"
	RelationshipReferenceSourceIDGroupOnly      RelationshipReferenceType = "SOURCE_ID_GROUP_ONLY"
	RelationshipReferenceBatchIDGroupOnly       RelationshipReferenceType = "BATCH_ID_GROUP_ONLY"
)

func (value RelationshipReferenceType) Valid() bool {
	switch value {
	case RelationshipReferenceSourceIDExactParent,
		RelationshipReferenceSourceIDReversalTarget,
		RelationshipReferenceSourceIDConversion,
		RelationshipReferenceSourceIDGroupOnly,
		RelationshipReferenceBatchIDGroupOnly:
		return true
	default:
		return false
	}
}

// CompanyFundLedgerTaskInput is the immutable task proposal emitted beside a
// provider fact. ProviderTransactionFactID is injected only after the worker
// has persisted that fact.
type CompanyFundLedgerTaskInput struct {
	Channel                      TransactionSource
	ProviderAccountKey           string
	Kind                         LedgerTaskKind
	ProviderTransactionFactID    int64
	SubjectProviderTransactionID string
	SubjectTransactionID         *int64
	RelationshipReferenceType    RelationshipReferenceType
	RelationshipReferenceKey     string
	RelationshipGroupKey         string
	EvidenceReference            string
	Proposal                     TransactionUpsertInput
	PolicyVersion                string
	RelationshipSLA              time.Duration
}

func (input CompanyFundLedgerTaskInput) validate(allowUnboundFact bool) error {
	if input.Channel != ChannelAirwallex {
		return fmt.Errorf("company-fund ledger tasks currently require the Airwallex channel")
	}
	if err := validateRequiredString("ledger task provider account key", input.ProviderAccountKey, maxProviderFactAccountKeyBytes); err != nil {
		return err
	}
	if !input.Kind.Valid() {
		return fmt.Errorf("unsupported company-fund ledger task kind %q", input.Kind)
	}
	if (!allowUnboundFact && input.ProviderTransactionFactID <= 0) || (allowUnboundFact && input.ProviderTransactionFactID < 0) {
		return fmt.Errorf("ledger task provider fact ID is invalid")
	}
	if err := validateRequiredString("ledger task subject provider transaction ID", input.SubjectProviderTransactionID, maxProviderFactReferenceBytes); err != nil {
		return err
	}
	if input.SubjectTransactionID != nil && *input.SubjectTransactionID <= 0 {
		return fmt.Errorf("ledger task subject transaction ID must be positive")
	}
	if input.RelationshipReferenceType != "" || input.RelationshipReferenceKey != "" {
		if !input.RelationshipReferenceType.Valid() {
			return fmt.Errorf("unsupported relationship reference type %q", input.RelationshipReferenceType)
		}
		if err := validateRequiredString("ledger task relationship reference key", input.RelationshipReferenceKey, maxProviderFactReferenceBytes); err != nil {
			return err
		}
	}
	if err := validateRequiredString("ledger task evidence reference", input.EvidenceReference, maxLedgerTaskEvidenceReferenceBytes); err != nil {
		return err
	}
	if input.RelationshipSLA <= 0 || input.RelationshipSLA.Microseconds() <= 0 {
		return fmt.Errorf("ledger task relationship SLA must be at least one microsecond")
	}
	if input.Proposal.MovementKey != "" {
		if err := input.validateDeferredProposal(); err != nil {
			return err
		}
	}
	if input.PolicyVersion != "" {
		if err := validateRequiredString("ledger task policy version", input.PolicyVersion, 64); err != nil {
			return err
		}
	}
	return nil
}

func (input CompanyFundLedgerTaskInput) validateDeferredProposal() error {
	proposal := input.Proposal
	proposal.RelationshipReferenceType = input.RelationshipReferenceType
	proposal.RelationshipReferenceKey = input.RelationshipReferenceKey
	proposal.RelationshipGroupKey = input.RelationshipGroupKey
	if proposal.Channel != input.Channel ||
		proposal.ProviderAccountKey != input.ProviderAccountKey ||
		proposal.ProviderTransactionID != input.SubjectProviderTransactionID {
		return fmt.Errorf("ledger task proposal identity does not match its subject")
	}
	switch input.Kind {
	case LedgerTaskKindFeeRelationship:
		if proposal.MovementKind != MovementKindFee {
			return fmt.Errorf("fee relationship task requires a fee movement proposal")
		}
		if proposal.ParentMovementKey == "" {
			proposal.ParentMovementKey = "pending-parent:" + input.RelationshipReferenceKey
		}
	case LedgerTaskKindReversalRelationship:
		if proposal.MovementKind != MovementKindReversal {
			return fmt.Errorf("reversal relationship task requires a reversal movement proposal")
		}
		if proposal.ReversalOfMovementKey == "" {
			proposal.ReversalOfMovementKey = "pending-reversal:" + input.RelationshipReferenceKey
		}
	case LedgerTaskKindConversionPair:
		if proposal.MovementKind != MovementKindConversion {
			return fmt.Errorf("conversion relationship task requires a conversion movement proposal")
		}
	default:
		return fmt.Errorf("ledger task kind %q cannot carry a deferred movement proposal", input.Kind)
	}
	if len(proposal.ParentMovementKey) > 256 || len(proposal.ReversalOfMovementKey) > 256 {
		return fmt.Errorf("ledger task relationship reference is too long")
	}
	return proposal.validate()
}

type CompanyFundLedgerTaskEnqueueResult struct {
	ID       int64
	Inserted bool
}

// ProviderEventDeferredMovement binds one future movement proposal to a fact
// in the same normalized event result. It is not visible in the ledger until a
// durable relationship task resolves its prerequisites.
type ProviderEventDeferredMovement struct {
	FactReference string
	Task          CompanyFundLedgerTaskInput
}

func (value ProviderEventDeferredMovement) validate(facts map[string]struct{}) error {
	if err := validateRequiredString("deferred movement fact reference", value.FactReference, maxProviderEventNormalizedFactReferenceBytes); err != nil {
		return NewPermanentProviderEventError(err)
	}
	if _, found := facts[value.FactReference]; !found {
		return NewPermanentProviderEventError(fmt.Errorf("deferred movement references unknown fact %q", value.FactReference))
	}
	if err := value.Task.validate(true); err != nil {
		return NewPermanentProviderEventError(fmt.Errorf("invalid deferred movement task: %w", err))
	}
	return nil
}

func safeLedgerTaskErrorCode(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > maxLedgerTaskErrorCodeBytes {
		return value[:maxLedgerTaskErrorCodeBytes]
	}
	return value
}
