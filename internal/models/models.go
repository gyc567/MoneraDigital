package models

import (
	"database/sql"
	"time"
)

// Enums
type LendingStatus string

const (
	LendingStatusActive     LendingStatus = "ACTIVE"
	LendingStatusCompleted  LendingStatus = "COMPLETED"
	LendingStatusTerminated LendingStatus = "TERMINATED"
)

type AddressType string

const (
	AddressTypeBTC  AddressType = "BTC"
	AddressTypeETH  AddressType = "ETH"
	AddressTypeUSDC AddressType = "USDC"
	AddressTypeUSDT AddressType = "USDT"
)

type WithdrawalStatus string

const (
	WithdrawalStatusPending    WithdrawalStatus = "PENDING"
	WithdrawalStatusProcessing WithdrawalStatus = "PROCESSING"
	WithdrawalStatusCompleted  WithdrawalStatus = "COMPLETED"
	WithdrawalStatusFailed     WithdrawalStatus = "FAILED"
)

// User model
type User struct {
	ID                   int            `json:"id" db:"id"`
	Email                string         `json:"email" db:"email"`
	Password             string         `json:"-" db:"password"`
	TwoFactorSecret      sql.NullString `json:"-" db:"two_factor_secret"`
	TwoFactorEnabled     bool           `json:"two_factor_enabled" db:"two_factor_enabled"`
	TwoFactorBackupCodes sql.NullString `json:"-" db:"two_factor_backup_codes"`
	CreatedAt            time.Time      `json:"created_at" db:"created_at"`
}

// LendingPosition model
type LendingPosition struct {
	ID           int           `json:"id" db:"id"`
	UserID       int           `json:"user_id" db:"user_id"`
	Asset        string        `json:"asset" db:"asset"`
	Amount       string        `json:"amount" db:"amount"` // Using string for decimal precision
	DurationDays int           `json:"duration_days" db:"duration_days"`
	Apy          string        `json:"apy" db:"apy"`
	Status       LendingStatus `json:"status" db:"status"`
	AccruedYield string        `json:"accrued_yield" db:"accrued_yield"`
	StartDate    time.Time     `json:"start_date" db:"start_date"`
	EndDate      time.Time     `json:"end_date" db:"end_date"`
}

// WithdrawalAddress model
type WithdrawalAddress struct {
	ID            int          `json:"id" db:"id"`
	UserID        int          `json:"user_id" db:"user_id"`
	Address       string       `json:"address" db:"address"`
	AddressType   AddressType  `json:"address_type" db:"address_type"`
	Label         string       `json:"label" db:"label"`
	IsVerified    bool         `json:"is_verified" db:"is_verified"`
	IsPrimary     bool         `json:"is_primary" db:"is_primary"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	VerifiedAt    sql.NullTime `json:"verified_at" db:"verified_at"`
	DeactivatedAt sql.NullTime `json:"deactivated_at" db:"deactivated_at"`
}

// AddressVerification model
type AddressVerification struct {
	ID         int          `json:"id" db:"id"`
	AddressID  int          `json:"address_id" db:"address_id"`
	Token      string       `json:"token" db:"token"`
	ExpiresAt  time.Time    `json:"expires_at" db:"expires_at"`
	VerifiedAt sql.NullTime `json:"verified_at" db:"verified_at"`
}

// Withdrawal model
type Withdrawal struct {
	ID            int              `json:"id" db:"id"`
	UserID        int              `json:"user_id" db:"user_id"`
	FromAddressID int              `json:"from_address_id" db:"from_address_id"`
	Amount        string           `json:"amount" db:"amount"`
	Asset         string           `json:"asset" db:"asset"`
	ToAddress     string           `json:"to_address" db:"to_address"`
	Status        WithdrawalStatus `json:"status" db:"status"`
	TxHash        sql.NullString   `json:"tx_hash" db:"tx_hash"`
	CreatedAt     time.Time        `json:"created_at" db:"created_at"`
	CompletedAt   sql.NullTime     `json:"completed_at" db:"completed_at"`
	FailureReason sql.NullString   `json:"failure_reason" db:"failure_reason"`
}

// Request/Response structs for API
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type ApplyLendingRequest struct {
	Asset        string `json:"asset" binding:"required"`
	Amount       string `json:"amount" binding:"required"`
	DurationDays int    `json:"duration_days" binding:"required,min=1"`
}

type AddAddressRequest struct {
	Address     string      `json:"address" binding:"required"`
	AddressType AddressType `json:"address_type" binding:"required,oneof=BTC ETH USDC USDT"`
	Label       string      `json:"label" binding:"required"`
}

type CreateWithdrawalRequest struct {
	AddressID int    `json:"address_id" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
	Asset     string `json:"asset" binding:"required"`
}

type Verify2FARequest struct {
	UserID int    `json:"user_id" binding:"required"`
	Token  string `json:"token" binding:"required,len=6"`
}
