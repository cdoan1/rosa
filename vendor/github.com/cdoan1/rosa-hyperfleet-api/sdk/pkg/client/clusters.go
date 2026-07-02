package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cdoan1/rosa-hyperfleet-api/sdk/pkg/types"
)

// ListClustersOptions contains options for listing clusters
type ListClustersOptions struct {
	Limit  int
	Offset int
	Status string
}

// ClusterList represents a paginated list of clusters
type ClusterList struct {
	Items  []types.Cluster `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// ListClusters retrieves a list of clusters
func (c *Client) ListClusters(ctx context.Context, opts *ListClustersOptions) (*ClusterList, error) {
	if opts == nil {
		opts = &ListClustersOptions{Limit: 50}
	}

	params := map[string]string{}
	if opts.Limit > 0 {
		params["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Offset > 0 {
		params["offset"] = strconv.Itoa(opts.Offset)
	}
	if opts.Status != "" {
		params["status"] = opts.Status
	}

	path := buildURL("/api/v0/clusters", params)

	var result ClusterList
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateCluster creates a new cluster
func (c *Client) CreateCluster(ctx context.Context, req *types.ClusterCreateRequest) (*types.Cluster, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	var result types.Cluster
	if err := c.do(ctx, http.MethodPost, "/api/v0/clusters", req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetCluster retrieves a cluster by ID
func (c *Client) GetCluster(ctx context.Context, clusterID string) (*types.Cluster, error) {
	if clusterID == "" {
		return nil, fmt.Errorf("cluster ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/clusters/%s", clusterID)

	var result types.Cluster
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateCluster updates an existing cluster
func (c *Client) UpdateCluster(ctx context.Context, clusterID string, req *types.ClusterUpdateRequest) (*types.Cluster, error) {
	if clusterID == "" {
		return nil, fmt.Errorf("cluster ID cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	path := fmt.Sprintf("/api/v0/clusters/%s", clusterID)

	var result types.Cluster
	if err := c.do(ctx, http.MethodPatch, path, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteClusterOptions contains options for deleting a cluster
type DeleteClusterOptions struct {
	Force bool
}

// DeleteCluster deletes a cluster
func (c *Client) DeleteCluster(ctx context.Context, clusterID string, opts *DeleteClusterOptions) error {
	if clusterID == "" {
		return fmt.Errorf("cluster ID cannot be empty")
	}

	params := map[string]string{}
	if opts != nil && opts.Force {
		params["force"] = "true"
	}

	path := buildURL(fmt.Sprintf("/api/v0/clusters/%s", clusterID), params)

	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// GetClusterStatus retrieves the status of a cluster
func (c *Client) GetClusterStatus(ctx context.Context, clusterID string) (*types.ClusterStatusResponse, error) {
	if clusterID == "" {
		return nil, fmt.Errorf("cluster ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/clusters/%s/statuses", clusterID)

	var result types.ClusterStatusResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
