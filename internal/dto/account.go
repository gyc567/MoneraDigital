package dto

// GetUserAccountsRequest defines the request for getting user accounts.
type GetUserAccountsRequest struct {
	UserID string `json:"userId"`
}

// GetUserAccountsResponse defines the response for getting user accounts.
type GetUserAccountsResponse struct {
	Code    string    `json:"code"`
	Message string    `json:"message"`
	Data    []Account `json:"data"`
}

// Account defines the structure for a user account.
type Account struct {
	AccountID        string `json:"accountId"`
	UserID           string `json:"userId"`
	AccountType      string `json:"accountType"`
	Currency         string `json:"currency"`
	Balance          string `json:"balance"`
	FrozenBalance    string `json:"frozenBalance"`
	AvailableBalance string `json:"availableBalance"`
	Status           string `json:"status"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

// CreateAccountRequest defines the request for creating an account.
type CreateAccountRequest struct {
	UserID      string `json:"userId"`
	AccountType string `json:"accountType"`
	Currency    string `json:"currency"`
}

// CreateAccountResponse defines the response for creating an account.
type CreateAccountResponse struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Data    Account `json:"data"`
}

// GetAccountHistoryRequest defines the request for getting account history.
type GetAccountHistoryRequest struct {
	AccountID string `json:"accountId"`
	UserID    string `json:"userId"`
	Currency  string `json:"currency"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Page      int    `json:"page"`
	Size      int    `json:"size"`
}

// GetAccountHistoryResponse defines the response for getting account history.
type GetAccountHistoryResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    []HistoryRecord `json:"data"`
}

// HistoryRecord defines the structure for an account history record.
type HistoryRecord struct {
	ID              string `json:"id"`
	AccountID       string `json:"accountId"`
	UserID          string `json:"userId"`
	Amount          string `json:"amount"`
	Currency        string `json:"currency"`
	TransactionType string `json:"transactionType"`
	Status          string `json:"status"`
	Description     string `json:"description"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

// FreezeBalanceRequest defines the request for freezing balance.
type FreezeBalanceRequest struct {
	AccountID string `json:"accountId"`
	UserID    string `json:"userId"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
	Reason    string `json:"reason"`
}

// FreezeBalanceResponse defines the response for freezing balance.
type FreezeBalanceResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Success bool `json:"success"`
	} `json:"data"`
}

// UnfreezeBalanceRequest defines the request for unfreezing balance.
type UnfreezeBalanceRequest struct {
	AccountID string `json:"accountId"`
	UserID    string `json:"userId"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
	Reason    string `json:"reason"`
}

// UnfreezeBalanceResponse defines the response for unfreezing balance.
type UnfreezeBalanceResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Success bool `json:"success"`
	} `json:"data"`
}

// TransferRequest defines the request for a transfer.
type TransferRequest struct {
	FromAccountID string `json:"fromAccountId"`
	ToAccountID   string `json:"toAccountId"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	Reason        string `json:"reason"`
}

// TransferResponse defines the response for a transfer.
type TransferResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Success bool `json:"success"`
	} `json:"data"`
}
