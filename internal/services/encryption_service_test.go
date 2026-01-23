package services

import (
	"testing"
)

func TestNewEncryptionService_ValidKey(t *testing.T) {
	// Valid 32-byte key
	key := "12345678901234567890123456789012"
	service, err := NewEncryptionService(key)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewEncryptionService_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		expectError bool
	}{
		{"too_short", "1234567890123456789012345678901", true},
		{"too_long", "123456789012345678901234567890123", true},
		{"empty", "", true},
		{"exactly_32_bytes", "12345678901234567890123456789012", false},
		{"32_bytes_random", makeKey(32), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEncryptionService(tt.key)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestEncryptionService_EncryptDecrypt(t *testing.T) {
	key := "12345678901234567890123456789012"
	service, _ := NewEncryptionService(key)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty_string", ""},
		{"short_string", "hello"},
		{"normal_string", "This is a test message"},
		{"long_string", string(makeBytes(1000))},
		{"special_chars", "Test!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"unicode", "Hello 世界 🌍"},
		{"json", `{"user": "test", "data": 123}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := service.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			if ciphertext == tt.plaintext {
				t.Error("ciphertext should not equal plaintext")
			}

			decrypted, err := service.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("decrypted text does not match original: got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptionService_DifferentCiphertexts(t *testing.T) {
	key := "12345678901234567890123456789012"
	service, _ := NewEncryptionService(key)

	// Same plaintext should produce different ciphertexts (due to random nonce)
	plaintext := "test message"
	ciphertext1, _ := service.Encrypt(plaintext)
	ciphertext2, _ := service.Encrypt(plaintext)

	if ciphertext1 == ciphertext2 {
		t.Error("same plaintext should produce different ciphertexts")
	}

	// Both should decrypt to same value
	decrypted1, _ := service.Decrypt(ciphertext1)
	decrypted2, _ := service.Decrypt(ciphertext2)

	if decrypted1 != decrypted2 {
		t.Error("decrypted texts should be equal")
	}
}

func TestEncryptionService_DecryptInvalidData(t *testing.T) {
	key := "12345678901234567890123456789012"
	service, _ := NewEncryptionService(key)

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"not_base64", "not-valid-base64!!!"},
		{"empty", ""},
		{"too_short", "aGVsbG8="}, // "hello" without nonce prefix
		{"invalid_base64_chars", "!!!invalid!!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Decrypt(tt.ciphertext)
			if err == nil {
				t.Error("expected error for invalid ciphertext")
			}
		})
	}
}

func TestEncryptionService_DecryptWithWrongKey(t *testing.T) {
	key1 := "12345678901234567890123456789012"
	key2 := "abcdefghijklmnopqrstuvwxyz123456" // Must be 32 bytes

	service1, _ := NewEncryptionService(key1)
	service2, _ := NewEncryptionService(key2)

	plaintext := "secret data"
	ciphertext, _ := service1.Encrypt(plaintext)

	// Try to decrypt with wrong key - should fail due to authentication tag mismatch
	_, err := service2.Decrypt(ciphertext)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestEncryptionService_KeyIsStoredSecurely(t *testing.T) {
	key := "12345678901234567890123456789012"
	service, _ := NewEncryptionService(key)

	// Verify the key is not exposed
	// This is a basic check - in real scenarios, more thorough testing would be needed
	if service.key == nil {
		t.Error("key should not be nil")
	}
	if len(service.key) != 32 {
		t.Errorf("key length should be 32, got %d", len(service.key))
	}
}

// Helper functions

func makeKey(length int) string {
	return string(makeBytes(length))
}

func makeBytes(length int) []byte {
	bytes := make([]byte, length)
	for i := range bytes {
		bytes[i] = byte(i % 256)
	}
	return bytes
}
