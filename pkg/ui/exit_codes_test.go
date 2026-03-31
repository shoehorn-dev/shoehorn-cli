package ui

import (
	"context"
	"fmt"
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
)

// We can't test os.Exit directly, but we can test the exit code determination
// by extracting the logic. Instead, we test the error classification.

func TestExitCodeMapping_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{
			name:     "nil error",
			err:      nil,
			wantCode: ExitSuccess,
		},
		{
			name:     "not authenticated",
			err:      fmt.Errorf("outer: %w", api.ErrNotAuthenticated),
			wantCode: ExitAuthRequired,
		},
		{
			name:     "not found",
			err:      fmt.Errorf("get entity: %w", api.ErrNotFound),
			wantCode: ExitNotFound,
		},
		{
			name:     "validation error",
			err:      fmt.Errorf("bad input: %w", api.ErrValidation),
			wantCode: ExitValidation,
		},
		{
			name:     "timeout",
			err:      fmt.Errorf("request: %w", api.ErrTimeout),
			wantCode: ExitTimeout,
		},
		{
			name:     "deadline exceeded",
			err:      fmt.Errorf("ctx: %w", context.DeadlineExceeded),
			wantCode: ExitTimeout,
		},
		{
			name:     "context cancelled",
			err:      fmt.Errorf("ctx: %w", context.Canceled),
			wantCode: ExitCancelled,
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("something went wrong"),
			wantCode: ExitError,
		},
		{
			name:     "api error 404 wraps ErrNotFound",
			err:      api.NewAPIError(404, "not found", ""),
			wantCode: ExitNotFound,
		},
		{
			name:     "api error 401 wraps ErrNotAuthenticated",
			err:      api.NewAPIError(401, "unauthorized", ""),
			wantCode: ExitAuthRequired,
		},
		{
			name:     "server error does not match specific code",
			err:      fmt.Errorf("server: %w", api.ErrServerError),
			wantCode: ExitError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyExitCode(tt.err)
			if got != tt.wantCode {
				t.Errorf("classifyExitCode(%v) = %d, want %d", tt.err, got, tt.wantCode)
			}
		})
	}
}
