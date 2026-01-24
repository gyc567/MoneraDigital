// internal/routes/routes.go
package routes

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"monera-digital/internal/account"
	"monera-digital/internal/container"
	"monera-digital/internal/docs"
	"monera-digital/internal/handlers"
	"monera-digital/internal/handlers/core"
	"monera-digital/internal/middleware"
)

// SetupRoutes configures all API routes with middleware
func SetupRoutes(router *gin.Engine, cont *container.Container) {
	// Add global middleware
	router.Use(middleware.RecoveryHandler())
	router.Use(middleware.ErrorHandler())
	router.Use(middleware.RateLimitMiddleware(cont.RateLimiter))

	// Initialize Swagger documentation
	docs.NewSwagger()

	// Swagger documentation endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Create handler
	h := handlers.NewHandler(
		cont.AuthService,
		cont.LendingService,
		cont.AddressService,
		cont.WithdrawalService,
		cont.DepositService,
		cont.WalletService,
		cont.WealthService,
	)

	// Create 2FA handler
	twofaHandler := handlers.NewTwoFAHandler(cont.TwoFAService)

	// Account System Client
	accountClient := account.NewClient("http://localhost:8081") // TODO: Use config
	accountHandler := &handlers.AccountHandler{Client: accountClient}

	// Public routes
	public := router.Group("/api")
	{
		auth := public.Group("/auth")
		{
			auth.POST("/register", h.Register)
			auth.POST("/login", h.Login)
			auth.POST("/refresh", h.RefreshToken)
			auth.POST("/logout", h.Logout)
		}

		webhooks := public.Group("/webhooks")
		{
			webhooks.POST("/core/deposit", h.HandleDepositWebhook)
		}

		// Account System Routes
		accounts := public.Group("/accounts")
		{
			accounts.GET("", accountHandler.GetUserAccounts)
			accounts.POST("", accountHandler.CreateAccount)
			accounts.GET("/history", accountHandler.GetAccountHistory)
			accounts.POST("/freeze", accountHandler.FreezeBalance)
			accounts.POST("/unfreeze", accountHandler.UnfreezeBalance)
			accounts.POST("/transfer", accountHandler.Transfer)
		}
	}

	// Protected routes
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware(cont.JWTSecret))
	{
		auth := protected.Group("/auth")
		{
			auth.GET("/me", h.GetMe)
			twofa := auth.Group("/2fa")
			{
				twofa.POST("/setup", twofaHandler.Setup2FA)
				twofa.POST("/enable", twofaHandler.Enable2FA)
				twofa.POST("/verify", twofaHandler.Verify2FA)
				twofa.POST("/disable", twofaHandler.Disable2FA)
				twofa.GET("/status", twofaHandler.Get2FAStatus)
			}
		}

		lending := protected.Group("/lending")
		{
			lending.POST("/apply", h.ApplyForLending)
			lending.GET("/positions", h.GetUserPositions)
		}

		wallet := protected.Group("/wallet")
		{
			wallet.POST("/create", h.CreateWallet)
			wallet.GET("/info", h.GetWalletInfo)
		}

		deposits := protected.Group("/deposits")
		{
			deposits.GET("", h.GetDeposits)
		}

		addresses := protected.Group("/addresses")
		{
			addresses.GET("", h.GetAddresses)
			addresses.POST("", h.AddAddress)
			addresses.POST("/:id/verify", h.VerifyAddress)
			addresses.POST("/:id/set-primary", h.SetPrimaryAddress)
			addresses.POST("/:id/deactivate", h.DeactivateAddress)
		}

		withdrawals := protected.Group("/withdrawals")
		{
			withdrawals.GET("", h.GetWithdrawals)
			withdrawals.POST("", h.CreateWithdrawal)
			withdrawals.GET("/fees", h.GetWithdrawalFees)
			withdrawals.GET("/:id", h.GetWithdrawalByID)
		}

		assets := protected.Group("/assets")
		{
			assets.GET("", h.GetAssets)
			assets.POST("/refresh-prices", h.RefreshPrices)
		}

		wealth := protected.Group("/wealth")
		{
			wealth.GET("/products", h.GetProducts)
			wealth.POST("/subscribe", h.Subscribe)
			wealth.GET("/orders", h.GetOrders)
			wealth.POST("/redeem", h.Redeem)
		}
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Core Account System Mock API
	core.SetupRoutes(router)
}
