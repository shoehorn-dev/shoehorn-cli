package commands

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
)

type fakeRunGetter struct {
	statuses []string
	calls    int
	err      error
}

func (f *fakeRunGetter) GetRun(ctx context.Context, runID string) (*api.ForgeRun, error) {
	if f.err != nil {
		return nil, f.err
	}
	idx := f.calls
	if idx >= len(f.statuses) {
		idx = len(f.statuses) - 1
	}
	f.calls++
	return &api.ForgeRun{ID: runID, Status: f.statuses[idx]}, nil
}

func TestWatchUntilTerminal_PollsUntilCompleted(t *testing.T) {
	getter := &fakeRunGetter{statuses: []string{"pending", "executing", "executing", "completed"}}

	var transitions []string
	run, err := watchUntilTerminal(context.Background(), getter, "run-1", time.Millisecond, func(s string) {
		transitions = append(transitions, s)
	})
	if err != nil {
		t.Fatalf("watchUntilTerminal error = %v", err)
	}
	if run.Status != "completed" {
		t.Errorf("final status = %q, want completed", run.Status)
	}
	want := []string{"pending", "executing", "completed"}
	if len(transitions) != len(want) {
		t.Fatalf("status transitions = %v, want %v (one per change, no repeats)", transitions, want)
	}
	for i := range want {
		if transitions[i] != want[i] {
			t.Errorf("transition[%d] = %q, want %q", i, transitions[i], want[i])
		}
	}
}

func TestWatchUntilTerminal_StopsOnFailedStatus(t *testing.T) {
	getter := &fakeRunGetter{statuses: []string{"executing", "failed"}}

	run, err := watchUntilTerminal(context.Background(), getter, "run-1", time.Millisecond, nil)
	if err != nil {
		t.Fatalf("watchUntilTerminal error = %v (terminal failure is the run's state, not a watch error)", err)
	}
	if run.Status != "failed" {
		t.Errorf("final status = %q, want failed", run.Status)
	}
}

func TestWatchUntilTerminal_ReturnsAPIError(t *testing.T) {
	getter := &fakeRunGetter{err: fmt.Errorf("boom")}

	_, err := watchUntilTerminal(context.Background(), getter, "run-1", time.Millisecond, nil)
	if err == nil {
		t.Fatal("expected API error to propagate")
	}
}

type flakyRunGetter struct {
	failures int
	calls    int
}

func (f *flakyRunGetter) GetRun(ctx context.Context, runID string) (*api.ForgeRun, error) {
	f.calls++
	if f.calls <= f.failures {
		return nil, fmt.Errorf("transient: connection reset")
	}
	return &api.ForgeRun{ID: runID, Status: "completed"}, nil
}

func TestWatchUntilTerminal_RetriesTransientErrors(t *testing.T) {
	getter := &flakyRunGetter{failures: 2}

	run, err := watchUntilTerminal(context.Background(), getter, "run-1", time.Millisecond, nil)
	if err != nil {
		t.Fatalf("watch should survive 2 transient errors, got: %v", err)
	}
	if run.Status != "completed" {
		t.Errorf("final status = %q, want completed", run.Status)
	}
}

func TestWatchUntilTerminal_GivesUpAfterConsecutiveErrors(t *testing.T) {
	getter := &fakeRunGetter{err: fmt.Errorf("boom")}

	_, err := watchUntilTerminal(context.Background(), getter, "run-1", time.Millisecond, nil)
	if err == nil {
		t.Fatal("persistent errors must eventually fail the watch")
	}
	if strings.Contains(err.Error(), "get run: get run") {
		t.Errorf("error should not be double-prefixed, got: %v", err)
	}
}

func TestPollInterval_SlowsDownForPendingApproval(t *testing.T) {
	base := 2 * time.Second

	if got := pollInterval(base, "executing"); got != base {
		t.Errorf("executing should poll at the base interval, got %v", got)
	}
	if got := pollInterval(base, "pending_approval"); got < 15*time.Second {
		t.Errorf("pending_approval is human-latency-bound and should poll slowly, got %v", got)
	}
	if got := pollInterval(30*time.Second, "pending_approval"); got != 30*time.Second {
		t.Errorf("a user-chosen slower interval should be kept, got %v", got)
	}
}

func TestPollInterval_ClampsBelowOneSecond(t *testing.T) {
	if got := pollInterval(0, "executing"); got < time.Second {
		t.Errorf("zero interval must clamp to at least 1s to avoid hot-looping, got %v", got)
	}
}

func TestWatchUntilTerminal_ContextCancellation(t *testing.T) {
	getter := &fakeRunGetter{statuses: []string{"executing"}}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	run, err := watchUntilTerminal(ctx, getter, "run-1", time.Hour, nil)
	if err == nil {
		t.Fatal("cancelled watch should return the context error so callers can print a resume hint")
	}
	if run == nil || run.Status != "executing" {
		t.Errorf("cancelled watch should return the last seen run, got %v", run)
	}
}
