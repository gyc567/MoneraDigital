package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"monera-digital/internal/models"
	"monera-digital/internal/services"
)

type Handler struct {
	AuthService       *services.AuthService
	LendingService    *services.LendingService
	AddressService    *services.AddressService
	WithdrawalService *services.WithdrawalService
}

func NewHandler(auth *services.AuthService, lending *services.LendingService, address *services.AddressService, withdrawal *services.WithdrawalService) *Handler {
	return &Handler{
		AuthService:       auth,
		LendingService:    lending,
		AddressService:    address,
		WithdrawalService: withdrawal,
	}
}

// Auth handlers
func (h *Handler) Login(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Login endpoint"})
}

func (h *Handler) Register(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Register endpoint"})
}

func (h *Handler) GetMe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get me endpoint"})
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

	var req models.ApplyLendingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	position, err := h.LendingService.ApplyForLending(userID.(int), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply for lending"})
		return
	}

	c.JSON(http.StatusCreated, position)
}

func (h *Handler) GetUserPositions(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	positions, err := h.LendingService.GetUserPositions(userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get positions"})
		return
	}

	c.JSON(http.StatusOK, positions)
}

// Address handlers
func (h *Handler) GetAddresses(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get addresses endpoint"})
}

func (h *Handler) AddAddress(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Add address endpoint"})
}

func (h *Handler) VerifyAddress(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Verify address endpoint"})
}

func (h *Handler) SetPrimaryAddress(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Set primary address endpoint"})
}

func (h *Handler) DeactivateAddress(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Deactivate address endpoint"})
}

// Withdrawal handlers
func (h *Handler) GetWithdrawals(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get withdrawals endpoint"})
}

func (h *Handler) CreateWithdrawal(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Create withdrawal endpoint"})
}

func (h *Handler) GetWithdrawalByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Get withdrawal by ID endpoint", "id": id})
}

// Docs handler
func (h *Handler) GetDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Docs endpoint"})
}