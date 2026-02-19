package coreapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Client is a client for the Core API (Monnaire Core System).
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// CoreAPIResponse represents the response from Core API.
type CoreAPIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
	Code    interface{} `json:"code"`
}

// NewClient creates a new Core API client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateWalletRequest represents the request to create a wallet.
type CreateWalletRequest struct {
	UserID      int    `json:"userId"`
	ProductCode string `json:"productCode"`
	Currency    string `json:"currency"`
}

// CreateWalletResponse represents the response from wallet creation.
type CreateWalletResponse struct {
	WalletID  string            `json:"walletId"`
	Address   string            `json:"address"`
	Addresses map[string]string `json:"addresses"`
	Status    string            `json:"status"`
	CreatedAt string            `json:"createdAt"`
}

// CreateWallet calls the Core API to create a wallet.
func (c *Client) CreateWallet(ctx context.Context, req CreateWalletRequest) (*CreateWalletResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/wallet/create", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	var resp CoreAPIResponse
	if err := c.doRequest(httpReq, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("wallet creation failed: %s", resp.Message)
	}

	respData, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var walletResp CreateWalletResponse
	if err := json.Unmarshal(respData, &walletResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &walletResp, nil
}

// GetIncomeHistoryRequest 请求结构
type GetIncomeHistoryRequest struct {
	Address string `json:"address"`
}

// GetAddressRequest 请求结构
type GetAddressRequest struct {
	UserID      string `json:"userId"`
	ProductCode string `json:"productCode"`
	Currency    string `json:"currency"`
}

// AddressInfo 钱包地址信息
type AddressInfo struct {
	Address     string  `json:"address"`
	AddressType *string `json:"addressType"`
	DerivePath  *string `json:"derivePath"`
}

// AddressIncomeRecord 收款记录结构
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

// GetAddress 获取钱包地址
func (c *Client) GetAddress(ctx context.Context, req GetAddressRequest) (*AddressInfo, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/wallet/address/get", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	var resp CoreAPIResponse
	if err := c.doRequest(httpReq, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("get address failed: %s", resp.Message)
	}

	respData, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var addressInfo AddressInfo
	if err := json.Unmarshal(respData, &addressInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &addressInfo, nil
}

// GetIncomeHistory 获取地址链上收款记录
func (c *Client) GetIncomeHistory(ctx context.Context, req GetIncomeHistoryRequest) ([]AddressIncomeRecord, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/wallet/address/incomeHistory", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	var resp CoreAPIResponse
	if err := c.doRequest(httpReq, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("income history query failed: %s", resp.Message)
	}

	respData, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var incomeRecords []AddressIncomeRecord
	if err := json.Unmarshal(respData, &incomeRecords); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return incomeRecords, nil
}

func (c *Client) doRequest(req *http.Request, v interface{}) error {
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// CoreAPIClientInterface defines the interface for Core API client operations.
// This allows for easier testing with mocks.
type CoreAPIClientInterface interface {
	CreateWallet(ctx context.Context, req CreateWalletRequest) (*CreateWalletResponse, error)
	GetAddress(ctx context.Context, req GetAddressRequest) (*AddressInfo, error)
	GetIncomeHistory(ctx context.Context, req GetIncomeHistoryRequest) ([]AddressIncomeRecord, error)
}

var _ CoreAPIClientInterface = (*Client)(nil)
