package account

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"monera-digital/internal/dto"
	"monera-digital/internal/logger"
	"net/url"

	"go.uber.org/zap"
)

// APIError represents an error from the account system API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error: status_code=%d, message=%s", e.StatusCode, e.Message)
}

// Client is a client for the account system API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.SugaredLogger
}

// NewClient creates a new account system API client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger.GetLogger().With("service", "AccountClient"),
	}
}

// GetUserAccounts retrieves all accounts for a given user.
func (c *Client) GetUserAccounts(ctx context.Context, req dto.GetUserAccountsRequest) (*dto.GetUserAccountsResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/accounts?userId=%s", c.baseURL, req.UserID), nil)
	if err != nil {
		return nil, err
	}

	var resp dto.GetUserAccountsResponse
	if err := c.doRequest(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateAccount creates a new account for a user.
func (c *Client) CreateAccount(ctx context.Context, req dto.CreateAccountRequest) (*dto.CreateAccountResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/accounts", c.baseURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	var resp dto.CreateAccountResponse
	if err := c.doRequest(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAccountHistory retrieves the transaction history for an account.
func (c *Client) GetAccountHistory(ctx context.Context, req dto.GetAccountHistoryRequest) (*dto.GetAccountHistoryResponse, error) {
	u, err := url.Parse(fmt.Sprintf("%s/accounts/history", c.baseURL))
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("accountId", req.AccountID)
	q.Set("userId", req.UserID)
	q.Set("currency", req.Currency)
	q.Set("startTime", req.StartTime)
	q.Set("endTime", req.EndTime)
	q.Set("page", fmt.Sprintf("%d", req.Page))
	q.Set("size", fmt.Sprintf("%d", req.Size))
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	var resp dto.GetAccountHistoryResponse
	if err := c.doRequest(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// FreezeBalance freezes a specified amount in an account.
func (c *Client) FreezeBalance(ctx context.Context, req dto.FreezeBalanceRequest) (*dto.FreezeBalanceResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/accounts/freeze", c.baseURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	var resp dto.FreezeBalanceResponse
	if err := c.doRequest(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UnfreezeBalance unfreezes a specified amount in an account.
func (c *Client) UnfreezeBalance(ctx context.Context, req dto.UnfreezeBalanceRequest) (*dto.UnfreezeBalanceResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/accounts/unfreeze", c.baseURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	var resp dto.UnfreezeBalanceResponse
	if err := c.doRequest(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Transfer moves funds between two accounts.
func (c *Client) Transfer(ctx context.Context, req dto.TransferRequest) (*dto.TransferResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/accounts/transfer", c.baseURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	var resp dto.TransferResponse
	if err := c.doRequest(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) doRequest(req *http.Request, v interface{}) error {
	req.Header.Set("Content-Type", "application/json")
	// TODO: Add authentication headers

	l := c.logger.With("method", req.Method, "url", req.URL.String())
	l.Debug("sending API request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		l.Errorw("API request failed", "error", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		l.Errorw("failed to read response body", "error", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		l.Warnw("API request returned non-200 status", "status_code", resp.StatusCode, "body", string(body))
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	if err := json.Unmarshal(body, v); err != nil {
		l.Errorw("failed to decode response", "error", err, "body", string(body))
		return fmt.Errorf("failed to decode response: %w", err)
	}

	l.Debug("API request successful")
	return nil
}
