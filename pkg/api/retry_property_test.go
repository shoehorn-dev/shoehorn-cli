package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Property: isRetryable returns true for all retryable HTTP status codes.
func TestIsRetryable_Property_RetryableStatuses(t *testing.T) {
	retryableCodes := []int{429, 502, 503, 504}
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.SampledFrom(retryableCodes).Draw(t, "code")
		msg := rapid.String().Draw(t, "msg")
		err := NewAPIError(code, msg, "")
		if !isRetryable(err) {
			t.Fatalf("isRetryable(APIError{%d}) = false, want true", code)
		}
	})
}

// Property: isRetryable returns false for all non-retryable HTTP status codes.
func TestIsRetryable_Property_NonRetryableStatuses(t *testing.T) {
	nonRetryable := []int{400, 401, 403, 404, 409, 422, 500}
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.SampledFrom(nonRetryable).Draw(t, "code")
		msg := rapid.String().Draw(t, "msg")
		err := NewAPIError(code, msg, "")
		if isRetryable(err) {
			t.Fatalf("isRetryable(APIError{%d}) = true, want false", code)
		}
	})
}

// Property: isRetryable returns false for context cancellation errors.
func TestIsRetryable_Property_ContextErrors(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		prefix := rapid.String().Draw(t, "prefix")
		// Wrap context.Canceled with arbitrary prefix
		err := fmt.Errorf("%s: %w", prefix, context.Canceled)
		if isRetryable(err) {
			t.Fatalf("isRetryable(wrapped context.Canceled) = true, want false")
		}
	})
}

// Property: isRetryable returns false for context deadline exceeded.
func TestIsRetryable_Property_DeadlineExceeded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		prefix := rapid.String().Draw(t, "prefix")
		err := fmt.Errorf("%s: %w", prefix, context.DeadlineExceeded)
		if isRetryable(err) {
			t.Fatalf("isRetryable(wrapped DeadlineExceeded) = true, want false")
		}
	})
}

// Property: retryWait is monotonically non-decreasing with attempt number.
func TestRetryWait_Property_Monotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.IntRange(0, 10).Draw(t, "attempt_a")
		b := rapid.IntRange(a, 10).Draw(t, "attempt_b")

		waitA := retryWait(a)
		waitB := retryWait(b)

		if waitB < waitA {
			t.Fatalf("retryWait(%d) = %v > retryWait(%d) = %v, not monotonic", a, waitA, b, waitB)
		}
	})
}

// Property: retryWait never exceeds retryMaxWait.
func TestRetryWait_Property_BoundedAbove(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		attempt := rapid.IntRange(0, 100).Draw(t, "attempt")
		wait := retryWait(attempt)
		if wait > retryMaxWait {
			t.Fatalf("retryWait(%d) = %v, exceeds max %v", attempt, wait, retryMaxWait)
		}
	})
}

// Property: retryWait is always positive.
func TestRetryWait_Property_AlwaysPositive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		attempt := rapid.IntRange(0, 100).Draw(t, "attempt")
		wait := retryWait(attempt)
		if wait <= 0 {
			t.Fatalf("retryWait(%d) = %v, want positive", attempt, wait)
		}
	})
}

// Property: retryWait at attempt 0 equals retryBaseWait.
func TestRetryWait_Property_BaseCase(t *testing.T) {
	if retryWait(0) != retryBaseWait {
		t.Fatalf("retryWait(0) = %v, want %v", retryWait(0), retryBaseWait)
	}
}

// Property: APIError.Unwrap is consistent -- same status code always maps to same sentinel.
func TestAPIError_Property_UnwrapDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.IntRange(200, 599).Draw(t, "code")
		msg1 := rapid.String().Draw(t, "msg1")
		msg2 := rapid.String().Draw(t, "msg2")

		err1 := NewAPIError(code, msg1, "")
		err2 := NewAPIError(code, msg2, "")

		sentinel1 := err1.Unwrap()
		sentinel2 := err2.Unwrap()

		if sentinel1 != sentinel2 {
			t.Fatalf("Unwrap for status %d is not deterministic: %v vs %v", code, sentinel1, sentinel2)
		}
	})
}

// Property: isRetryable is deterministic -- same error always gives same result.
func TestIsRetryable_Property_Deterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.IntRange(200, 599).Draw(t, "code")
		msg := rapid.String().Draw(t, "msg")
		err := NewAPIError(code, msg, "")

		r1 := isRetryable(err)
		r2 := isRetryable(err)
		if r1 != r2 {
			t.Fatalf("isRetryable not deterministic for status %d", code)
		}
	})
}

// Property: retryWait for attempts 0..2 follows exact exponential pattern.
func TestRetryWait_Property_ExponentialPattern(t *testing.T) {
	for i := range 3 {
		expected := retryBaseWait << uint(i)
		if expected > retryMaxWait {
			expected = retryMaxWait
		}
		got := retryWait(i)
		if got != expected {
			t.Errorf("retryWait(%d) = %v, want %v", i, got, expected)
		}
	}
	// Verify the actual values: 500ms, 1s, 2s
	if retryWait(0) != 500*time.Millisecond {
		t.Errorf("retryWait(0) = %v, want 500ms", retryWait(0))
	}
	if retryWait(1) != 1*time.Second {
		t.Errorf("retryWait(1) = %v, want 1s", retryWait(1))
	}
	if retryWait(2) != 2*time.Second {
		t.Errorf("retryWait(2) = %v, want 2s", retryWait(2))
	}
}
