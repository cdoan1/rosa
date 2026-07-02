package client

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// Options contains configuration options for the client
type Options struct {
	BaseURL    string
	Region     string
	AWSConfig  aws.Config
	HTTPClient *http.Client
	UserAgent  string
	// AccountID is the AWS account ID to send in X-Amz-Account-Id header
	AccountID string
}

// Option is a functional option for configuring the client
type Option func(*Options)

// WithRegion sets the AWS region for API Gateway
func WithRegion(region string) Option {
	return func(o *Options) {
		o.Region = region
	}
}

// WithBaseURL sets the base URL for the API
func WithBaseURL(baseURL string) Option {
	return func(o *Options) {
		o.BaseURL = baseURL
	}
}

// WithAWSConfig sets the AWS config for authentication
func WithAWSConfig(cfg aws.Config) Option {
	return func(o *Options) {
		o.AWSConfig = cfg
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) Option {
	return func(o *Options) {
		o.HTTPClient = client
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		if o.HTTPClient == nil {
			o.HTTPClient = &http.Client{}
		}
		o.HTTPClient.Timeout = timeout
	}
}

// WithUserAgent sets a custom user agent
func WithUserAgent(userAgent string) Option {
	return func(o *Options) {
		o.UserAgent = userAgent
	}
}

// WithAccountID sets the AWS account ID for the X-Amz-Account-Id header
func WithAccountID(accountID string) Option {
	return func(o *Options) {
		o.AccountID = accountID
	}
}

// defaultOptions returns the default options
func defaultOptions() Options {
	return Options{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UserAgent: "rosa-hyperfleet-api-sdk/0.1.0",
		Region:    "us-east-1",
	}
}
