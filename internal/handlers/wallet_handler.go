package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"monera-digital/internal/dto"
	"monera-digital/internal/models"
	"monera-digital/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// parseTokenFromHeader extracts and parses userId from JWT token in Authorization header
func parseTokenFromHeader(authHeader string) (int, error) {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, jwt.ErrSignatureInvalid
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &models.TokenClaims{})
	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(*models.TokenClaims)
	if !ok {
		return 0, jwt.ErrTokenMalformed
	}

	if claims.UserID == 0 {
		return 0, jwt.ErrTokenMalformed
	}

	return claims.UserID, nil
}

func (h *Handler) CreateWallet(c *gin.Context) {
	bodyBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.CreateWalletResponse{
			Code:      "400",
			Message:   "Failed to read request body",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	var req dto.CreateWalletRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		c.JSON(http.StatusBadRequest, dto.CreateWalletResponse{
			Code:      "400",
			Message:   "Invalid JSON format",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	// If userId not provided in request body, try to extract from JWT token in Authorization header
	if req.UserID == "" {
		authHeader := c.GetHeader("Authorization")
		log.Printf("Authorization header: %s", authHeader)
		if authHeader != "" {
			userID, err := parseTokenFromHeader(authHeader)
			if err == nil && userID != 0 {
				req.UserID = strconv.Itoa(userID)
			}
		}
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
	if req.ProductCode != "X_FINANCE" {
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
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not authenticated",
			"code":    "UNAUTHORIZED",
		})
		return
	}

	info, err := h.WalletService.GetWalletInfo(c.Request.Context(), userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": err.Error(),
			"code":    "WALLET_INFO_ERROR",
		})
		return
	}

	if info == nil {
		c.JSON(http.StatusOK, gin.H{"status": "NONE"})
		return
	}

	c.JSON(http.StatusOK, info)
}

func (h *Handler) AddWalletAddress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not authenticated",
			"code":    "UNAUTHORIZED",
		})
		return
	}

	var req dto.AddWalletAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
			"code":    "INVALID_REQUEST",
		})
		return
	}

	if req.Chain == "" || req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "chain and token are required",
			"code":    "MISSING_FIELDS",
		})
		return
	}

	wallet, err := h.WalletService.AddAddress(c.Request.Context(), userID.(int), services.AddAddressRequest{
		Chain: req.Chain,
		Token: req.Token,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": err.Error(),
			"code":    "ADD_ADDRESS_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, wallet)
}
