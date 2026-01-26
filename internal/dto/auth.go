// internal/dto/auth.go
package dto

import "time"

// RegisterRequest DTO for user registration
type RegisterRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginRequest DTO for user login
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse DTO for login response
type LoginResponse struct {
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	User         *UserInfo `json:"user,omitempty"`
	Token        string    `json:"token,omitempty"`
	Requires2FA  bool      `json:"requires2FA,omitempty"`
	UserID       int       `json:"userId,omitempty"`
}

// UserInfo DTO for user information
type UserInfo struct {
	ID               int    `json:"id"`
	Email            string `json:"email"`
	TwoFactorEnabled bool   `json:"twoFactorEnabled"`
}

// RefreshTokenRequest DTO for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse DTO for token refresh response
type RefreshTokenResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresIn   int       `json:"expires_in"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// LogoutRequest DTO for logout
type LogoutRequest struct {
	Token string `json:"token" binding:"required"`
}

// Setup2FARequest DTO for 2FA setup
type Setup2FARequest struct {
	Email string `json:"email" binding:"required,email"`
}

// Setup2FAResponse DTO for 2FA setup response
type Setup2FAResponse struct {
	Secret string `json:"secret"`
	QRCode string `json:"qr_code"`
}

// Enable2FARequest DTO for enabling 2FA
type Enable2FARequest struct {
	Token string `json:"token" binding:"required,len=6"`
}

// Verify2FALoginRequest DTO for 2FA verification during login
type Verify2FALoginRequest struct {
	Email string `json:"email" binding:"required,email"`
	Token string `json:"token" binding:"required,len=6"`
}
