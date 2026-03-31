package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Confirm asks the user for confirmation before a destructive operation.
//
// When yesFlag is true (--yes passed), returns true immediately without prompting.
// When reader is non-nil, reads a line from it for the answer (y/yes = true, anything else = false).
// When reader is nil and yesFlag is false, returns an error telling the user to pass --yes
// (this covers non-interactive/CI mode where stdin is not a terminal).
func Confirm(prompt string, yesFlag bool, reader io.Reader) (bool, error) {
	if yesFlag {
		return true, nil
	}

	if reader == nil {
		return false, fmt.Errorf("confirmation required: use --yes to skip interactive prompt")
	}

	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)

	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("read confirmation: %w", err)
		}
		return false, nil // EOF = no
	}

	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}
