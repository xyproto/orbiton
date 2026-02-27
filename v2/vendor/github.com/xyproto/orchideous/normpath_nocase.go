//go:build darwin || windows

package orchideous

import (
	"path/filepath"
	"strings"
)

// normalizePath returns a canonical, case-folded path for deduplication on
// case-insensitive file systems (macOS, Windows).
func normalizePath(s string) string {
	return strings.ToLower(filepath.Clean(s))
}
