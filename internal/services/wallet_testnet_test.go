package services

import (
	"testing"
)

func TestIsTestnetCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     bool
	}{
		{
			name:     "USDT_TRON_TESTNET is testnet",
			currency: "USDT_TRON_TESTNET",
			want:     true,
		},
		{
			name:     "USDC_TRON_TESTNET is testnet",
			currency: "USDC_TRON_TESTNET",
			want:     true,
		},
		{
			name:     "USDT_TRC20 is not testnet",
			currency: "USDT_TRC20",
			want:     false,
		},
		{
			name:     "USDT_ERC20 is not testnet",
			currency: "USDT_ERC20",
			want:     false,
		},
		{
			name:     "empty string is not testnet",
			currency: "",
			want:     false,
		},
		{
			name:     "TRX(SHASTA)_TRON_TESTNET is testnet",
			currency: "TRX(SHASTA)_TRON_TESTNET",
			want:     true,
		},
		{
			name:     "lowercase testnet is testnet",
			currency: "usdt_tron_testnet",
			want:     true,
		},
		{
			name:     "GOERLI is testnet",
			currency: "USDT_GOERLI",
			want:     true,
		},
		{
			name:     "SEPOLIA is testnet",
			currency: "USDC_SEPOLIA",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTestnetCurrency(tt.currency)
			if got != tt.want {
				t.Errorf("isTestnetCurrency(%q) = %v, want %v", tt.currency, got, tt.want)
			}
		})
	}
}

func TestGenerateTestnetAddress(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		userID   int
		wantLen  int
		prefix   string
	}{
		{
			name:     "TRON testnet address",
			currency: "USDT_TRON_TESTNET",
			userID:   123,
			wantLen:  34,
			prefix:   "T",
		},
		{
			name:     "TRC20 testnet address",
			currency: "USDT_TRC20_TESTNET",
			userID:   456,
			wantLen:  34,
			prefix:   "T",
		},
		{
			name:     "BEP20 testnet address",
			currency: "USDT_BEP20_TESTNET",
			userID:   789,
			wantLen:  42,
			prefix:   "0x",
		},
		{
			name:     "ERC20 testnet address",
			currency: "USDT_ERC20_TESTNET",
			userID:   1000,
			wantLen:  42,
			prefix:   "0x",
		},
		{
			name:     "Default testnet address",
			currency: "UNKNOWN_TESTNET",
			userID:   1001,
			wantLen:  42,
			prefix:   "0x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateTestnetAddress(tt.currency, tt.userID)
			if len(got) != tt.wantLen {
				t.Errorf("generateTestnetAddress(%q, %d) length = %d, want %d", tt.currency, tt.userID, len(got), tt.wantLen)
			}
			if got[:len(tt.prefix)] != tt.prefix {
				t.Errorf("generateTestnetAddress(%q, %d) prefix = %q, want %q", tt.currency, tt.userID, got[:len(tt.prefix)], tt.prefix)
			}
		})
	}
}

func TestGenerateTestnetAddress_Deterministic(t *testing.T) {
	// Test that the same inputs produce the same address
	currency := "USDT_TRON_TESTNET"
	userID := 12345

	addr1 := generateTestnetAddress(currency, userID)
	addr2 := generateTestnetAddress(currency, userID)

	if addr1 != addr2 {
		t.Errorf("generateTestnetAddress should be deterministic: %q != %q", addr1, addr2)
	}
}

func TestGenerateTestnetAddress_DifferentUsers(t *testing.T) {
	// Test that different userIDs produce different addresses
	currency := "USDT_TRON_TESTNET"

	addr1 := generateTestnetAddress(currency, 1)
	addr2 := generateTestnetAddress(currency, 2)

	if addr1 == addr2 {
		t.Errorf("Different userIDs should produce different addresses: both got %q", addr1)
	}
}

func TestGeneratePadding(t *testing.T) {
	tests := []struct {
		length int
		seed   string
	}{
		{length: 0, seed: "test"},
		{length: 5, seed: "test"},
		{length: 10, seed: "USDT_TRON_TESTNET"},
		{length: 20, seed: "USDT"},
	}

	for _, tt := range tests {
		got := generatePadding(tt.length, tt.seed)
		if len(got) != tt.length {
			t.Errorf("generatePadding(%d, %q) length = %d, want %d", tt.length, tt.seed, len(got), tt.length)
		}
	}
}

func TestGenerateHexPadding(t *testing.T) {
	tests := []struct {
		length int
	}{
		{length: 0},
		{length: 5},
		{length: 10},
		{length: 20},
	}

	for _, tt := range tests {
		got := generateHexPadding(tt.length)
		if len(got) != tt.length {
			t.Errorf("generateHexPadding(%d) length = %d, want %d", tt.length, len(got), tt.length)
		}
		// Verify all characters are valid hex
		for _, c := range got {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("generateHexPadding(%d) contains non-hex character: %c", tt.length, c)
			}
		}
	}
}

func TestGeneratePadding_Deterministic(t *testing.T) {
	// Same inputs should produce same output
	p1 := generatePadding(10, "seed")
	p2 := generatePadding(10, "seed")
	if p1 != p2 {
		t.Errorf("generatePadding should be deterministic: %q != %q", p1, p2)
	}
}

func TestGenerateHexPadding_Deterministic(t *testing.T) {
	// Same length should produce same output
	p1 := generateHexPadding(10)
	p2 := generateHexPadding(10)
	if p1 != p2 {
		t.Errorf("generateHexPadding should be deterministic: %q != %q", p1, p2)
	}
}
