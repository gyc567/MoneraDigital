// internal/repository/repository.go
package repository

import (
	"context"
	"errors"
)

// User 用户仓储接口
type User interface {
	// GetByEmail 根据邮箱获取用户
	GetByEmail(ctx context.Context, email string) (*UserModel, error)

	// GetByID 根据ID获取用户
	GetByID(ctx context.Context, id int) (*UserModel, error)

	// Create 创建用户
	Create(ctx context.Context, email, passwordHash string) (*UserModel, error)

	// Update 更新用户
	Update(ctx context.Context, user *UserModel) error

	// Delete 删除用户
	Delete(ctx context.Context, id int) error
}

// UserModel 用户模型
type UserModel struct {
	ID                   int
	Email                string
	Password             string
	TwoFactorEnabled     bool
	TwoFactorSecret      string
	TwoFactorBackupCodes string
	CreatedAt            string
	UpdatedAt            string
}

// Lending 借贷仓储接口
type Lending interface {
	// CreatePosition 创建借贷头寸
	CreatePosition(ctx context.Context, position *LendingPositionModel) (*LendingPositionModel, error)

	// GetPositionsByUserID 获取用户的借贷头寸
	GetPositionsByUserID(ctx context.Context, userID int) ([]*LendingPositionModel, error)

	// GetPositionByID 根据ID获取借贷头寸
	GetPositionByID(ctx context.Context, id int) (*LendingPositionModel, error)

	// UpdatePosition 更新借贷头寸
	UpdatePosition(ctx context.Context, position *LendingPositionModel) error
}

// LendingPositionModel 借贷头寸模型
type LendingPositionModel struct {
	ID           int
	UserID       int
	Asset        string
	Amount       string
	DurationDays int
	APY          string
	AccruedYield string
	Status       string
	StartDate    string
	EndDate      string
	CreatedAt    string
}

// Address 地址仓储接口
type Address interface {
	// CreateAddress 创建地址
	CreateAddress(ctx context.Context, address *WithdrawalAddressModel) (*WithdrawalAddressModel, error)

	// GetAddressesByUserID 获取用户的地址
	GetAddressesByUserID(ctx context.Context, userID int) ([]*WithdrawalAddressModel, error)

	// GetAddressByID 根据ID获取地址
	GetAddressByID(ctx context.Context, id int) (*WithdrawalAddressModel, error)

	// UpdateAddress 更新地址
	UpdateAddress(ctx context.Context, address *WithdrawalAddressModel) error

	// DeleteAddress 删除地址
	DeleteAddress(ctx context.Context, id int) error
}

// WithdrawalAddressModel 提现地址模型
type WithdrawalAddressModel struct {
	ID            int
	UserID        int
	Address       string
	AddressType   string
	Label         string
	IsVerified    bool
	IsPrimary     bool
	CreatedAt     string
	VerifiedAt    string
	DeactivatedAt string
}

// Withdrawal 提现仓储接口
type Withdrawal interface {
	// CreateWithdrawal 创建提现
	CreateWithdrawal(ctx context.Context, withdrawal *WithdrawalModel) (*WithdrawalModel, error)

	// GetWithdrawalsByUserID 获取用户的提现
	GetWithdrawalsByUserID(ctx context.Context, userID int) ([]*WithdrawalModel, error)

	// GetWithdrawalByID 根据ID获取提现
	GetWithdrawalByID(ctx context.Context, id int) (*WithdrawalModel, error)

	// UpdateWithdrawal 更新提现
	UpdateWithdrawal(ctx context.Context, withdrawal *WithdrawalModel) error
}

// WithdrawalModel 提现模型
type WithdrawalModel struct {
	ID            int
	UserID        int
	FromAddressID int
	Amount        string
	Asset         string
	ToAddress     string
	Status        string
	TxHash        string
	CreatedAt     string
	CompletedAt   string
	FailureReason string
}

// Repository 仓储容器
type Repository struct {
	User       User
	Lending    Lending
	Address    Address
	Withdrawal Withdrawal
}

// Common errors
var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
	ErrInvalidInput  = errors.New("invalid input")
)
