// internal/services/auth_refresh.go
package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"monera-digital/internal/models"
)

// RefreshToken 刷新令牌
func (s *AuthService) RefreshToken(refreshToken string) (*models.TokenPair, error) {
	// 1. 验证刷新令牌有效性
	claims := &models.TokenClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	// 2. 检查令牌类型
	if claims.TokenType != "refresh" {
		return nil, errors.New("invalid token type")
	}

	// 3. 检查是否在黑名单中
	if s.tokenBlacklist != nil && s.tokenBlacklist.IsBlacklisted(refreshToken) {
		return nil, errors.New("refresh token has been revoked")
	}

	// 4. 生成新的访问令牌
	accessToken, err := s.generateAccessToken(claims.UserID, claims.Email)
	if err != nil {
		return nil, err
	}

	// 5. 可选：生成新的刷新令牌（每次刷新都生成新的）
	newRefreshToken, err := s.generateRefreshToken(claims.UserID, claims.Email)
	if err != nil {
		return nil, err
	}

	// 6. 将旧的刷新令牌加入黑名单
	if s.tokenBlacklist != nil {
		s.tokenBlacklist.Add(refreshToken, time.Unix(claims.ExpiresAt, 0))
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 分钟
		ExpiresAt:    time.Now().Add(15 * time.Minute),
	}, nil
}

// generateAccessToken 生成访问令牌（15 分钟过期）
func (s *AuthService) generateAccessToken(userID int, email string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	claims := &models.TokenClaims{
		UserID:    userID,
		Email:     email,
		TokenType: "access",
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// generateRefreshToken 生成刷新令牌（7 天过期）
func (s *AuthService) generateRefreshToken(userID int, email string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(7 * 24 * time.Hour)

	claims := &models.TokenClaims{
		UserID:    userID,
		Email:     email,
		TokenType: "refresh",
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// Logout 登出（将令牌加入黑名单）
func (s *AuthService) Logout(token string) error {
	if s.tokenBlacklist == nil {
		return errors.New("token blacklist not initialized")
	}

	// 解析令牌获取过期时间
	claims := &models.TokenClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		// 即使令牌无效，也将其加入黑名单
		s.tokenBlacklist.Add(token, time.Now().Add(24*time.Hour))
		return nil
	}

	// 将令牌加入黑名单
	s.tokenBlacklist.Add(token, time.Unix(claims.ExpiresAt, 0))
	return nil
}
