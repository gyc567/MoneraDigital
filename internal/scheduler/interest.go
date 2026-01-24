package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"monera-digital/internal/binance"
	"monera-digital/internal/logger"
	"monera-digital/internal/repository"
)

type InterestScheduler struct {
	repo         repository.Wealth
	accountRepo  repository.AccountV2
	journalRepo  repository.Journal
	priceService *binance.PriceService
	metrics      *SchedulerMetrics
}

func NewInterestScheduler(wealthRepo repository.Wealth, accountRepo repository.AccountV2, journalRepo repository.Journal) *InterestScheduler {
	return &InterestScheduler{
		repo:         wealthRepo,
		accountRepo:  accountRepo,
		journalRepo:  journalRepo,
		priceService: binance.NewPriceService(),
		metrics:      NewSchedulerMetrics(),
	}
}

func (s *InterestScheduler) Start() {
	loc := GetShanghaiLocation()

	nextMidnight := time.Now().In(loc)
	if nextMidnight.Hour() >= 0 {
		nextMidnight = nextMidnight.AddDate(0, 0, 1)
	}
	nextMidnight = time.Date(nextMidnight.Year(), nextMidnight.Month(), nextMidnight.Day(), 0, 0, 0, 0, loc)

	duration := nextMidnight.Sub(time.Now().In(loc))
	logger.Info("[InterestScheduler] First run scheduled",
		"scheduled_time", nextMidnight.Format("2006-01-02 15:04:05"),
		"delay_seconds", duration.Seconds())

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	time.Sleep(duration)

	logger.Info("[InterestScheduler] Started - running daily at 00:00 Asia/Shanghai")

	for range ticker.C {
		now := NowInShanghai()
		logger.Info("[InterestScheduler] Execution started", "timestamp", now.Format("2006-01-02 15:04:05"))

		ctx := context.Background()

		// Step 1: Calculate daily interest
		ordersProcessed, interestAccrued, err := s.CalculateDailyInterest(ctx)

		// Step 2: Settle expired orders
		settledCount, settleErr := s.SettleExpiredOrders(ctx)

		success := err == nil && settleErr == nil
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}
		if settleErr != nil {
			if errorMsg != "" {
				errorMsg += "; "
			}
			errorMsg += fmt.Sprintf("settle error: %v", settleErr)
		}

		s.metrics.RecordInterestRun(success, ordersProcessed, interestAccrued, errorMsg)

		if !success {
			logger.Error("[InterestScheduler] Execution failed", "error", errorMsg)
		} else {
			logger.Info("[InterestScheduler] Execution completed",
				"orders_processed", ordersProcessed,
				"interest_accrued", interestAccrued,
				"orders_settled", settledCount)
		}
	}
}

func (s *InterestScheduler) CalculateDailyInterest(ctx context.Context) (int, float64, error) {
	logger.Info("[InterestScheduler] Calculating daily interest...")

	today := TodayInShanghai()

	orders, err := s.repo.GetActiveOrders(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get active orders: %v", err)
	}

	logger.Info("[InterestScheduler] Found active orders", "count", len(orders))

	ordersProcessed := 0
	totalInterestAccrued := 0.0

	for _, order := range orders {
		if order.StartDate == today {
			logger.Debug("[InterestScheduler] Order skipped - start date is today",
				"order_id", order.ID, "start_date", today)
			continue
		}

		if order.LastInterestDate == today {
			logger.Debug("[InterestScheduler] Order skipped - interest already calculated today",
				"order_id", order.ID)
			continue
		}

		product, err := s.repo.GetProductByID(ctx, order.ProductID)
		if err != nil {
			logger.Error("[InterestScheduler] Failed to get product",
				"order_id", order.ID, "error", err.Error())
			continue
		}

		apy, _ := strconv.ParseFloat(product.APY, 64)
		amount, _ := strconv.ParseFloat(order.Amount, 64)

		dailyInterest := amount * (apy / 100) / 365

		err = s.repo.AccrueInterest(ctx, order.ID, strconv.FormatFloat(dailyInterest, 'f', -1, 64), today)
		if err != nil {
			logger.Error("[InterestScheduler] Failed to accrue interest",
				"order_id", order.ID, "error", err.Error())
			continue
		}

		ordersProcessed++
		totalInterestAccrued += dailyInterest

		logger.Info("[InterestScheduler] Interest accrued",
			"order_id", order.ID,
			"interest", dailyInterest,
			"currency", order.Currency,
			"apy", apy,
			"amount", amount)
	}

	logger.Info("[InterestScheduler] Daily interest calculation completed",
		"orders_processed", ordersProcessed,
		"total_interest", totalInterestAccrued)

	return ordersProcessed, totalInterestAccrued, nil
}

