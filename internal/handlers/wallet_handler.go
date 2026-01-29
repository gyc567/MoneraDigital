package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) CreateWallet(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	req, err := h.WalletService.CreateWallet(c.Request.Context(), userID.(int))
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, req)
	"net/http"
	"strconv"
	"time"

	"monera-digital/internal/dto"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateWallet(c *gin.Context) {
	var req dto.CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.CreateWalletResponse{
			Code:      "400",
			Message:   "Invalid request parameters",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	// Validate required fields
	if req.UserID == "" || req.ProductCode == "" || req.Currency == "" {
		c.JSON(http.StatusBadRequest, dto.CreateWalletResponse{
			Code:      "400",
			Message:   "userId, productCode and currency are required",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	// Validate product code
	if req.ProductCode != "BANK_ACCOUNT" {
		c.JSON(http.StatusBadRequest, dto.CreateWalletResponse{
			Code:      "400",
			Message:   "Invalid product code",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	// Convert userId string to int
	userID, err := strconv.Atoi(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.CreateWalletResponse{
			Code:      "400",
			Message:   "Invalid userId format",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	wallet, err := h.WalletService.CreateWallet(c.Request.Context(), userID, req.ProductCode, req.Currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.CreateWalletResponse{
			Code:      "500",
			Message:   "Failed to create wallet",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	status := "NORMAL"
	if wallet.Status != "SUCCESS" {
		status = "PENDING"
	}

	createdAt := wallet.CreatedAt.Format("2006-01-02 15:04:05")

	c.JSON(http.StatusOK, dto.CreateWalletResponse{
		Code:    "200",
		Message: "Success",
		Data: dto.WalletResponseData{
			UserID:      req.UserID,
			ProductCode: req.ProductCode,
			Currency:    req.Currency,
			Status:      status,
			CreatedAt:   createdAt,
		},
		Success:   true,
		Timestamp: time.Now().UnixMilli(),
	})
}

func (h *Handler) GetWalletInfo(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	info, err := h.WalletService.GetWalletInfo(c.Request.Context(), userID.(int))
	if err != nil {
		c.Error(err)
		return
	}

	if info == nil {
		c.JSON(http.StatusOK, gin.H{"status": "NONE"})
		return
	}

	c.JSON(http.StatusOK, info)
}
