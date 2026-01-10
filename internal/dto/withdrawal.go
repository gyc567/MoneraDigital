// internal/dto/withdrawal.go
package dto

import "time"

// CreateWithdrawalRequest DTO for creating a withdrawal
type CreateWithdrawalRequest struct {
	FromAddressID int     `json:"from_address_id" binding:"required,gt=0"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Asset         string  `json:"asset" binding:"required,oneof=BTC ETH USDC USDT"`
	ToAddress     string  `json:"to_address" binding:"required,min=20,max=100"`
}

// WithdrawalResponse DTO for withdrawal response
type WithdrawalResponse struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	FromAddressID int       `json:"from_address_id"`
	Amount        float64   `json:"amount"`
	Asset         string    `json:"asset"`
	ToAddress     string    `json:"to_address"`
	Status        string    `json:"status"`
	TxHash        *string   `json:"tx_hash,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	FailureReason *string   `json:"failure_reason,omitempty"`
}

// WithdrawalsListResponse DTO for list of withdrawals
type WithdrawalsListResponse struct {
	Withdrawals []WithdrawalResponse `json:"withdrawals"`
	Total       int                  `json:"total"`
	Count       int                  `json:"count"`
}

// CancelWithdrawalRequest DTO for canceling a withdrawal
type CancelWithdrawalRequest struct {
	WithdrawalID int `json:"withdrawal_id" binding:"required,gt=0"`
}
