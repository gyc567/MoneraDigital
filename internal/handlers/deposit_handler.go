package handlers

import (
    "net/http"
    "strconv"
    "github.com/gin-gonic/gin"
)

func (h *Handler) GetDeposits(c *gin.Context) {
    userID, exists := c.Get("userID")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    limit := 20
    offset := 0
    if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
        limit = l
    }
    if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
        offset = o
    }

    deposits, total, err := h.DepositService.GetDeposits(c.Request.Context(), userID.(int), limit, offset)
    if err != nil {
        c.Error(err)
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "total":    total,
        "deposits": deposits,
    })
}

func (h *Handler) HandleDepositWebhook(c *gin.Context) {
    // Webhook logic
    // Verify signature
    // Call service
    c.JSON(http.StatusOK, gin.H{"success": true})
}
