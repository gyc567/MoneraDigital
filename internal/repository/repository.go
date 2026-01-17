// internal/repository/repository.go
package repository

import (
	"context"
	"errors"
	"monera-digital/internal/models"
)

// User 用户仓储接口
type User interface {
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id int) (*models.User, error)
	Create(ctx context.Context, email, passwordHash string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id int) error
}

// Account 账户仓储接口
type Account interface {
	GetByUserIDAndType(ctx context.Context, userID int, accountType string) (*models.Account, error)
	Create(ctx context.Context, account *models.Account) error
	UpdateFrozenBalance(ctx context.Context, userID int, amount float64) error // Add to frozen
	ReleaseFrozenBalance(ctx context.Context, userID int, amount float64) error // Remove from frozen
	DeductBalance(ctx context.Context, userID int, amount float64) error // Deduct from balance and frozen
}

// Lending 借贷仓储接口
type Lending interface {
	CreatePosition(ctx context.Context, position *models.LendingPosition) (*models.LendingPosition, error)
	GetPositionsByUserID(ctx context.Context, userID int) ([]*models.LendingPosition, error)
	GetPositionByID(ctx context.Context, id int) (*models.LendingPosition, error)
	UpdatePosition(ctx context.Context, position *models.LendingPosition) error
}

// Address 地址仓储接口
type Address interface {
	CreateAddress(ctx context.Context, address *models.WithdrawalAddress) (*models.WithdrawalAddress, error)
	GetAddressesByUserID(ctx context.Context, userID int) ([]*models.WithdrawalAddress, error)
	GetAddressByID(ctx context.Context, id int) (*models.WithdrawalAddress, error)
	UpdateAddress(ctx context.Context, address *models.WithdrawalAddress) error
	DeleteAddress(ctx context.Context, id int) error
	GetByAddressAndChain(ctx context.Context, address, chain string) (*models.WithdrawalAddress, error)
}

// Withdrawal 提现仓储接口
type Withdrawal interface {
	CreateOrder(ctx context.Context, order *models.WithdrawalOrder) (*models.WithdrawalOrder, error)
	GetOrdersByUserID(ctx context.Context, userID int) ([]*models.WithdrawalOrder, error)
	GetOrderByID(ctx context.Context, id int) (*models.WithdrawalOrder, error)
	UpdateOrder(ctx context.Context, order *models.WithdrawalOrder) error
	CreateRequest(ctx context.Context, request *models.WithdrawalRequest) error
	GetRequestByID(ctx context.Context, requestID string) (*models.WithdrawalRequest, error)
}

// Deposit 充值仓储接口
type Deposit interface {
	Create(ctx context.Context, deposit *models.Deposit) error
	GetByTxHash(ctx context.Context, txHash string) (*models.Deposit, error)
	GetByUserID(ctx context.Context, userID int, limit, offset int) ([]*models.Deposit, int64, error)
	UpdateStatus(ctx context.Context, id int, status string, confirmedAt string) error
}

// Wallet 钱包仓储接口
type Wallet interface {
	CreateRequest(ctx context.Context, req *models.WalletCreationRequest) error
	GetRequestByUserID(ctx context.Context, userID int) (*models.WalletCreationRequest, error)
	UpdateRequest(ctx context.Context, req *models.WalletCreationRequest) error
	GetActiveWalletByUserID(ctx context.Context, userID int) (*models.WalletCreationRequest, error)
}

// Repository 仓储容器
type Repository struct {
	User       User
	Account    Account
	Lending    Lending
	Address    Address
	Withdrawal Withdrawal
	Deposit    Deposit
	Wallet     Wallet
}

// Common errors
var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrInsufficientBalance = errors.New("insufficient balance")
)
