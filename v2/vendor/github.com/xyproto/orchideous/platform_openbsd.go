//go:build openbsd

package orchideous

func isLinux() bool  { return false }
func isDarwin() bool { return false }

const platformCDefine = "-D_BSD_SOURCE"

func extraLDLibPaths() []string { return []string{"-L/usr/local/lib"} }

func prependAsNeededFlag(ldflags []string) []string {
	return prependUnique(ldflags, "-Wl,--as-needed")
}

// detectPlatformType returns "openbsd" when pkg_info(1) is present.
func detectPlatformType() string {
	if fileExists("/usr/sbin/pkg_info") {
		return "openbsd"
	}
	return "generic"
}
