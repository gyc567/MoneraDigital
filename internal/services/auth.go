package services

import (
	"database/sql"
	"monera-digital/internal/models"
)

type AuthService struct{
	DB *sql.DB
}

func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{DB: db}
}

type LoginResponse struct {
	User        *models.User `json:"user,omitempty"`
	Token       string       `json:"token,omitempty"`
	Requires2FA bool         `json:"requires_2fa,omitempty"`
	UserID      int          `json:"user_id,omitempty"`
}

func (s *AuthService) Register(req models.RegisterRequest) (*models.User, error) {
	// TODO: Implement
	return &models.User{}, nil
}

func (s *AuthService) Login(req models.LoginRequest) (*LoginResponse, error) {
	// TODO: Implement
	return &LoginResponse{}, nil
}

func (s *AuthService) Verify2FAAndLogin(userID int, token string) (*LoginResponse, error) {
	// TODO: Implement
	return &LoginResponse{}, nil
}

func (s *AuthService) GetUserByID(userID int) (*models.User, error) {
	// TODO: Implement
	return &models.User{}, nil
}