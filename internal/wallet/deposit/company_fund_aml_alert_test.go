package deposit

import (
	"context"
	"strings"
	"testing"
	"time"
)

type companyFundAMLAlertHandlerStub struct {
	result CompanyFundAMLAlertResult
	err    error
	calls  []CompanyFundAMLAlertInput
}

func (stub *companyFundAMLAlertHandlerStub) HandleCompanyFundAMLAlert(_ context.Context, input CompanyFundAMLAlertInput) (CompanyFundAMLAlertResult, error) {
	stub.calls = append(stub.calls, input)
	return stub.result, stub.err
}

func TestProcessKYTAlert_CompanyFundAlertIsAppliedWithoutCustomerOrphan(t *testing.T) {
	repo := newMockRepo()
	handler := &companyFundAMLAlertHandlerStub{result: CompanyFundAMLAlertApplied}
	svc := newKYTSvc(t, repo, nil, true)
	svc.SetCompanyFundAMLAlertHandler(handler)

	repo.pending = []*Event{{
		ID:             701,
		EventID:        "company-aml-event",
		EventType:      "AML_KYT_ALERT",
		RawPayload:     []byte(`{"eventType":"AML_KYT_ALERT","eventDetail":{"txKey":"company-tx","amlList":[{"provider":"MistTrack","riskLevel":"LOW"}]}}`),
		SafeheronTxKey: "company-tx",
	}}

	processed, err := svc.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("ProcessOne() = %v, %v", processed, err)
	}
	if len(handler.calls) != 1 {
		t.Fatalf("company AML handler calls = %d, want 1", len(handler.calls))
	}
	if got := handler.calls[0]; got.TransactionKey != "company-tx" || got.ScreeningState != "TRIGGERED" || got.RiskLevel != KytLow {
		t.Fatalf("company AML handler input = %#v", got)
	}
	if len(repo.doneIDs) != 1 || len(repo.errorIDs) != 0 || len(repo.noTxIncrements) != 0 {
		t.Fatalf("company AML event must be finalized without orphan handling: done=%v errors=%v increments=%v", repo.doneIDs, repo.errorIDs, repo.noTxIncrements)
	}
}

func TestProcessKYTAlert_IgnoredRoutingFinishesWithoutCustomerOrphan(t *testing.T) {
	repo := newMockRepo()
	handler := &companyFundAMLAlertHandlerStub{result: CompanyFundAMLAlertIgnored}
	svc := newKYTSvc(t, repo, nil, true)
	svc.SetCompanyFundAMLAlertHandler(handler)
	repo.pending = []*Event{{
		ID:             706,
		EventID:        "ignored-routing-aml-event",
		EventType:      "AML_KYT_ALERT",
		RawPayload:     []byte(`{"eventType":"AML_KYT_ALERT","eventDetail":{"txKey":"ignored-routing-tx","amlList":[{"provider":"MistTrack","riskLevel":"LOW"}]}}`),
		SafeheronTxKey: "ignored-routing-tx",
	}}

	processed, err := svc.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("ProcessOne() = %v, %v", processed, err)
	}
	if len(repo.doneIDs) != 1 || repo.doneIDs[0] != 706 || len(repo.noTxIncrements) != 0 {
		t.Fatalf("ignored routing event must finish without orphan retry: done=%v increments=%v", repo.doneIDs, repo.noTxIncrements)
	}
}

func TestProcessKYTAlert_IgnoredRoutingFinalizationErrors(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		configure   func(*mockRepo)
		wantErrPart string
	}{
		{name: "mark done", configure: func(repo *mockRepo) { repo.markDoneErr = context.Canceled }, wantErrPart: "mark ignored routing AML event done"},
		{name: "commit", configure: func(repo *mockRepo) { repo.commitErr = context.Canceled }, wantErrPart: "commit ignored routing AML event"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			repo := newMockRepo()
			testCase.configure(repo)
			svc := newKYTSvc(t, repo, nil, true)
			svc.SetCompanyFundAMLAlertHandler(&companyFundAMLAlertHandlerStub{result: CompanyFundAMLAlertIgnored})
			repo.pending = []*Event{{
				ID: 707, EventID: "ignored-routing-finalization", EventType: "AML_KYT_ALERT",
				RawPayload: []byte(`{"eventType":"AML_KYT_ALERT","eventDetail":{"txKey":"ignored-routing-tx","amlList":[{"provider":"MistTrack","riskLevel":"LOW"}]}}`),
			}}

			_, err := svc.ProcessOne(context.Background())
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("ProcessOne() error = %v, want %q", err, testCase.wantErrPart)
			}
		})
	}
}

