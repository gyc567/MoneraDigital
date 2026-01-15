// internal/middleware/error_handler.go
package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"monera-digital/internal/validator"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ErrorHandler middleware for handling errors consistently
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			handleError(c, err.Err)
		}
	}
}

// handleError maps errors to appropriate HTTP status codes and responses
func handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// Check for validation errors
	var validationErr *validator.ValidationError
	if errors.As(err, &validationErr) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: validationErr.Error(),
			Details: validationErr.Field,
		})
		return
	}

	// Check for specific error messages
	errMsg := err.Error()

	switch errMsg {
	case "email not found":
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    "EMAIL_NOT_FOUND",
			Message: "Email input error or does not exist",
		})
	case "invalid credentials":
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    "INVALID_CREDENTIALS",
			Message: "Invalid email or password",
		})
	case "email already registered":
		c.JSON(http.StatusConflict, ErrorResponse{
			Code:    "EMAIL_ALREADY_EXISTS",
			Message: "Email is already registered",
		})
	case "invalid refresh token":
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    "INVALID_REFRESH_TOKEN",
			Message: "Refresh token is invalid or expired",
		})
	case "refresh token has been revoked":
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    "TOKEN_REVOKED",
			Message: "Refresh token has been revoked",
		})
	case "token blacklist not initialized":
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Token service is not properly initialized",
		})
	case "unauthorized":
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: "Authentication required",
		})
	case "not found":
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    "NOT_FOUND",
			Message: "Resource not found",
		})
	default:
		// Generic internal server error
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "An internal server error occurred",
			Details: errMsg,
		})
	}
}

// RecoveryHandler middleware for recovering from panics
func RecoveryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Code:    "PANIC_RECOVERED",
					Message: "An unexpected error occurred",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