func (s *InterestScheduler) SettleOrder(ctx context.Context, orderID int64) error {
	logger.Info("[InterestScheduler] Settling order", "order_id", orderID)

	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %v", err)
	}

	if order.Status != 1 {
		return fmt.Errorf("order status is not active: %d", order.Status)
	}

	account, err := s.accountRepo.GetAccountByUserIDAndCurrency(ctx, order.UserID, order.Currency)
	if err != nil {
		return fmt.Errorf("failed to get account: %v", err)
	}

	now := time.Now()

	// Step 1: Unfreeze principal
	err = s.accountRepo.UnfreezeBalance(ctx, account.ID, order.Amount)
	if err != nil {
		return fmt.Errorf("failed to unfreeze balance: %v", err)
	}

	// Generate journal record for principal unfreeze
	balance, _ := strconv.ParseFloat(account.Balance, 64)
	principalAmt, _ := strconv.ParseFloat(order.Amount, 64)
	newBalance := balance + principalAmt

	principalJournal := &repository.JournalModel{
		SerialNo:        fmt.Sprintf("SETTLE-PRINCIPAL-%s-%d", now.Format("20060102150405"), order.ID),
		UserID:          order.UserID,
		AccountID:       account.ID,
		Amount:          order.Amount,
		BalanceSnapshot: strconv.FormatFloat(newBalance, 'f', -1, 64),
		BizType:         "REDEEM_UNFREEZE",
		RefID:           &order.ID,
		CreatedAt:       now.Format(time.RFC3339),
	}
	err = s.journalRepo.CreateJournalRecord(ctx, principalJournal)
	if err != nil {
		logger.Error("[InterestScheduler] Failed to create principal journal record",
			"order_id", orderID, "error", err.Error())
	}

	// Step 2: Pay interest if accrued
	interestPaid, _ := strconv.ParseFloat(order.InterestAccrued, 64)
	if interestPaid > 0 {
		err = s.accountRepo.AddBalance(ctx, account.ID, strconv.FormatFloat(interestPaid, 'f', -1, 64))
		if err != nil {
			return fmt.Errorf("failed to add interest to balance: %v", err)
		}

		// Generate journal record for interest payout
		balanceAfterInterest := newBalance + interestPaid
		interestJournal := &repository.JournalModel{
			SerialNo:        fmt.Sprintf("SETTLE-INTEREST-%s-%d", now.Format("20060102150405"), order.ID),
			UserID:          order.UserID,
			AccountID:       account.ID,
			Amount:          strconv.FormatFloat(interestPaid, 'f', -1, 64),
			BalanceSnapshot: strconv.FormatFloat(balanceAfterInterest, 'f', -1, 64),
			BizType:         "INTEREST_PAYOUT",
			RefID:           &order.ID,
			CreatedAt:       now.Format(time.RFC3339),
		}
		err = s.journalRepo.CreateJournalRecord(ctx, interestJournal)
		if err != nil {
			logger.Error("[InterestScheduler] Failed to create interest journal record",
				"order_id", orderID, "error", err.Error())
		}
	}

	// Step 3: Update order status
	err = s.repo.SettleOrder(ctx, orderID, strconv.FormatFloat(interestPaid, 'f', -1, 64))
	if err != nil {
		return fmt.Errorf("failed to settle order: %v", err)
	}

	logger.Info("[InterestScheduler] Order settled",
		"order_id", orderID,
		"amount_unfrozen", order.Amount,
		"currency", order.Currency,
		"interest_paid", interestPaid)

	return nil
}

// SettleExpiredOrders Find and settle all orders that have expired
func (s *InterestScheduler) SettleExpiredOrders(ctx context.Context) (int, error) {
	today := TodayInShanghai()

	logger.Info("[InterestScheduler] Settling expired orders", "date", today)

	orders, err := s.repo.GetExpiredOrders(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get expired orders: %v", err)
	}

	settledCount := 0
	renewedCount := 0

	for _, order := range orders {
		if order.AutoRenew {
			err = s.RenewOrder(ctx, order)
			if err != nil {
				logger.Error("[InterestScheduler] Failed to renew order",
					"order_id", order.ID, "error", err.Error())
				continue
			}
			renewedCount++
			logger.Info("[InterestScheduler] Order auto-renewed",
				"order_id", order.ID,
				"user_id", order.UserID,
				"amount", order.Amount,
				"currency", order.Currency)
		} else {
			err = s.SettleOrder(ctx, order.ID)
			if err != nil {
				logger.Error("[InterestScheduler] Failed to settle order",
					"order_id", order.ID, "error", err.Error())
				continue
			}
			settledCount++
			logger.Info("[InterestScheduler] Order auto-settled",
				"order_id", order.ID,
				"user_id", order.UserID,
				"amount", order.Amount,
				"currency", order.Currency)
		}
	}

	logger.Info("[InterestScheduler] Expired orders settlement completed",
		"settled_count", settledCount,
		"renewed_count", renewedCount)
	return settledCount + renewedCount, nil
}

