package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"monera-digital/internal/logger"
	"monera-digital/internal/safeheron"
	"monera-digital/internal/wallet/deposit"
)

type safeheronWebhookVerifierFunc func([]byte) (*safeheron.WebhookEvent, error)

func (verify safeheronWebhookVerifierFunc) WebhookConvert(body []byte) (*safeheron.WebhookEvent, error) {
	return verify(body)
}

type failingWebhookBody struct{}

func (failingWebhookBody) Read([]byte) (int, error) { return 0, errors.New("body read details") }
func (failingWebhookBody) Close() error             { return nil }

func observeSafeheronWebhookLogs(t *testing.T) (*observer.ObservedLogs, *bytes.Buffer) {
	t.Helper()
	previousLogger := logger.Logger
	core, observed := observer.New(zap.DebugLevel)
	logger.Logger = zap.New(core).Sugar()
	legacy := &bytes.Buffer{}
	previousWriter := log.Writer()
	log.SetOutput(legacy)
	t.Cleanup(func() {
		logger.Logger = previousLogger
		log.SetOutput(previousWriter)
	})
	return observed, legacy
}

func TestWebhook_SuccessEmitsOneStructuredTerminalLog(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		inserted bool
		result   string
	}{
		{name: "stored", inserted: true, result: "stored"},
		{name: "duplicate", inserted: false, result: "duplicate"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			observed, legacy := observeSafeheronWebhookLogs(t)
			h := NewSafeheronWebhookHandler(
				&fakeVerifier{convertFn: func(_ []byte) (*safeheron.WebhookEvent, error) {
					return &safeheron.WebhookEvent{
						EventType: "TRANSACTION_STATUS_CHANGED",
						EventDetail: safeheron.EventDetail{
							TxKey:             "tx-log-result",
							TransactionStatus: "COMPLETED",
						},
					}, nil
				}},
				&fakeRecorder{insertFn: func(_ context.Context, _ *deposit.Event) (bool, error) {
					return testCase.inserted, nil
				}},
				nil,
			)

			w := runWebhook(h, `{"signed":"envelope"}`)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			entries := observed.All()
			if len(entries) != 1 {
				t.Fatalf("structured log entries = %d, want exactly one: %#v", len(entries), entries)
			}
			entry := entries[0]
			if entry.Level != zap.InfoLevel || entry.Message != "safeheron webhook processed" {
				t.Fatalf("terminal log = %s %q, want INFO processed", entry.Level, entry.Message)
			}
			fields := entry.ContextMap()
			for key, want := range map[string]any{
				"component":              "safeheron_webhook",
				"result":                 testCase.result,
				"httpStatus":             int64(http.StatusOK),
				"eventType":              "TRANSACTION_STATUS_CHANGED",
				"txKey":                  "tx-log-result",
				"providerStatus":         "COMPLETED",
				"companyFundEligibility": "not_configured",
				"bridgeResult":           "not_applicable",
			} {
				if got := fields[key]; got != want {
					t.Errorf("field %s = %#v, want %#v", key, got, want)
				}
			}
			if got, ok := fields["eventIdHash"].(string); !ok || len(got) != 12 {
				t.Fatalf("eventIdHash = %#v, want 12-character summary", fields["eventIdHash"])
			}
			if _, exists := fields["eventId"]; exists {
				t.Fatal("success log must not contain the full event ID")
			}
			if _, exists := fields["rawPayload"]; exists {
				t.Fatal("success log must not contain raw provider payload")
			}
			if strings.TrimSpace(legacy.String()) != "" {
				t.Fatalf("success path emitted legacy process logs:\n%s", legacy.String())
			}
		})
	}
}

