//go:build !darwin

package orchideous

// platformHints is a no-op on non-macOS platforms.
func platformHints(_ []string) {}
