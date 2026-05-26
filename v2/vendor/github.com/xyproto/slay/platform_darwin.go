//go:build darwin

package slay

import "github.com/xyproto/files"

func isLinux() bool  { return false }
func isDarwin() bool { return true }

const platformCDefine = "-D_XOPEN_SOURCE=700"

func extraLDLibPaths() []string { return nil }

// macOS linker does not support --as-needed.
func prependAsNeededFlag(ldflags []string) []string { return ldflags }

// detectPlatformType returns "brew" when Homebrew is available, else "generic".
func detectPlatformType() string {
	if files.WhichCached("brew") != "" {
		return "brew"
	}
	return "generic"
}