func TestWebhook_RejectionEmitsOneStructuredWarningWithoutPayload(t *testing.T) {
	observed, legacy := observeSafeheronWebhookLogs(t)
	const sensitiveEnvelope = `{"secret":"must-not-appear"}`
	h := NewSafeheronWebhookHandler(
		&fakeVerifier{convertFn: func(_ []byte) (*safeheron.WebhookEvent, error) {
			return nil, errors.New("signature details must not appear")
		}},
		&fakeRecorder{insertFn: func(_ context.Context, _ *deposit.Event) (bool, error) {
			t.Fatal("recorder should not be called")
			return false, nil
		}},
		nil,
	)

	w := runWebhook(h, sensitiveEnvelope)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("structured log entries = %d, want exactly one", len(entries))
	}
	entry := entries[0]
	fields := entry.ContextMap()
	if entry.Level != zap.WarnLevel || entry.Message != "safeheron webhook rejected" ||
		fields["result"] != "rejected" || fields["reason"] != "verification_failed" ||
		fields["httpStatus"] != int64(http.StatusUnauthorized) {
		t.Fatalf("rejection log = %s %q %#v", entry.Level, entry.Message, fields)
	}
	allLogText := entry.Message + legacy.String()
	for _, value := range fields {
		allLogText += " " + strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(toString(value)), "\n", " "))
	}
	if strings.Contains(allLogText, sensitiveEnvelope) || strings.Contains(allLogText, "signature details") {
		t.Fatalf("rejection log leaked provider or verifier details: %s", allLogText)
	}
	if strings.TrimSpace(legacy.String()) != "" {
		t.Fatalf("rejection emitted legacy process logs:\n%s", legacy.String())
	}
}

