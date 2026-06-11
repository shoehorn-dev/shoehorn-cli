// Package browser opens URLs in the user's default browser.
package browser

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

// Open launches the default browser for an http(s) URL.
// Non-http(s) URLs are rejected.
func Open(rawURL string) error {
	if err := validateURL(rawURL); err != nil {
		return err
	}
	args := commandArgs(runtime.GOOS, rawURL)
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}

func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("refusing to open %q: only http and https URLs are supported", rawURL)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid URL %q: missing host", rawURL)
	}
	return nil
}

func commandArgs(goos, rawURL string) []string {
	switch goos {
	case "windows":
		return []string{"rundll32", "url.dll,FileProtocolHandler", rawURL}
	case "darwin":
		return []string{"open", rawURL}
	default:
		return []string{"xdg-open", rawURL}
	}
}
