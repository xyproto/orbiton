package distrodetector

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
)

const defaultName = "Unknown"

// Used when checking for Linux distros and BSDs (and NAME= is not defined in /etc)
var distroNames = []string{"Arch Linux", "Debian", "Ubuntu", "Void Linux", "FreeBSD", "NetBSD", "OpenBSD", "Manjaro", "Mint", "Elementary", "MX Linuyx", "Fedora", "openSUSE", "Solus", "Zorin", "CentOS", "KDE neon", "Lite", "Kali", "Antergos", "antiX", "Lubuntu", "PCLinuxOS", "Endless", "Peppermint", "SmartOS", "TrueOS", "Arco", "SparkyLinux", "deepin", "Puppy", "Slackware", "Bodhi", "Tails", "Xubuntu", "Archman", "Bluestar", "Mageia", "Deuvan", "Parrot", "Pop!", "ArchLabs", "Q4OS", "Kubuntu", "Nitrux", "Red Hat", "4MLinux", "Gentoo", "Pinguy", "LXLE", "KaOS", "Ultimate", "Alpine", "Feren", "KNOPPIX", "Robolinux", "Voyager", "Netrunner", "GhostBSD", "Budgie", "ClearOS", "Gecko", "SwagArch", "Emmabuntüs", "Scientific", "Omarine", "Neptune", "NixOS", "Slax", "Clonezilla", "DragonFly", "ExTiX", "OpenBSD", "Redcore", "Ubuntu Studio", "BunsenLabs", "BlackArch", "NuTyX", "ArchBang", "BackBox", "Sabayon", "AUSTRUMI", "Container", "ROSA", "SteamOS", "Tiny Core", "Kodachi", "Qubes", "siduction", "Parabola", "Trisquel", "Vector", "SolydXK", "Elive", "AV Linux", "Artix", "Raspbian", "Porteus"}

// Distro represents the platform, contents of /etc/*release* and name of the
// detected Linux distribution or BSD.
type Distro struct {
	platform    string
	etcContents string
	name        string
	codename    string
	version     string
}

// readEtc returns the contents of /etc/*release* + /etc/issue, or an empty string
func readEtc() string {
	filenames, err := filepath.Glob("/etc/*release*")
	if err != nil {
		return ""
	}
	filenames = append(filenames, "/etc/issue")
	var bs strings.Builder
	for _, filename := range filenames {
		// Try reading all the files
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			continue
		}
		bs.Write(data)
	}
	return bs.String()
}

// expand expands "void" to "Void Linux",
// but shortens "Debian GNU/Linux" to just "Debian".
func expand(name string) string {
	rdict := map[string]string{"void": "Void Linux", "Debian GNU/Linux": "Debian"}
	if _, found := rdict[name]; found {
		return rdict[name]
	}
	return name
}

// Remove parenthesis and capitalize words within parenthesis
func nopar(s string) string {
	if strings.Contains(s, "(") && strings.Contains(s, ")") {
		fields := strings.SplitN(s, "(", 2)
		a := fields[0] + capitalize(fields[1])
		fields = strings.SplitN(a, ")", 2)
		return fields[0] + fields[1]
	}
	return s
}

