// internal/dto/withdrawal.go
package dto

import "time"

// CreateWithdrawalRequest DTO for creating a withdrawal
type CreateWithdrawalRequest struct {
	FromAddressID int     `json:"fromAddressId" binding:"required,gt=0"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Asset         string  `json:"asset" binding:"required,oneof=BTC ETH USDC USDT"`
	ToAddress     string  `json:"toAddress" binding:"required,min=20,max=100"`
}

// WithdrawalResponse DTO for withdrawal response
type WithdrawalResponse struct {
	ID            int        `json:"id"`
	UserID        int        `json:"userId"`
	FromAddressID int        `json:"fromAddressId"`
	Amount        float64    `json:"amount"`
	Asset         string     `json:"asset"`
	ToAddress     string     `json:"toAddress"`
	Status        string     `json:"status"`
	TxHash        *string    `json:"txHash,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	FailureReason *string    `json:"failureReason,omitempty"`
}

// WithdrawalsListResponse DTO for list of withdrawals
type WithdrawalsListResponse struct {
	Withdrawals []WithdrawalResponse `json:"withdrawals"`
	Total       int                  `json:"total"`
	Count       int                  `json:"count"`
}

// CancelWithdrawalRequest DTO for canceling a withdrawal
type CancelWithdrawalRequest struct {
	WithdrawalID int `json:"withdrawalId" binding:"required,gt=0"`
}
