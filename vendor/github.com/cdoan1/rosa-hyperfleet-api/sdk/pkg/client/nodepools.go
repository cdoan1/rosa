package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cdoan1/rosa-hyperfleet-api/sdk/pkg/types"
)

// ListNodePoolsOptions contains options for listing nodepools
type ListNodePoolsOptions struct {
	Limit     int
	Offset    int
	ClusterID string
}

// NodePoolList represents a paginated list of nodepools
type NodePoolList struct {
	Items  []types.NodePool `json:"items"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// ListNodePools retrieves a list of nodepools
func (c *Client) ListNodePools(ctx context.Context, opts *ListNodePoolsOptions) (*NodePoolList, error) {
	if opts == nil {
		opts = &ListNodePoolsOptions{Limit: 50}
	}

	params := map[string]string{}
	if opts.Limit > 0 {
		params["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Offset > 0 {
		params["offset"] = strconv.Itoa(opts.Offset)
	}
	if opts.ClusterID != "" {
		params["clusterId"] = opts.ClusterID
	}

	path := buildURL("/api/v0/nodepools", params)

	var result NodePoolList
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateNodePool creates a new nodepool
func (c *Client) CreateNodePool(ctx context.Context, req *types.NodePoolCreateRequest) (*types.NodePool, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	var result types.NodePool
	if err := c.do(ctx, http.MethodPost, "/api/v0/nodepools", req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetNodePool retrieves a nodepool by ID
func (c *Client) GetNodePool(ctx context.Context, nodePoolID string) (*types.NodePool, error) {
	if nodePoolID == "" {
		return nil, fmt.Errorf("nodepool ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/nodepools/%s", nodePoolID)

	var result types.NodePool
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateNodePool updates an existing nodepool
func (c *Client) UpdateNodePool(ctx context.Context, nodePoolID string, req *types.NodePoolUpdateRequest) (*types.NodePool, error) {
	if nodePoolID == "" {
		return nil, fmt.Errorf("nodepool ID cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	path := fmt.Sprintf("/api/v0/nodepools/%s", nodePoolID)

	var result types.NodePool
	if err := c.do(ctx, http.MethodPut, path, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteNodePool deletes a nodepool
func (c *Client) DeleteNodePool(ctx context.Context, nodePoolID string) error {
	if nodePoolID == "" {
		return fmt.Errorf("nodepool ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/nodepools/%s", nodePoolID)

	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// GetNodePoolStatus retrieves the status of a nodepool
func (c *Client) GetNodePoolStatus(ctx context.Context, nodePoolID string) (*types.NodePoolStatusResponse, error) {
	if nodePoolID == "" {
		return nil, fmt.Errorf("nodepool ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/nodepools/%s/status", nodePoolID)

	var result types.NodePoolStatusResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
