package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"monera-digital/internal/models"
	"monera-digital/internal/services"
)

var (
	authService    = &services.AuthService{}
	lendingService = &services.LendingService{}
)

// Auth handlers
func Login(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Login endpoint"})
}

func Register(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Register endpoint"})
}

func GetMe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get me endpoint"})
}

func Setup2FA(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Setup 2FA endpoint"})
}

func Enable2FA(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Enable 2FA endpoint"})
}

func Verify2FALogin(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Verify 2FA login endpoint"})
}

// Lending handlers
func ApplyForLending(c *gin.Context) {
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

	position, err := lendingService.ApplyForLending(userID.(int), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply for lending"})
		return
	}

	c.JSON(http.StatusCreated, position)
}

func GetUserPositions(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	positions, err := lendingService.GetUserPositions(userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get positions"})
		return
	}

	c.JSON(http.StatusOK, positions)
}

// Address handlers
func GetAddresses(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get addresses endpoint"})
}

func AddAddress(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Add address endpoint"})
}

func VerifyAddress(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Verify address endpoint"})
}

func SetPrimaryAddress(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Set primary address endpoint"})
}

func DeactivateAddress(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Deactivate address endpoint"})
}

// Withdrawal handlers
func GetWithdrawals(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get withdrawals endpoint"})
}

func CreateWithdrawal(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Create withdrawal endpoint"})
}

func GetWithdrawalByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Get withdrawal by ID endpoint", "id": id})
}

// Docs handler
func GetDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Docs endpoint"})
}
