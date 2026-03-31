//go:build windows

package config

// warnLoosePermissions is a no-op on Windows where POSIX file permissions
// do not apply. Windows ACLs are managed separately.
func warnLoosePermissions(_ string) {}