// detectFromExecutables tries to detect distro information by looking for
// or using existing binaries on the system.
func (d *Distro) detectFromExecutables() {
	// TODO: Generate a list of all files in PATH before performing these checks
	// Executables related to package managers
	if Has("xbps-query") {
		d.name = "Void Linux"
	} else if Has("pacman") {
		d.name = "Arch Linux"
	} else if Has("dnf") {
		d.name = "Fedora"
	} else if Has("yum") {
		d.name = "Fedora"
	} else if Has("zypper") {
		d.name = "openSUSE"
	} else if Has("emerge") {
		d.name = "Gentoo"
	} else if Has("apk") {
		d.name = "Alpine"
	} else if Has("slapt-get") || Has("slackpkg") {
		d.name = "Slackware"
	} else if d.platform == "Darwin" {
		productName := strings.TrimSpace(Run("sw_vers -productName"))
		// Set the platform to either "macOS" or "OS X", if it is in the product name
		if strings.HasPrefix(productName, "Mac OS X") {
			d.platform = "OS X"
		} else if strings.Contains(productName, "macOS") {
			d.platform = "macOS"
		} else {
			d.platform = productName
		}
		// Version number
		d.version = strings.TrimSpace(Run("sw_vers -productVersion"))
		// Codename (like "High Sierra")
		d.codename = AppleCodename(d.version)
		// Mac doesn't really have a distro name, use the platform name
		d.name = d.platform
	} else if Has("/usr/sbin/pkg") {
		d.name = "FreeBSD"
		d.version = strings.TrimSpace(Run("/bin/freebsd-version -u"))
		// Only keep the version number, such as "11.2", ignore the "-RELEASE" part
		if strings.Contains(d.version, "-") {
			d.version = d.version[:strings.LastIndex(d.version, "-")]
		}
		// rpm and dpkg-query should come last, since many distros may include them
	} else if Has("rpm") {
		d.name = "Red Hat"
	} else if Has("dpkg-query") {
		d.name = "Debian"
	}
}

func (d *Distro) detectFromEtc() {
	// First check for Linux distros and BSD distros by grepping in /etc/*release* + /etc/issue
	for _, distroName := range distroNames {
		if d.Grep(distroName) {
			d.name = distroName
			break
		}
	}
	// Examine all lines of text in /etc/*release* + /etc/issue
	for _, line := range strings.Split(d.etcContents, "\n") {
		// Check if NAME= is defined in /etc/*release* + /etc/issue
		if strings.HasPrefix(line, "NAME=") {
			fields := strings.SplitN(strings.TrimSpace(line), "=", 2)
			name := fields[1]
			if name != "" {
				if strings.HasPrefix(name, "\"") && strings.HasSuffix(name, "\"") {
					d.name = name[1 : len(name)-1]
					continue
				}
				d.name = name
			}
			// Check if DISTRIB_CODENAME= (Ubuntu) or VERSION= (Debian) is defined in /etc/*release* + /etc/issue
		} else if strings.HasPrefix(line, "DISTRIB_CODENAME=") || (d.codename == "" && strings.HasPrefix(line, "VERSION=")) {
			fields := strings.SplitN(strings.TrimSpace(line), "=", 2)
			codename := fields[1]
			if codename != "" {
				if strings.HasPrefix(codename, "\"") && strings.HasSuffix(codename, "\"") {
					d.codename = nopar(capitalize(codename[1 : len(codename)-1]))
					continue
				}
				d.codename = nopar(capitalize(codename))
			}
			// Check if DISTRIBVER = is defined in /etc/*release* (NetBSD)
		} else if strings.Contains(line, "DISTRIBVER =") {
			fields := strings.SplitN(strings.TrimSpace(line), "=", 2)
			version := strings.TrimSpace(fields[1])
			if version != "" {
				if strings.HasPrefix(version, "'") && strings.HasSuffix(version, "'") {
					if containsDigit(version) {
						d.version = version[1 : len(version)-1]
					}
					continue
				}
				if containsDigit(version) {
					d.version = version
				}
			}
			// Check if DISTRIB_RELEASE= is defined in /etc/*release*
		} else if strings.HasPrefix(line, "DISTRIB_RELEASE=") {
			fields := strings.SplitN(strings.TrimSpace(line), "=", 2)
			version := fields[1]
			if version != "" {
				if strings.HasPrefix(version, "\"") && strings.HasSuffix(version, "\"") {
					if containsDigit(version) {
						d.version = version[1 : len(version)-1]
					}
					continue
				}
				if containsDigit(version) {
					d.version = version
				}
			}
		} else if d.version == "" && strings.HasPrefix(line, "OS_MAJOR_VERSION=") {
			fields := strings.SplitN(strings.TrimSpace(line), "=", 2)
			version := fields[1]
			if version != "" {
				if strings.HasPrefix(version, "\"") && strings.HasSuffix(version, "\"") {
					if containsDigit(version) {
						d.version = version[1 : len(version)-1]
					}
					continue
				}
				if containsDigit(version) {
					d.version = version
				}
			}
		} else if d.version != "" && !strings.Contains(d.version, ".") && strings.HasPrefix(line, "OS_MINOR_VERSION=") {
			fields := strings.SplitN(strings.TrimSpace(line), "=", 2)
			version := fields[1]
			if version != "" {
				if strings.HasPrefix(version, "\"") && strings.HasSuffix(version, "\"") {
					if containsDigit(version) {
						d.version += "." + version[1:len(version)-1]
					}
					continue
				}
				if containsDigit(version) {
					d.version += "." + version
				}
			}
		}
	}
	// If the codename starts with a word with a digit, followed by a space, use that as the version number,
	// if the currently detected version number is empty. If not, strip the number from the codename.
	if strings.Contains(d.codename, " ") {
		fields := strings.SplitN(d.codename, " ", 2)
		if containsDigit(fields[0]) {
			if d.version == "" {
				d.version = fields[0]
			}
			d.codename = fields[1]
		}
	}
}

