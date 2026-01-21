package handlers

import (
	"net/http"
	"strconv"

	"monera-digital/internal/account"
	"monera-digital/internal/dto"
	"monera-digital/internal/logger"

	"github.com/gin-gonic/gin"
)

// AccountHandler handles requests for the external account system.
// @title Account System API
// @description API for interacting with the external account system
// @version 1.0
// @host api.monera-digital.com
// @basePath /api
// @schemes https
type AccountHandler struct {
	Client *account.Client
}

// NewAccountHandler creates a new AccountHandler.
// @Summary Get user accounts
// @Description Retrieves all accounts for a given user
// @Tags accounts
// @Accept json
// @Produce json
// @Param userId query string true "User ID"
// @Success 200 {object} dto.GetUserAccountsResponse
// @Router /accounts [get]
func (h *AccountHandler) GetUserAccounts(c *gin.Context) {
	// Parse userId from query
	userID := c.Query("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing userId parameter"})
		return
	}

	resp, err := h.Client.GetUserAccounts(c.Request.Context(), dto.GetUserAccountsRequest{UserID: userID})
	if err != nil {
		logger.Error("failed to get user accounts", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// CreateAccount handles POST /accounts
// @Summary Create a new account
// @Description Creates a new account for a user
// @Tags accounts
// @Accept json
// @Produce json
// @Param request body dto.CreateAccountRequest true "Create Account Request"
// @Success 200 {object} dto.CreateAccountResponse
// @Router /accounts [post]
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req dto.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	resp, err := h.Client.CreateAccount(c.Request.Context(), req)
	if err != nil {
		logger.Error("failed to create account", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetAccountHistory handles GET /accounts/history
// @Summary Get account history
// @Description Retrieves transaction history for an account
// @Tags accounts
// @Accept json
// @Produce json
// @Param accountId query string true "Account ID"
// @Param userId query string true "User ID"
// @Param currency query string false "Currency"
// @Param startTime query string false "Start Time"
// @Param endTime query string false "End Time"
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Success 200 {object} dto.GetAccountHistoryResponse
// @Router /accounts/history [get]
func (h *AccountHandler) GetAccountHistory(c *gin.Context) {
	// Parse query params
	accountID := c.Query("accountId")
	userID := c.Query("userId")
	currency := c.Query("currency")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	req := dto.GetAccountHistoryRequest{
		AccountID: accountID,
		UserID:    userID,
		Currency:  currency,
		StartTime: startTime,
		EndTime:   endTime,
		Page:      page,
		Size:      size,
	}

	resp, err := h.Client.GetAccountHistory(c.Request.Context(), req)
	if err != nil {
		logger.Error("failed to get account history", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// FreezeBalance handles POST /accounts/freeze
// @Summary Freeze balance
// @Description Freezes a specified amount in an account
// @Tags accounts
// @Accept json
// @Produce json
// @Param request body dto.FreezeBalanceRequest true "Freeze Balance Request"
// @Success 200 {object} dto.FreezeBalanceResponse
// @Router /accounts/freeze [post]
func (h *AccountHandler) FreezeBalance(c *gin.Context) {
	var req dto.FreezeBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	resp, err := h.Client.FreezeBalance(c.Request.Context(), req)
	if err != nil {
		logger.Error("failed to freeze balance", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UnfreezeBalance handles POST /accounts/unfreeze
// @Summary Unfreeze balance
// @Description Unfreezes a specified amount in an account
// @Tags accounts
// @Accept json
// @Produce json
// @Param request body dto.UnfreezeBalanceRequest true "Unfreeze Balance Request"
// @Success 200 {object} dto.UnfreezeBalanceResponse
// @Router /accounts/unfreeze [post]
func (h *AccountHandler) UnfreezeBalance(c *gin.Context) {
	var req dto.UnfreezeBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	resp, err := h.Client.UnfreezeBalance(c.Request.Context(), req)
	if err != nil {
		logger.Error("failed to unfreeze balance", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Transfer handles POST /accounts/transfer
// @Summary Transfer funds
// @Description Moves funds between two accounts
// @Tags accounts
// @Accept json
// @Produce json
// @Param request body dto.TransferRequest true "Transfer Request"
// @Success 200 {object} dto.TransferResponse
// @Router /accounts/transfer [post]
func (h *AccountHandler) Transfer(c *gin.Context) {
	var req dto.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	resp, err := h.Client.Transfer(c.Request.Context(), req)
	if err != nil {
		logger.Error("failed to transfer", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, resp)
}
