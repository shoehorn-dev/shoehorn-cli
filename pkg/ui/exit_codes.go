package ui

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
)

// Exit codes for the CLI
const (
	ExitSuccess      = 0 // Command executed successfully
	ExitError        = 1 // Generic error
	ExitAuthRequired = 2 // Authentication required
	ExitNotFound     = 3 // Resource not found
	ExitValidation   = 4 // Validation error
	ExitTimeout      = 5 // Operation timeout
	ExitCancelled    = 6 // User cancelled operation
)

// Exit terminates the program with the given exit code
func Exit(code int) {
	os.Exit(code)
}

// ExitWithError prints the error and exits with the appropriate code.
// Exit code is determined by inspecting the error chain with errors.Is,
// not by string matching (which is fragile and error-prone).
func ExitWithError(err error) {
	if err == nil {
		Exit(ExitSuccess)
		return
	}

	// Print error to stderr
	RenderError(err)

	Exit(classifyExitCode(err))
}

// classifyExitCode determines the appropriate exit code for an error
// by inspecting the error chain with errors.Is.
func classifyExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	switch {
	case errors.Is(err, api.ErrNotAuthenticated):
		return ExitAuthRequired
	case errors.Is(err, api.ErrNotFound):
		return ExitNotFound
	case errors.Is(err, api.ErrValidation):
		return ExitValidation
	case errors.Is(err, api.ErrTimeout),
		errors.Is(err, context.DeadlineExceeded):
		return ExitTimeout
	case errors.Is(err, context.Canceled):
		return ExitCancelled
	default:
		return ExitError
	}
}

// ExitWithMessage prints a message and exits with the given code
func ExitWithMessage(code int, message string, args ...any) {
	if code == ExitSuccess {
		fmt.Fprintf(os.Stdout, message+"\n", args...)
	} else {
		fmt.Fprintf(os.Stderr, message+"\n", args...)
	}
	Exit(code)
}
