package services

import (
	"os"
	"testing"
)

func TestGenerateAddress_EnvConfig(t *testing.T) {
	service := &WalletService{}

	// Case 1: Default address when env var not set
	os.Unsetenv("WALLET_ADDR_USDT_ERC20")
	addr1 := service.generateAddress("USDT_ERC20")
	if addr1 == "" {
		t.Error("Expected default address, got empty")
	}

	// Case 2: Custom address from env var
	expectedAddr := "0xCustomAddress123"
	os.Setenv("WALLET_ADDR_USDT_ERC20", expectedAddr)
	defer os.Unsetenv("WALLET_ADDR_USDT_ERC20")

	addr2 := service.generateAddress("USDT_ERC20")
	if addr2 != expectedAddr {
		t.Errorf("Expected custom address %s, got %s", expectedAddr, addr2)
	}
}
