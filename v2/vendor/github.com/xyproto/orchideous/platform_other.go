//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd && !solaris && !illumos && !windows

package orchideous

func isLinux() bool  { return false }
func isDarwin() bool { return false }

const platformCDefine = "-D_XOPEN_SOURCE=700"

func extraLDLibPaths() []string { return nil }

func prependAsNeededFlag(ldflags []string) []string {
	return prependUnique(ldflags, "-Wl,--as-needed")
}

// detectPlatformType returns "generic" on platforms without a known package manager.
func detectPlatformType() string { return "generic" }
