package redemption

import (
	"fmt"
	"time"
)

// RedemptionService wires the redemption use-cases with a repository
type RedemptionService struct {
	repo RedemptionRepository
}

func NewRedemptionService(repo RedemptionRepository) *RedemptionService {
	if repo == nil {
		repo = NewInMemoryRedemptionRepository()
	}
	return &RedemptionService{repo: repo}
}

func (s *RedemptionService) computeInterest(principal, apy float64, days int) float64 {
	if principal <= 0 || apy < 0 || days <= 0 {
		return 0
	}
	return principal * apy * float64(days) / 365.0
}

func (s *RedemptionService) CreateRedemption(userID, productID string, principal float64, autoRenew bool) (*RedemptionRecord, error) {
	if principal <= 0 {
		return nil, fmt.Errorf("invalid principal")
	}
	product, ok := GetProduct(productID)
	if !ok {
		return nil, fmt.Errorf("product not found")
	}
	durationDays := product.DurationDays
	apy := product.APY

	now := time.Now()
	endDate := now.AddDate(0, 0, durationDays)
	interestTotal := s.computeInterest(principal, apy, durationDays)
	redemptionAmount := principal + interestTotal

	rec := &RedemptionRecord{
		UserID:           userID,
		ProductID:        productID,
		Principal:        principal,
		APY:              apy,
		DurationDays:     durationDays,
		Status:           StatusHolding,
		StartDate:        now,
		EndDate:          endDate,
		AutoRenew:        autoRenew,
		InterestTotal:    interestTotal,
		RedemptionAmount: redemptionAmount,
	}
	if err := s.repo.Create(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *RedemptionService) RedeemMaturity(id string) (*RedemptionRecord, error) {
	rec, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("redemption not found")
	}
	if rec.Status != StatusHolding {
		return nil, fmt.Errorf("invalid redemption state")
	}

	now := time.Now()
	product, ok := GetProduct(rec.ProductID)
	if !ok {
		return nil, fmt.Errorf("product not found")
	}
	if rec.AutoRenew {
		renewedPrincipal := rec.RedemptionAmount
		renewedEnd := now.AddDate(0, 0, product.DurationDays)
		renewedInterest := s.computeInterest(renewedPrincipal, product.APY, product.DurationDays)
		renewedAmount := renewedPrincipal + renewedInterest

		renewedRecord := &RedemptionRecord{
			UserID:           rec.UserID,
			ProductID:        rec.ProductID,
			Principal:        renewedPrincipal,
			APY:              product.APY,
			DurationDays:     product.DurationDays,
			Status:           StatusHolding,
			StartDate:        now,
			EndDate:          renewedEnd,
			AutoRenew:        rec.AutoRenew,
			InterestTotal:    renewedInterest,
			RedemptionAmount: renewedAmount,
		}
		if err := s.repo.Create(renewedRecord); err != nil {
			return nil, err
		}

		rec.Status = StatusRedeemed
		t := now
		rec.RedeemedAt = &t
		rec.RenewedToOrderID = &renewedRecord.ID
		if err := s.repo.Update(rec); err != nil {
			return nil, err
		}
		return renewedRecord, nil
	}
	rec.Status = StatusRedeemed
	t := now
	rec.RedeemedAt = &t
	if err := s.repo.Update(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *RedemptionService) GetRedemption(id string) (*RedemptionRecord, error) {
	return s.repo.Get(id)
}
