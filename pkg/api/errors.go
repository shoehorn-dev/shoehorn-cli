// Package api - errors.go defines typed errors for the Shoehorn API client.
//
// APIError carries HTTP status code and message, enabling callers to use
// errors.As() for structured error handling instead of string matching.
// Sentinel errors (ErrNotAuthenticated, ErrNotFound, etc.) support errors.Is().
package api

import (
	"errors"
	"fmt"
)

// Sentinel errors for common API failure modes.
// Callers should use errors.Is(err, api.ErrNotAuthenticated) rather than
// string matching to check for specific failure conditions.
var (
	// ErrNotAuthenticated indicates the request lacked valid credentials (HTTP 401).
	ErrNotAuthenticated = errors.New("not authenticated")
	// ErrNotFound indicates the requested resource does not exist (HTTP 404).
	ErrNotFound = errors.New("resource not found")
	// ErrForbidden indicates the caller is authenticated but not authorized (HTTP 403).
	ErrForbidden = errors.New("forbidden")
	// ErrValidation indicates the request payload failed validation (HTTP 400/422).
	ErrValidation = errors.New("validation error")
	// ErrTimeout indicates the request or upstream gateway timed out (HTTP 408/504).
	ErrTimeout = errors.New("request timeout")
	// ErrConflict indicates a resource conflict, e.g. duplicate slug (HTTP 409).
	ErrConflict = errors.New("conflict")
	// ErrRateLimit indicates the client is sending too many requests (HTTP 429).
	ErrRateLimit = errors.New("rate limited")
	// ErrServerError indicates an internal server failure (HTTP 5xx).
	ErrServerError = errors.New("server error")
)

// APIError represents a structured error from the Shoehorn API.
// It carries the HTTP status code and optional error code for programmatic handling.
type APIError struct {
	StatusCode int    // HTTP status code (e.g., 404, 500)
	Message    string // Human-readable error message
	Code       string // Machine-readable error code from API (optional)
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("API error (%d/%s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
}

// Unwrap returns the appropriate sentinel error based on status code,
// enabling errors.Is(err, api.ErrNotFound) to work.
func (e *APIError) Unwrap() error {
	switch {
	case e.StatusCode == 401:
		return ErrNotAuthenticated
	case e.StatusCode == 403:
		return ErrForbidden
	case e.StatusCode == 404:
		return ErrNotFound
	case e.StatusCode == 400 || e.StatusCode == 422:
		return ErrValidation
	case e.StatusCode == 409:
		return ErrConflict
	case e.StatusCode == 429:
		return ErrRateLimit
	case e.StatusCode == 408 || e.StatusCode == 504:
		return ErrTimeout
	case e.StatusCode >= 500:
		return ErrServerError
	default:
		return nil
	}
}

// NewAPIError creates an APIError with the given HTTP status code, human-readable
// message, and optional machine-readable error code. The code parameter may be
// empty when the upstream response does not include a structured error body.
func NewAPIError(statusCode int, message, code string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Code:       code,
	}
}

// IsNotFound returns true if the error is a 404 API error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsNotAuthenticated returns true if the error is a 401 API error.
func IsNotAuthenticated(err error) bool {
	return errors.Is(err, ErrNotAuthenticated)
}
