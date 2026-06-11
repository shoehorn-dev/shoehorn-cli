package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
)

// runGetter is the API surface the watch loop needs.
type runGetter interface {
	GetRun(ctx context.Context, runID string) (*api.ForgeRun, error)
}

// watchUntilTerminal polls a run until it reaches a terminal status.
// onStatus fires once per status change. On context cancellation the last
// seen run is returned together with the context error.
func watchUntilTerminal(ctx context.Context, client runGetter, runID string, interval time.Duration, onStatus func(status string)) (*api.ForgeRun, error) {
	var last *api.ForgeRun
	var lastStatus string

	for {
		run, err := client.GetRun(ctx, runID)
		if err != nil {
			return last, fmt.Errorf("get run: %w", err)
		}
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

		timer := time.NewTimer(interval)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return last, ctx.Err()
		}
	}
}
