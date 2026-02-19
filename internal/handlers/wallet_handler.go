package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"monera-digital/internal/dto"
	"monera-digital/internal/logger"
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
	// Debug logging for account opening flow
	logger.Info("[DEBUG-ACCOUNT-OPENING] CreateWallet handler started")

	bodyBytes, err := c.GetRawData()
	if err != nil {
		logger.Error("[DEBUG-ACCOUNT-OPENING] Failed to read request body", "error", err.Error())
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
		logger.Error("[DEBUG-ACCOUNT-OPENING] Failed to parse request body", "error", err.Error())
		c.JSON(http.StatusBadRequest, dto.CreateWalletResponse{
			Code:      "400",
			Message:   "Invalid JSON format",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	logger.Info("[DEBUG-ACCOUNT-OPENING] CreateWallet request parsed", "userId", req.UserID, "productCode", req.ProductCode, "currency", req.Currency)

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
		logger.Error("[DEBUG-ACCOUNT-OPENING] CreateWallet failed", "error", err.Error(), "userId", userID)
		c.JSON(http.StatusBadRequest, dto.CreateWalletResponse{
			Code:      "WALLET_CREATE_FAILED",
			Message:   err.Error(),
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	logger.Info("[DEBUG-ACCOUNT-OPENING] CreateWallet completed", "walletId", wallet.ID, "status", wallet.Status)

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

	logger.Info("[DEBUG] GetWalletInfo called", "userId", userID)

	info, err := h.WalletService.GetWalletInfo(c.Request.Context(), userID.(int))
	if err != nil {
		logger.Error("[DEBUG] GetWalletInfo error", "userId", userID, "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": err.Error(),
			"code":    "WALLET_INFO_ERROR",
		})
		return
	}

	if info == nil {
		logger.Info("[DEBUG] GetWalletInfo returning NONE", "userId", userID)
		c.JSON(http.StatusOK, gin.H{"status": "NONE"})
		return
	}

	logger.Info("[DEBUG] GetWalletInfo returning status", "userId", userID, "status", info.Status, "requestId", info.RequestID)
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
		// Return 400 for business logic errors (wallet not found, etc.)
		errMsg := err.Error()
		if strings.Contains(errMsg, "wallet not found") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "WALLET_NOT_FOUND",
				"message": "Please create a wallet first before adding addresses",
				"code":    "WALLET_NOT_FOUND",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": errMsg,
			"code":    "ADD_ADDRESS_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"walletId":    wallet.WalletID,
		"currency":    wallet.Currency,
		"address":     wallet.Address,
		"addressType": wallet.AddressType.String,
		"derivePath":  wallet.DerivePath.String,
		"status":      string(wallet.Status),
		"isPrimary":   wallet.IsPrimary,
		"createdAt":   wallet.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

func (h *Handler) GetAddressIncomeHistory(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.GetAddressIncomeHistoryResponse{
			Code:      "401",
			Message:   "Unauthorized",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	var req dto.GetAddressIncomeHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.GetAddressIncomeHistoryResponse{
			Code:      "400",
			Message:   "Invalid request: address is required",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	// 调用 Core API 获取收入历史
	records, err := h.WalletService.GetAddressIncomeHistory(c.Request.Context(), userID.(int), req.Address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.GetAddressIncomeHistoryResponse{
			Code:      "500",
			Message:   "Failed to get income history: " + err.Error(),
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	// 转换为 DTO 类型
	data := make([]dto.AddressIncomeRecord, len(records))
	for i, r := range records {
		data[i] = dto.AddressIncomeRecord{
			TxKey:             r.TxKey,
			TxHash:            r.TxHash,
			CoinKey:           r.CoinKey,
			TxAmount:          r.TxAmount,
			Address:           r.Address,
			TransactionStatus: r.TransactionStatus,
			BlockHeight:       r.BlockHeight,
			CreateTime:        r.CreateTime,
			CompletedTime:     r.CompletedTime,
		}
	}

	c.JSON(http.StatusOK, dto.GetAddressIncomeHistoryResponse{
		Code:      "200",
		Message:   "成功",
		Data:      data,
		Success:   true,
		Timestamp: time.Now().UnixMilli(),
	})
}

func (h *Handler) GetWalletAddress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.GetWalletAddressResponse{
			Code:      "401",
			Message:   "Unauthorized",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	var req dto.GetWalletAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.GetWalletAddressResponse{
			Code:      "400",
			Message:   "Invalid request format",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	// 如果请求中没有 userId，从 JWT token 中获取
	if req.UserID == "" {
		req.UserID = strconv.Itoa(userID.(int))
	}

	// 验证必填字段
	if req.UserID == "" || req.ProductCode == "" || req.Currency == "" {
		c.JSON(http.StatusBadRequest, dto.GetWalletAddressResponse{
			Code:      "400",
			Message:   "userId, productCode and currency are required",
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	wallet, err := h.WalletService.GetWalletAddress(c.Request.Context(), userID.(int), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.GetWalletAddressResponse{
			Code:      "500",
			Message:   "Failed to get wallet address: " + err.Error(),
			Success:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.GetWalletAddressResponse{
		Code:      "200",
		Message:   "成功",
		Data:      *wallet,
		Success:   true,
		Timestamp: time.Now().UnixMilli(),
	})
}
