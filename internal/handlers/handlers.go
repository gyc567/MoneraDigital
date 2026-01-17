package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"monera-digital/internal/dto"
	"monera-digital/internal/models"
	"monera-digital/internal/services"
	"monera-digital/internal/validator"
)

type Handler struct {
	AuthService       *services.AuthService
	LendingService    *services.LendingService
	AddressService    *services.AddressService
	WithdrawalService *services.WithdrawalService
	DepositService    *services.DepositService
	WalletService     *services.WalletService
	Validator         validator.Validator
}

func NewHandler(auth *services.AuthService, lending *services.LendingService, address *services.AddressService, withdrawal *services.WithdrawalService, deposit *services.DepositService, wallet *services.WalletService) *Handler {
	return &Handler{
		AuthService:       auth,
		LendingService:    lending,
		AddressService:    address,
		WithdrawalService: withdrawal,
		DepositService:    deposit,
		WalletService:     wallet,
		Validator:         validator.NewValidator(),
	}
}

// Auth handlers
func (h *Handler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate input
	if err := h.Validator.ValidateEmail(req.Email); err != nil {
		c.Error(err)
		return
	}

	// Convert DTO to model for service
	modelReq := models.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	resp, err := h.AuthService.Login(modelReq)
	if err != nil {
		c.Error(err)
		return
	}

	// Convert response to DTO
	dtoResp := dto.LoginResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		ExpiresIn:    resp.ExpiresIn,
		ExpiresAt:    resp.ExpiresAt,
		Token:        resp.Token,
		Requires2FA:  resp.Requires2FA,
		UserID:       resp.UserID,
	}

	if resp.User != nil {
		dtoResp.User = &dto.UserInfo{
			ID:    resp.User.ID,
			Email: resp.User.Email,
		}
	}

	c.JSON(http.StatusOK, dtoResp)
}

func (h *Handler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate input
	if err := h.Validator.ValidateEmail(req.Email); err != nil {
		c.Error(err)
		return
	}
	if err := h.Validator.ValidatePassword(req.Password); err != nil {
		c.Error(err)
		return
	}

	// Convert DTO to model for service
	modelReq := models.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	user, err := h.AuthService.Register(modelReq)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, dto.UserInfo{
		ID:    user.ID,
		Email: user.Email,
	})
}

func (h *Handler) GetMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	email, _ := c.Get("email")
	c.JSON(http.StatusOK, dto.UserInfo{
		ID:    userID.(int),
		Email: email.(string),
	})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	// Temporarily disabled - not implemented in AuthService
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Token refresh not yet implemented"})
}

func (h *Handler) Logout(c *gin.Context) {
	// Temporarily disabled - not implemented in AuthService
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Logout not yet implemented"})
}

func (h *Handler) Setup2FA(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Setup 2FA endpoint"})
}

func (h *Handler) Enable2FA(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Enable 2FA endpoint"})
}

func (h *Handler) Verify2FALogin(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Verify 2FA login endpoint"})
}

// Lending handlers
func (h *Handler) ApplyForLending(c *gin.Context) {
	// Temporarily simplified - not fully implemented
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Apply for lending endpoint", "user_id": userID})
}

func (h *Handler) GetUserPositions(c *gin.Context) {
	// Temporarily simplified - not fully implemented
	c.JSON(http.StatusOK, gin.H{"positions": []interface{}{}, "total": 0, "count": 0})
}

// Address handlers
func (h *Handler) GetAddresses(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	addresses, err := h.AddressService.GetAddresses(c.Request.Context(), userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"addresses": addresses, "total": len(addresses), "count": len(addresses)})
}

func (h *Handler) AddAddress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.AddAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	addr, err := h.AddressService.AddAddress(c.Request.Context(), userID.(int), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, addr)
}

func (h *Handler) VerifyAddress(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	// Simplified verification (mock)
	c.JSON(http.StatusOK, gin.H{"message": "Address verification triggered"})
}

func (h *Handler) SetPrimaryAddress(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

func (h *Handler) DeactivateAddress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.AddressService.DeleteAddress(c.Request.Context(), userID.(int), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Address deleted"})
}

// Withdrawal handlers
func (h *Handler) GetWithdrawals(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orders, err := h.WithdrawalService.GetWithdrawalHistory(c.Request.Context(), userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"withdrawals": orders, "total": len(orders), "count": len(orders)})
}

func (h *Handler) CreateWithdrawal(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.WithdrawalService.CreateWithdrawal(c.Request.Context(), userID.(int), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Withdrawal created", "order": order})
}

func (h *Handler) GetWithdrawalByID(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	order, err := h.WithdrawalService.GetWithdrawalByID(c.Request.Context(), userID.(int), id)
	if err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *Handler) GetWithdrawalFees(c *gin.Context) {
	asset := c.Query("asset")
	chain := c.Query("chain")
	amount := c.Query("amount")

	fee, received, err := h.WithdrawalService.EstimateFee(c.Request.Context(), asset, chain, amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"fee": fee, "receivedAmount": received})
}

// Docs handler
func (h *Handler) GetDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Docs endpoint"})
}
