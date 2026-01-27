package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"monera-digital/internal/dto"
	"monera-digital/internal/models"
	"monera-digital/internal/services"
	"monera-digital/internal/validator"
)

const (
	minDurationDays = 30
	maxDurationDays = 360
)

var supportedAssets = map[string]bool{
	"BTC":  true,
	"ETH":  true,
	"USDT": true,
	"USDC": true,
	"SOL":  true,
}

// Handler contains all HTTP handlers
type Handler struct {
	AuthService       *services.AuthService
	LendingService    *services.LendingService
	AddressService    *services.AddressService
	WithdrawalService *services.WithdrawalService
	DepositService    *services.DepositService
	WalletService     *services.WalletService
	WealthService     *services.WealthService
	Validator         validator.Validator
}

// Wealth handlers
func (h *Handler) GetAssets(c *gin.Context) {
	// TODO: Implement wealth GetAssets handler
	c.JSON(200, gin.H{"message": "GetAssets not implemented yet"})
}

func (h *Handler) RefreshPrices(c *gin.Context) {
	// TODO: Implement wealth RefreshPrices handler
	c.JSON(200, gin.H{"message": "RefreshPrices not implemented yet"})
}

func (h *Handler) GetProducts(c *gin.Context) {
	// TODO: Implement wealth GetProducts handler
	c.JSON(200, gin.H{"message": "GetProducts not implemented yet"})
}

func (h *Handler) Subscribe(c *gin.Context) {
	// TODO: Implement wealth Subscribe handler
	c.JSON(200, gin.H{"message": "Subscribe not implemented yet"})
}

func (h *Handler) GetOrders(c *gin.Context) {
	// TODO: Implement wealth GetOrders handler
	c.JSON(200, gin.H{"message": "GetOrders not implemented yet"})
}

func (h *Handler) Redeem(c *gin.Context) {
	// TODO: Implement wealth Redeem handler
	c.JSON(200, gin.H{"message": "Redeem not implemented yet"})
}

