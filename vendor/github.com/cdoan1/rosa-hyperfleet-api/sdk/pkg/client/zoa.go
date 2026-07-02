package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// TrustedActionInfo represents information about a trusted action
type TrustedActionInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// TrustedActionRequest represents a request to run a trusted action
type TrustedActionRequest struct {
	ClusterID string                 `json:"cluster_id,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
}

// TrustedActionResponse represents the response from running a trusted action
type TrustedActionResponse struct {
	ExecutionID string                 `json:"execution_id"`
	Status      string                 `json:"status"`
	Output      map[string]interface{} `json:"output,omitempty"`
}

// ListTrustedActionsOptions contains options for listing trusted actions
type ListTrustedActionsOptions struct {
	Page int
	Size int
}

// TrustedActionList represents a paginated list of trusted actions
type TrustedActionList struct {
	Kind  string              `json:"kind"`
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
	Total int                 `json:"total"`
	Items []TrustedActionInfo `json:"items"`
}

// RunTrustedAction executes a trusted action
func (c *Client) RunTrustedAction(ctx context.Context, action string, req *TrustedActionRequest) (*TrustedActionResponse, error) {
	if action == "" {
		return nil, fmt.Errorf("action name cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	path := fmt.Sprintf("/api/v0/trusted-actions/%s/run", action)

	var result TrustedActionResponse
	if err := c.do(ctx, http.MethodPost, path, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListTrustedActions retrieves a list of available trusted actions
func (c *Client) ListTrustedActions(ctx context.Context, opts *ListTrustedActionsOptions) (*TrustedActionList, error) {
	if opts == nil {
		opts = &ListTrustedActionsOptions{Page: 1, Size: 100}
	}

	params := map[string]string{}
	if opts.Page > 0 {
		params["page"] = strconv.Itoa(opts.Page)
	}
	if opts.Size > 0 {
		params["size"] = strconv.Itoa(opts.Size)
	}

	path := buildURL("/api/v0/trusted-actions", params)

	var result TrustedActionList
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetTrustedAction retrieves details about a specific trusted action
func (c *Client) GetTrustedAction(ctx context.Context, action string) (*TrustedActionInfo, error) {
	if action == "" {
		return nil, fmt.Errorf("action name cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/trusted-actions/%s", action)

	var result TrustedActionInfo
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ExecutionResponse represents the response for a trusted action execution
type ExecutionResponse struct {
	ExecutionID string                 `json:"execution_id"`
	Status      string                 `json:"status"`
	Output      map[string]interface{} `json:"output,omitempty"`
	CreatedAt   string                 `json:"created_at,omitempty"`
	CompletedAt string                 `json:"completed_at,omitempty"`
}

// GetTrustedActionRun retrieves the details of a specific trusted action run
func (c *Client) GetTrustedActionRun(ctx context.Context, runID string) (*ExecutionResponse, error) {
	if runID == "" {
		return nil, fmt.Errorf("run ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/trusted-actions/runs/%s", runID)

	var result ExecutionResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
