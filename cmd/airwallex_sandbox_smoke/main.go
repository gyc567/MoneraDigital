// Command airwallex_sandbox_smoke exercises the Airwallex sandbox path locally
// against monera_local. It is a delivery gate for stage, not a production tool.
//
// Modes:
//
//	go run ./cmd/airwallex_sandbox_smoke
//	  login + list FT + in-memory normalize (no provider-event writes)
//
//	go run ./cmd/airwallex_sandbox_smoke -persist
//	  reconcile FT window → encrypted inbox → drain worker → facts/txns
//
//	go run ./cmd/airwallex_sandbox_smoke -webhook
//	  sign a local webhook envelope → handler Receive → worker IGNORED
//	  (webhook is audit+wake only; ledger facts still come from -persist)
//
//	go run ./cmd/airwallex_sandbox_smoke -persist -webhook
//	  full local gate: REST ingest + webhook path
//
// Required env:
//
//	AIRWALLEX_BASE_URL, AIRWALLEX_CLIENT_ID, AIRWALLEX_API_KEY, AIRWALLEX_LOGIN_AS
//
// Optional env:
//
//	AIRWALLEX_API_VERSION (default 2026-07-17)
//	AIRWALLEX_RUNTIME_CONFIG_PATH (default secrets/airwallex-sandbox-runtime.json)
//	AIRWALLEX_WEBHOOK_SECRET / AIRWALLEX_WEBHOOK_SECRET_PATH
//	  (default secrets/airwallex-webhook-secret.local, auto-created for -webhook)
//	AIRWALLEX_WEBHOOK_VERSION (default event-v1; must match runtime event_version)
//	COMPANY_FUND_PAYLOAD_KEY_PATH (default secrets/company-fund-payload.key)
//	COMPANY_FUND_PAYLOAD_KEY_VERSION (default payload-v1)
//	MONERA_DATABASE_URL (smoke override) or DATABASE_URL
//	  (default postgresql://linden@localhost:5432/monera_local?sslmode=disable)
//
// Never prints API keys, tokens, webhook secrets, or full account ids.
package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"monera-digital/internal/companyfund"
	"monera-digital/internal/db"
	"monera-digital/internal/handlers"
)

func main() {
	persist := flag.Bool("persist", false, "write sandbox FT snapshots into monera_local and drain the provider-event worker")
	webhook := flag.Bool("webhook", false, "exercise signed webhook ingress + worker IGNORED path against monera_local")
	lookback := flag.Duration("lookback", 72*time.Hour, "reconcile window length ending at now")
	pageSize := flag.Int("page-size", 100, "Financial Transactions page size")
	flag.Parse()

	if os.Getenv("APP_ENV") != "production" {
		_ = godotenv.Overload(".env")
	}

	if err := run(*persist, *webhook, *lookback, *pageSize); err != nil {
		fmt.Fprintf(os.Stderr, "airwallex sandbox smoke failed: %v\n", err)
		os.Exit(1)
	}
}