func NewHandler(auth *services.AuthService, lending *services.LendingService, address *services.AddressService, withdrawal *services.WithdrawalService, deposit *services.DepositService, wallet *services.WalletService, wealth *services.WealthService) *Handler {
	return &Handler{
		AuthService:       auth,
		LendingService:    lending,
		AddressService:    address,
		WithdrawalService: withdrawal,
		DepositService:    deposit,
		WalletService:     wallet,
		WealthService:     wealth,
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

	if err := h.Validator.ValidateEmail(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.AuthService.Login(models.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

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
			ID:               resp.User.ID,
			Email:            resp.User.Email,
			TwoFactorEnabled: resp.User.TwoFactorEnabled,
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

	if err := h.Validator.ValidateEmail(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Validator.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.AuthService.Register(models.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.UserInfo{
		ID:               user.ID,
		Email:            user.Email,
		TwoFactorEnabled: user.TwoFactorEnabled,
	})
}

func (h *Handler) GetMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := h.AuthService.GetUserByID(userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user profile"})
		return
	}

	c.JSON(http.StatusOK, dto.UserInfo{
		ID:               user.ID,
		Email:            user.Email,
		TwoFactorEnabled: user.TwoFactorEnabled,
	})
}

// Verify2FALogin verifies 2FA token during login and completes authentication
func (h *Handler) Verify2FALogin(c *gin.Context) {
	var req models.Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.AuthService.Verify2FAAndLogin(req.UserID, req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	dtoResp := dto.LoginResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		ExpiresIn:    resp.ExpiresIn,
		ExpiresAt:    resp.ExpiresAt,
		Token:        resp.Token,
		User: &dto.UserInfo{
			ID:               resp.User.ID,
			Email:            resp.User.Email,
			TwoFactorEnabled: resp.User.TwoFactorEnabled,
		},
	}

	c.JSON(http.StatusOK, dtoResp)
}

func (h *Handler) RefreshToken(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Token refresh not yet implemented"})
}

func (h *Handler) Logout(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Logout not yet implemented"})
}

// Lending handlers

func (h *Handler) ApplyForLending(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req dto.ApplyLendingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be positive"})
		return
	}
	if req.DurationDays < minDurationDays || req.DurationDays > maxDurationDays {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Duration must be between %d and %d days", minDurationDays, maxDurationDays)})
		return
	}
	if !supportedAssets[req.Asset] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported asset. Supported assets: BTC, ETH, USDT, USDC, SOL"})
		return
	}

	position, err := h.LendingService.ApplyForLending(userID, models.ApplyLendingRequest{
		Asset:        req.Asset,
		Amount:       fmt.Sprintf("%.8f", req.Amount),
		DurationDays: req.DurationDays,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toLendingPositionResponse(position))
}

func (h *Handler) GetUserPositions(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	positions, err := h.LendingService.GetUserPositions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	dtoPositions := make([]dto.LendingPositionResponse, len(positions))
	for i, pos := range positions {
		dtoPositions[i] = toLendingPositionResponse(&pos)
	}

	total := 0.0
	for _, pos := range dtoPositions {
		total += pos.AccruedYield
	}

	c.JSON(http.StatusOK, dto.LendingPositionsListResponse{
		Positions: dtoPositions,
		Total:     int(total),
		Count:     len(dtoPositions),
	})
}

// Address handlers

func (h *Handler) GetAddresses(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	addresses, err := h.AddressService.GetAddresses(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"addresses": addresses, "total": len(addresses), "count": len(addresses)})
}

func (h *Handler) AddAddress(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req models.AddAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	addr, err := h.AddressService.AddAddress(c.Request.Context(), userID, req)
	if err != nil {
		if err.Error() == "address already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "Address already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, addr)
}

func (h *Handler) VerifyAddress(c *gin.Context) {
	if _, err := h.getUserID(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Address verification triggered"})
}

func (h *Handler) SetPrimaryAddress(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

func (h *Handler) DeactivateAddress(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.AddressService.DeleteAddress(c.Request.Context(), userID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Address deleted"})
}

// Withdrawal handlers

func (h *Handler) GetWithdrawals(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	orders, err := h.WithdrawalService.GetWithdrawalHistory(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"withdrawals": orders, "total": len(orders), "count": len(orders)})
}

func (h *Handler) CreateWithdrawal(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req models.CreateWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.WithdrawalService.CreateWithdrawal(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Withdrawal created", "order": order})
}

func (h *Handler) GetWithdrawalByID(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	order, err := h.WithdrawalService.GetWithdrawalByID(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, errors.New("unauthorized")) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *Handler) GetWithdrawalFees(c *gin.Context) {
	fee, received, err := h.WithdrawalService.EstimateFee(c.Request.Context(), c.Query("asset"), c.Query("chain"), c.Query("amount"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"fee": fee, "receivedAmount": received})
}

func (h *Handler) GetDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Docs endpoint"})
}

// Helper functions

func (h *Handler) getUserID(c *gin.Context) (int, error) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, errors.New("Unauthorized")
	}
	return userID.(int), nil
}

func toLendingPositionResponse(position *models.LendingPosition) dto.LendingPositionResponse {
	apy, _ := strconv.ParseFloat(position.Apy, 64)
	amount, _ := strconv.ParseFloat(position.Amount, 64)
	accruedYield, _ := strconv.ParseFloat(position.AccruedYield, 64)

	return dto.LendingPositionResponse{
		ID:           position.ID,
		UserID:       position.UserID,
		Asset:        position.Asset,
		Amount:       amount,
		DurationDays: position.DurationDays,
		APY:          apy,
		Status:       string(position.Status),
		AccruedYield: accruedYield,
		StartDate:    position.StartDate,
		EndDate:      position.EndDate,
		CreatedAt:    position.StartDate,
	}
}
