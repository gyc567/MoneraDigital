// internal/models/token.go
package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenPair 令牌对（访问令牌 + 刷新令牌）
type TokenPair struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	TokenType    string    `json:"tokenType"`
	ExpiresIn    int       `json:"expiresIn"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// TokenClaims JWT 令牌声明
type TokenClaims struct {
	UserID    int    `json:"userId"`
	Email     string `json:"email"`
	TokenType string `json:"tokenType"` // "access" 或 "refresh"
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

// Implement jwt.Claims interface
func (tc *TokenClaims) GetAudience() (jwt.ClaimStrings, error) {
	return jwt.ClaimStrings{}, nil
}

func (tc *TokenClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	if tc.ExpiresAt == 0 {
		return nil, nil
	}
	t := jwt.NewNumericDate(time.Unix(tc.ExpiresAt, 0))
	return t, nil
}

func (tc *TokenClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	if tc.IssuedAt == 0 {
		return nil, nil
	}
	t := jwt.NewNumericDate(time.Unix(tc.IssuedAt, 0))
	return t, nil
}

func (tc *TokenClaims) GetIssuer() (string, error) {
	return "", nil
}

func (tc *TokenClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

func (tc *TokenClaims) GetSubject() (string, error) {
	return "", nil
}

// RefreshTokenRequest 刷新令牌请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// RefreshTokenResponse 刷新令牌响应
type RefreshTokenResponse struct {
	AccessToken string    `json:"accessToken"`
	ExpiresIn   int       `json:"expiresIn"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	Token string `json:"token" binding:"required"`
}
