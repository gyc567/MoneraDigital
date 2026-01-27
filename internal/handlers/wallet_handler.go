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