func run(persist, webhook bool, lookback time.Duration, pageSize int) error {
	cfg, err := loadSmokeConfig(webhook)
	if err != nil {
		return err
	}

	client, err := companyfund.NewAirwallexClient(companyfund.AirwallexClientConfig{
		BaseURL:    cfg.BaseURL,
		ClientID:   cfg.ClientID,
		APIKey:     cfg.APIKey,
		APIVersion: cfg.APIVersion,
		LoginAs:    cfg.LoginAs,
	})
	if err != nil {
		return fmt.Errorf("airwallex client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	sqlDB, err := db.InitDB(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open monera db: %w", err)
	}
	defer sqlDB.Close()

	loader := companyfund.NewPostgresAccountRegistryLoader(sqlDB)
	registry := companyfund.NewAccountRegistry(loader, time.Minute)
	if err := registry.Load(ctx); err != nil {
		return fmt.Errorf("load account registry: %w", err)
	}
	account, ok := companyfund.ResolveAirwallexSingleAccountScope(registry.Snapshot(), cfg.LoginAs)
	if !ok {
		return fmt.Errorf("DB does not have exactly one enabled AIRWALLEX account matching LOGIN_AS")
	}
	fmt.Printf("registry ok  account_id=%d key=%s…\n", account.ID, shortID(account.ProviderAccountKey))

	runtimeRaw, err := os.ReadFile(cfg.RuntimeConfigPath)
	if err != nil {
		return fmt.Errorf("read runtime config %s: %w", cfg.RuntimeConfigPath, err)
	}
	runtimeConfig, err := companyfund.ParseAirwallexFinancialTransactionsRuntimeConfigJSON(runtimeRaw)
	if err != nil {
		return fmt.Errorf("parse runtime config: %w", err)
	}
	if !runtimeConfig.Enabled {
		return fmt.Errorf("runtime config enabled=false")
	}
	if runtimeConfig.APIVersion != cfg.APIVersion {
		return fmt.Errorf("runtime api_version %q != client pin %q", runtimeConfig.APIVersion, cfg.APIVersion)
	}
	if cfg.WebhookVersion != "" && cfg.WebhookVersion != runtimeConfig.EventVersion {
		return fmt.Errorf("webhook version %q must equal runtime event_version %q", cfg.WebhookVersion, runtimeConfig.EventVersion)
	}

	bundle, err := companyfund.NewAirwallexFinancialTransactionsScopedRuntimeBundle(runtimeConfig, registry, cfg.LoginAs)
	if err != nil {
		return fmt.Errorf("runtime bundle: %w", err)
	}
	if !bundle.Enabled || bundle.ProviderEvents == nil || bundle.FinancialTransactions == nil {
		return fmt.Errorf("runtime bundle inactive")
	}
	fmt.Printf("runtime ok  rules=%d mapping=%s event=%s schema=%s\n",
		bundle.Resolvers.RuleCount(), runtimeConfig.MappingVersion, runtimeConfig.EventVersion, runtimeConfig.SchemaVersion)

	now := time.Now().UTC()
	windowStart := now.Add(-lookback)
	page, err := client.ListFinancialTransactions(ctx, companyfund.AirwallexFinancialTransactionsRequest{
		FromCreatedAt: windowStart,
		ToCreatedAt:   now,
		PageNum:       0,
		PageSize:      pageSize,
	})
	if err != nil {
		return fmt.Errorf("list financial transactions: %w", err)
	}
	fmt.Printf("login+list ok  base=%s api_version=%s items=%d has_more=%v window=%s..%s\n",
		redactHost(cfg.BaseURL), cfg.APIVersion, len(page.Items), page.HasMore,
		windowStart.Format(time.RFC3339), now.Format(time.RFC3339))

	if err := normalizeInMemory(bundle, runtimeConfig, account, page.Items); err != nil {
		return err
	}
	if !persist && !webhook {
		fmt.Println("dry-run complete (no provider-event writes). re-run with -persist and/or -webhook.")
		return nil
	}

	return runMutating(ctx, cfg, client, runtimeConfig, sqlDB, account, lookback, pageSize, now, persist, webhook)
}

