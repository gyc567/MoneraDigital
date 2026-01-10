// internal/dto/lending.go
package dto

import "time"

// ApplyLendingRequest DTO for lending application
type ApplyLendingRequest struct {
	Asset        string  `json:"asset" binding:"required,oneof=BTC ETH USDC USDT"`
	Amount       float64 `json:"amount" binding:"required,gt=0"`
	DurationDays int     `json:"duration_days" binding:"required,gt=0,lte=365"`
}

// LendingPositionResponse DTO for lending position response
type LendingPositionResponse struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	Asset        string    `json:"asset"`
	Amount       float64   `json:"amount"`
	DurationDays int       `json:"duration_days"`
	APY          float64   `json:"apy"`
	Status       string    `json:"status"`
	AccruedYield float64   `json:"accrued_yield"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	CreatedAt    time.Time `json:"created_at"`
}

// LendingPositionsListResponse DTO for list of lending positions
type LendingPositionsListResponse struct {
	Positions []LendingPositionResponse `json:"positions"`
	Total     int                       `json:"total"`
	Count     int                       `json:"count"`
}

// CloseLendingRequest DTO for closing a lending position
type CloseLendingRequest struct {
	PositionID int `json:"position_id" binding:"required,gt=0"`
}
