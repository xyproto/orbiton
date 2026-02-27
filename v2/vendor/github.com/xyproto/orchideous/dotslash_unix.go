//go:build !windows

package orchideous

import (
	"path/filepath"
	"strings"
)

// dotSlash prepends ./ to a relative path to make it executable.
func dotSlash(name string) string {
	if filepath.IsAbs(name) || strings.HasPrefix(name, "./") {
		return name
	}
	return "./" + name
}
