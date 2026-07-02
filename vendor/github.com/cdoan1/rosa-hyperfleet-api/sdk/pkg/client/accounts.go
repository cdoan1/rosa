package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cdoan1/rosa-hyperfleet-api/sdk/pkg/types"
)

// ListAccountsOptions contains options for listing accounts
type ListAccountsOptions struct {
	Page int
	Size int
}

// AccountList represents a paginated list of accounts
type AccountList struct {
	Kind  string          `json:"kind"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
	Total int             `json:"total"`
	Items []types.Account `json:"items"`
}

// EnableAccount enables an account for use with the platform
func (c *Client) EnableAccount(ctx context.Context, req *types.EnableAccountRequest) (*types.Account, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	var result types.Account
	if err := c.do(ctx, http.MethodPost, "/api/v0/accounts", req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListAccounts retrieves a list of accounts
func (c *Client) ListAccounts(ctx context.Context, opts *ListAccountsOptions) (*AccountList, error) {
	if opts == nil {
		opts = &ListAccountsOptions{Page: 1, Size: 100}
	}

	params := map[string]string{}
	if opts.Page > 0 {
		params["page"] = strconv.Itoa(opts.Page)
	}
	if opts.Size > 0 {
		params["size"] = strconv.Itoa(opts.Size)
	}

	path := buildURL("/api/v0/accounts", params)

	var result AccountList
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetAccount retrieves an account by ID
func (c *Client) GetAccount(ctx context.Context, accountID string) (*types.Account, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/accounts/%s", accountID)

	var result types.Account
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DisableAccount disables an account
func (c *Client) DisableAccount(ctx context.Context, accountID string) error {
	if accountID == "" {
		return fmt.Errorf("account ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/accounts/%s", accountID)

	return c.do(ctx, http.MethodDelete, path, nil, nil)
}
