//go:build !windows

package config

import (
	"fmt"
	"os"
)

// warnLoosePermissions checks that the config file is not readable by
// group or others. Tokens stored in the config would be exposed if
// permissions are looser than 0600.
func warnLoosePermissions(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	perm := info.Mode().Perm()
	if perm&0077 != 0 {
		fmt.Fprintf(os.Stderr, "WARNING: %s has loose permissions (%04o). Run: chmod 600 %s\n",
			path, perm, path)
	}
}