// New detects the platform and distro/BSD, then returns a pointer to
// a Distro struct.
func New() *Distro {
	var d Distro
	d.platform = capitalize(runtime.GOOS)
	d.etcContents = readEtc()
	// Distro name, if not detected
	d.name = defaultName
	d.codename = ""
	d.version = ""

	d.detectFromEtc()
	// Replacements
	d.name = expand(d.name)
	if d.name == defaultName {
		// This is only called if no distro has been detected so far
		d.detectFromExecutables()
	}
	return &d
}

// Grep /etc/*release* for the given string.
// If the search fails, a case-insensitive string search is attempted.
// The contents of /etc/*release* is cached.
func (d *Distro) Grep(name string) bool {
	return strings.Contains(d.etcContents, name) || strings.Contains(strings.ToLower(d.etcContents), strings.ToLower(name))
}

// Platform returns the name of the current platform.
// This is the same as `runtime.GOOS`, but capitalized.
func (d *Distro) Platform() string {
	return d.platform
}

// Name returns the detected name of the current distro/BSD, or "Unknown".
func (d *Distro) Name() string {
	return d.name
}

// Codename returns the detected codename of the current distro/BSD,
// or an empty string.
func (d *Distro) Codename() string {
	return d.codename
}

// Version returns the detected release version of the current distro/BSD,
// or an empty string.
func (d *Distro) Version() string {
	return d.version
}

// EtcRelease returns the contents of /etc/*release + /etc/issue, or an empty string.
// The contents are cached.
func (d *Distro) EtcRelease() string {
	return d.etcContents
}

// String returns a string with the current platform, distro
// codename and release version (if available).
// Example strings:
//
//	Linux (Ubuntu Bionic 18.04)
//	Darwin (10.13.3)
func (d *Distro) String() string {
	var sb strings.Builder
	sb.WriteString(d.platform)
	sb.WriteString(" ")
	if d.name != "" || d.codename != "" || d.version != "" {
		sb.WriteString("(")
		needSpace := false
		if d.name != defaultName && d.name != "" {
			sb.WriteString(d.name)
			needSpace = true
		}
		if d.codename != "" {
			if needSpace {
				sb.WriteString(" ")
			}
			sb.WriteString(d.codename)
			needSpace = true
		}
		if d.version != "" {
			if needSpace {
				sb.WriteString(" ")
			}
			sb.WriteString(d.version)
		}
		sb.WriteString(")")
	}
	return sb.String()
}
