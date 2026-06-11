package commands

import (
	"context"
	"time"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
)

// runGetter is the API surface the watch loop needs.
type runGetter interface {
	GetRun(ctx context.Context, runID string) (*api.ForgeRun, error)
}

// maxConsecutivePollErrors is how many GetRun failures in a row the watch
// tolerates before giving up. Single blips (API restart, network hiccup)
// must not kill a watch that may have been running for a long time.
const maxConsecutivePollErrors = 3

// pollInterval returns the effective polling interval for a status.
// pending_approval is human-latency-bound, so it polls slowly regardless of
// the base interval. Anything below 1s is clamped to avoid hot loops.
func pollInterval(base time.Duration, status string) time.Duration {
	if base < time.Second {
		base = time.Second
	}
	if status == "pending_approval" && base < 15*time.Second {
		return 15 * time.Second
	}
	return base
}

// watchUntilTerminal polls a run until it reaches a terminal status.
// onStatus fires once per status change. On context cancellation the last
// seen run is returned together with the context error.
func watchUntilTerminal(ctx context.Context, client runGetter, runID string, interval time.Duration, onStatus func(status string)) (*api.ForgeRun, error) {
	var last *api.ForgeRun
	var lastStatus string
	consecutiveErrors := 0

	for {
		run, err := client.GetRun(ctx, runID)
		if err != nil {
			if ctx.Err() != nil {
				return last, ctx.Err()
			}
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutivePollErrors {
				return last, err
			}
		} else {
			consecutiveErrors = 0
			last = run

			if run.Status != lastStatus {
				lastStatus = run.Status
				if onStatus != nil {
					onStatus(run.Status)
				}
			}

			if isTerminalStatus(run.Status) {
				return run, nil
			}
		}

		timer := time.NewTimer(pollInterval(interval, lastStatus))
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return last, ctx.Err()
		}
	}
}
