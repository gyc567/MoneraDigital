package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"monera-digital/internal/services"
)

// TwoFAHandler handles 2FA HTTP endpoints
type TwoFAHandler struct {
	base         *BaseHandler
	twofaService *services.TwoFactorService
}

// NewTwoFAHandler creates a new 2FA handler
func NewTwoFAHandler(twofa *services.TwoFactorService) *TwoFAHandler {
	return &TwoFAHandler{
		base:         &BaseHandler{},
		twofaService: twofa,
	}
}

// Setup2FA generates a new 2FA secret, QR code, and backup codes for the user
// POST /api/auth/2fa/setup
func (h *TwoFAHandler) Setup2FA(c *gin.Context) {
	userID, ok := h.base.requireUserID(c)
	if !ok {
		return
	}

	email, ok := h.base.getUserEmail(c)
	if !ok {
		h.base.errorResponse(c, http.StatusBadRequest, "INVALID_EMAIL", "User email not found")
		return
	}

	setup, err := h.twofaService.Setup(userID, email)
	if err != nil {
		h.base.errorResponse(c, http.StatusInternalServerError, "SETUP_FAILED", err.Error())
		return
	}

	h.base.successResponse(c, gin.H{
		"secret":      setup.Secret,
		"qrCodeUrl":   setup.QRCode,
		"backupCodes": setup.BackupCodes,
		"message":     "2FA setup successful. Scan the QR code with your authenticator app.",
	})
}

// Enable2FA verifies the TOTP token and enables 2FA for the user
// POST /api/auth/2fa/enable
func (h *TwoFAHandler) Enable2FA(c *gin.Context) {
	userID, ok := h.base.requireUserID(c)
	if !ok {
		return
	}

	token, ok := h.base.bindTokenRequest(c)
	if !ok {
		return
	}

	if err := h.twofaService.Enable(userID, token); err != nil {
		h.base.errorResponse(c, http.StatusBadRequest, "ENABLE_FAILED", err.Error())
		return
	}

	h.base.successResponse(c, gin.H{
		"enabled": true,
		"message": "2FA has been enabled successfully",
	})
}

// Disable2FA disables 2FA for the user
// POST /api/auth/2fa/disable
func (h *TwoFAHandler) Disable2FA(c *gin.Context) {
	userID, ok := h.base.requireUserID(c)
	if !ok {
		return
	}

	token, ok := h.base.bindTokenRequest(c)
	if !ok {
		return
	}

	if err := h.twofaService.Disable(userID, token); err != nil {
		h.base.errorResponse(c, http.StatusBadRequest, "DISABLE_FAILED", err.Error())
		return
	}

	h.base.successResponse(c, gin.H{
		"enabled": false,
		"message": "2FA has been disabled successfully",
	})
}

// Verify2FA verifies a 2FA token (TOTP or backup code)
// POST /api/auth/2fa/verify
func (h *TwoFAHandler) Verify2FA(c *gin.Context) {
	userID, ok := h.base.requireUserID(c)
	if !ok {
		return
	}

	token, ok := h.base.bindTokenRequest(c)
	if !ok {
		return
	}

	valid, err := h.twofaService.Verify(userID, token)
	if err != nil {
		h.base.errorResponse(c, http.StatusBadRequest, "VERIFY_FAILED", err.Error())
		return
	}

	if !valid {
		h.base.errorResponse(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid verification code")
		return
	}

	h.base.successResponse(c, gin.H{
		"valid":   true,
		"message": "Token is valid",
	})
}

// Get2FAStatus returns whether 2FA is enabled for the user
// GET /api/auth/2fa/status
func (h *TwoFAHandler) Get2FAStatus(c *gin.Context) {
	userID, ok := h.base.requireUserID(c)
	if !ok {
		return
	}

	enabled, err := h.twofaService.IsEnabled(userID)
	if err != nil {
		h.base.errorResponse(c, http.StatusInternalServerError, "STATUS_FAILED", err.Error())
		return
	}

	h.base.successResponse(c, gin.H{
		"enabled": enabled,
	})
}
