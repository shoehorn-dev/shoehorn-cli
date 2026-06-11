package commands

import (
	"context"
	"fmt"
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
