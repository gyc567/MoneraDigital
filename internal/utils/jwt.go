package utils

import (
	"errors"
	"time"

	"monera-digital/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(userID int, email string, secret string) (string, error) {
	now := time.Now()
	claims := models.TokenClaims{
		UserID:    userID,
		Email:     email,
		TokenType: "access",
		ExpiresAt: now.Add(time.Hour * 24).Unix(),
		IssuedAt:  now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	return token.SignedString([]byte(secret))
}

// ParseJWT parses and validates a JWT token, returning the claims
func ParseJWT(tokenString, secret string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return map[string]interface{}(claims), nil
	}

	return nil, errors.New("invalid token")
}
