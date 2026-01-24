package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"monera-digital/internal/binance"
	"monera-digital/internal/repository"
)

var (
	ErrInsufficientBalance   = errors.New("insufficient balance")
	ErrProductNotFound       = errors.New("product not found")
	ErrOrderNotFound         = errors.New("order not found")
	ErrProductNotAvailable   = errors.New("product not available")
	ErrAmountBelowMin        = errors.New("amount below minimum")
	ErrAmountAboveMax        = errors.New("amount above maximum")
	ErrQuotaExceeded         = errors.New("quota exceeded")
	ErrOrderAlreadyRedeemed  = errors.New("order already redeemed")
	ErrInvalidRedemptionType = errors.New("invalid redemption type")
	ErrPriceFetchFailed      = errors.New("failed to fetch price")
)

type WealthService struct {
	repo        repository.Wealth
	accountRepo repository.AccountV2
	journalRepo repository.Journal
}

func NewWealthService(wealthRepo repository.Wealth, accountRepo repository.AccountV2, journalRepo repository.Journal) *WealthService {
	return &WealthService{
		repo:        wealthRepo,
		accountRepo: accountRepo,
		journalRepo: journalRepo,
	}
}

type Asset struct {
	Currency      string  `json:"currency"`
	Total         string  `json:"total"`
	Available     string  `json:"available"`
	FrozenBalance string  `json:"frozenBalance"`
	UsdValue      float64 `json:"usdValue"`
}

func (s *WealthService) GetAssets(ctx context.Context, userID int) ([]*Asset, error) {
	accounts, err := s.accountRepo.GetAccountsByUserID(ctx, int64(userID))
	if err != nil {
		return nil, err
	}

	var currencies []string
	for _, a := range accounts {
		if a.Currency != "USDT" && a.Currency != "USDC" && a.Currency != "DAI" {
			currencies = append(currencies, a.Currency)
		}
	}
	prices := binance.NewPriceService().GetPricesFromCache(currencies)

	var result []*Asset
	for _, a := range accounts {
		available := subtractStrings(a.Balance, a.FrozenBalance)
		availableFloat, _ := strconv.ParseFloat(available, 64)
		usdValue := availableFloat

		if a.Currency == "USDT" || a.Currency == "USDC" || a.Currency == "DAI" {
			usdValue = availableFloat
		} else if price, ok := prices[a.Currency]; ok {
			usdValue = availableFloat * price
		}

		result = append(result, &Asset{
			Currency:      a.Currency,
			Total:         a.Balance,
			Available:     available,
			FrozenBalance: a.FrozenBalance,
			UsdValue:      usdValue,
		})
	}
	return result, nil
}

func subtractStrings(a, b string) string {
	aFloat, errA := strconv.ParseFloat(a, 64)
	bFloat, errB := strconv.ParseFloat(b, 64)

	if errA != nil {
		return a
	}
	if errB != nil {
		return "0"
	}

	diff := aFloat - bFloat
	if diff < 0 {
		diff = 0
	}

	return strconv.FormatFloat(diff, 'f', -1, 64)
}

type Product struct {
	ID               int64   `json:"id"`
	Title            string  `json:"title"`
	Currency         string  `json:"currency"`
	APY              float64 `json:"apy"`
	Duration         int     `json:"duration"`
	MinAmount        string  `json:"minAmount"`
	MaxAmount        string  `json:"maxAmount"`
	RemainingQuota   string  `json:"remainingQuota"`
	AutoRenewAllowed bool    `json:"autoRenewAllowed"`
}

type Order struct {
	ID               int64  `json:"id"`
	ProductTitle     string `json:"productTitle"`
	Currency         string `json:"currency"`
	Amount           string `json:"amount"`
	InterestExpected string `json:"interestExpected"`
	InterestPaid     string `json:"interestPaid"`
	InterestAccrued  string `json:"interestAccrued"`
	StartDate        string `json:"startDate"`
	EndDate          string `json:"endDate"`
	AutoRenew        bool   `json:"autoRenew"`
	Status           int    `json:"status"`
	RedemptionAmount string `json:"redemptionAmount,omitempty"`
	CreatedAt        string `json:"createdAt"`
}

func (s *WealthService) GetProducts(ctx context.Context, page, pageSize int) ([]*Product, int64, error) {
	products, err := s.repo.GetActiveProducts(ctx)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(products))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(products) {
		return []*Product{}, total, nil
	}
	if end > len(products) {
		end = len(products)
	}

	var result []*Product
	for _, p := range products[start:end] {
		apy, _ := strconv.ParseFloat(p.APY, 64)
		result = append(result, &Product{
			ID:               p.ID,
			Title:            p.Title,
			Currency:         p.Currency,
			APY:              apy,
			Duration:         p.Duration,
			MinAmount:        p.MinAmount,
			MaxAmount:        p.MaxAmount,
			RemainingQuota:   p.TotalQuota,
			AutoRenewAllowed: p.AutoRenewAllowed,
		})
	}
	return result, total, nil
}

