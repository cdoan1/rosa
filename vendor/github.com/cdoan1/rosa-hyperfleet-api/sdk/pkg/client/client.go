package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	// HeaderAccountID is the AWS account ID header
	HeaderAccountID = "X-Amz-Account-Id"
	// HeaderCallerARN is the AWS caller ARN header
	HeaderCallerARN = "X-Amz-Caller-Arn"
	// HeaderUserID is the AWS user ID header
	HeaderUserID = "X-Amz-User-Id"
)

// Client provides access to the ROSA Regional Platform API
type Client struct {
	baseURL    string
	httpClient *http.Client
	signer     *awsSigner
	userAgent  string
	accountID  string
}

// NewClient creates a new API client with the given options
func NewClient(opts ...Option) (*Client, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	if options.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	// Load AWS config if not provided
	var awsConfig aws.Config
	if options.AWSConfig.Region == "" {
		cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(options.Region))
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
		awsConfig = cfg
	} else {
		awsConfig = options.AWSConfig
	}

	// Get AWS credentials
	credentials, err := awsConfig.Credentials.Retrieve(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	client := &Client{
		baseURL:    strings.TrimSuffix(options.BaseURL, "/"),
		httpClient: options.HTTPClient,
		signer:     newAWSSigner(credentials, options.Region),
		userAgent:  options.UserAgent,
		accountID:  options.AccountID,
	}

	return client, nil
}

// do performs an HTTP request with AWS SigV4 signing
func (c *Client) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyBytes []byte
	var bodyReader io.Reader

	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if c.accountID != "" {
		req.Header.Set(HeaderAccountID, c.accountID)
	}

	// Sign request with AWS SigV4
	if err := c.signer.signRequest(ctx, req, bodyBytes); err != nil {
		return err
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error response
	if resp.StatusCode >= 400 {
		return parseErrorResponse(resp, respBody)
	}

	// Parse successful response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// buildURL constructs a URL with query parameters
func buildURL(path string, queryParams map[string]string) string {
	if len(queryParams) == 0 {
		return path
	}

	params := url.Values{}
	for k, v := range queryParams {
		if v != "" {
			params.Add(k, v)
		}
	}

	if len(params) > 0 {
		return path + "?" + params.Encode()
	}
	return path
}