func TestProcessKYTAlert_DeferredRoutingScheduleError(t *testing.T) {
	repo := newMockRepo()
	repo.deferEventErr = context.DeadlineExceeded
	svc := newKYTSvc(t, repo, nil, true)
	svc.SetCompanyFundAMLAlertHandler(&companyFundAMLAlertHandlerStub{result: CompanyFundAMLAlertDeferred})
	repo.pending = []*Event{{
		ID: 708, EventID: "deferred-routing-schedule-error", EventType: "AML_KYT_ALERT",
		RawPayload: []byte(`{"eventType":"AML_KYT_ALERT","eventDetail":{"txKey":"deferred-routing-tx","amlList":[{"provider":"MistTrack","riskLevel":"LOW"}]}}`),
	}}

	_, err := svc.ProcessOne(context.Background())
	if err == nil || !strings.Contains(err.Error(), "schedule deferred company-fund AML event") {
		t.Fatalf("ProcessOne() error = %v", err)
	}
}

func TestProcessKYTAlert_DeferredRoutingCommitError(t *testing.T) {
	repo := newMockRepo()
	repo.commitErr = context.Canceled
	svc := newKYTSvc(t, repo, nil, true)
	svc.SetCompanyFundAMLAlertHandler(&companyFundAMLAlertHandlerStub{result: CompanyFundAMLAlertDeferred})
	repo.pending = []*Event{{
		ID: 709, EventID: "deferred-routing-rollback-error", EventType: "AML_KYT_ALERT",
		RawPayload: []byte(`{"eventType":"AML_KYT_ALERT","eventDetail":{"txKey":"deferred-routing-tx","amlList":[{"provider":"MistTrack","riskLevel":"LOW"}]}}`),
	}}

	_, err := svc.ProcessOne(context.Background())
	if err == nil || !strings.Contains(err.Error(), "commit deferred company-fund AML event") {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if len(repo.deferredEventIDs) != 1 {
		t.Fatalf("retry update must precede the failed atomic commit: %v", repo.deferredEventIDs)
	}
}

func TestProcessKYTAlert_CompanyFundAlertDefersWithoutCustomerOrphanRetry(t *testing.T) {
	repo := newMockRepo()
	handler := &companyFundAMLAlertHandlerStub{result: CompanyFundAMLAlertDeferred}
	svc := newKYTSvc(t, repo, nil, true)
	svc.SetCompanyFundAMLAlertHandler(handler)

	repo.pending = []*Event{{
		ID:             702,
		EventID:        "company-aml-before-projection",
		EventType:      "AML_KYT_ALERT",
		RawPayload:     []byte(`{"eventType":"AML_KYT_ALERT","eventDetail":{"txKey":"company-before-projection","amlList":[{"provider":"MistTrack","riskLevel":"LOW"}]}}`),
		SafeheronTxKey: "company-before-projection",
	}}

	processed, err := svc.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("ProcessOne() = %v, %v; want durable defer counted as processed work", processed, err)
	}
	if len(handler.calls) != 1 || len(repo.doneIDs) != 0 || len(repo.errorIDs) != 0 || len(repo.noTxIncrements) != 0 || len(repo.deferredEventIDs) != 1 {
		t.Fatalf("deferred company AML event must be durably delayed without orphan handling: calls=%d done=%v errors=%v increments=%v deferred=%v", len(handler.calls), repo.doneIDs, repo.errorIDs, repo.noTxIncrements, repo.deferredEventIDs)
	}
	if repo.eventRetryInterval != time.Minute {
		t.Fatalf("deferred retry interval = %s, want 1m", repo.eventRetryInterval)
	}
}

