package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	redemp "monera-digital/internal/redemption"
)

type CreateRedemptionRequest struct {
	UserID    string  `json:"userId" binding:"required"`
	ProductID string  `json:"productId" binding:"required"`
	Principal float64 `json:"principal" binding:"required"`
	AutoRenew bool    `json:"autoRenew"`
}

// RegisterRedemptionRoutes wires redemption endpoints into the provided Gin engine
func RegisterRedemptionRoutes(r *gin.Engine, svc *redemp.RedemptionService) {
	// Create Redemption
	r.POST("/api/redemption", func(c *gin.Context) {
		var req CreateRedemptionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		rec, err := svc.CreateRedemption(req.UserID, req.ProductID, req.Principal, req.AutoRenew)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":               rec.ID,
			"status":           rec.Status,
			"redemptionAmount": rec.RedemptionAmount,
			"startDate":        rec.StartDate.Format(time.RFC3339),
			"endDate":          rec.EndDate.Format(time.RFC3339),
		})
	})

	// Get Redemption
	r.GET("/api/redemption/:id", func(c *gin.Context) {
		id := c.Param("id")
		rec, err := svc.GetRedemption(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, rec)
	})
}