func normalizeInMemory(
	bundle *companyfund.AirwallexFinancialTransactionsRuntimeBundle,
	runtimeConfig companyfund.AirwallexFinancialTransactionsRuntimeConfig,
	account companyfund.CompanyFundAccount,
	items []companyfund.AirwallexFinancialTransaction,
) error {
	apply, ignore, quarantine, fail := 0, 0, 0, 0
	combos := map[string]int{}
	reasons := map[string]int{}
	for _, item := range items {
		key := fmt.Sprintf("%s|%s|%s|%s", item.TransactionType, item.SourceType, item.Status, item.Currency)
		combos[key]++
		result := bundle.FinancialTransactions.Normalize(companyfund.AirwallexFinancialTransactionNormalizationInput{
			SchemaVersion:      runtimeConfig.SchemaVersion,
			EventVersion:       runtimeConfig.EventVersion,
			ProviderAccountKey: account.ProviderAccountKey,
			ConfiguredAccount:  account,
			Source: companyfund.AirwallexFinancialTransactionSourceMetadata{
				ProviderEventID:       "smoke-" + item.ProviderID,
				ProviderEventRecordID: 1,
				PayloadDigest:         strings.Repeat("a", 64),
				FactSource:            companyfund.ProviderSourceReconciliation,
				SeenSource:            companyfund.TransactionSeenSourceReconciliation,
			},
			FinancialTransaction: item,
		})
		switch result.Disposition {
		case companyfund.AirwallexFinancialTransactionDispositionApply:
			apply++
		case companyfund.AirwallexFinancialTransactionDispositionIgnore:
			ignore++
			if result.Reason != "" {
				reasons["ignore:"+result.Reason]++
			}
		case companyfund.AirwallexFinancialTransactionDispositionQuarantine:
			quarantine++
			if result.Reason != "" {
				reasons["quarantine:"+result.Reason]++
			}
		default:
			fail++
			fmt.Printf("  normalize UNKNOWN disposition id=%s combo=%s disp=%q reason=%s\n",
				shortID(item.ProviderID), key, result.Disposition, result.Reason)
		}
	}
	fmt.Printf("normalize summary  apply=%d ignore=%d quarantine=%d fail=%d\n", apply, ignore, quarantine, fail)
	for k, n := range combos {
		fmt.Printf("  combo %s count=%d\n", k, n)
	}
	for k, n := range reasons {
		fmt.Printf("  reason %s count=%d\n", k, n)
	}
	if fail > 0 {
		return fmt.Errorf("in-memory normalize failures: %d", fail)
	}
	return nil
}

func runMutating(
	ctx context.Context,
	cfg smokeConfig,
	client *companyfund.AirwallexClient,
	runtimeConfig companyfund.AirwallexFinancialTransactionsRuntimeConfig,
	sqlDB *sql.DB,
	account companyfund.CompanyFundAccount,
	lookback time.Duration,
	pageSize int,
	now time.Time,
	persist, webhook bool,
) error {
	payloadKey, err := loadPayloadKey(cfg.PayloadKeyPath)
	if err != nil {
		return err
	}
	cipher, err := companyfund.NewAES256GCMPayloadCipher(map[string][]byte{cfg.PayloadKeyVersion: payloadKey})
	if err != nil {
		return fmt.Errorf("payload cipher: %w", err)
	}

	repo := companyfund.NewDBRepository(sqlDB)
	payloads, err := companyfund.NewOwnedProviderPayloadService(repo, cipher, time.Now)
	if err != nil {
		return fmt.Errorf("payload service: %w", err)
	}

	// Re-resolve scope against a fresh registry snapshot so a concurrent MGT
	// enable/disable cannot widen the smoke write path beyond LOGIN_AS.
	loader := companyfund.NewPostgresAccountRegistryLoader(sqlDB)
	liveRegistry := companyfund.NewAccountRegistry(loader, time.Minute)
	if err := liveRegistry.Load(ctx); err != nil {
		return fmt.Errorf("reload account registry: %w", err)
	}
	live, ok := companyfund.ResolveAirwallexSingleAccountScope(liveRegistry.Snapshot(), cfg.LoginAs)
	if !ok {
		return fmt.Errorf("DB no longer has exactly one enabled AIRWALLEX account matching LOGIN_AS")
	}
	account = live

	bundle, err := companyfund.NewAirwallexFinancialTransactionsScopedRuntimeBundle(runtimeConfig, liveRegistry, cfg.LoginAs)
	if err != nil {
		return fmt.Errorf("scoped runtime bundle: %w", err)
	}
	if !bundle.Enabled || bundle.ProviderEvents == nil {
		return fmt.Errorf("runtime bundle inactive for mutating path")
	}

	if persist {
		if err := runPersist(ctx, cfg, client, runtimeConfig, sqlDB, repo, payloads, bundle, account, lookback, pageSize, now); err != nil {
			return err
		}
	}
	if webhook {
		if err := runWebhook(ctx, cfg, sqlDB, repo, payloads, bundle, account); err != nil {
			return err
		}
	}
	return printDBCounts(ctx, sqlDB, persist)
}

