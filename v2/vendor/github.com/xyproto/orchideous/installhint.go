package orchideous

import (
	"fmt"
	"sync"

	"github.com/xyproto/distrodetector"
)

var (
	distroOnce sync.Once
	detected   *distrodetector.Distro
)

func getDistro() *distrodetector.Distro {
	distroOnce.Do(func() {
		detected = distrodetector.New()
	})
	return detected
}

// packageNames maps a tool to its package name per package manager.
// Keys: "apt", "dnf", "pacman", "brew", "apk", "zypper", "xbps", "emerge"
type packageNames map[string]string

var toolPackages = map[string]packageNames{
	"gcc": {
		"apt":    "build-essential",
		"dnf":    "gcc gcc-c++",
		"pacman": "base-devel",
		"brew":   "gcc",
		"apk":    "build-base",
		"zypper": "gcc gcc-c++",
		"xbps":   "base-devel",
		"emerge": "sys-devel/gcc",
	},
	"ninja": {
		"apt":    "ninja-build",
		"dnf":    "ninja-build",
		"pacman": "ninja",
		"brew":   "ninja",
		"apk":    "ninja",
		"zypper": "ninja",
		"xbps":   "ninja",
		"emerge": "dev-build/ninja",
	},
	"make": {
		"apt":    "build-essential",
		"dnf":    "make",
		"pacman": "base-devel",
		"brew":   "make",
		"apk":    "build-base",
		"zypper": "make",
		"xbps":   "base-devel",
		"emerge": "sys-devel/make",
	},
	"cmake": {
		"apt":    "cmake",
		"dnf":    "cmake",
		"pacman": "cmake",
		"brew":   "cmake",
		"apk":    "cmake",
		"zypper": "cmake",
		"xbps":   "cmake",
		"emerge": "dev-build/cmake",
	},
	"clang-format": {
		"apt":    "clang-format",
		"dnf":    "clang-tools-extra",
		"pacman": "clang",
		"brew":   "clang-format",
		"apk":    "clang-extra-tools",
		"zypper": "clang",
		"xbps":   "clang-tools-extra",
		"emerge": "sys-devel/clang",
	},
	"valgrind": {
		"apt":    "valgrind",
		"dnf":    "valgrind",
		"pacman": "valgrind",
		"brew":   "valgrind",
		"apk":    "valgrind",
		"zypper": "valgrind",
		"xbps":   "valgrind",
		"emerge": "dev-debug/valgrind",
	},
	"gdb": {
		"apt":    "gdb",
		"dnf":    "gdb",
		"pacman": "gdb",
		"brew":   "gdb",
		"apk":    "gdb",
		"zypper": "gdb",
		"xbps":   "gdb",
		"emerge": "dev-debug/gdb",
	},
}

// distroToManager maps detected distro names to the package manager command.
func distroToManager(name string) string {
	switch name {
	case "Arch Linux", "Manjaro", "EndeavourOS", "ArcoLinux", "Garuda Linux",
		"ArchLabs", "ArchBang", "BlackArch", "Artix", "Parabola", "CachyOS":
		return "pacman"
	case "Debian", "Ubuntu", "Mint", "Pop!_OS", "Elementary", "Kubuntu",
		"Xubuntu", "Lubuntu", "Ubuntu Studio", "Kali", "MX Linux",
		"antiX", "Devuan", "Tails", "Raspbian", "BunsenLabs", "KDE neon",
		"Peppermint", "LXLE", "Q4OS", "Bodhi", "SparkyLinux", "Zorin":
		return "apt"
	case "Fedora", "Red Hat", "CentOS", "Rocky Linux", "AlmaLinux",
		"Oracle Linux", "Nobara", "Bazzite":
		return "dnf"
	case "openSUSE", "Gecko":
		return "zypper"
	case "Void Linux":
		return "xbps"
	case "Alpine":
		return "apk"
	case "Gentoo":
		return "emerge"
	case "macOS", "OS X":
		return "brew"
	default:
		return ""
	}
}

// installCommand returns the full install command for a given tool on the current distro.
func installCommand(manager, tool string) string {
	pkg, ok := toolPackages[tool]
	if !ok {
		return ""
	}
	pkgName, ok := pkg[manager]
	if !ok {
		return ""
	}
	switch manager {
	case "apt":
		return "apt install " + pkgName
	case "dnf":
		return "dnf install " + pkgName
	case "pacman":
		return "pacman -S " + pkgName
	case "brew":
		return "brew install " + pkgName
	case "apk":
		return "apk add " + pkgName
	case "zypper":
		return "zypper install " + pkgName
	case "xbps":
		return "xbps-install " + pkgName
	case "emerge":
		return "emerge " + pkgName
	}
	return ""
}

// installHint returns a human-readable install hint for the given tool,
// using the detected distro's package manager. Falls back to a generic message.
func installHint(tool string) string {
	d := getDistro()
	manager := distroToManager(d.Name())
	if manager == "" {
		return fmt.Sprintf("install %s using your package manager", tool)
	}
	cmd := installCommand(manager, tool)
	if cmd == "" {
		return fmt.Sprintf("install %s using your package manager", tool)
	}
	return cmd
}
