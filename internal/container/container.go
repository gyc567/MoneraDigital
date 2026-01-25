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

// ContainerOption 配置选项函数
type ContainerOption func(*Container)

// WithEncryption 配置加密服务和 2FA 服务
func WithEncryption(key string) ContainerOption {
	return func(c *Container) {
		// Normalize encryption key (support hex-encoded or raw format)
		normalizedKey, err := services.DecodeEncryptionKey(key)
		if err != nil {
			log.Printf("Warning: Invalid encryption key format: %v", err)
			return
		}

		encryptionService, err := services.NewEncryptionService(normalizedKey)
		if err != nil {
			log.Printf("Warning: Failed to initialize encryption service: %v", err)
			return
		}
		c.EncryptionService = encryptionService
		c.TwoFAService = services.NewTwoFactorService(c.DB, encryptionService)
	}
}

// Container 依赖注入容器
type Container struct {
	// 基础设施
	DB *sql.DB

	// 配置
	JWTSecret string

	// 缓存
	TokenBlacklist *cache.TokenBlacklist
	RateLimiter    *middleware.RateLimiter

	// 仓储
	Repository *repository.Repository

	// 服务
	AuthService       *services.AuthService
	LendingService    *services.LendingService
	AddressService    *services.AddressService
	WithdrawalService *services.WithdrawalService
	DepositService    *services.DepositService
	WalletService     *services.WalletService
	WealthService     *services.WealthService
	EncryptionService *services.EncryptionService
	TwoFAService      *services.TwoFactorService

	// 中间件
	RateLimitMiddleware *middleware.PerEndpointRateLimiter
}

// NewContainer 创建依赖注入容器
func NewContainer(db *sql.DB, jwtSecret string, opts ...ContainerOption) *Container {
	c := &Container{DB: db, JWTSecret: jwtSecret}

	// 初始化缓存
	c.TokenBlacklist = cache.NewTokenBlacklist()
	c.RateLimiter = middleware.NewRateLimiter(5, 60)

	// 初始化仓储
	c.Repository = &repository.Repository{
		User:       postgres.NewUserRepository(db),
		Deposit:    postgres.NewDepositRepository(db),
		Wallet:     postgres.NewWalletRepository(db),
		Account:    postgres.NewAccountRepositoryV1(db), // Legacy interface
		AccountV2:  postgres.NewAccountRepository(db),   // New detailed interface
		Address:    postgres.NewAddressRepository(db),
		Withdrawal: postgres.NewWithdrawalRepository(db),
	}

	// 初始化核心服务
	c.AuthService = services.NewAuthService(db, jwtSecret)
	c.AuthService.SetTokenBlacklist(c.TokenBlacklist)

	// 注入TwoFactorService依赖（如果已初始化）
	if c.TwoFAService != nil {
		c.AuthService.SetTwoFactorService(c.TwoFAService)
	}

	c.LendingService = services.NewLendingService(db)
	c.AddressService = services.NewAddressService(c.Repository.Address)
	c.WithdrawalService = services.NewWithdrawalService(db, c.Repository, services.NewSafeheronService())
	c.DepositService = services.NewDepositService(c.Repository.Deposit)
	c.WalletService = services.NewWalletService(c.Repository.Wallet)
	c.WealthService = services.NewWealthService(c.Repository.Wealth, c.Repository.AccountV2, c.Repository.Journal)

	// 应用配置选项 (按顺序执行)
	for _, opt := range opts {
		opt(c)
	}

	// 初始化中间件
	c.RateLimitMiddleware = middleware.NewPerEndpointRateLimiter()
	c.RateLimitMiddleware.AddEndpoint("/api/auth/register", 5, 60)
	c.RateLimitMiddleware.AddEndpoint("/api/auth/login", 5, 60)
	c.RateLimitMiddleware.AddEndpoint("/api/auth/refresh", 10, 60)

	return c
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

	// 验证核心服务初始化
	services := []struct {
		name  string
		value interface{}
	}{
		{"AuthService", c.AuthService},
		{"LendingService", c.LendingService},
		{"AddressService", c.AddressService},
		{"WithdrawalService", c.WithdrawalService},
		{"DepositService", c.DepositService},
		{"WalletService", c.WalletService},
	}

	for _, s := range services {
		if s.value == nil {
			log.Printf("%s not initialized", s.name)
		}
	}

	log.Println("Container verification passed")
	return nil
}
