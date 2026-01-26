package config

import (
	"os"
	"strconv"

	"github.com/pquerna/otp"
)

// TwoFactorConfig holds 2FA configuration
type TwoFactorConfig struct {
	Issuer          string        // TOTP issuer name
	Period          uint          // TOTP period in seconds (default: 30)
	Digits          int           // Number of digits in TOTP code (default: 6)
	SecretSize      int           // Secret key size in bytes (default: 20 = 160 bits)
	Algorithm       otp.Algorithm // HMAC algorithm (default: SHA1)
	Skew            uint          // Time window for validation (default: 1 = ±30s)
	BackupCodeCount int           // Number of backup codes to generate (default: 10)
}

// LoadTwoFactorConfig loads 2FA configuration from environment variables
func LoadTwoFactorConfig() *TwoFactorConfig {
	return &TwoFactorConfig{
		Issuer:          getEnvOrDefault("TWOFA_ISSUER", "Monera Digital"),
		Period:          getEnvUintOrDefault("TWOFA_PERIOD", 30),
		Digits:          getEnvIntOrDefault("TWOFA_DIGITS", 6),
		SecretSize:      getEnvIntOrDefault("TWOFA_SECRET_SIZE", 20), // 160 bits
		Algorithm:       otp.AlgorithmSHA1,                            // Google Authenticator standard
		Skew:            getEnvUintOrDefault("TWOFA_SKEW", 1),         // ±30 seconds
		BackupCodeCount: getEnvIntOrDefault("TWOFA_BACKUP_COUNT", 10),
	}
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvUintOrDefault(key string, defaultValue uint) uint {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil && intVal >= 0 {
			return uint(intVal)
		}
	}
	return defaultValue
}
