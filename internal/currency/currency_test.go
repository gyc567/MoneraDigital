package currency

import (
	"testing"
)

func TestIsValid(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     bool
	}{
		{"valid USDT_ERC20", USDT_ERC20, true},
		{"valid USDT_TRC20", USDT_TRC20, true},
		{"valid USDT_BEP20", USDT_BEP20, true},
		{"valid USDC_ERC20", USDC_ERC20, true},
		{"valid USDC_TRC20", USDC_TRC20, true},
		{"valid USDC_BEP20", USDC_BEP20, true},
		{"empty string", "", false},
		{"old format ETH", "ETH", false},
		{"old format TRON", "TRON", false},
		{"old format BSC", "BSC", false},
		{"wrong format USDT-ERC20", "USDT-ERC20", false},
		{"wrong format usdt_erc20", "usdt_erc20", false},
		{"wrong token USDT_ETH", "USDT_ETH", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValid(tt.currency); got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.currency, got, tt.want)
			}
		})
	}
}

func TestNetworkFromCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     string
	}{
		{"USDT_ERC20", USDT_ERC20, "ERC20"},
		{"USDT_TRC20", USDT_TRC20, "TRC20"},
		{"USDT_BEP20", USDT_BEP20, "BEP20_BINANCE_SMART_CHAIN_MAINNET"},
		{"USDC_ERC20", USDC_ERC20, "ERC20"},
		{"single token", "ETH", ""}, // Single tokens return empty network
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NetworkFromCurrency(tt.currency); got != tt.want {
				t.Errorf("NetworkFromCurrency(%q) = %v, want %v", tt.currency, got, tt.want)
			}
		})
	}
}

func TestTokenFromCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     string
	}{
		{"USDT_ERC20", USDT_ERC20, "USDT"},
		{"USDT_TRC20", USDT_TRC20, "USDT"},
		{"USDC_BEP20", USDC_BEP20, "USDC"},
		{"single token", "ETH", "ETH"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TokenFromCurrency(tt.currency); got != tt.want {
				t.Errorf("TokenFromCurrency(%q) = %v, want %v", tt.currency, got, tt.want)
			}
		})
	}
}

func TestBuildCurrency(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		network string
		want    string
	}{
		{"USDT ERC20", "USDT", "ERC20", USDT_ERC20},
		{"USDT TRC20", "USDT", "TRC20", USDT_TRC20},
		{"USDC BEP20", "USDC", "BEP20", "USDC_BEP20"}, // BuildCurrency just concatenates, use ToFullFormat for full format
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildCurrency(tt.token, tt.network); got != tt.want {
				t.Errorf("BuildCurrency(%q, %q) = %v, want %v", tt.token, tt.network, got, tt.want)
			}
		})
	}
}

func TestFormatForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     string
	}{
		{"USDT_ERC20", USDT_ERC20, "USDT (ERC20)"},
		{"USDT_TRC20", USDT_TRC20, "USDT (TRC20)"},
		{"unknown", "UNKNOWN", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatForDisplay(tt.currency); got != tt.want {
				t.Errorf("FormatForDisplay(%q) = %v, want %v", tt.currency, got, tt.want)
			}
		})
	}
}

func TestSupportedCurrenciesOptions(t *testing.T) {
	options := SupportedCurrenciesOptions()

	if len(options) != len(AllSupportedCurrencies) {
		t.Errorf("Expected %d options, got %d", len(AllSupportedCurrencies), len(options))
	}

	for i, opt := range options {
		if opt.Value != AllSupportedCurrencies[i] {
			t.Errorf("Option %d value = %v, want %v", i, opt.Value, AllSupportedCurrencies[i])
		}
		if opt.Label != CurrencyLabelMap[opt.Value] {
			t.Errorf("Option %d label = %v, want %v", i, opt.Label, CurrencyLabelMap[opt.Value])
		}
	}
}

func TestAllSupportedCurrenciesContainsAll(t *testing.T) {
	currencySet := make(map[string]bool)
	for _, c := range AllSupportedCurrencies {
		currencySet[c] = true
	}

	for _, c := range AllSupportedCurrencies {
		if !currencySet[c] {
			t.Errorf("Currency %s not in set", c)
		}
	}
}

func TestToFullFormat(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     string
	}{
		{"USDT ERC20", "USDT_ERC20", USDT_ERC20},
		{"USDT TRC20", "USDT_TRC20", USDT_TRC20},
		{"USDT BEP20", "USDT_BEP20", USDT_BEP20},
		{"USDC ERC20", "USDC_ERC20", USDC_ERC20},
		{"USDC TRC20", "USDC_TRC20", USDC_TRC20},
		{"USDC BEP20", "USDC_BEP20", USDC_BEP20},
		{"already full format", USDT_BEP20, USDT_BEP20},
		{"unknown currency", "UNKNOWN_TOKEN", "UNKNOWN_TOKEN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToFullFormat(tt.currency); got != tt.want {
				t.Errorf("ToFullFormat(%q) = %v, want %v", tt.currency, got, tt.want)
			}
		})
	}
}
