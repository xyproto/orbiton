//go:build !darwin && !windows

package slay

import "path/filepath"

// normalizePath returns a canonical path for deduplication on case-sensitive
// file systems (Linux, BSDs, etc.).
func normalizePath(s string) string {
	return filepath.Clean(s)
}
