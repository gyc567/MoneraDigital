package services

import (
	"context"
)

type SafeheronService struct {
	// config...
}

func NewSafeheronService() *SafeheronService {
	return &SafeheronService{}
}

type SafeheronWithdrawalResponse struct {
	TxHash           string
	SafeheronOrderID string
	NetworkFee       string
}

func (s *SafeheronService) Withdraw(ctx context.Context, req SafeheronWithdrawalRequest) (*SafeheronWithdrawalResponse, error) {
	// Stub implementation
	// In real life, call Safeheron API
	return &SafeheronWithdrawalResponse{
		TxHash:           "0xmocktxhash",
		SafeheronOrderID: "mock-sh-id",
		NetworkFee:       "1.0",
	}, nil
}

type SafeheronWithdrawalRequest struct {
	CoinType  string
	ChainType string
	ToAddress string
	Amount    string
	RequestID string
}
