// internal/container/container.go
package container

import (
	"database/sql"
	"log"

	"monera-digital/internal/cache"
	"monera-digital/internal/middleware"
	"monera-digital/internal/repository"
	"monera-digital/internal/repository/postgres"
	"monera-digital/internal/services"
)

// Container 依赖注入容器
type Container struct {
	// 基础设施
	DB *sql.DB

	// 缓存
	TokenBlacklist *cache.TokenBlacklist
	RateLimiter    *middleware.RateLimiter

	// 仓储
	Repository *repository.Repository

	// 服务

	AuthService *services.AuthService

	LendingService *services.LendingService

	AddressService *services.AddressService

	WithdrawalService *services.WithdrawalService

	DepositService *services.DepositService

	WalletService *services.WalletService

	// 中间件

	RateLimitMiddleware *middleware.PerEndpointRateLimiter
}

// NewContainer 创建依赖注入容器

func NewContainer(db *sql.DB, jwtSecret string) *Container {

	// 初始化缓存

	tokenBlacklist := cache.NewTokenBlacklist()

	rateLimiter := middleware.NewRateLimiter(5, 60) // 5 请求/分钟

	// 初始化仓储
	repo := &repository.Repository{
		User:       postgres.NewUserRepository(db),
		Deposit:    postgres.NewDepositRepository(db),
		Wallet:     postgres.NewWalletRepository(db),
		Account:    postgres.NewAccountRepository(db),
		Address:    postgres.NewAddressRepository(db),
		Withdrawal: postgres.NewWithdrawalRepository(db),
		// Lending:    postgres.NewLendingRepository(db),
	}

	// 初始化服务
	authService := services.NewAuthService(db, jwtSecret)
	authService.SetTokenBlacklist(tokenBlacklist)

	lendingService := services.NewLendingService(db)

	addressService := services.NewAddressService(repo.Address)

	safeheronService := services.NewSafeheronService()

	withdrawalService := services.NewWithdrawalService(db, repo, safeheronService)

	depositService := services.NewDepositService(repo.Deposit)

	walletService := services.NewWalletService(repo.Wallet)

	// 初始化中间件

	rateLimitMiddleware := middleware.NewPerEndpointRateLimiter()

	rateLimitMiddleware.AddEndpoint("/api/auth/register", 5, 60) // 5 请求/分钟

	rateLimitMiddleware.AddEndpoint("/api/auth/login", 5, 60) // 5 请求/分钟

	rateLimitMiddleware.AddEndpoint("/api/auth/refresh", 10, 60) // 10 请求/分钟

	return &Container{

		DB: db,

		TokenBlacklist: tokenBlacklist,

		RateLimiter: rateLimiter,

		Repository: repo,

		AuthService: authService,

		LendingService: lendingService,

		AddressService: addressService,

		WithdrawalService: withdrawalService,

		DepositService: depositService,

		WalletService: walletService,

		RateLimitMiddleware: rateLimitMiddleware,
	}

}

// Close 关闭容器中的资源

func (c *Container) Close() error {

	if c.TokenBlacklist != nil {

		c.TokenBlacklist.Close()

	}

	if c.DB != nil {

		return c.DB.Close()

	}

	return nil

}

// Verify 验证容器中的所有依赖

func (c *Container) Verify() error {

	// 验证数据库连接

	if err := c.DB.Ping(); err != nil {

		log.Printf("Database connection failed: %v", err)

		return err

	}

	// 验证服务初始化

	if c.AuthService == nil {

		return log.New(nil, "", 0).Output(0, "AuthService not initialized")

	}

	if c.LendingService == nil {

		return log.New(nil, "", 0).Output(0, "LendingService not initialized")

	}

	if c.AddressService == nil {

		return log.New(nil, "", 0).Output(0, "AddressService not initialized")

	}

	if c.WithdrawalService == nil {

		return log.New(nil, "", 0).Output(0, "WithdrawalService not initialized")

	}

	if c.DepositService == nil {

		return log.New(nil, "", 0).Output(0, "DepositService not initialized")

	}

	if c.WalletService == nil {

		return log.New(nil, "", 0).Output(0, "WalletService not initialized")

	}

	log.Println("Container verification passed")

	return nil

}
