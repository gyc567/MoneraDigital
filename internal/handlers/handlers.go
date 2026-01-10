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
	Validator         validator.Validator
}

func NewHandler(auth *services.AuthService, lending *services.LendingService, address *services.AddressService, withdrawal *services.WithdrawalService) *Handler {
	return &Handler{
		AuthService:       auth,
		LendingService:    lending,
		AddressService:    address,
		WithdrawalService: withdrawal,
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
		User: dto.UserInfo{
			ID:    resp.User.ID,
			Email: resp.User.Email,
		},
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
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.AuthService.RefreshToken(req.RefreshToken)
	if err != nil {
		c.Error(err)
		return
	}

	dtoResp := dto.RefreshTokenResponse{
		AccessToken: resp.AccessToken,
		TokenType:   resp.TokenType,
		ExpiresIn:   resp.ExpiresIn,
		ExpiresAt:   resp.ExpiresAt,
	}

	c.JSON(http.StatusOK, dtoResp)
}

func (h *Handler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.AuthService.Logout(req.Token)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
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
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req dto.ApplyLendingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate input
	if err := h.Validator.ValidateAmount(req.Amount); err != nil {
		c.Error(err)
		return
	}
	if err := h.Validator.ValidateAsset(req.Asset); err != nil {
		c.Error(err)
		return
	}
	if err := h.Validator.ValidateDuration(req.DurationDays); err != nil {
		c.Error(err)
		return
	}

	// Convert DTO to model for service
	modelReq := models.ApplyLendingRequest{
		Asset:        req.Asset,
		Amount:       req.Amount,
		DurationDays: req.DurationDays,
	}

	position, err := h.LendingService.ApplyForLending(userID.(int), modelReq)
	if err != nil {
		c.Error(err)
		return
	}

	// Convert response to DTO
	dtoResp := dto.LendingPositionResponse{
		ID:           position.ID,
		UserID:       position.UserID,
		Asset:        position.Asset,
		Amount:       position.Amount,
		DurationDays: position.DurationDays,
		APY:          position.APY,
		Status:       position.Status,
		AccruedYield: position.AccruedYield,
		StartDate:    position.StartDate,
		EndDate:      position.EndDate,
		CreatedAt:    position.CreatedAt,
	}

	c.JSON(http.StatusCreated, dtoResp)
}

func (h *Handler) GetUserPositions(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	positions, err := h.LendingService.GetUserPositions(userID.(int))
	if err != nil {
		c.Error(err)
		return
	}

	// Convert responses to DTOs
	dtoPositions := make([]dto.LendingPositionResponse, len(positions))
	for i, pos := range positions {
		dtoPositions[i] = dto.LendingPositionResponse{
			ID:           pos.ID,
			UserID:       pos.UserID,
			Asset:        pos.Asset,
			Amount:       pos.Amount,
			DurationDays: pos.DurationDays,
			APY:          pos.APY,
			Status:       pos.Status,
			AccruedYield: pos.AccruedYield,
			StartDate:    pos.StartDate,
			EndDate:      pos.EndDate,
			CreatedAt:    pos.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, dto.LendingPositionsListResponse{
		Positions: dtoPositions,
		Total:     len(dtoPositions),
		Count:     len(dtoPositions),
	})
}

// Address handlers
func (h *Handler) GetAddresses(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// TODO: Implement GetAddresses in AddressService
	c.JSON(http.StatusOK, dto.WithdrawalAddressesListResponse{
		Addresses: []dto.WithdrawalAddressResponse{},
		Total:     0,
		Count:     0,
	})
}

func (h *Handler) AddAddress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req dto.AddAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate input
	if err := h.Validator.ValidateAddress(req.Address); err != nil {
		c.Error(err)
		return
	}
	if err := h.Validator.ValidateAsset(req.AddressType); err != nil {
		c.Error(err)
		return
	}

	// TODO: Implement AddAddress in AddressService
	c.JSON(http.StatusCreated, dto.WithdrawalAddressResponse{
		ID:        1,
		UserID:    userID.(int),
		Address:   req.Address,
		Type:      req.AddressType,
		Label:     req.Label,
		IsVerified: false,
		IsPrimary: false,
	})
}

func (h *Handler) VerifyAddress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req dto.VerifyAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Implement VerifyAddress in AddressService
	c.JSON(http.StatusOK, gin.H{"message": "Address verified"})
}

func (h *Handler) SetPrimaryAddress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req dto.SetPrimaryAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Implement SetPrimaryAddress in AddressService
	c.JSON(http.StatusOK, gin.H{"message": "Primary address set"})
}

func (h *Handler) DeactivateAddress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req dto.DeactivateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Implement DeactivateAddress in AddressService
	c.JSON(http.StatusOK, gin.H{"message": "Address deactivated"})
}

// Withdrawal handlers
func (h *Handler) GetWithdrawals(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// TODO: Implement GetWithdrawals in WithdrawalService
	c.JSON(http.StatusOK, dto.WithdrawalsListResponse{
		Withdrawals: []dto.WithdrawalResponse{},
		Total:       0,
		Count:       0,
	})
}

func (h *Handler) CreateWithdrawal(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req dto.CreateWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate input
	if err := h.Validator.ValidateAmount(req.Amount); err != nil {
		c.Error(err)
		return
	}
	if err := h.Validator.ValidateAsset(req.Asset); err != nil {
		c.Error(err)
		return
	}
	if err := h.Validator.ValidateAddress(req.ToAddress); err != nil {
		c.Error(err)
		return
	}

	// TODO: Implement CreateWithdrawal in WithdrawalService
	c.JSON(http.StatusCreated, dto.WithdrawalResponse{
		ID:            1,
		UserID:        userID.(int),
		FromAddressID: req.FromAddressID,
		Amount:        req.Amount,
		Asset:         req.Asset,
		ToAddress:     req.ToAddress,
		Status:        "pending",
	})
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

	// TODO: Implement GetWithdrawalByID in WithdrawalService
	c.JSON(http.StatusOK, dto.WithdrawalResponse{
		ID:        id,
		UserID:    userID.(int),
		Status:    "pending",
	})
}

// Docs handler
func (h *Handler) GetDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Docs endpoint"})
}