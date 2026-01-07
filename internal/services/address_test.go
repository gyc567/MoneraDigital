package services

import (
	"testing"
)

func TestGenerateVerificationToken(t *testing.T) {
	token1, err := generateVerificationToken()
	if err != nil {
		t.Fatalf("generateVerificationToken() returned error: %v", err)
	}

	token2, err := generateVerificationToken()
	if err != nil {
		t.Fatalf("generateVerificationToken() returned error: %v", err)
	}

	// Tokens should be unique
	if token1 == token2 {
		t.Error("generateVerificationToken() should generate unique tokens")
	}

	// Token should be 64 characters (32 bytes hex encoded)
	if len(token1) != 64 {
		t.Errorf("generateVerificationToken() returned token of length %d; expected 64", len(token1))
	}
}