func runPersist(
	ctx context.Context,
	cfg smokeConfig,
	client *companyfund.AirwallexClient,
	runtimeConfig companyfund.AirwallexFinancialTransactionsRuntimeConfig,
	sqlDB *sql.DB,
	repo *companyfund.DBRepository,
	payloads *companyfund.OwnedProviderPayloadService,
	bundle *companyfund.AirwallexFinancialTransactionsRuntimeBundle,
	account companyfund.CompanyFundAccount,
	lookback time.Duration,
	pageSize int,
	now time.Time,
) error {
	syncAdapter, err := companyfund.NewCompanyFundReconciliationSyncRunAdapter(repo, companyfund.CompanyFundReconciliationSyncRunAdapterConfig{
		LeaseOwner:    "airwallex-sandbox-smoke",
		LeaseDuration: 5 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("sync-run adapter: %w", err)
	}

	reconciler, err := companyfund.NewAirwallexFinancialTransactionsReconciler(
		client,
		payloads,
		syncAdapter,
		companyfund.AirwallexFinancialTransactionsReconcilerConfig{
			APIVersion:        runtimeConfig.APIVersion,
			SchemaVersion:     runtimeConfig.SchemaVersion,
			EventVersion:      runtimeConfig.EventVersion,
			PageSize:          pageSize,
			MaxPages:          20,
			PayloadKeyVersion: cfg.PayloadKeyVersion,
			PayloadRetention:  48 * time.Hour,
		},
	)
	if err != nil {
		return fmt.Errorf("reconciler: %w", err)
	}

	windowStart := now.Add(-lookback)
	result, err := reconciler.Reconcile(ctx, companyfund.AirwallexFinancialTransactionsReconcileInput{
		Account:            account,
		ProviderAccountKey: account.ProviderAccountKey,
		WindowStart:        windowStart,
		WindowEnd:          now,
		APIVersion:         runtimeConfig.APIVersion,
		SchemaVersion:      runtimeConfig.SchemaVersion,
		EventVersion:       runtimeConfig.EventVersion,
	})
	if err != nil {
		return fmt.Errorf("reconcile: %w", err)
	}
	fmt.Printf("reconcile ok  run_id=%d pages=%d created=%d existing=%d seen=%d\n",
		result.RunID, result.PagesFetched, result.EventsCreated, result.EventsExisting, result.CandidatesSeen)

	payloadReader, err := companyfund.NewProviderEventSourceBytesReader(
		companyfund.NewPostgresSafeheronWebhookPayloadReader(sqlDB),
		payloads,
	)
	if err != nil {
		return fmt.Errorf("payload reader: %w", err)
	}
	worker, err := companyfund.NewProviderEventWorker(
		repo,
		payloadReader,
		map[companyfund.TransactionSource]companyfund.ProviderEventNormalizer{
			companyfund.ChannelAirwallex: bundle.ProviderEvents,
		},
		companyfund.ProviderEventWorkerConfig{
			Owner:         "airwallex-sandbox-smoke",
			LeaseDuration: time.Minute,
			RenewInterval: 20 * time.Second,
			RetryPolicy: companyfund.ProviderEventRetryPolicy{
				InitialDelay: 30 * time.Second,
				MaxDelay:     time.Hour,
			},
			Now: time.Now,
		},
	)
	if err != nil {
		return fmt.Errorf("provider event worker: %w", err)
	}
	claimed, err := drainWorker(ctx, worker, 500)
	if err != nil {
		return err
	}
	fmt.Printf("worker drained claimed=%d\n", claimed)
	fmt.Println("persist complete")
	return nil
}

func runWebhook(
	ctx context.Context,
	cfg smokeConfig,
	sqlDB *sql.DB,
	repo *companyfund.DBRepository,
	payloads *companyfund.OwnedProviderPayloadService,
	bundle *companyfund.AirwallexFinancialTransactionsRuntimeBundle,
	account companyfund.CompanyFundAccount,
) error {
	secret, err := loadOrCreateWebhookSecret(cfg)
	if err != nil {
		return err
	}

	verifier, err := companyfund.NewAirwallexWebhookVerifier(companyfund.AirwallexWebhookVerifierConfig{
		Secret: secret,
		MaxAge: 5 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("webhook verifier: %w", err)
	}

	woke := false
	handler, err := handlers.NewCompanyFundAirwallexWebhookHandler(handlers.CompanyFundAirwallexWebhookHandlerConfig{
		Verifier:             verifier,
		Ingestor:             payloads,
		Wake:                 func() { woke = true },
		Eligible:             func() bool { return true },
		ProviderEventVersion: cfg.WebhookVersion,
		KeyVersion:           cfg.PayloadKeyVersion,
		Retention:            48 * time.Hour,
		LegalHold:            false,
	})
	if err != nil {
		return fmt.Errorf("webhook handler: %w", err)
	}

	eventID := fmt.Sprintf("smoke-wh-%d", time.Now().UTC().UnixNano())
	body, err := json.Marshal(map[string]any{
		"id":         eventID,
		"name":       "balance.updated",
		"account_id": account.ProviderAccountKey,
		"org_id":     "org_smoke_local",
		"data": map[string]any{
			"note": "sandbox-smoke local envelope; must not create ledger facts",
		},
	})
	if err != nil {
		return fmt.Errorf("marshal webhook body: %w", err)
	}
	timestamp := strconv.FormatInt(time.Now().UTC().UnixMilli(), 10)
	signature := signAirwallexWebhook(secret, timestamp, body)

	// Snapshot movement counts before webhook so we can prove envelope-only path
	// never materializes ledger rows.
	var factsBefore, txnsBefore int
	if err := sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM company_fund_provider_transaction_facts WHERE channel = 'AIRWALLEX'`).Scan(&factsBefore); err != nil {
		return err
	}
	if err := sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM company_fund_transactions WHERE channel = 'AIRWALLEX'`).Scan(&txnsBefore); err != nil {
		return err
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/webhooks/airwallex", bytes.NewReader(body))
	ginCtx.Request.Header.Set("x-timestamp", timestamp)
	ginCtx.Request.Header.Set("x-signature", signature)
	handler.Receive(ginCtx)
	if recorder.Code != http.StatusOK {
		return fmt.Errorf("webhook handler status=%d, want 200", recorder.Code)
	}
	if !woke {
		return fmt.Errorf("webhook handler did not invoke wake callback")
	}
	fmt.Printf("webhook ingest ok  event_id=%s status=%d woke=%t\n", shortID(eventID), recorder.Code, woke)

	// Idempotent replay must still 200 and must not double-insert.
	recorder2 := httptest.NewRecorder()
	ginCtx2, _ := gin.CreateTestContext(recorder2)
	ginCtx2.Request = httptest.NewRequest(http.MethodPost, "/api/webhooks/airwallex", bytes.NewReader(body))
	ginCtx2.Request.Header.Set("x-timestamp", timestamp)
	ginCtx2.Request.Header.Set("x-signature", signature)
	handler.Receive(ginCtx2)
	if recorder2.Code != http.StatusOK {
		return fmt.Errorf("webhook replay status=%d, want 200", recorder2.Code)
	}
	fmt.Println("webhook replay ok  status=200")

	payloadReader, err := companyfund.NewProviderEventSourceBytesReader(
		companyfund.NewPostgresSafeheronWebhookPayloadReader(sqlDB),
		payloads,
	)
	if err != nil {
		return fmt.Errorf("webhook payload reader: %w", err)
	}
	worker, err := companyfund.NewProviderEventWorker(
		repo,
		payloadReader,
		map[companyfund.TransactionSource]companyfund.ProviderEventNormalizer{
			companyfund.ChannelAirwallex: bundle.ProviderEvents,
		},
		companyfund.ProviderEventWorkerConfig{
			Owner:         "airwallex-sandbox-smoke-webhook",
			LeaseDuration: time.Minute,
			RenewInterval: 20 * time.Second,
			RetryPolicy: companyfund.ProviderEventRetryPolicy{
				InitialDelay: 30 * time.Second,
				MaxDelay:     time.Hour,
			},
			Now: time.Now,
		},
	)
	if err != nil {
		return fmt.Errorf("webhook worker: %w", err)
	}

	// Drain until the just-ingested envelope is terminal IGNORED, or fail closed.
	var sawIgnored bool
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !sawIgnored {
		step, stepErr := worker.ProcessNext(ctx)
		if stepErr != nil {
			return fmt.Errorf("webhook worker process: %w", stepErr)
		}
		if step.Claimed {
			fmt.Printf("  webhook worker event_id=%d outcome=%s facts=%d movements=%d\n",
				step.EventID, step.Outcome, step.FactCount, step.MovementCount)
			switch step.Outcome {
			case companyfund.ProviderEventFinalizeIgnored:
				sawIgnored = true
			case companyfund.ProviderEventFinalizeProcessed:
				return fmt.Errorf("webhook envelope was PROCESSED; must be IGNORED")
			default:
				return fmt.Errorf("webhook envelope worker outcome=%s, want IGNORED", step.Outcome)
			}
			continue
		}
		var status string
		err := sqlDB.QueryRowContext(ctx, `
			SELECT event_state
			FROM company_fund_provider_events
			WHERE channel = 'AIRWALLEX' AND provider_event_id = $1
		`, eventID).Scan(&status)
		if err != nil {
			return fmt.Errorf("lookup webhook provider event: %w", err)
		}
		switch status {
		case "IGNORED":
			sawIgnored = true
		case "PROCESSED":
			return fmt.Errorf("webhook envelope was PROCESSED; must be IGNORED")
		case "DEAD_LETTER", "FAILED":
			return fmt.Errorf("webhook envelope event_state=%s, want IGNORED", status)
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
	if !sawIgnored {
		var status string
		if err := sqlDB.QueryRowContext(ctx, `
			SELECT event_state FROM company_fund_provider_events
			WHERE channel = 'AIRWALLEX' AND provider_event_id = $1
		`, eventID).Scan(&status); err != nil {
			return fmt.Errorf("final lookup webhook provider event: %w", err)
		}
		return fmt.Errorf("webhook envelope event_state=%s after drain, want IGNORED", status)
	}

	var factsAfter, txnsAfter, eventRows int
	if err := sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM company_fund_provider_transaction_facts WHERE channel = 'AIRWALLEX'`).Scan(&factsAfter); err != nil {
		return err
	}
	if err := sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM company_fund_transactions WHERE channel = 'AIRWALLEX'`).Scan(&txnsAfter); err != nil {
		return err
	}
	if err := sqlDB.QueryRowContext(ctx, `
		SELECT count(*) FROM company_fund_provider_events
		WHERE channel = 'AIRWALLEX' AND provider_event_id = $1
	`, eventID).Scan(&eventRows); err != nil {
		return err
	}
	if eventRows != 1 {
		return fmt.Errorf("webhook event rows=%d, want exactly 1 (idempotent ingest)", eventRows)
	}
	if factsAfter != factsBefore || txnsAfter != txnsBefore {
		return fmt.Errorf("webhook path mutated ledger facts/txns (facts %d→%d, txns %d→%d)",
			factsBefore, factsAfter, txnsBefore, txnsAfter)
	}
	fmt.Printf("webhook path ok  ignored=%t ledger_unchanged facts=%d txns=%d\n", sawIgnored, factsAfter, txnsAfter)
	return nil
}

func drainWorker(ctx context.Context, worker *companyfund.ProviderEventWorker, limit int) (int, error) {
	processed, emptyStreak := 0, 0
	for processed < limit && emptyStreak < 3 {
		step, stepErr := worker.ProcessNext(ctx)
		if stepErr != nil {
			return processed, fmt.Errorf("worker process: %w", stepErr)
		}
		if !step.Claimed {
			emptyStreak++
			time.Sleep(50 * time.Millisecond)
			continue
		}
		emptyStreak = 0
		processed++
		fmt.Printf("  worker event_id=%d outcome=%s facts=%d movements=%d\n",
			step.EventID, step.Outcome, step.FactCount, step.MovementCount)
	}
	return processed, nil
}

func printDBCounts(ctx context.Context, sqlDB *sql.DB, requireEvents bool) error {
	var eventN, factN, txnN, runN, ignoredN int
	if err := sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM company_fund_provider_events WHERE channel = 'AIRWALLEX'`).Scan(&eventN); err != nil {
		return err
	}
	if err := sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM company_fund_provider_events WHERE channel = 'AIRWALLEX' AND event_state = 'IGNORED'`).Scan(&ignoredN); err != nil {
		return err
	}
	if err := sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM company_fund_provider_transaction_facts WHERE channel = 'AIRWALLEX'`).Scan(&factN); err != nil {
		return err
	}
	if err := sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM company_fund_transactions WHERE channel = 'AIRWALLEX'`).Scan(&txnN); err != nil {
		return err
	}
	if err := sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM company_fund_sync_runs WHERE channel = 'AIRWALLEX'`).Scan(&runN); err != nil {
		return err
	}
	fmt.Printf("db counts  events=%d ignored=%d facts=%d txns=%d sync_runs=%d\n", eventN, ignoredN, factN, txnN, runN)
	if requireEvents && eventN == 0 {
		return fmt.Errorf("expected AIRWALLEX provider events after persist")
	}
	return nil
}

type smokeConfig struct {
	BaseURL           string
	ClientID          string
	APIKey            string
	LoginAs           string
	APIVersion        string
	RuntimeConfigPath string
	PayloadKeyPath    string
	PayloadKeyVersion string
	WebhookSecret     string
	WebhookSecretPath string
	WebhookVersion    string
	DatabaseURL       string
}

func loadSmokeConfig(needWebhook bool) (smokeConfig, error) {
	cfg := smokeConfig{
		BaseURL:           strings.TrimSpace(os.Getenv("AIRWALLEX_BASE_URL")),
		ClientID:          strings.TrimSpace(os.Getenv("AIRWALLEX_CLIENT_ID")),
		APIKey:            strings.TrimSpace(os.Getenv("AIRWALLEX_API_KEY")),
		LoginAs:           strings.TrimSpace(os.Getenv("AIRWALLEX_LOGIN_AS")),
		APIVersion:        strings.TrimSpace(os.Getenv("AIRWALLEX_API_VERSION")),
		RuntimeConfigPath: strings.TrimSpace(os.Getenv("AIRWALLEX_RUNTIME_CONFIG_PATH")),
		PayloadKeyPath:    strings.TrimSpace(os.Getenv("COMPANY_FUND_PAYLOAD_KEY_PATH")),
		PayloadKeyVersion: strings.TrimSpace(os.Getenv("COMPANY_FUND_PAYLOAD_KEY_VERSION")),
		WebhookSecret:     strings.TrimSpace(os.Getenv("AIRWALLEX_WEBHOOK_SECRET")),
		WebhookSecretPath: strings.TrimSpace(os.Getenv("AIRWALLEX_WEBHOOK_SECRET_PATH")),
		WebhookVersion:    strings.TrimSpace(os.Getenv("AIRWALLEX_WEBHOOK_VERSION")),
		// MONERA_DATABASE_URL is a smoke-only override for operators isolating
		// monera_local; otherwise fall back to the repo-standard DATABASE_URL.
		DatabaseURL: firstNonEmptyEnv("MONERA_DATABASE_URL", "DATABASE_URL"),
	}
	if cfg.APIVersion == "" {
		cfg.APIVersion = "2026-07-17"
	}
	if cfg.RuntimeConfigPath == "" {
		cfg.RuntimeConfigPath = filepath.Join("secrets", "airwallex-sandbox-runtime.json")
	}
	if cfg.PayloadKeyPath == "" {
		cfg.PayloadKeyPath = filepath.Join("secrets", "company-fund-payload.key")
	}
	if cfg.PayloadKeyVersion == "" {
		cfg.PayloadKeyVersion = "payload-v1"
	}
	if cfg.WebhookSecretPath == "" {
		cfg.WebhookSecretPath = filepath.Join("secrets", "airwallex-webhook-secret.local")
	}
	if cfg.WebhookVersion == "" {
		cfg.WebhookVersion = "event-v1"
	}
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = "postgresql://linden@localhost:5432/monera_local?sslmode=disable"
	}
	if cfg.BaseURL == "" || cfg.ClientID == "" || cfg.APIKey == "" || cfg.LoginAs == "" {
		return smokeConfig{}, fmt.Errorf("AIRWALLEX_BASE_URL/CLIENT_ID/API_KEY/LOGIN_AS are required")
	}
	if !strings.Contains(strings.ToLower(cfg.BaseURL), "sandbox") {
		return smokeConfig{}, fmt.Errorf("refusing non-sandbox AIRWALLEX_BASE_URL %q", redactHost(cfg.BaseURL))
	}
	if needWebhook && strings.TrimSpace(cfg.WebhookVersion) == "" {
		return smokeConfig{}, fmt.Errorf("AIRWALLEX_WEBHOOK_VERSION is required for -webhook")
	}
	return cfg, nil
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func loadPayloadKey(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read payload key %s: %w", path, err)
	}
	hexKey := strings.TrimSpace(string(raw))
	if len(hexKey) != 64 {
		return nil, fmt.Errorf("payload key file must contain 64 hex chars, got len=%d", len(hexKey))
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("payload key hex is invalid")
	}
	return key, nil
}

func loadOrCreateWebhookSecret(cfg smokeConfig) (string, error) {
	if cfg.WebhookSecret != "" {
		return cfg.WebhookSecret, nil
	}
	if raw, err := os.ReadFile(cfg.WebhookSecretPath); err == nil {
		secret := strings.TrimSpace(string(raw))
		if secret != "" {
			return secret, nil
		}
	}
	// Local-only auto-create so -webhook works without console secret.
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate local webhook secret: %w", err)
	}
	secret := hex.EncodeToString(buf)
	if err := os.MkdirAll(filepath.Dir(cfg.WebhookSecretPath), 0o700); err != nil {
		return "", fmt.Errorf("create secrets dir: %w", err)
	}
	if err := os.WriteFile(cfg.WebhookSecretPath, []byte(secret+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write local webhook secret: %w", err)
	}
	fmt.Printf("webhook secret created at %s (local only; not for stage console)\n", cfg.WebhookSecretPath)
	return secret, nil
}

func signAirwallexWebhook(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func redactHost(raw string) string {
	raw = strings.TrimSpace(raw)
	_, rest, ok := strings.Cut(raw, "://")
	if !ok {
		return raw
	}
	if j := strings.IndexAny(rest, "/?"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

func shortID(v string) string {
	v = strings.TrimSpace(v)
	if len(v) <= 10 {
		return v
	}
	return v[:10]
}
