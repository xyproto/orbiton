package slay

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// The compile flags of the previous build are remembered outside the project
// directory, so that a change to CXXFLAGS (or to what slay detects) triggers a
// rebuild without leaving stray files behind.

// flagStamp returns a string that represents the compile flags of a build.
func flagStamp(flags BuildFlags) string {
	parts := []string{flags.Compiler, flags.Std, flags.ContainerImage}
	parts = append(parts, flags.CFlags...)
	parts = append(parts, flags.Defines...)
	parts = append(parts, flags.IncPaths...)
	parts = append(parts, flags.LDFlags...)
	return strings.Join(parts, " ")
}

// stampPath returns the file that holds the flag stamp for the given output
// executable in the current directory, or an empty string if unavailable.
func stampPath(output string) string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(cacheDir, "slay")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return ""
	}
	sum := sha256.Sum256([]byte(filepath.Join(mustGetwd(), output)))
	return filepath.Join(dir, hex.EncodeToString(sum[:8])+".flags")
}

// flagsChanged reports whether the compile flags differ from the previous build.
// Builds that were not made by slay have no stamp and are left alone.
func flagsChanged(output string, flags BuildFlags) bool {
	path := stampPath(output)
	if path == "" {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return string(data) != flagStamp(flags)
}

// saveFlagStamp records the compile flags of a successful build.
func saveFlagStamp(output string, flags BuildFlags) {
	if path := stampPath(output); path != "" {
		os.WriteFile(path, []byte(flagStamp(flags)), 0o600)
	}
}
