package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BaseHandler 封装通用的 HTTP 处理逻辑
type BaseHandler struct{}

// getUserID 从上下文中获取用户 ID
func (h *BaseHandler) getUserID(c *gin.Context) (int, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	id, ok := userID.(int)
	return id, ok
}

// getUserEmail 从上下文中获取用户邮箱
func (h *BaseHandler) getUserEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("email")
	if !exists {
		return "", false
	}
	emailStr, ok := email.(string)
	return emailStr, ok
}

// requireUserID 确保用户已认证，未认证返回 401
func (h *BaseHandler) requireUserID(c *gin.Context) (int, bool) {
	id, ok := h.getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
			"code":  "AUTH_REQUIRED",
		})
		return 0, false
	}
	return id, true
}

// bindTokenRequest 绑定并验证 TOTP token 请求
func (h *BaseHandler) bindTokenRequest(c *gin.Context) (string, bool) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Token is required",
			"code":  "INVALID_REQUEST",
		})
		return "", false
	}

	if len(req.Token) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Token must be 6 digits",
			"code":  "INVALID_TOKEN_FORMAT",
		})
		return "", false
	}

	return req.Token, true
}

// successResponse 返回成功响应
func (h *BaseHandler) successResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// errorResponse 返回错误响应
func (h *BaseHandler) errorResponse(c *gin.Context, status int, code string, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
