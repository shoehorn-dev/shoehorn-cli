package api

import (
	"errors"
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// Metamorphic: Error message content does not affect sentinel mapping.
// Same status code with different messages should produce the same sentinel.
func TestAPIError_Metamorphic_MessageDoesNotAffectSentinel(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.IntRange(200, 599).Draw(t, "code")
		msg1 := rapid.String().Draw(t, "msg1")
		msg2 := rapid.String().Draw(t, "msg2")

		err1 := NewAPIError(code, msg1, "")
		err2 := NewAPIError(code, msg2, "")

		if err1.Unwrap() != err2.Unwrap() {
			t.Fatalf("different messages changed sentinel for status %d", code)
		}
	})
}

// Metamorphic: IsNotFound and IsNotAuthenticated are mutually exclusive.
func TestAPIError_Metamorphic_SentinelsMutuallyExclusive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.IntRange(200, 599).Draw(t, "code")
		err := NewAPIError(code, "test", "")

		notFound := errors.Is(err, ErrNotFound)
		notAuth := errors.Is(err, ErrNotAuthenticated)
		forbidden := errors.Is(err, ErrForbidden)
		validation := errors.Is(err, ErrValidation)
		conflict := errors.Is(err, ErrConflict)
		rateLimit := errors.Is(err, ErrRateLimit)
		timeout := errors.Is(err, ErrTimeout)
		serverErr := errors.Is(err, ErrServerError)

		sentinels := []bool{notFound, notAuth, forbidden, validation, conflict, rateLimit, timeout, serverErr}
		count := 0
		for _, s := range sentinels {
			if s {
				count++
			}
		}

		if count > 1 {
			t.Fatalf("status %d matched %d sentinels, want at most 1", code, count)
		}
	})
}

// Metamorphic: Wrapping an APIError preserves its sentinel through errors.Is.
func TestAPIError_Metamorphic_WrappingPreservesSentinel(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.SampledFrom([]int{401, 403, 404, 400, 409, 429, 408, 500, 502}).Draw(t, "code")
		err := NewAPIError(code, "test", "")
		sentinel := err.Unwrap()
		if sentinel == nil {
			return // no sentinel for this code
		}

		// Wrap the error the way production code does (fmt.Errorf with %w)
		wrapped := fmt.Errorf("outer: %w", err)
		if !errors.Is(wrapped, sentinel) {
			t.Fatalf("wrapped error for status %d lost sentinel %v", code, sentinel)
		}
	})
}