func TestWebhook_PersistenceFailureEmitsOneStructuredError(t *testing.T) {
	observed, legacy := observeSafeheronWebhookLogs(t)
	h := NewSafeheronWebhookHandler(
		&fakeVerifier{convertFn: func(_ []byte) (*safeheron.WebhookEvent, error) {
			return &safeheron.WebhookEvent{
				EventType:   "TRANSACTION_CREATED",
				EventDetail: safeheron.EventDetail{TxKey: "tx-log-error", TransactionStatus: "CONFIRMING"},
			}, nil
		}},
		&fakeRecorder{insertFn: func(_ context.Context, _ *deposit.Event) (bool, error) {
			return false, errors.New("database credentials must not appear")
		}},
		nil,
	)

	w := runWebhook(h, `{}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("structured log entries = %d, want exactly one", len(entries))
	}
	entry := entries[0]
	fields := entry.ContextMap()
	if entry.Level != zap.ErrorLevel || entry.Message != "safeheron webhook failed" ||
		fields["result"] != "failed" || fields["stage"] != "event_persistence" ||
		fields["reason"] != "insert_failed" || fields["httpStatus"] != int64(http.StatusInternalServerError) {
		t.Fatalf("failure log = %s %q %#v", entry.Level, entry.Message, fields)
	}
	if strings.Contains(entry.Message+legacy.String(), "database credentials") {
		t.Fatal("failure log leaked underlying error text")
	}
	if strings.TrimSpace(legacy.String()) != "" {
		t.Fatalf("failure emitted legacy process logs:\n%s", legacy.String())
	}
}

func TestWebhook_RejectionReasonsRemainSingleStructuredWarnings(t *testing.T) {
	noVerify := func(t *testing.T) WebhookVerifier {
		return safeheronWebhookVerifierFunc(func([]byte) (*safeheron.WebhookEvent, error) {
			t.Fatal("verifier must not be called")
			return nil, nil
		})
	}
	testCases := []struct {
		name   string
		reason string
		status int
		run    func(*testing.T) *httptest.ResponseRecorder
	}{
		{
			name: "blocked IP", reason: "ip_not_allowed", status: http.StatusForbidden,
			run: func(t *testing.T) *httptest.ResponseRecorder {
				h := NewSafeheronWebhookHandler(noVerify(t), &fakeRecorder{}, []string{"1.2.3.4"})
				return runWebhookWithIP(h, `{}`, "9.9.9.9")
			},
		},
		{
			name: "empty body", reason: "empty_body", status: http.StatusBadRequest,
			run: func(t *testing.T) *httptest.ResponseRecorder {
				h := NewSafeheronWebhookHandler(noVerify(t), &fakeRecorder{}, nil)
				return runWebhook(h, "")
			},
		},
		{
			name: "missing tx key", reason: "missing_tx_key", status: http.StatusBadRequest,
			run: func(t *testing.T) *httptest.ResponseRecorder {
				h := NewSafeheronWebhookHandler(
					&fakeVerifier{convertFn: func([]byte) (*safeheron.WebhookEvent, error) {
						return &safeheron.WebhookEvent{EventType: "TRANSACTION_CREATED"}, nil
					}},
					&fakeRecorder{},
					nil,
				)
				return runWebhook(h, `{}`)
			},
		},
		{
			name: "body too large", reason: "body_too_large", status: http.StatusRequestEntityTooLarge,
			run: func(t *testing.T) *httptest.ResponseRecorder {
				h := NewSafeheronWebhookHandler(noVerify(t), &fakeRecorder{}, nil)
				body := io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), MaxWebhookBodyBytes+1)))
				return runWebhookWithRequestBody(h, body)
			},
		},
		{
			name: "body read failure", reason: "body_read_failed", status: http.StatusBadRequest,
			run: func(t *testing.T) *httptest.ResponseRecorder {
				h := NewSafeheronWebhookHandler(noVerify(t), &fakeRecorder{}, nil)
				return runWebhookWithRequestBody(h, failingWebhookBody{})
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			observed, legacy := observeSafeheronWebhookLogs(t)
			response := testCase.run(t)
			if response.Code != testCase.status {
				t.Fatalf("status = %d, want %d", response.Code, testCase.status)
			}
			entries := observed.All()
			if len(entries) != 1 {
				t.Fatalf("log entries = %d, want exactly one", len(entries))
			}
			entry := entries[0]
			fields := entry.ContextMap()
			if entry.Level != zap.WarnLevel || entry.Message != "safeheron webhook rejected" ||
				fields["reason"] != testCase.reason || fields["httpStatus"] != int64(testCase.status) {
				t.Fatalf("rejection log = %s %q %#v", entry.Level, entry.Message, fields)
			}
			if strings.TrimSpace(legacy.String()) != "" {
				t.Fatalf("rejection emitted legacy process logs:\n%s", legacy.String())
			}
		})
	}
}

func TestWebhook_EmptyDecryptedPayloadEmitsOneStructuredError(t *testing.T) {
	observed, legacy := observeSafeheronWebhookLogs(t)
	h := NewSafeheronWebhookHandler(
		safeheronWebhookVerifierFunc(func([]byte) (*safeheron.WebhookEvent, error) {
			return &safeheron.WebhookEvent{
				EventType:   "TRANSACTION_CREATED",
				EventDetail: safeheron.EventDetail{TxKey: "tx-empty-payload"},
			}, nil
		}),
		&fakeRecorder{insertFn: func(context.Context, *deposit.Event) (bool, error) {
			t.Fatal("recorder must not be called")
			return false, nil
		}},
		nil,
	)

	response := runWebhook(h, `{}`)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	entries := observed.All()
	if len(entries) != 1 || entries[0].Level != zap.ErrorLevel ||
		entries[0].ContextMap()["stage"] != "decryption" {
		t.Fatalf("empty payload log = %#v", entries)
	}
	if strings.TrimSpace(legacy.String()) != "" {
		t.Fatalf("empty payload emitted legacy logs:\n%s", legacy.String())
	}
}

func TestWebhook_CompanyFundSourceFailureEmitsOneStructuredError(t *testing.T) {
	observed, legacy := observeSafeheronWebhookLogs(t)
	body := []byte(`{"eventType":"TRANSACTION_STATUS_CHANGED"}`)
	h := newSafeheronCompanyFundBridgeHandler(
		&fakeRecorder{insertFn: func(context.Context, *deposit.Event) (bool, error) { return true, nil }},
		&safeheronCompanyFundSourceStub{err: errors.New("source database details")},
		&safeheronCompanyFundBridgeStub{},
		&safeheronCompanyFundEligibilityStub{},
		body,
	)

	response := runWebhook(h, `{}`)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	entries := observed.All()
	if len(entries) != 1 || entries[0].Level != zap.ErrorLevel ||
		entries[0].ContextMap()["stage"] != "company_fund_eligibility_source" {
		t.Fatalf("source failure log = %#v", entries)
	}
	if strings.Contains(entries[0].Message+legacy.String(), "source database details") {
		t.Fatal("source failure log leaked underlying error text")
	}
	if strings.TrimSpace(legacy.String()) != "" {
		t.Fatalf("source failure emitted legacy logs:\n%s", legacy.String())
	}
}

func TestEmitSafeheronWebhookLogIsSafeBeforeLoggerInitialization(t *testing.T) {
	previousLogger := logger.Logger
	logger.Logger = nil
	t.Cleanup(func() { logger.Logger = previousLogger })
	emitSafeheronWebhookLog(
		safeheronWebhookLogInfo,
		"safeheron webhook processed",
		"stored",
		http.StatusOK,
		time.Now(),
	)
}

func runWebhookWithRequestBody(h *SafeheronWebhookHandler, body io.ReadCloser) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	request := httptest.NewRequest(http.MethodPost, "/api/webhooks/safeheron", nil)
	request.Body = body
	context.Request = request
	h.Receive(context)
	return response
}

func toString(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
