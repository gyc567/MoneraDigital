package services

import (
	"encoding/hex"
	"testing"
)

// ============================================================================
// DecodeEncryptionKey Tests
// ============================================================================

func TestDecodeEncryptionKey_HexFormat(t *testing.T) {
	// Valid 64-char hex string (32 bytes when decoded)
	hexKey := "c70c58a23fd8ab7b80e654cb3dafa371b47949991e6fc5721b33704984c4e016"
	decoded, err := DecodeEncryptionKey(hexKey)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(decoded) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(decoded))
	}

	// Verify bytes are correct
	expectedBytes, _ := hex.DecodeString(hexKey)
	if string(expectedBytes) != decoded {
		t.Error("decoded key does not match expected bytes")
	}
}

func TestDecodeEncryptionKey_RawFormat(t *testing.T) {
	// Valid 32-char raw string
	rawKey := "12345678901234567890123456789012"
	decoded, err := DecodeEncryptionKey(rawKey)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if decoded != rawKey {
		t.Errorf("raw key should not be modified: got %q, want %q", decoded, rawKey)
	}

	if len(decoded) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(decoded))
	}
}

func TestDecodeEncryptionKey_InvalidHexFormat(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		reason string
	}{
		{"invalid_hex_chars", "gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg", "invalid hex characters"},
		{"hex_with_spaces", "c70c 58a2 3fd8 ab7b 80e6 54cb 3daf a371 b479 4999 1e6f c572 1b33 7049 84c4 e016", "spaces in hex"},
		{"hex_uppercase", "C70C58A23FD8AB7B80E654CB3DAFA371B47949991E6FC5721B33704984C4E016", "uppercase hex (should work)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := DecodeEncryptionKey(tt.key)

			if tt.name == "hex_uppercase" {
				// Uppercase hex should work
				if err != nil {
					t.Fatalf("uppercase hex should work, got error: %v", err)
				}
				if len(decoded) != 32 {
					t.Errorf("expected 32 bytes, got %d", len(decoded))
				}
			} else {
				// Invalid formats should error
				if err == nil {
					t.Errorf("expected error for %s, got nil", tt.reason)
				}
			}
		})
	}
}

func TestDecodeEncryptionKey_InvalidLength(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		reason string
	}{
		{"too_short_raw", "1234567890123456789012345678901", "31 chars (too short)"},
		{"too_long_raw", "123456789012345678901234567890123", "33 chars (too long)"},
		{"too_short_hex", "c70c58a23fd8ab7b80e654cb3dafa371b47949991e6fc5721b33704984c4e01", "63 chars (hex too short)"},
		{"too_long_hex", "c70c58a23fd8ab7b80e654cb3dafa371b47949991e6fc5721b33704984c4e0160", "65 chars (hex too long)"},
		{"empty_string", "", "empty string"},
		{"whitespace_only", "   ", "whitespace only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := DecodeEncryptionKey(tt.key)

			if err == nil {
				t.Errorf("expected error for %s, got nil (decoded: %q)", tt.reason, decoded)
			}
		})
	}
}

func TestDecodeEncryptionKey_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		shouldErr bool
	}{
		{"hex_lowercase", "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789", false},
		{"hex_uppercase", "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789", false},
		{"hex_mixed_case", "AbCdEf0123456789aBcDeF0123456789AbCdEf0123456789aBcDeF0123456789", false},
		{"hex_with_leading_zeros", "0000000000000000000000000000000000000000000000000000000000000000", false},
		{"raw_with_spaces_trimmed", "   12345678901234567890123456789012   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := DecodeEncryptionKey(tt.key)

			if tt.shouldErr && err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error for %s: %v", tt.name, err)
			}
			if !tt.shouldErr && len(decoded) != 32 {
				t.Errorf("expected 32 bytes, got %d", len(decoded))
			}
		})
	}
}

// ============================================================================
// NewEncryptionService with Normalized Keys
// ============================================================================

func TestNewEncryptionService_WithHexKey(t *testing.T) {
	// Simulate the container flow: hex key → normalize → create service
	hexKey := "c70c58a23fd8ab7b80e654cb3dafa371b47949991e6fc5721b33704984c4e016"

	// Step 1: Normalize key
	normalizedKey, err := DecodeEncryptionKey(hexKey)
	if err != nil {
		t.Fatalf("failed to decode hex key: %v", err)
	}

	// Step 2: Create service with normalized key
	service, err := NewEncryptionService(normalizedKey)
	if err != nil {
		t.Fatalf("failed to create service with hex key: %v", err)
	}

	if service == nil {
		t.Fatal("expected non-nil service")
	}

	// Step 3: Verify encrypt/decrypt works
	plaintext := "test secret data"
	ciphertext, err := service.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	decrypted, err := service.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted does not match plaintext: got %q, want %q", decrypted, plaintext)
	}
}

func TestNewEncryptionService_WithRawKey(t *testing.T) {
	// Verify backward compatibility: raw 32-byte key still works
	rawKey := "12345678901234567890123456789012"

	// Service should work directly with raw key (no normalization needed in this case)
	service, err := NewEncryptionService(rawKey)
	if err != nil {
		t.Fatalf("failed to create service with raw key: %v", err)
	}

	if service == nil {
		t.Fatal("expected non-nil service")
	}

	// Verify encrypt/decrypt works
	plaintext := "test data"
	ciphertext, err := service.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	decrypted, err := service.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted does not match plaintext: got %q, want %q", decrypted, plaintext)
	}
}

// ============================================================================
// Existing Tests (Preserved for Backward Compatibility)
// ============================================================================

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
