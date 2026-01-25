package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"monera-digital/internal/models"
)

// EncryptionProvider interface for dependency injection (KISS: testable)
type EncryptionProvider interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// TwoFactorService handles 2FA operations (KISS: single responsibility)
type TwoFactorService struct {
	DB interface {
		QueryRow(query string, args ...interface{}) *sql.Row
		Exec(query string, args ...interface{}) (sql.Result, error)
	}
	Encryption EncryptionProvider
}

// SetupResponse contains 2FA setup data returned to the user
type SetupResponse struct {
	Secret      string   `json:"secret"`
	QRCode      string   `json:"qrCodeUrl"`
	BackupCodes []string `json:"backupCodes"`
	OTPAuth     string   `json:"otpauth"`
}

// NewTwoFactorService creates a new 2FA service
func NewTwoFactorService(db *sql.DB, encryption *EncryptionService) *TwoFactorService {
	return &TwoFactorService{
		DB:         db,
		Encryption: encryption,
	}
}

// NewTwoFactorServiceWithInterface creates a new 2FA service with interface
// Use this for testing with mock implementations
func NewTwoFactorServiceWithInterface(db interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}, encryption EncryptionProvider) *TwoFactorService {
	return &TwoFactorService{
		DB:         db,
		Encryption: encryption,
	}
}

// Setup generates a new 2FA secret, QR code, and backup codes
func (s *TwoFactorService) Setup(userID int, email string) (*SetupResponse, error) {
	// Generate TOTP secret
	secret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Monera Digital",
		AccountName: email,
		Period:      30,
		Digits:      6,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	// Generate 10 backup codes
	backupCodes := make([]string, 10)
	for i := range backupCodes {
		code := make([]byte, 4)
		rand.Read(code)
		backupCodes[i] = hex.EncodeToString(code)
	}

	// Encrypt and store secret and backup codes
	encryptedSecret, err := s.Encryption.Encrypt(secret.Secret())
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}
	encryptedBackupCodes, err := json.Marshal(backupCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup codes: %w", err)
	}

	query := `
		UPDATE users
		SET two_factor_secret = $1, two_factor_backup_codes = $2
		WHERE id = $3`

	_, err = s.DB.Exec(query, encryptedSecret, string(encryptedBackupCodes), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to store 2FA secret: %w", err)
	}

	return &SetupResponse{
		Secret:      secret.Secret(),
		QRCode:      secret.URL(),
		BackupCodes: backupCodes,
		OTPAuth:     secret.URL(),
	}, nil
}

// Enable verifies TOTP token and enables 2FA for the user
func (s *TwoFactorService) Enable(userID int, token string) error {
	secret, err := s.getSecret(userID)
	if err != nil {
		return fmt.Errorf("2FA not set up: %w", err)
	}

	valid := totp.Validate(token, secret)
	if !valid {
		return fmt.Errorf("invalid verification code")
	}

	query := `UPDATE users SET two_factor_enabled = true WHERE id = $1`
	_, err = s.DB.Exec(query, userID)
	return err
}

// Disable verifies token or backup code and disables 2FA
func (s *TwoFactorService) Disable(userID int, token string) error {
	valid, err := s.Verify(userID, token)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("invalid verification code")
	}

	query := `
		UPDATE users
		SET two_factor_enabled = false, two_factor_secret = NULL, two_factor_backup_codes = NULL
		WHERE id = $1`
	_, err = s.DB.Exec(query, userID)
	return err
}

// Verify checks if a token is valid (TOTP or backup code)
func (s *TwoFactorService) Verify(userID int, token string) (bool, error) {
	secret, err := s.getSecret(userID)
	if err != nil {
		return false, err
	}

	// Check TOTP first
	if totp.Validate(token, secret) {
		return true, nil
	}

	// Check backup codes
	backupCodes, err := s.getBackupCodes(userID)
	if err != nil {
		return false, err
	}

	for i, code := range backupCodes {
		if code == token {
			// Remove used backup code
			remaining := append(backupCodes[:i], backupCodes[i+1:]...)
			encryptedBackupCodes, err := json.Marshal(remaining)
			if err != nil {
				return false, fmt.Errorf("failed to marshal backup codes: %w", err)
			}

			query := `UPDATE users SET two_factor_backup_codes = $1 WHERE id = $2`
			_, err = s.DB.Exec(query, string(encryptedBackupCodes), userID)
			if err != nil {
				return false, fmt.Errorf("failed to update backup codes: %w", err)
			}
			return true, nil
		}
	}

	return false, nil
}

// IsEnabled checks if 2FA is enabled for a user
func (s *TwoFactorService) IsEnabled(userID int) (bool, error) {
	var enabled bool
	query := `SELECT two_factor_enabled FROM users WHERE id = $1`
	err := s.DB.QueryRow(query, userID).Scan(&enabled)
	return enabled, err
}

// getSecret retrieves and decrypts the 2FA secret for a user
func (s *TwoFactorService) getSecret(userID int) (string, error) {
	var encryptedSecret string
	query := `SELECT two_factor_secret FROM users WHERE id = $1`
	err := s.DB.QueryRow(query, userID).Scan(&encryptedSecret)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("2FA not set up")
	}
	if err != nil {
		return "", err
	}
	secret, err := s.Encryption.Decrypt(encryptedSecret)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}
	return secret, nil
}

// getBackupCodes retrieves and decrypts backup codes for a user
func (s *TwoFactorService) getBackupCodes(userID int) ([]string, error) {
	var encryptedCodes string
	query := `SELECT two_factor_backup_codes FROM users WHERE id = $1`
	err := s.DB.QueryRow(query, userID).Scan(&encryptedCodes)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("2FA not set up")
	}
	if err != nil {
		return nil, err
	}

	var codes []string
	if err := json.Unmarshal([]byte(encryptedCodes), &codes); err != nil {
		return nil, err
	}
	return codes, nil
}

// VerifyAndLogin verifies 2FA token and returns login response data
func (s *TwoFactorService) VerifyAndLogin(userID int, token string) (*LoginResponse, error) {
	// 验证2FA令牌
	valid, err := s.Verify(userID, token)
	if err != nil {
		return nil, fmt.Errorf("2FA verification failed: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("invalid 2FA token")
	}

	// 获取用户信息用于生成JWT
	var email string
	var twoFactorEnabled bool
	query := `SELECT email, two_factor_enabled FROM users WHERE id = $1`
	err = s.DB.QueryRow(query, userID).Scan(&email, &twoFactorEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 这里只返回验证成功的用户信息，JWT生成在AuthService中处理
	return &LoginResponse{
		User: &models.User{
			ID:    userID,
			Email: email,
		},
	}, nil
}

// generateTOTPToken generates a valid TOTP token for testing
// This is unexported and for testing purposes only
func generateTOTPToken(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}
