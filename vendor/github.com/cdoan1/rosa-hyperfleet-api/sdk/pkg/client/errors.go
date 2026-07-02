package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// APIError represents an error response from the API
type APIError struct {
	Kind   string `json:"kind"`
	Code   string `json:"code"`
	Reason string `json:"reason"`
	Status int    `json:"-"` // HTTP status code
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Status, e.Code, e.Reason)
	}
	return fmt.Sprintf("[%d] %s", e.Status, e.Reason)
}

// IsNotFound returns true if the error is a 404 Not Found
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Status == http.StatusNotFound
	}
	return false
}

// IsConflict returns true if the error is a 409 Conflict
func IsConflict(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Status == http.StatusConflict
	}
	return false
}

// IsForbidden returns true if the error is a 403 Forbidden
func IsForbidden(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Status == http.StatusForbidden
	}
	return false
}

// IsUnauthorized returns true if the error is a 401 Unauthorized
func IsUnauthorized(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Status == http.StatusUnauthorized
	}
	return false
}

// IsBadRequest returns true if the error is a 400 Bad Request
func IsBadRequest(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Status == http.StatusBadRequest
	}
	return false
}

// parseErrorResponse attempts to parse an error response from the API
func parseErrorResponse(resp *http.Response, body []byte) error {
	apiErr := &APIError{
		Status: resp.StatusCode,
		Reason: resp.Status,
	}

	// Try to parse JSON error response
	if len(body) > 0 {
		var errResp struct {
			Kind   string `json:"kind"`
			Code   string `json:"code"`
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil {
			apiErr.Kind = errResp.Kind
			apiErr.Code = errResp.Code
			if errResp.Reason != "" {
				apiErr.Reason = errResp.Reason
			}
		}
	}

	return apiErr
}