func (s *WealthService) Subscribe(ctx context.Context, userID int, productID int64, amount string, autoRenew bool) (string, error) {
	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return "", ErrProductNotFound
	}

	if product.Status != 1 {
		return "", ErrProductNotFound
	}

	available, _ := strconv.ParseFloat(amount, 64)
	if available <= 0 {
		return "", ErrAmountBelowMin
	}

	minAmount, _ := strconv.ParseFloat(product.MinAmount, 64)
	if available < minAmount {
		return "", ErrAmountBelowMin
	}

	maxAmount, _ := strconv.ParseFloat(product.MaxAmount, 64)
	if available > maxAmount {
		return "", ErrAmountAboveMax
	}

	account, err := s.accountRepo.GetAccountByUserIDAndCurrency(ctx, int64(userID), product.Currency)
	if err != nil {
		return "", ErrInsufficientBalance
	}

	balance, _ := strconv.ParseFloat(account.Balance, 64)
	frozen, _ := strconv.ParseFloat(account.FrozenBalance, 64)
	availableBalance := balance - frozen
	if available > availableBalance {
		return "", ErrInsufficientBalance
	}

	err = s.accountRepo.FreezeBalance(ctx, account.ID, amount)
	if err != nil {
		return "", err
	}

	now := time.Now()
	startDate := now.Format("2006-01-02")
	endDate := now.AddDate(0, 0, product.Duration).Format("2006-01-02")

	apy, _ := strconv.ParseFloat(product.APY, 64)
	amountFloat, _ := strconv.ParseFloat(amount, 64)
	dailyInterest := amountFloat * (apy / 100) / 365
	interestExpected := strconv.FormatFloat(dailyInterest*float64(product.Duration), 'f', -1, 64)

	order := &repository.WealthOrderModel{
		UserID:            int64(userID),
		ProductID:         productID,
		ProductTitle:      product.Title,
		Currency:          product.Currency,
		Amount:            amount,
		AutoRenew:         autoRenew,
		Status:            1,
		StartDate:         startDate,
		EndDate:           endDate,
		PrincipalRedeemed: "0",
		InterestExpected:  interestExpected,
		InterestPaid:      "0",
		InterestAccrued:   "0",
		LastInterestDate:  "",
		CreatedAt:         now.Format(time.RFC3339),
		UpdatedAt:         now.Format(time.RFC3339),
	}

	err = s.repo.CreateOrder(ctx, order)
	if err != nil {
		s.accountRepo.UnfreezeBalance(ctx, account.ID, amount)
		return "", err
	}

	serialNo := fmt.Sprintf("SUBSCRIBE-%s-%d", now.Format("20060102150405"), order.ID)
	balanceAfterFreeze := balance - available
	journalRecord := &repository.JournalModel{
		SerialNo:        serialNo,
		UserID:          int64(userID),
		AccountID:       account.ID,
		Amount:          "-" + amount,
		BalanceSnapshot: strconv.FormatFloat(balanceAfterFreeze, 'f', -1, 64),
		BizType:         "SUBSCRIBE_FREEZE",
		RefID:           &order.ID,
		CreatedAt:       now.Format(time.RFC3339),
	}

	err = s.journalRepo.CreateJournalRecord(ctx, journalRecord)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create journal record: %v\n", err)
	}

	return interestExpected, nil
}

func (s *WealthService) GetOrders(ctx context.Context, userID int, page, pageSize int) ([]*Order, int64, error) {
	orders, err := s.repo.GetOrdersByUserID(ctx, int64(userID))
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(orders))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(orders) {
		return []*Order{}, total, nil
	}
	if end > len(orders) {
		end = len(orders)
	}

	var result []*Order
	for _, o := range orders[start:end] {
		result = append(result, &Order{
			ID:               o.ID,
			ProductTitle:     o.ProductTitle,
			Currency:         o.Currency,
			Amount:           o.Amount,
			InterestExpected: o.InterestExpected,
			InterestPaid:     o.InterestPaid,
			InterestAccrued:  o.InterestAccrued,
			StartDate:        o.StartDate,
			EndDate:          o.EndDate,
			AutoRenew:        o.AutoRenew,
			Status:           o.Status,
			CreatedAt:        o.CreatedAt,
		})
	}
	return result, total, nil
}

