package dto

// CreateWalletRequest defines the request for creating a wallet.
type CreateWalletRequest struct {
	UserID      string `json:"userId"`
	ProductCode string `json:"productCode"`
	Currency    string `json:"currency"`
}

// CreateWalletResponse defines the response for creating a wallet.
type CreateWalletResponse struct {
	Code      string             `json:"code"`
	Message   string             `json:"message"`
	Data      WalletResponseData `json:"data"`
	Success   bool               `json:"success"`
	Timestamp int64              `json:"timestamp"`
}

// WalletResponseData defines the wallet data in response.
type WalletResponseData struct {
	UserID      string `json:"userId"`
	ProductCode string `json:"productCode"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
}
