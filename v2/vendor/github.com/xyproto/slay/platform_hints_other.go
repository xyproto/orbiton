//go:build !darwin

package slay

// platformHints is a no-op on non-macOS platforms.
func platformHints(_ []string) {}
