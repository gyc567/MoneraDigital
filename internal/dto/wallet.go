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

// AddWalletAddressRequest defines the request for adding a wallet address.
type AddWalletAddressRequest struct {
	Chain string `json:"chain"`
	Token string `json:"token"`
}

// GetAddressIncomeHistoryRequest defines the request for getting address income history.
type GetAddressIncomeHistoryRequest struct {
	Address string `json:"address" binding:"required"`
}

// AddressIncomeRecord defines the address income record.
type AddressIncomeRecord struct {
	TxKey             string `json:"txKey"`
	TxHash            string `json:"txHash"`
	CoinKey           string `json:"coinKey"`
	TxAmount          string `json:"txAmount"`
	Address           string `json:"address"`
	TransactionStatus string `json:"transactionStatus"`
	BlockHeight       int64  `json:"blockHeight"`
	CreateTime        string `json:"createTime"`
	CompletedTime     string `json:"completedTime"`
}

// GetAddressIncomeHistoryResponse defines the response for getting address income history.
type GetAddressIncomeHistoryResponse struct {
	Code      string                `json:"code"`
	Message   string                `json:"message"`
	Data      []AddressIncomeRecord `json:"data"`
	Success   bool                  `json:"success"`
	Timestamp int64                 `json:"timestamp"`
}

// GetWalletAddressRequest defines the request for getting wallet address.
type GetWalletAddressRequest struct {
	UserID      string `json:"userId"`
	ProductCode string `json:"productCode"`
	Currency    string `json:"currency"`
}

// WalletAddress defines the wallet address information.
type WalletAddress struct {
	Address     string  `json:"address"`
	AddressType *string `json:"addressType,omitempty"`
	DerivePath  *string `json:"derivePath,omitempty"`
}

// GetWalletAddressResponse defines the response for getting wallet address.
type GetWalletAddressResponse struct {
	Code      string        `json:"code"`
	Message   string        `json:"message"`
	Data      WalletAddress `json:"data"`
	Success   bool          `json:"success"`
	Timestamp int64         `json:"timestamp"`
}