func (s *WealthService) Redeem(ctx context.Context, userID int, orderID int64, redemptionType string) error {
	fmt.Printf("[DEBUG] Redeem - userID: %d, orderID: %d, redemptionType: %s\n", userID, orderID, redemptionType)
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		fmt.Printf("[DEBUG] Redeem - GetOrderByID error: %v\n", err)
		return ErrOrderNotFound
	}
	fmt.Printf("[DEBUG] Redeem - order found: ID=%d, UserID=%d, Status=%d, Amount=%s\n", order.ID, order.UserID, order.Status, order.Amount)

	if order.UserID != int64(userID) {
		return ErrOrderNotFound
	}

	if order.Status == 3 {
		return ErrOrderAlreadyRedeemed
	}

	now := time.Now()
	endDate, _ := time.Parse("2006-01-02", order.EndDate)
	isExpired := now.After(endDate) || now.Equal(endDate)

	if !isExpired {
		fmt.Printf("[DEBUG] Early redemption for order %d - only unfreezing principal\n", order.ID)
	}

	isFull := redemptionType == "full"
	if isFull {
		account, err := s.accountRepo.GetAccountByUserIDAndCurrency(ctx, int64(userID), order.Currency)
		if err != nil {
			return err
		}

		err = s.accountRepo.UnfreezeBalance(ctx, account.ID, order.Amount)
		if err != nil {
			return err
		}

		order.Status = 3
		order.RedemptionAmount = order.Amount
		order.RedeemedAt = now.Format(time.RFC3339)

		if isExpired {
			interestAccrued, _ := strconv.ParseFloat(order.InterestAccrued, 64)
			if interestAccrued > 0 {
				err = s.accountRepo.AddBalance(ctx, account.ID, order.InterestAccrued)
				if err != nil {
					return err
				}

				balance, _ := strconv.ParseFloat(account.Balance, 64)
				newBalance := balance + interestAccrued

				journalRecord := &repository.JournalModel{
					SerialNo:        fmt.Sprintf("REDEEM-PRINCIPAL-%s-%d", now.Format("20060102150405"), order.ID),
					UserID:          int64(userID),
					AccountID:       account.ID,
					Amount:          order.Amount,
					BalanceSnapshot: strconv.FormatFloat(newBalance, 'f', -1, 64),
					BizType:         "REDEEM_UNFREEZE",
					RefID:           &order.ID,
					CreatedAt:       now.Format(time.RFC3339),
				}
				err = s.journalRepo.CreateJournalRecord(ctx, journalRecord)
				if err != nil {
					fmt.Printf("[ERROR] Failed to create principal journal record: %v\n", err)
				}

				interestJournalRecord := &repository.JournalModel{
					SerialNo:        fmt.Sprintf("REDEEM-INTEREST-%s-%d", now.Format("20060102150405"), order.ID),
					UserID:          int64(userID),
					AccountID:       account.ID,
					Amount:          order.InterestAccrued,
					BalanceSnapshot: strconv.FormatFloat(newBalance+interestAccrued, 'f', -1, 64),
					BizType:         "INTEREST_PAYOUT",
					RefID:           &order.ID,
					CreatedAt:       now.Format(time.RFC3339),
				}
				err = s.journalRepo.CreateJournalRecord(ctx, interestJournalRecord)
				if err != nil {
					fmt.Printf("[ERROR] Failed to create interest journal record: %v\n", err)
				}

				order.InterestPaid = order.InterestAccrued
				order.InterestAccrued = "0"
				fmt.Printf("[DEBUG] Paid interest %.8f %s for order %d\n", interestAccrued, order.Currency, order.ID)
			} else {
				balance, _ := strconv.ParseFloat(account.Balance, 64)
				newBalance := balance + interestAccrued
				journalRecord := &repository.JournalModel{
					SerialNo:        fmt.Sprintf("REDEEM-PRINCIPAL-%s-%d", now.Format("20060102150405"), order.ID),
					UserID:          int64(userID),
					AccountID:       account.ID,
					Amount:          order.Amount,
					BalanceSnapshot: strconv.FormatFloat(newBalance, 'f', -1, 64),
					BizType:         "REDEEM_UNFREEZE",
					RefID:           &order.ID,
					CreatedAt:       now.Format(time.RFC3339),
				}
				err = s.journalRepo.CreateJournalRecord(ctx, journalRecord)
				if err != nil {
					fmt.Printf("[ERROR] Failed to create principal journal record: %v\n", err)
				}
			}
		} else {
			balance, _ := strconv.ParseFloat(account.Balance, 64)
			principalAmt, _ := strconv.ParseFloat(order.Amount, 64)
			newBalance := balance + principalAmt
			journalRecord := &repository.JournalModel{
				SerialNo:        fmt.Sprintf("REDEEM-PRINCIPAL-%s-%d", now.Format("20060102150405"), order.ID),
				UserID:          int64(userID),
				AccountID:       account.ID,
				Amount:          order.Amount,
				BalanceSnapshot: strconv.FormatFloat(newBalance, 'f', -1, 64),
				BizType:         "REDEEM_UNFREEZE",
				RefID:           &order.ID,
				CreatedAt:       now.Format(time.RFC3339),
			}
			err = s.journalRepo.CreateJournalRecord(ctx, journalRecord)
			if err != nil {
				fmt.Printf("[ERROR] Failed to create principal journal record: %v\n", err)
			}
		}
	}

	return s.repo.UpdateOrder(ctx, order)
}
