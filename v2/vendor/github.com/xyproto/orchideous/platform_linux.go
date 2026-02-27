//go:build linux

package orchideous

func isLinux() bool  { return true }
func isDarwin() bool { return false }

const platformCDefine = "-D_GNU_SOURCE"

func extraLDLibPaths() []string { return nil }

func prependAsNeededFlag(ldflags []string) []string {
	return prependUnique(ldflags, "-Wl,--as-needed")
}

// detectPlatformType returns the package-manager type for the current distro.
func detectPlatformType() string {
	// Arch Linux uses pacman; Debian/Ubuntu use dpkg.
	if fileExists("/usr/bin/pacman") {
		return "arch"
	}
	if fileExists("/usr/bin/dpkg-query") {
		return "deb"
	}
	return "generic"
}
