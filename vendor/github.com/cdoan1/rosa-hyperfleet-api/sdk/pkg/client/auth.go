package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const (
	// emptyPayloadHash is the SHA256 of an empty string
	emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// serviceName is the AWS service name for API Gateway
	serviceName = "execute-api"
)

// awsSigner handles AWS SigV4 request signing
type awsSigner struct {
	credentials aws.Credentials
	region      string
	signer      *v4.Signer
}

// newAWSSigner creates a new AWS SigV4 signer
func newAWSSigner(credentials aws.Credentials, region string) *awsSigner {
	return &awsSigner{
		credentials: credentials,
		region:      region,
		signer:      v4.NewSigner(),
	}
}

// signRequest signs an HTTP request with AWS SigV4
func (s *awsSigner) signRequest(ctx context.Context, req *http.Request, body []byte) error {
	payloadHash := emptyPayloadHash
	if len(body) > 0 {
		sum := sha256.Sum256(body)
		payloadHash = hex.EncodeToString(sum[:])
	}

	err := s.signer.SignHTTP(ctx, s.credentials, req, payloadHash, serviceName, s.region, time.Now())
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	return nil
}
