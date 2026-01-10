// internal/models/token.go
package models

import "time"

// TokenPair 令牌对（访问令牌 + 刷新令牌）
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// TokenClaims JWT 令牌声明
type TokenClaims struct {
	UserID    int    `json:"user_id"`
	Email     string `json:"email"`
	TokenType string `json:"token_type"` // "access" 或 "refresh"
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

// RefreshTokenRequest 刷新令牌请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse 刷新令牌响应
type RefreshTokenResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	Token string `json:"token" binding:"required"`
}
