package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// EncryptionService provides AES-256-GCM encryption for sensitive data
type EncryptionService struct {
	key []byte
}

// DecodeEncryptionKey normalizes encryption key from environment
// Supports two formats:
// - Hex-encoded: 64 hex characters (e.g., "c70c58a23fd8ab7b...") → decodes to 32 bytes
// - Raw string: exactly 32 characters (e.g., "12345678901234567890123456789012") → used as-is
// Returns error if format is invalid or key is wrong length
func DecodeEncryptionKey(keyString string) (string, error) {
	keyString = strings.TrimSpace(keyString)

	if keyString == "" {
		return "", errors.New("encryption key cannot be empty")
	}

	// Try to detect format: hex-encoded (64 chars) vs raw (32 chars)
	if len(keyString) == 64 {
		// Likely hex-encoded, try to decode
		decodedBytes, err := hex.DecodeString(keyString)
		if err != nil {
			return "", fmt.Errorf("invalid hex format: %w", err)
		}
		if len(decodedBytes) != 32 {
			return "", fmt.Errorf("decoded hex key must be 32 bytes, got %d", len(decodedBytes))
		}
		return string(decodedBytes), nil
	}

	if len(keyString) == 32 {
		// Raw 32-byte string, use as-is
		return keyString, nil
	}

	// Invalid length
	return "", fmt.Errorf("encryption key must be either 32 characters (raw) or 64 characters (hex-encoded), got %d", len(keyString))
}

// NewEncryptionService creates a new encryption service
// The key must be 32 bytes for AES-256 (after normalization)
func NewEncryptionService(key string) (*EncryptionService, error) {
	keyBytes := []byte(key)
	if len(keyBytes) != 32 {
		return nil, errors.New("encryption key must be exactly 32 bytes")
	}
	return &EncryptionService{key: keyBytes}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func (s *EncryptionService) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (s *EncryptionService) Decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("invalid base64: %w", err)
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}
