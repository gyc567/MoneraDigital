package currency

import "strings"

// Supported currencies following token_network format
const (
	USDT_ERC20 = "USDT_ERC20"
	USDT_TRC20 = "USDT_TRC20"
	USDT_BEP20 = "USDT_BEP20_BINANCE_SMART_CHAIN_MAINNET"
	USDC_ERC20 = "USDC_ERC20"
	USDC_TRC20 = "USDC_TRC20"
	USDC_BEP20 = "USDC_BEP20_BINANCE_SMART_CHAIN_MAINNET"
)

// ShortFormatToFull maps short currency codes to their full backend format
var ShortFormatToFull = map[string]string{
	"USDT_ERC20": USDT_ERC20,
	"USDT_TRC20": USDT_TRC20,
	"USDT_BEP20": USDT_BEP20,
	"USDC_ERC20": USDC_ERC20,
	"USDC_TRC20": USDC_TRC20,
	"USDC_BEP20": USDC_BEP20,
}

// SupportedNetworks contains all valid network identifiers
var SupportedNetworks = []string{
	"ERC20",
	"TRC20",
	"BEP20",
}

// NetworkAliasMap maps common aliases to standard network names
var NetworkAliasMap = map[string]string{
	"TRON": "TRC20",
	"BSC":  "BEP20",
	"ETH":  "ERC20",
}

// NormalizeNetwork converts network aliases to standard names
func NormalizeNetwork(network string) string {
	if normalized, ok := NetworkAliasMap[network]; ok {
		return normalized
	}
	return network
}

// SupportedCurrencies contains all valid currency tokens
var SupportedCurrencies = []string{
	"USDT",
	"USDC",
}

// AllSupportedCurrencies returns all valid currency strings
var AllSupportedCurrencies = []string{
	USDT_ERC20,
	USDT_TRC20,
	USDT_BEP20,
	USDC_ERC20,
	USDC_TRC20,
	USDC_BEP20,
}

// CurrencyLabelMap provides display labels for currencies (uses full format as key)
var CurrencyLabelMap = map[string]string{
	USDT_ERC20: "USDT (ERC20)",
	USDT_TRC20: "USDT (TRC20)",
	USDT_BEP20: "USDT (BEP20)",
	USDC_ERC20: "USDC (ERC20)",
	USDC_TRC20: "USDC (TRC20)",
	USDC_BEP20: "USDC (BEP20)",
}

// ShortFormatLabelMap provides display labels for short currency codes
var ShortFormatLabelMap = map[string]string{
	"USDT_ERC20": "USDT (ERC20)",
	"USDT_TRC20": "USDT (TRC20)",
	"USDT_BEP20": "USDT (BEP20)",
	"USDC_ERC20": "USDC (ERC20)",
	"USDC_TRC20": "USDC (TRC20)",
	"USDC_BEP20": "USDC (BEP20)",
}

// NetworkFromCurrency extracts the network part from currency (e.g., "ERC20" from "USDT_ERC20")
// For single-token formats like "ETH", returns an empty string
func NetworkFromCurrency(currency string) string {
	parts := strings.SplitN(currency, "_", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// TokenFromCurrency extracts the token part from currency (e.g., "USDT" from "USDT_ERC20")
func TokenFromCurrency(currency string) string {
	parts := strings.SplitN(currency, "_", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return currency
}

// IsValid checks if the currency format is valid (TOKEN_NETWORK)
func IsValid(currency string) bool {
	if currency == "" {
		return false
	}

	// Check if it's in the supported list (full format)
	for _, c := range AllSupportedCurrencies {
		if c == currency {
			return true
		}
	}

	// Also accept short format (will be converted to full format)
	for _, c := range ShortFormatToFull {
		if c == currency {
			return true
		}
	}
	return false
}

// ToFullFormat converts short currency format to full backend format
// e.g., "USDT_BEP20" -> "USDT_BEP20_BINANCE_SMART_CHAIN_MAINNET"
func ToFullFormat(currency string) string {
	if full, ok := ShortFormatToFull[currency]; ok {
		return full
	}
	return currency
}

// BuildCurrency creates a currency string from token and network
func BuildCurrency(token, network string) string {
	return token + "_" + network
}

// FormatForDisplay returns a human-readable label for the currency
// Handles both full format (USDC_BEP20_BINANCE_SMART_CHAIN_MAINNET) and short format (USDC_BEP20)
func FormatForDisplay(currency string) string {
	// Try full format first
	if label, ok := CurrencyLabelMap[currency]; ok {
		return label
	}
	// Try short format
	if label, ok := ShortFormatLabelMap[currency]; ok {
		return label
	}
	return currency
}

// SupportedCurrenciesOptions returns frontend options for currency selection
func SupportedCurrenciesOptions() []struct {
	Value string
	Label string
} {
	options := make([]struct {
		Value string
		Label string
	}, len(AllSupportedCurrencies))

	for i, c := range AllSupportedCurrencies {
		options[i] = struct {
			Value string
			Label string
		}{
			Value: c,
			Label: CurrencyLabelMap[c],
		}
	}
	return options
}
