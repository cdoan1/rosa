package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cdoan1/rosa-hyperfleet-api/sdk/pkg/types"
)

// CheckAuthorization checks if a principal is authorized to perform an action
func (c *Client) CheckAuthorization(ctx context.Context, req *types.CheckAuthorizationRequest) (*types.CheckAuthorizationResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	var result types.CheckAuthorizationResponse
	if err := c.do(ctx, http.MethodPost, "/api/v0/authz/check", req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListPoliciesOptions contains options for listing policies
type ListPoliciesOptions struct {
	Page int
	Size int
}

// PolicyList represents a paginated list of policies
type PolicyList struct {
	Kind  string         `json:"kind"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
	Total int            `json:"total"`
	Items []types.Policy `json:"items"`
}

// CreatePolicy creates a new authorization policy
func (c *Client) CreatePolicy(ctx context.Context, req *types.CreatePolicyRequest) (*types.Policy, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	var result types.Policy
	if err := c.do(ctx, http.MethodPost, "/api/v0/authz/policies", req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListPolicies retrieves a list of authorization policies
func (c *Client) ListPolicies(ctx context.Context, opts *ListPoliciesOptions) (*PolicyList, error) {
	if opts == nil {
		opts = &ListPoliciesOptions{Page: 1, Size: 100}
	}

	params := map[string]string{}
	if opts.Page > 0 {
		params["page"] = strconv.Itoa(opts.Page)
	}
	if opts.Size > 0 {
		params["size"] = strconv.Itoa(opts.Size)
	}

	path := buildURL("/api/v0/authz/policies", params)

	var result PolicyList
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetPolicy retrieves a policy by ID
func (c *Client) GetPolicy(ctx context.Context, policyID string) (*types.Policy, error) {
	if policyID == "" {
		return nil, fmt.Errorf("policy ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/authz/policies/%s", policyID)

	var result types.Policy
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdatePolicy updates an existing policy
func (c *Client) UpdatePolicy(ctx context.Context, policyID string, req *types.UpdatePolicyRequest) (*types.Policy, error) {
	if policyID == "" {
		return nil, fmt.Errorf("policy ID cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	path := fmt.Sprintf("/api/v0/authz/policies/%s", policyID)

	var result types.Policy
	if err := c.do(ctx, http.MethodPut, path, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeletePolicy deletes a policy
func (c *Client) DeletePolicy(ctx context.Context, policyID string) error {
	if policyID == "" {
		return fmt.Errorf("policy ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/authz/policies/%s", policyID)

	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// ListGroupsOptions contains options for listing groups
type ListGroupsOptions struct {
	Page int
	Size int
}

// GroupList represents a paginated list of groups
type GroupList struct {
	Kind  string        `json:"kind"`
	Page  int           `json:"page"`
	Size  int           `json:"size"`
	Total int           `json:"total"`
	Items []types.Group `json:"items"`
}

// CreateGroup creates a new authorization group
func (c *Client) CreateGroup(ctx context.Context, req *types.CreateGroupRequest) (*types.Group, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	var result types.Group
	if err := c.do(ctx, http.MethodPost, "/api/v0/authz/groups", req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListGroups retrieves a list of authorization groups
func (c *Client) ListGroups(ctx context.Context, opts *ListGroupsOptions) (*GroupList, error) {
	if opts == nil {
		opts = &ListGroupsOptions{Page: 1, Size: 100}
	}

	params := map[string]string{}
	if opts.Page > 0 {
		params["page"] = strconv.Itoa(opts.Page)
	}
	if opts.Size > 0 {
		params["size"] = strconv.Itoa(opts.Size)
	}

	path := buildURL("/api/v0/authz/groups", params)

	var result GroupList
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetGroup retrieves a group by ID
func (c *Client) GetGroup(ctx context.Context, groupID string) (*types.Group, error) {
	if groupID == "" {
		return nil, fmt.Errorf("group ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/authz/groups/%s", groupID)

	var result types.Group
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteGroup deletes a group
func (c *Client) DeleteGroup(ctx context.Context, groupID string) error {
	if groupID == "" {
		return fmt.Errorf("group ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v0/authz/groups/%s", groupID)

	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// UpdateGroupMembers updates the members of a group
func (c *Client) UpdateGroupMembers(ctx context.Context, groupID string, req *types.UpdateGroupMembersRequest) error {
	if groupID == "" {
		return fmt.Errorf("group ID cannot be empty")
	}
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	path := fmt.Sprintf("/api/v0/authz/groups/%s/members", groupID)

	return c.do(ctx, http.MethodPut, path, req, nil)
}
