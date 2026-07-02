package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cdoan1/rosa-hyperfleet-api/sdk/pkg/types"
)

// ListManagementClustersOptions contains options for listing management clusters
type ListManagementClustersOptions struct {
	Page int
	Size int
}

// ManagementClusterList represents a paginated list of management clusters
type ManagementClusterList struct {
	Kind  string                    `json:"kind"`
	Page  int                       `json:"page"`
	Size  int                       `json:"size"`
	Total int                       `json:"total"`
	Items []types.ManagementCluster `json:"items"`
}

// CreateManagementCluster creates a new management cluster
func (c *Client) CreateManagementCluster(ctx context.Context, req *types.ManagementClusterRequest) (*types.ManagementCluster, error) {
	var result types.ManagementCluster
	if err := c.do(ctx, http.MethodPost, "/api/v0/management_clusters", req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListManagementClusters retrieves a list of management clusters
func (c *Client) ListManagementClusters(ctx context.Context, opts *ListManagementClustersOptions) (*ManagementClusterList, error) {
	if opts == nil {
		opts = &ListManagementClustersOptions{Page: 1, Size: 100}
	}

	params := map[string]string{}
	if opts.Page > 0 {
		params["page"] = strconv.Itoa(opts.Page)
	}
	if opts.Size > 0 {
		params["size"] = strconv.Itoa(opts.Size)
	}

	path := buildURL("/api/v0/management_clusters", params)

	var result ManagementClusterList
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetManagementCluster retrieves a management cluster by ID
func (c *Client) GetManagementCluster(ctx context.Context, id string) (*types.ManagementCluster, error) {
	if id == "" {
		return nil, fmt.Errorf("management cluster ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/management_clusters/%s", id)

	var result types.ManagementCluster
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
