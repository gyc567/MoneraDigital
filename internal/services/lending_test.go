package services

import (
	"math"
	"testing"
)

func TestLendingService_CalculateAPY(t *testing.T) {
	ls := &LendingService{}

	tests := []struct {
		asset        string
		durationDays int
		expected     string
	}{
		{"BTC", 30, "4.50"},
		{"ETH", 90, "5.72"},
		{"USDT", 180, "10.62"},
		{"USDC", 360, "12.30"},
		{"UNKNOWN", 30, "5.00"},
	}

	for _, test := range tests {
		result := ls.CalculateAPY(test.asset, test.durationDays)
		if result != test.expected {
			t.Errorf("CalculateAPY(%s, %d) = %s; expected %s", test.asset, test.durationDays, result, test.expected)
		}
	}
}

func TestLendingService_CalculateEstimatedYield(t *testing.T) {
	ls := &LendingService{}

	tests := []struct {
		amount       float64
		apy          float64
		durationDays int
		expected     float64
	}{
		{1000, 5.0, 365, 50.0},   // Correct calculation: (1000 * 0.05 * 365) / 365 = 50
		{1000, 10.0, 182, 49.86}, // (1000 * 0.1 * 182) / 365 ≈ 49.86
	}

	for _, test := range tests {
		result := ls.CalculateEstimatedYield(test.amount, test.apy, test.durationDays)
		if math.Abs(result-test.expected) > 0.01 {
			t.Errorf("CalculateEstimatedYield(%.2f, %.2f, %d) = %.2f; expected %.2f",
				test.amount, test.apy, test.durationDays, result, test.expected)
		}
	}
}
