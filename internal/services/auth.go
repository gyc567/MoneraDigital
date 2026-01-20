package services

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"monera-digital/internal/cache"
	"monera-digital/internal/config"
	"monera-digital/internal/models"
	"monera-digital/internal/utils"
)

// AuthService provides authentication functionality
type AuthService struct {
	DB             *sql.DB
	jwtSecret      string
	tokenBlacklist *cache.TokenBlacklist
}

// NewAuthService creates a new AuthService instance
func NewAuthService(db *sql.DB, jwtSecret string) *AuthService {
	return &AuthService{
		DB:        db,
		jwtSecret: jwtSecret,
	}
}

// SetTokenBlacklist sets the token blacklist for logout functionality
func (s *AuthService) SetTokenBlacklist(tb *cache.TokenBlacklist) {
	s.tokenBlacklist = tb
}

// LoginResponse represents the login API response
type LoginResponse struct {
	User         *models.User `json:"user,omitempty"`
	Token        string       `json:"token,omitempty"`
	AccessToken  string       `json:"access_token,omitempty"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	TokenType    string       `json:"token_type,omitempty"`
	ExpiresIn    int          `json:"expires_in,omitempty"`
	ExpiresAt    time.Time    `json:"expires_at,omitempty"`
	Requires2FA  bool         `json:"requires_2fa,omitempty"`
	UserID       int          `json:"user_id,omitempty"`
}

// Register handles user registration
func (s *AuthService) Register(req models.RegisterRequest) (*models.User, error) {
	// Check if email already exists
	var exists bool
	err := s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Insert user into database
	var user models.User
	query := `
		INSERT INTO users (email, password, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id, email, created_at, two_factor_enabled`

	err = s.DB.QueryRow(query, req.Email, hashedPassword).Scan(
		&user.ID, &user.Email, &user.CreatedAt, &user.TwoFactorEnabled,
	)
	if err != nil {
		return nil, err
	}

	// Create account in Core Account System (fire and forget)
	_, _ = s.createCoreAccount(user.ID, req.Email)

	return &user, nil
}

// createCoreAccount creates an account in the Core Account System
func (s *AuthService) createCoreAccount(userID int, email string) (string, error) {
	accountReq := map[string]interface{}{
		"externalId":  strconv.Itoa(userID),
		"accountType": "INDIVIDUAL",
		"profile": map[string]interface{}{
			"email":     email,
			"firstName": "",
			"lastName":  "",
		},
		"metadata": map[string]interface{}{
			"source": "monera_web",
		},
	}

	jsonData, err := json.Marshal(accountReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	cfg := config.Load()
	coreAPIURL := fmt.Sprintf("http://localhost:%s/api/core/accounts/create", cfg.Port)

	resp, err := http.Post(coreAPIURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Sprintf("core_simulated_%d", userID), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("core account creation failed with status: %d", resp.StatusCode)
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			AccountID string `json:"accountId"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Data.AccountID, nil
}

// Login handles user authentication
func (s *AuthService) Login(req models.LoginRequest) (*LoginResponse, error) {
	var user models.User
	var hashedPassword string

	query := `SELECT id, email, password, two_factor_enabled FROM users WHERE email = $1`
	err := s.DB.QueryRow(query, req.Email).Scan(&user.ID, &user.Email, &hashedPassword, &user.TwoFactorEnabled)

	if err == sql.ErrNoRows {
		return nil, errors.New("invalid credentials")
	} else if err != nil {
		return nil, err
	}

	// Verify password
	if !utils.CheckPasswordHash(req.Password, hashedPassword) {
		return nil, errors.New("invalid credentials")
	}

	// Check if 2FA is required
	if user.TwoFactorEnabled {
		return &LoginResponse{
			Requires2FA: true,
			UserID:      user.ID,
		}, nil
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID, user.Email, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	return &LoginResponse{
		User:        &user,
		Token:       token,
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   86400,
		ExpiresAt:   expiresAt,
	}, nil
}

// Verify2FAAndLogin verifies 2FA token and completes login
func (s *AuthService) Verify2FAAndLogin(userID int, token string) (*LoginResponse, error) {
	// TODO: Implement 2FA verification
	return &LoginResponse{}, nil
}

// GetUserByID retrieves a user by their ID
func (s *AuthService) GetUserByID(userID int) (*models.User, error) {
	// TODO: Implement user retrieval
	return &models.User{}, nil
}
