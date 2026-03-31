package api

import (
	"errors"
	"fmt"
	"testing"
)

// TestAPIError_Error tests the Error() string output.
func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name       string
		err        *APIError
		wantSubstr string
	}{
		{
			name:       "with code",
			err:        &APIError{StatusCode: 404, Message: "entity not found", Code: "NOT_FOUND"},
			wantSubstr: "API error (404/NOT_FOUND): entity not found",
		},
		{
			name:       "without code",
			err:        &APIError{StatusCode: 500, Message: "internal error"},
			wantSubstr: "API error (500): internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantSubstr {
				t.Errorf("APIError.Error() = %q, want %q", got, tt.wantSubstr)
			}
		})
	}
}

// TestAPIError_Unwrap_ErrorsIs tests that errors.Is works with sentinel errors.
func TestAPIError_Unwrap_ErrorsIs(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		sentinel   error
		wantMatch  bool
	}{
		{"401 matches ErrNotAuthenticated", 401, ErrNotAuthenticated, true},
		{"401 does not match ErrNotFound", 401, ErrNotFound, false},
		{"403 matches ErrForbidden", 403, ErrForbidden, true},
		{"404 matches ErrNotFound", 404, ErrNotFound, true},
		{"400 matches ErrValidation", 400, ErrValidation, true},
		{"422 matches ErrValidation", 422, ErrValidation, true},
		{"408 matches ErrTimeout", 408, ErrTimeout, true},
		{"504 matches ErrTimeout", 504, ErrTimeout, true},
		{"500 matches ErrServerError", 500, ErrServerError, true},
		{"502 matches ErrServerError", 502, ErrServerError, true},
		{"200 matches nothing", 200, ErrNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := &APIError{StatusCode: tt.statusCode, Message: "test"}
			got := errors.Is(apiErr, tt.sentinel)
			if got != tt.wantMatch {
				t.Errorf("errors.Is(APIError{%d}, %v) = %v, want %v",
					tt.statusCode, tt.sentinel, got, tt.wantMatch)
			}
		})
	}
}

// TestAPIError_ErrorsAs tests that errors.As works to extract APIError.
func TestAPIError_ErrorsAs(t *testing.T) {
	originalErr := &APIError{StatusCode: 404, Message: "not found", Code: "NOT_FOUND"}
	wrappedErr := fmt.Errorf("get entity: %w", originalErr)

	var apiErr *APIError
	if !errors.As(wrappedErr, &apiErr) {
		t.Fatal("errors.As failed to extract APIError from wrapped error")
	}

	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Code != "NOT_FOUND" {
		t.Errorf("Code = %q, want %q", apiErr.Code, "NOT_FOUND")
	}
}

// TestAPIError_WrappedChain tests that errors.Is works through wrapping chains.
func TestAPIError_WrappedChain(t *testing.T) {
	apiErr := &APIError{StatusCode: 401, Message: "unauthorized"}
	wrapped := fmt.Errorf("get entity: %w", fmt.Errorf("api call: %w", apiErr))

	if !errors.Is(wrapped, ErrNotAuthenticated) {
		t.Error("errors.Is should find ErrNotAuthenticated through wrapping chain")
	}

	if !IsNotAuthenticated(wrapped) {
		t.Error("IsNotAuthenticated should return true for wrapped 401")
	}
}

// TestNewAPIError tests the constructor.
func TestNewAPIError(t *testing.T) {
	err := NewAPIError(500, "server broke", "INTERNAL")
	if err.StatusCode != 500 || err.Message != "server broke" || err.Code != "INTERNAL" {
		t.Errorf("NewAPIError = %+v, unexpected values", err)
	}
}

// TestIsNotFound_Helper tests the convenience function.
func TestIsNotFound_Helper(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", &APIError{StatusCode: 404, Message: "gone"})
	if !IsNotFound(err) {
		t.Error("IsNotFound should return true for wrapped 404")
	}
	if IsNotFound(fmt.Errorf("random error")) {
		t.Error("IsNotFound should return false for non-API error")
	}
}

// TestAPIError_Metamorphic_StatusCodeDeterminesSentinel tests the metamorphic
// property: same status code always maps to same sentinel regardless of message.
func TestAPIError_Metamorphic_StatusCodeDeterminesSentinel(t *testing.T) {
	messages := []string{"error a", "error b", "completely different", ""}
	for _, msg := range messages {
		err1 := &APIError{StatusCode: 404, Message: msg}
		if !errors.Is(err1, ErrNotFound) {
			t.Errorf("404 with message %q should always match ErrNotFound", msg)
		}
	}
}