// RenewOrder Auto-renew an expired order
func (s *InterestScheduler) RenewOrder(ctx context.Context, order *repository.WealthOrderModel) error {
	logger.Info("[InterestScheduler] Renewing order", "order_id", order.ID)

	// Get product info
	product, err := s.repo.GetProductByID(ctx, order.ProductID)
	if err != nil {
		return fmt.Errorf("failed to get product: %v", err)
	}

	// Check if product is still available
	if product.Status != 1 {
		logger.Warn("[InterestScheduler] Product not available for renewal, settling normally",
			"order_id", order.ID, "product_id", product.ID)
		return s.SettleOrder(ctx, order.ID)
	}

	// Check if auto-renew is still allowed
	if !product.AutoRenewAllowed {
		logger.Warn("[InterestScheduler] Auto-renew not allowed for product, settling normally",
			"order_id", order.ID, "product_id", product.ID)
		return s.SettleOrder(ctx, order.ID)
	}

	// Get user account
	account, err := s.accountRepo.GetAccountByUserIDAndCurrency(ctx, order.UserID, order.Currency)
	if err != nil {
		return fmt.Errorf("failed to get account: %v", err)
	}

	now := time.Now()

	// Step 1: Pay interest from old order
	interestPaid, _ := strconv.ParseFloat(order.InterestAccrued, 64)
	if interestPaid > 0 {
		err = s.accountRepo.AddBalance(ctx, account.ID, strconv.FormatFloat(interestPaid, 'f', -1, 64))
		if err != nil {
			return fmt.Errorf("failed to add interest: %v", err)
		}

		// Generate journal record for interest payout
		balance, _ := strconv.ParseFloat(account.Balance, 64)
		interestJournal := &repository.JournalModel{
			SerialNo:        fmt.Sprintf("RENEW-INTEREST-%s-%d", now.Format("20060102150405"), order.ID),
			UserID:          order.UserID,
			AccountID:       account.ID,
			Amount:          strconv.FormatFloat(interestPaid, 'f', -1, 64),
			BalanceSnapshot: strconv.FormatFloat(balance+interestPaid, 'f', -1, 64),
			BizType:         "INTEREST_PAYOUT",
			RefID:           &order.ID,
			CreatedAt:       now.Format(time.RFC3339),
		}
		err = s.journalRepo.CreateJournalRecord(ctx, interestJournal)
		if err != nil {
			logger.Error("[InterestScheduler] Failed to create interest journal record",
				"order_id", order.ID, "error", err.Error())
		}
	}

	// Step 2: Create new order (principal stays frozen)
	newOrder, err := s.repo.RenewOrder(ctx, order, product)
	if err != nil {
		return fmt.Errorf("failed to create renewed order: %v", err)
	}

	// Generate journal record for new subscription
	balanceAfterInterest, _ := strconv.ParseFloat(account.Balance, 64)
	subscribeJournal := &repository.JournalModel{
		SerialNo:        fmt.Sprintf("RENEW-SUBSCRIBE-%s-%d", now.Format("20060102150405"), newOrder.ID),
		UserID:          order.UserID,
		AccountID:       account.ID,
		Amount:          "-" + newOrder.Amount,
		BalanceSnapshot: strconv.FormatFloat(balanceAfterInterest-float64(interestPaid), 'f', -1, 64),
		BizType:         "SUBSCRIBE_FREEZE",
		RefID:           &newOrder.ID,
		CreatedAt:       now.Format(time.RFC3339),
	}
	err = s.journalRepo.CreateJournalRecord(ctx, subscribeJournal)
	if err != nil {
		logger.Error("[InterestScheduler] Failed to create subscription journal record",
			"order_id", newOrder.ID, "error", err.Error())
	}

	// Step 3: Update old order status
	err = s.repo.SettleOrder(ctx, order.ID, strconv.FormatFloat(interestPaid, 'f', -1, 64))
	if err != nil {
		logger.Error("[InterestScheduler] Failed to update old order status",
			"order_id", order.ID, "error", err.Error())
	}

	logger.Info("[InterestScheduler] Order renewed successfully",
		"old_order_id", order.ID,
		"new_order_id", newOrder.ID,
		"amount", order.Amount,
		"currency", order.Currency,
		"interest_paid", interestPaid)

	return nil
}

func (s *InterestScheduler) GetMetrics() *SchedulerMetrics {
	return s.metrics
}