func TestProcessKYTAlert_CustomerDepositStillCreditsWhenCompanyHandlerIsInstalled(t *testing.T) {
	repo := newMockRepo()
	handler := &companyFundAMLAlertHandlerStub{result: CompanyFundAMLAlertNotCompany}
	svc := newKYTSvc(t, repo, nil, true)
	svc.SetCompanyFundAMLAlertHandler(handler)
	repo.deposits["customer-tx"] = &DepositRow{
		ID: 41, UserID: 7, SafeheronTxKey: "customer-tx", Amount: "1.25", Asset: "USDT", Status: DepositStatusKYTPending,
	}
	repo.pending = []*Event{{
		ID:         703,
		EventID:    "customer-aml-event",
		EventType:  "AML_KYT_ALERT",
		RawPayload: []byte(`{"eventType":"AML_KYT_ALERT","eventDetail":{"txKey":"customer-tx","amlList":[{"provider":"MistTrack","riskLevel":"LOW"}]}}`),
	}}

	processed, err := svc.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("ProcessOne() = %v, %v", processed, err)
	}
	if repo.deposits["customer-tx"].Status != DepositStatusCredited {
		t.Fatalf("customer deposit status = %s, want CREDITED", repo.deposits["customer-tx"].Status)
	}
	if len(handler.calls) != 1 || len(repo.doneIDs) != 1 {
		t.Fatalf("customer AML event must preserve user processing: calls=%d done=%v", len(handler.calls), repo.doneIDs)
	}
}

func TestProcessKYTAlert_CustomerHighRiskStillRequiresManualReviewWhenCompanyHandlerIsInstalled(t *testing.T) {
	repo := newMockRepo()
	handler := &companyFundAMLAlertHandlerStub{result: CompanyFundAMLAlertNotCompany}
	svc := newKYTSvc(t, repo, nil, true)
	svc.SetCompanyFundAMLAlertHandler(handler)
	repo.deposits["customer-high-risk-tx"] = &DepositRow{
		ID: 43, UserID: 9, SafeheronTxKey: "customer-high-risk-tx", Amount: "1.25", Asset: "USDT", Status: DepositStatusKYTPending,
	}
	repo.pending = []*Event{{
		ID:         705,
		EventID:    "customer-high-risk-aml-event",
		EventType:  "AML_KYT_ALERT",
		RawPayload: []byte(`{"eventType":"AML_KYT_ALERT","eventDetail":{"txKey":"customer-high-risk-tx","amlList":[{"provider":"MistTrack","riskLevel":"HIGH"}]}}`),
	}}

	processed, err := svc.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("ProcessOne() = %v, %v", processed, err)
	}
	if repo.deposits["customer-high-risk-tx"].Status != DepositStatusManualReview {
		t.Fatalf("high-risk customer deposit status = %s, want MANUAL_REVIEW", repo.deposits["customer-high-risk-tx"].Status)
	}
	if len(handler.calls) != 1 || len(repo.doneIDs) != 1 {
		t.Fatalf("customer AML event must preserve user manual-review handling: calls=%d done=%v", len(handler.calls), repo.doneIDs)
	}
}

func TestProcessKYTAlert_DualRoutingWaitsForBothCustomerAndCompanyProjection(t *testing.T) {
	repo := newMockRepo()
	handler := &companyFundAMLAlertHandlerStub{result: CompanyFundAMLAlertDeferred}
	svc := newKYTSvc(t, repo, nil, true)
	svc.SetCompanyFundAMLAlertHandler(handler)
	repo.deposits["dual-tx"] = &DepositRow{
		ID: 42, UserID: 8, SafeheronTxKey: "dual-tx", Amount: "2.5", Asset: "USDT", Status: DepositStatusKYTPending,
	}
	repo.pending = []*Event{{
		ID:         704,
		EventID:    "dual-aml-event",
		EventType:  "AML_KYT_ALERT",
		RawPayload: []byte(`{"eventType":"AML_KYT_ALERT","eventDetail":{"txKey":"dual-tx","amlList":[{"provider":"MistTrack","riskLevel":"LOW"}]}}`),
	}}

	processed, err := svc.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("ProcessOne() = %v, %v; want durable defer counted as processed work", processed, err)
	}
	if repo.deposits["dual-tx"].Status != DepositStatusKYTPending {
		t.Fatalf("dual customer deposit must wait for complete routing, got %s", repo.deposits["dual-tx"].Status)
	}
	if len(handler.calls) != 1 || len(repo.doneIDs) != 0 || len(repo.errorIDs) != 0 || len(repo.noTxIncrements) != 0 || len(repo.deferredEventIDs) != 1 {
		t.Fatalf("dual AML event must remain pending on a durable retry schedule: calls=%d done=%v errors=%v increments=%v deferred=%v", len(handler.calls), repo.doneIDs, repo.errorIDs, repo.noTxIncrements, repo.deferredEventIDs)
	}
}
