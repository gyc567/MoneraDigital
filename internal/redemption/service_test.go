package redemption

import (
	"math"
	"testing"
	"time"
)

func TestCreateRedemption(t *testing.T) {
	repo := NewInMemoryRedemptionRepository()
	svc := NewRedemptionService(repo)
	rec, err := svc.CreateRedemption("u1", "prod-7d", 1000, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec == nil {
		t.Fatalf("expected redemption record, got nil")
	}
	if rec.UserID != "u1" {
		t.Fatalf("unexpected userID: %s", rec.UserID)
	}
	if rec.ProductID != "prod-7d" {
		t.Fatalf("unexpected productID: %s", rec.ProductID)
	}
	if rec.Principal != 1000 {
		t.Fatalf("unexpected principal: %v", rec.Principal)
	}
	if rec.Status != StatusHolding {
		t.Fatalf("expected status HOLDING, got %s", rec.Status)
	}
	expectedInterest := 1000 * 0.07 * 7 / 365.0
	if rec.InterestTotal == 0 {
		t.Fatalf("interestTotal should be non-zero")
	}
	if math.Abs(rec.InterestTotal-expectedInterest) > 1e-9 {
		t.Fatalf("unexpected interestTotal: got %f expected %f", rec.InterestTotal, expectedInterest)
	}
	if rec.RedemptionAmount <= 1000 {
		// should be principal + interest
		t.Fatalf("redemptionAmount should be greater than principal, got %f", rec.RedemptionAmount)
	}
	_ = time.Now()
}

func TestRedeemMaturityNoAutoRenew(t *testing.T) {
	repo := NewInMemoryRedemptionRepository()
	svc := NewRedemptionService(repo)
	rec, err := svc.CreateRedemption("u1", "prod-7d", 1000, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	redeemed, err := svc.RedeemMaturity(rec.ID)
	if err != nil {
		t.Fatalf("redeem failed: %v", err)
	}
	if redeemed == nil {
		// should not be nil
		// but if nil, it's an error
		t.Fatalf("redeemMaturity returned nil record")
	}
	if redeemed.Status != StatusRedeemed {
		t.Fatalf("expected redeemed status, got %s", redeemed.Status)
	}

	// ensure redeemedAt is set
	if redeemed.RedeemedAt == nil {
		t.Fatalf("redeemedAt should be set")
	}
}

func TestRedeemMaturityWithAutoRenew(t *testing.T) {
	repo := NewInMemoryRedemptionRepository()
	svc := NewRedemptionService(repo)
	rec, err := svc.CreateRedemption("u2", "prod-7d", 1000, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	renewed, err := svc.RedeemMaturity(rec.ID)
	if err != nil {
		t.Fatalf("redeem with auto renew failed: %v", err)
	}
	if renewed == nil {
		t.Fatalf("expected renewed record, got nil")
	}
	if renewed.Status != StatusHolding {
		t.Fatalf("expected renewed record status HOLDING, got %s", renewed.Status)
	}
}
