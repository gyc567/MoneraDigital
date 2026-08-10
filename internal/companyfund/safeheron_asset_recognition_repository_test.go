package companyfund

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDBRepositoryListsBoundedSafeheronUnrecognizedAssets(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(selectSafeheronUnrecognizedAssetCandidatesSQL)).
		WithArgs(int64(300), 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "provider_asset_key"}).
			AddRow(385, "USDT_ERC20").
			AddRow(386, "USDT_ERC20"))

	rows, err := NewDBRepository(db).ListSafeheronUnrecognizedAssetCandidates(context.Background(), 300, 20)
	if err != nil || len(rows) != 2 || rows[0].TransactionID != 385 || rows[1].ProviderAssetKey != "USDT_ERC20" {
		t.Fatalf("candidates = %#v, %v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDBRepositoryAppliesSafeheronRecognitionOnlyWhileEvidenceStillMatches(t *testing.T) {
	patch := SafeheronAssetRecognitionPatch{
		TransactionID:            385,
		ExpectedProviderAssetKey: "USDT_ERC20",
		Asset: AssetIdentity{
			Currency: "USDT", ChainCode: "ETHEREUM", ProviderAssetKey: "USDT_ERC20", ContractAddress: "0xdac17f",
		},
	}

	t.Run("applied", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectQuery(regexp.QuoteMeta(applySafeheronAssetRecognitionSQL)).
			WithArgs(int64(385), "USDT_ERC20", "USDT", "ETHEREUM", "0xdac17f").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(385))

		applied, err := NewDBRepository(db).ApplySafeheronAssetRecognition(context.Background(), patch)
		if err != nil || !applied {
			t.Fatalf("ApplySafeheronAssetRecognition() = %t, %v", applied, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("stale predicate is idempotent", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectQuery(regexp.QuoteMeta(applySafeheronAssetRecognitionSQL)).
			WithArgs(int64(385), "USDT_ERC20", "USDT", "ETHEREUM", "0xdac17f").
			WillReturnError(sql.ErrNoRows)

		applied, err := NewDBRepository(db).ApplySafeheronAssetRecognition(context.Background(), patch)
		if err != nil || applied {
			t.Fatalf("ApplySafeheronAssetRecognition() = %t, %v", applied, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSafeheronRecognitionRepairSQLCannotRewriteFinanceOrIdentityFields(t *testing.T) {
	lower := strings.ToLower(applySafeheronAssetRecognitionSQL)
	for _, required := range []string{
		"channel = 'safeheron'", "is_unrecognized_asset = true", "provider_asset_key = $2",
		"currency = $3", "chain_code = $4", "asset_contract = nullif($5, '')", "is_unrecognized_asset = false",
	} {
		if !strings.Contains(lower, required) {
			t.Fatalf("recognition repair SQL missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"amount =", "movement_key =", "provider_transaction_id =", "transaction_direction =",
		"category_level1", "category_level2", "include_in_operating", "usd_value =", "current_valuation_history_id =",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("recognition repair SQL rewrites protected field %q", forbidden)
		}
	}
}

func TestDBRepositorySafeheronRecognitionRepairValidatesInputs(t *testing.T) {
	repository := &DBRepository{}
	if _, err := repository.ListSafeheronUnrecognizedAssetCandidates(context.Background(), -1, 1); err == nil {
		t.Fatal("negative cursor accepted")
	}
	if _, err := repository.ListSafeheronUnrecognizedAssetCandidates(context.Background(), 0, 0); err == nil {
		t.Fatal("zero limit accepted")
	}
	for _, patch := range []SafeheronAssetRecognitionPatch{
		{},
		{TransactionID: 1, ExpectedProviderAssetKey: "USDT_ERC20", Asset: AssetIdentity{Currency: "USDT", ChainCode: "ETHEREUM", ProviderAssetKey: "OTHER"}},
	} {
		if _, err := repository.ApplySafeheronAssetRecognition(context.Background(), patch); err == nil {
			t.Fatalf("invalid patch accepted: %#v", patch)
		}
	}
	if _, err := repository.ListSafeheronUnrecognizedAssetCandidates(context.Background(), 0, 1); err == nil {
		t.Fatal("valid list input accepted without database")
	}
	if _, err := repository.ApplySafeheronAssetRecognition(context.Background(), SafeheronAssetRecognitionPatch{
		TransactionID: 1, ExpectedProviderAssetKey: "USDT_ERC20",
		Asset: AssetIdentity{Currency: "USDT", ChainCode: "ETHEREUM", ProviderAssetKey: "USDT_ERC20"},
	}); err == nil {
		t.Fatal("valid patch accepted without database")
	}
}

func TestDBRepositorySafeheronRecognitionRepairPropagatesDatabaseFailures(t *testing.T) {
	t.Run("list query", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		queryErr := errors.New("query failed")
		mock.ExpectQuery(regexp.QuoteMeta(selectSafeheronUnrecognizedAssetCandidatesSQL)).
			WithArgs(int64(0), 1).WillReturnError(queryErr)
		if _, err := NewDBRepository(db).ListSafeheronUnrecognizedAssetCandidates(t.Context(), 0, 1); !errors.Is(err, queryErr) {
			t.Fatalf("query failure = %v", err)
		}
	})

	for _, testCase := range []struct {
		name string
		rows *sqlmock.Rows
	}{
		{name: "scan", rows: sqlmock.NewRows([]string{"id", "provider_asset_key"}).AddRow("invalid", "USDT_ERC20")},
		{name: "invalid persisted row", rows: sqlmock.NewRows([]string{"id", "provider_asset_key"}).AddRow(1, " USDT_ERC20")},
		{name: "row iterator", rows: sqlmock.NewRows([]string{"id", "provider_asset_key"}).AddRow(1, "USDT_ERC20").RowError(0, errors.New("rows failed"))},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectQuery(regexp.QuoteMeta(selectSafeheronUnrecognizedAssetCandidatesSQL)).
				WithArgs(int64(0), 1).WillReturnRows(testCase.rows)
			if _, err := NewDBRepository(db).ListSafeheronUnrecognizedAssetCandidates(t.Context(), 0, 1); err == nil {
				t.Fatal("database row failure was ignored")
			}
		})
	}

	validPatch := SafeheronAssetRecognitionPatch{
		TransactionID: 1, ExpectedProviderAssetKey: "USDT_ERC20",
		Asset: AssetIdentity{Currency: "USDT", ChainCode: "ETHEREUM", ProviderAssetKey: "USDT_ERC20"},
	}
	t.Run("update query", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		queryErr := errors.New("update failed")
		mock.ExpectQuery(regexp.QuoteMeta(applySafeheronAssetRecognitionSQL)).
			WithArgs(int64(1), "USDT_ERC20", "USDT", "ETHEREUM", "").WillReturnError(queryErr)
		if _, err := NewDBRepository(db).ApplySafeheronAssetRecognition(t.Context(), validPatch); !errors.Is(err, queryErr) {
			t.Fatalf("update failure = %v", err)
		}
	})

	t.Run("different returned row", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		mock.ExpectQuery(regexp.QuoteMeta(applySafeheronAssetRecognitionSQL)).
			WithArgs(int64(1), "USDT_ERC20", "USDT", "ETHEREUM", "").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
		if _, err := NewDBRepository(db).ApplySafeheronAssetRecognition(t.Context(), validPatch); err == nil {
			t.Fatal("different updated row accepted")
		}
	})
}
