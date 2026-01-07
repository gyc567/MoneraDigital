package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"monera-digital/internal/config"
	"monera-digital/internal/db"
	"monera-digital/internal/handlers"
	"monera-digital/internal/middleware"
	"monera-digital/internal/services"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	database, err := db.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize services
	authService := services.NewAuthService(database)
	lendingService := services.NewLendingService(database)
	addressService := services.NewAddressService(database)
	withdrawalService := services.NewWithdrawalService(database)

	// Initialize handlers
	h := handlers.NewHandler(authService, lendingService, addressService, withdrawalService)

	// Initialize Gin router
	r := gin.Default()

	// Add middleware
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())

	// Auth routes
	authGroup := r.Group("/api/auth")
	{
		authGroup.POST("/login", h.Login)
		authGroup.POST("/register", h.Register)
		authGroup.GET("/me", middleware.AuthRequired(cfg.JWTSecret), h.GetMe)
		authGroup.POST("/2fa/setup", middleware.AuthRequired(cfg.JWTSecret), h.Setup2FA)
		authGroup.POST("/2fa/enable", middleware.AuthRequired(cfg.JWTSecret), h.Enable2FA)
		authGroup.POST("/2fa/verify-login", h.Verify2FALogin)
	}

	// Lending routes
	lendingGroup := r.Group("/api/lending")
	lendingGroup.Use(middleware.AuthRequired(cfg.JWTSecret))
	{
		lendingGroup.POST("/apply", h.ApplyForLending)
		lendingGroup.GET("/positions", h.GetUserPositions)
	}

	// Addresses routes
	addressesGroup := r.Group("/api/addresses")
	addressesGroup.Use(middleware.AuthRequired(cfg.JWTSecret))
	{
		addressesGroup.GET("", h.GetAddresses)
		addressesGroup.POST("", h.AddAddress)
		addressesGroup.POST("/:id/verify", h.VerifyAddress)
		addressesGroup.POST("/:id/primary", h.SetPrimaryAddress)
		addressesGroup.DELETE("/:id", h.DeactivateAddress)
	}

	// Withdrawals routes
	withdrawalsGroup := r.Group("/api/withdrawals")
	withdrawalsGroup.Use(middleware.AuthRequired(cfg.JWTSecret))
	{
		withdrawalsGroup.GET("", h.GetWithdrawals)
		withdrawalsGroup.POST("", h.CreateWithdrawal)
		withdrawalsGroup.GET("/:id", h.GetWithdrawalByID)
	}

	// Docs route
	r.GET("/api/docs", h.GetDocs)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	r.Run(":" + cfg.Port)
}