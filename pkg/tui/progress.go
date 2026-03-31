package tui

import (
	"fmt"
	"os"
	"sync"
)

// RunProgress runs a function that processes multiple items, displaying progress.
// The function receives an update callback: update(done, failed) to report progress.
// The update callback is safe for concurrent use from multiple goroutines.
// In plain mode (non-interactive), prints a line per update to stderr.
func RunProgress(total int, fn func(update func(done, failed int)) error) error {
	if total == 0 {
		return fn(func(_, _ int) {})
	}

	var mu sync.Mutex
	update := func(done, failed int) {
		mu.Lock()
		defer mu.Unlock()

		if done > total {
			done = total
		}
		if failed > 0 {
			fmt.Fprintf(os.Stderr, "\r  [%d/%d] Processing... (%d failed)", done, total, failed)
		} else {
			fmt.Fprintf(os.Stderr, "\r  [%d/%d] Processing...", done, total)
		}
		if done >= total {
			fmt.Fprintln(os.Stderr)
		}
	}

	return fn(update)
}
