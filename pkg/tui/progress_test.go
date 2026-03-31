package tui

import (
	"fmt"
	"testing"
)

func TestRunProgress_AllSucceed(t *testing.T) {
	processed := 0
	err := RunProgress(5, func(update func(done, failed int)) error {
		for i := range 5 {
			processed++
			update(i+1, 0)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if processed != 5 {
		t.Errorf("processed = %d, want 5", processed)
	}
}

func TestRunProgress_PartialFailure(t *testing.T) {
	err := RunProgress(3, func(update func(done, failed int)) error {
		update(1, 0) // success
		update(2, 1) // one failed
		update(3, 1) // done
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunProgress_FunctionError(t *testing.T) {
	err := RunProgress(1, func(update func(done, failed int)) error {
		return fmt.Errorf("network error")
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunProgress_ZeroTotal_CallbackInvoked(t *testing.T) {
	called := false
	err := RunProgress(0, func(update func(done, failed int)) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("callback was not invoked for zero total")
	}
}

func TestRunProgress_DoneExceedsTotal_Clamped(t *testing.T) {
	// Should not panic or produce garbled output
	err := RunProgress(2, func(update func(done, failed int)) error {
		update(1, 0)
		update(5, 0) // exceeds total -- should be clamped to 2
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
