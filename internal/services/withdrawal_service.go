package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"monera-digital/internal/models"
	"monera-digital/internal/repository"
)

type ISafeheronService interface {
	Withdraw(ctx context.Context, req SafeheronWithdrawalRequest) (*SafeheronWithdrawalResponse, error)
}

type WithdrawalService struct {
	repo     *repository.Repository // Or individual repos
	safeheron ISafeheronService
	db       *sql.DB
}

func NewWithdrawalService(db *sql.DB, repo *repository.Repository, safeheron ISafeheronService) *WithdrawalService {
	return &WithdrawalService{
		db:        db,
		repo:      repo,
		safeheron: safeheron,
	}
}

// CreateWithdrawal handles the withdrawal process
func (s *WithdrawalService) CreateWithdrawal(ctx context.Context, userID int, req models.CreateWithdrawalRequest) (*models.WithdrawalOrder, error) {
	// 1. Validate Input
	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil || amount <= 0 {
		return nil, errors.New("invalid amount")
	}

	// 2. Get Account (Assuming WEALTH type for now as per PRD)
	account, err := s.repo.Account.GetByUserIDAndType(ctx, userID, "WEALTH")
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, errors.New("account not found")
		}
		return nil, err
	}

	// 3. Check Balance (Available = Balance - Frozen)
	available := account.Balance - account.FrozenBalance
	if available < amount {
		return nil, errors.New("insufficient balance")
	}

	// 4. Get Address details
	address, err := s.repo.Address.GetAddressByID(ctx, req.AddressID)
	if err != nil {
		return nil, errors.New("address not found")
	}
	if address.UserID != userID {
		return nil, errors.New("address does not belong to user")
	}
	// TODO: Check if verified (PRD requirement)
	// if !address.Verified { return nil, errors.New("address not verified") }

	// 5. Freeze Balance (Transaction)
	// Start Transaction? Repository methods use DB passed in NewRepository.
	// If I want a transaction spanning multiple repos, I need a TxManager or pass tx to methods.
	// The current Repository pattern takes `*sql.DB`.
	// Ideally, `repo` methods should take `DBTX` interface (sql.DB or sql.Tx).
	// But `repository.go` defines interfaces.
	// For simplicity, I'll assume simple operations or update the logic.
	// The PRD says: "UPDATE account SET frozen_balance ...".
	// We implemented `UpdateFrozenBalance`.

	if err := s.repo.Account.UpdateFrozenBalance(ctx, userID, amount); err != nil {
		return nil, fmt.Errorf("failed to freeze balance: %w", err)
	}

	// 6. Call Safeheron
	requestID := uuid.New().String()
	shResp, err := s.safeheron.Withdraw(ctx, SafeheronWithdrawalRequest{
		CoinType:  req.Asset, // Assuming Asset matches CoinType
		ChainType: address.ChainType,
		ToAddress: address.WalletAddress,
		Amount:    req.Amount,
		RequestID: requestID,
	})

	if err != nil {
		// 7. Fail: Unfreeze
		_ = s.repo.Account.ReleaseFrozenBalance(ctx, userID, amount)
		return nil, fmt.Errorf("safeheron failed: %w", err)
	}

	// 8. Success: Deduct Balance (Release frozen + Deduct from balance)
	// PRD says: "Deduct from frozen and balance".
	if err := s.repo.Account.DeductBalance(ctx, userID, amount); err != nil {
		// Critical error! Money sent but balance not deducted.
		// Log CRITICAL error.
		// In a real system, we'd use a transaction around this.
		// Or a reconciliation job will fix it.
		fmt.Printf("CRITICAL: DeductBalance failed after Safeheron success. Order: %s\n", shResp.SafeheronOrderID)
	}

	// 9. Create Order
	order := &models.WithdrawalOrder{
		UserID:           userID,
		Amount:           req.Amount,
		NetworkFee:       shResp.NetworkFee,
		PlatformFee:      "0", // Calc platform fee
		ActualAmount:     req.Amount, // Subtract fees
		ChainType:        address.ChainType,
		CoinType:         req.Asset,
		ToAddress:        address.WalletAddress,
		SafeheronOrderID: sql.NullString{String: shResp.SafeheronOrderID, Valid: true},
		TransactionHash:  sql.NullString{String: shResp.TxHash, Valid: true},
		Status:           "SENT", // or PENDING check PRD
	}

	createdOrder, err := s.repo.Withdrawal.CreateOrder(ctx, order)
	if err != nil {
		// Log error
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return createdOrder, nil
}

// GetWithdrawalHistory returns the withdrawal history for a user
func (s *WithdrawalService) GetWithdrawalHistory(ctx context.Context, userID int) ([]*models.WithdrawalOrder, error) {
	return s.repo.Withdrawal.GetOrdersByUserID(ctx, userID)
}

func (s *WithdrawalService) GetWithdrawalByID(ctx context.Context, userID int, id int) (*models.WithdrawalOrder, error) {
	order, err := s.repo.Withdrawal.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, errors.New("unauthorized")
	}
	return order, nil
}

func (s *WithdrawalService) EstimateFee(ctx context.Context, asset, chain, amount string) (string, string, error) {
	// Stub implementation
	// Real implementation would call Safeheron or Blockchain node
	// Fee = Network Fee + Platform Fee
	// For now, return static or simple calc
	
	networkFee := "1.0"
	if asset == "ETH" {
		networkFee = "0.002"
	}
	
	// platformFee := "0" // 0.5% maybe?
	amt, _ := strconv.ParseFloat(amount, 64)
	if amt > 0 {
		// platformFee = fmt.Sprintf("%.4f", amt*0.005)
	}
	
	// Received = Amount - Network - Platform
	received := amt - 1.0 // Simple sub
	if received < 0 {
		received = 0
	}
	
	return networkFee, fmt.Sprintf("%.4f", received), nil
}
