package binance

import (
	"sync"
)

// PriceService handles price fetching and caching
type PriceService struct {
	prices map[string]float64
	mutex  sync.RWMutex
}

// NewPriceService creates a new price service instance
func NewPriceService() *PriceService {
	return &PriceService{
		prices: make(map[string]float64),
		mutex:  sync.RWMutex{},
	}
}

// GetPricesFromCache returns prices for the given currencies
// For now, returns mock prices for testing
func (ps *PriceService) GetPricesFromCache(currencies []string) map[string]float64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	result := make(map[string]float64)
	for _, currency := range currencies {
		if price, exists := ps.prices[currency]; exists {
			result[currency] = price
		} else {
			// Return mock prices for testing
			result[currency] = ps.getMockPrice(currency)
		}
	}
	return result
}

// getMockPrice returns a mock price for a currency (for testing)
func (ps *PriceService) getMockPrice(currency string) float64 {
	mockPrices := map[string]float64{
		"BTC":   45000.0,
		"ETH":   3000.0,
		"ADA":   0.5,
		"SOL":   100.0,
		"DOT":   8.0,
		"LINK":  15.0,
		"UNI":   7.0,
		"AAVE":  100.0,
		"COMP":  60.0,
		"SUSHI": 3.0,
		"YFI":   8000.0,
		"MKR":   1500.0,
		"CRV":   1.5,
		"REN":   0.15,
		"KNC":   1.2,
		"ZRX":   0.3,
		"BAL":   5.0,
		"REP":   25.0,
		"GNT":   0.1,
		"STORJ": 0.8,
		"ANT":   3.5,
		"BAT":   0.25,
		"OMG":   2.0,
		"LRC":   0.4,
		"RLC":   2.5,
	}

	if price, exists := mockPrices[currency]; exists {
		return price
	}

	// Default mock price for unknown currencies
	return 1.0
}

// UpdatePrices would update the price cache (placeholder for future implementation)
func (ps *PriceService) UpdatePrices(prices map[string]float64) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	for currency, price := range prices {
		ps.prices[currency] = price
	}
}
