//go:build netbsd

package orchideous

func isLinux() bool  { return false }
func isDarwin() bool { return false }

const platformCDefine = "-D_BSD_SOURCE"

func extraLDLibPaths() []string { return []string{"-L/usr/pkg/lib"} }

func prependAsNeededFlag(ldflags []string) []string {
	return prependUnique(ldflags, "-Wl,--as-needed")
}

// detectPlatformType returns "generic" on NetBSD.
func detectPlatformType() string { return "generic" }
