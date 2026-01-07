package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"monera-digital/internal/config"
	"monera-digital/internal/handlers"
	"monera-digital/internal/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize Gin router
	r := gin.Default()

	// Add middleware
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())

	// Auth routes
	authGroup := r.Group("/api/auth")
	{
		authGroup.POST("/login", handlers.Login)
		authGroup.POST("/register", handlers.Register)
		authGroup.GET("/me", middleware.AuthRequired(), handlers.GetMe)
		authGroup.POST("/2fa/setup", middleware.AuthRequired(), handlers.Setup2FA)
		authGroup.POST("/2fa/enable", middleware.AuthRequired(), handlers.Enable2FA)
		authGroup.POST("/2fa/verify-login", handlers.Verify2FALogin)
	}

	// Lending routes
	lendingGroup := r.Group("/api/lending")
	lendingGroup.Use(middleware.AuthRequired())
	{
		lendingGroup.POST("/apply", handlers.ApplyForLending)
		lendingGroup.GET("/positions", handlers.GetUserPositions)
	}

	// Addresses routes
	addressesGroup := r.Group("/api/addresses")
	addressesGroup.Use(middleware.AuthRequired())
	{
		addressesGroup.GET("", handlers.GetAddresses)
		addressesGroup.POST("", handlers.AddAddress)
		addressesGroup.POST("/:id/verify", handlers.VerifyAddress)
		addressesGroup.POST("/:id/primary", handlers.SetPrimaryAddress)
		addressesGroup.DELETE("/:id", handlers.DeactivateAddress)
	}

	// Withdrawals routes
	withdrawalsGroup := r.Group("/api/withdrawals")
	withdrawalsGroup.Use(middleware.AuthRequired())
	{
		withdrawalsGroup.GET("", handlers.GetWithdrawals)
		withdrawalsGroup.POST("", handlers.CreateWithdrawal)
		withdrawalsGroup.GET("/:id", handlers.GetWithdrawalByID)
	}

	// Docs route
	r.GET("/api/docs", handlers.GetDocs)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	r.Run(":" + cfg.Port)
}
