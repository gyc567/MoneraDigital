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
		{"valid USDT_TRON_TESTNET", USDT_TRON_TESTNET, true},
		{"valid USDC_ERC20", USDC_ERC20, true},
		{"valid USDC_TRC20", USDC_TRC20, true},
		{"valid USDC_BEP20 (full format)", USDC_BEP20, true}, // 长格式
		{"valid USDC_BEP20 (short format)", "USDC_BEP20", true}, // 短格式也接受
		{"valid USDC_TRON_TESTNET", USDC_TRON_TESTNET, true},
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
		{"USDT_BEP20", USDT_BEP20, "BEP20"},
		{"USDC_ERC20", USDC_ERC20, "ERC20"},
		{"USDC_TRC20", USDC_TRC20, "TRC20"},
		{"USDC_BEP20 (full format)", USDC_BEP20, "BEP20"}, // 长格式也返回BEP20
		{"single token", "ETH", ""},
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
		{"USDT_BEP20", USDT_BEP20, "USDT"},
		{"USDC_ERC20", USDC_ERC20, "USDC"},
		{"USDC_TRC20", USDC_TRC20, "USDC"},
		{"USDC_BEP20 (full format)", USDC_BEP20, "USDC"}, // 长格式返回USDC
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
		{"USDT ERC20", "USDT", "ERC20", "USDT_ERC20"},
		{"USDT TRC20", "USDT", "TRC20", "USDT_TRC20"},
		{"USDT BEP20", "USDT", "BEP20", "USDT_BEP20"},
		{"USDC ERC20", "USDC", "ERC20", "USDC_ERC20"},
		{"USDC TRC20", "USDC", "TRC20", "USDC_TRC20"},
		{"USDC BEP20", "USDC", "BEP20", "USDC_BEP20"}, // 返回短格式
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
		{"USDT_BEP20", USDT_BEP20, "USDT (BEP20)"},
		{"USDC_BEP20 (full format)", USDC_BEP20, "USDC (BEP20)"},
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
		{"USDT ERC20 (short)", "USDT_ERC20", "USDT_ERC20"}, // 不需要转换
		{"USDT TRC20 (short)", "USDT_TRC20", "USDT_TRC20"}, // 不需要转换
		{"USDT BEP20 (short)", "USDT_BEP20", "USDT_BEP20"}, // 不需要转换
		{"USDC ERC20 (short)", "USDC_ERC20", "USDC_ERC20"}, // 不需要转换
		{"USDC TRC20 (short)", "USDC_TRC20", "USDC_TRC20"}, // 不需要转换
		{"USDC BEP20 (short to full)", "USDC_BEP20", USDC_BEP20}, // 特例：需要转换
		{"USDC BEP20 (already full)", USDC_BEP20, USDC_BEP20}, // 已经是长格式
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

func TestToShortFormat(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     string
	}{
		{"USDT ERC20", "USDT_ERC20", "USDT_ERC20"}, // 不需要转换
		{"USDT BEP20", "USDT_BEP20", "USDT_BEP20"}, // 不需要转换
		{"USDC BEP20 (full to short)", USDC_BEP20, "USDC_BEP20"}, // 特例：需要转换
		{"USDC BEP20 (already short)", "USDC_BEP20", "USDC_BEP20"}, // 已经是短格式
		{"unknown currency", "UNKNOWN_TOKEN", "UNKNOWN_TOKEN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToShortFormat(tt.currency); got != tt.want {
				t.Errorf("ToShortFormat(%q) = %v, want %v", tt.currency, got, tt.want)
			}
		})
	}
}
