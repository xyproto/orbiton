//go:build solaris || illumos

package orchideous

func isLinux() bool  { return false }
func isDarwin() bool { return false }

const platformCDefine = "-D_XOPEN_SOURCE=700"

func extraLDLibPaths() []string { return nil }

func prependAsNeededFlag(ldflags []string) []string {
	return prependUnique(ldflags, "-Wl,-zignore")
}

// detectPlatformType returns "generic" on Solaris/illumos.
func detectPlatformType() string { return "generic" }
