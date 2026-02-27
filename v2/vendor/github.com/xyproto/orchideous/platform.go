package orchideous

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xyproto/files"
)

// skipPackages are packages that should be skipped when resolving includes.
var skipPackages = map[string]bool{
	"glibc": true, "gcc": true, "wine": true,
}

// cachedPCFiles caches package -> .pc file list lookups.
var cachedPCFiles = make(map[string][]string)

// resolveIncludesViaPackageManager resolves unresolved includes using the platform's
// package manager. Returns additional cflags and ldflags.
func resolveIncludesViaPackageManager(includes []string, systemIncDirs []string, win64 bool, cxx string) (cflags, ldflags []string) {
	if !hasPkgConfig() {
		return nil, nil
	}

	platform := detectPlatformType()
	resolved := make(map[string]bool)

	for _, inc := range includes {
		if resolved[inc] {
			continue
		}
		// First pass: direct path lookup
		for _, sysDir := range systemIncDirs {
			incPath := filepath.Join(sysDir, inc)
			flags := platformResolve(platform, incPath, cxx)
			if flags != "" {
				cflags, ldflags = mergeFlags(cflags, ldflags, flags)
				resolved[inc] = true
				break
			}
		}
		// Second pass: deeper search with find
		if !resolved[inc] {
			for _, sysDir := range systemIncDirs {
				incPath := findIncludeFile(sysDir, inc)
				if incPath == "" {
					continue
				}
				flags := platformResolve(platform, incPath, cxx)
				if flags != "" {
					cflags, ldflags = mergeFlags(cflags, ldflags, flags)
					resolved[inc] = true
					break
				}
			}
		}
	}
	return cflags, ldflags
}

// platformResolve dispatches to the correct platform-specific resolver.
func platformResolve(platform, incPath, cxx string) string {
	switch platform {
	case "arch":
		return archIncludePathToFlags(incPath)
	case "deb":
		return debIncludePathToFlags(incPath, cxx)
	case "freebsd":
		return freebsdIncludePathToFlags(incPath)
	case "openbsd":
		return openbsdIncludePathToFlags(incPath)
	case "brew":
		return brewIncludePathToFlags(incPath)
	case "msys2":
		return msys2IncludePathToFlags(incPath)
	case "vcpkg":
		return vcpkgIncludePathToFlags(incPath)
	default:
		return genericIncludePathToFlags(incPath)
	}
}

// findIncludeFile searches for an include file under a system include directory.
func findIncludeFile(sysDir, inc string) string {
	// Direct check first (works on all platforms)
	direct := filepath.Join(sysDir, inc)
	if fileExists(direct) {
		return direct
	}
	// Try recursive walk (up to 3 levels deep) — portable alternative to Unix find
	maxDepth := 3
	target := filepath.ToSlash(inc)
	var found string
	filepath.WalkDir(sysDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		// Enforce max depth
		rel, _ := filepath.Rel(sysDir, path)
		depth := strings.Count(filepath.ToSlash(rel), "/")
		if d.IsDir() && depth >= maxDepth {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(filepath.ToSlash(path), target) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// pcFilesToFlags takes a list of .pc file paths and returns combined pkg-config flags.
func pcFilesToFlags(pcFiles []string) string {
	var allFlags []string
	seen := make(map[string]bool)
	for _, pcFile := range pcFiles {
		pcName := strings.TrimSuffix(filepath.Base(pcFile), ".pc")
		flags := pkgConfigFlags(pcName)
		if flags == "" && pcName != "glm" && pcName != "libglvnd" && pcName != "RapidJSON" {
			flags = "-l" + pcName
		}
		for f := range strings.FieldsSeq(flags) {
			if !seen[f] {
				seen[f] = true
				allFlags = append(allFlags, f)
			}
		}
	}
	return strings.Join(allFlags, " ")
}

// pcFilesToFlagsWithDir is like pcFilesToFlags but sets PKG_CONFIG_PATH per-file (for brew).
func pcFilesToFlagsWithDir(pcFiles []string) string {
	var allFlags []string
	seen := make(map[string]bool)
	for _, pcFile := range pcFiles {
		pcName := strings.TrimSuffix(filepath.Base(pcFile), ".pc")
		pcDir := filepath.Dir(pcFile)
		out, err := exec.Command("sh", "-c",
			fmt.Sprintf(`PKG_CONFIG_PATH="%s" pkg-config --cflags --libs %s 2>/dev/null`, pcDir, pcName)).Output()
		flags := ""
		if err == nil {
			flags = strings.TrimSpace(string(out))
		}
		if flags == "" && pcName != "glm" && pcName != "libglvnd" && pcName != "RapidJSON" {
			flags = "-l" + pcName
		}
		for f := range strings.FieldsSeq(flags) {
			if !seen[f] {
				seen[f] = true
				allFlags = append(allFlags, f)
			}
		}
	}
	return strings.Join(allFlags, " ")
}

// tryLibFallback tries to find a matching .so in lib directories when no .pc files exist.
func tryLibFallback(includePath, packageName string, libPaths []string) string {
	booststyle := ""
	parts := strings.Split(includePath, "/")
	if len(parts) >= 2 {
		base := strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1]))
		booststyle = parts[len(parts)-2] + "_" + base
	}
	baseName := strings.TrimSuffix(filepath.Base(includePath), filepath.Ext(filepath.Base(includePath)))
	candidates := []string{packageName, booststyle, strings.ToUpper(packageName), baseName}

	for _, name := range candidates {
		if name == "" {
			continue
		}
		for _, libPath := range libPaths {
			soPath := filepath.Join(libPath, "lib"+name+".so")
			if fileExists(soPath) {
				result := "-l" + name
				incDir := filepath.Dir(includePath)
				if fileExists(incDir) {
					result += " -I" + incDir
				}
				ppPath := filepath.Join(libPath, "lib"+name+"++.so")
				if fileExists(ppPath) {
					result += " -l" + name + "++"
				}
				return result
			}
		}
	}
	return ""
}

// lookupPCFiles queries the package manager for .pc files belonging to a package.
func lookupPCFiles(platform, pkg string) []string {
	if cached, ok := cachedPCFiles[pkg]; ok {
		return cached
	}
	var cmd string
	switch platform {
	case "arch":
		cmd = fmt.Sprintf(`/usr/bin/pacman -Ql -- %s 2>/dev/null | /usr/bin/grep '\.pc$' | /usr/bin/cut -d' ' -f2-`, pkg)
	case "deb":
		cmd = fmt.Sprintf(`LC_ALL=C /usr/bin/dpkg-query -L %s 2>/dev/null | grep '\.pc$'`, pkg)
	case "freebsd":
		cmd = fmt.Sprintf(`/usr/sbin/pkg list %s 2>/dev/null | /usr/bin/grep '\.pc$'`, pkg)
	case "openbsd":
		cmd = fmt.Sprintf(`/usr/sbin/pkg_info -L %s 2>/dev/null | grep '\.pc$'`, pkg)
	case "brew":
		cmd = fmt.Sprintf(`LC_ALL=C brew ls --verbose %s 2>/dev/null | grep '\.pc$'`, pkg)
	default:
		return nil
	}
	out, err := exec.Command("sh", "-c", cmd).Output()
	var pcFiles []string
	if err == nil {
		for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
			if line != "" {
				pcFiles = append(pcFiles, line)
			}
		}
	}
	cachedPCFiles[pkg] = pcFiles
	return pcFiles
}

// lookupPackageOwner queries the package manager for who owns a file.
func lookupPackageOwner(platform, filePath string) string {
	var cmd string
	switch platform {
	case "arch":
		cmd = "LC_ALL=C /usr/bin/pacman -Qo -- " + filePath + " 2>/dev/null | /usr/bin/cut -d' ' -f5"
	case "deb":
		cmd = "LC_ALL=C /usr/bin/dpkg-query -S " + filePath + " 2>/dev/null | /usr/bin/cut -d: -f1"
	case "freebsd":
		cmd = "/usr/sbin/pkg which -q " + filePath + " 2>/dev/null | cut -d- -f1-"
	case "openbsd":
		cmd = "/usr/sbin/pkg_info -E " + filePath + " 2>/dev/null | head -1 | cut -d' ' -f2 | cut -d- -f1"
	default:
		return ""
	}
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// archIncludePathToFlags resolves an include path to flags on Arch Linux.
func archIncludePathToFlags(includePath string) string {
	if includePath == "" || !fileExists(includePath) {
		return ""
	}
	pkg := lookupPackageOwner("arch", includePath)
	if pkg == "" || skipPackages[pkg] {
		return ""
	}
	pcFiles := lookupPCFiles("arch", pkg)
	if len(pcFiles) == 0 {
		result := tryLibFallback(includePath, pkg, []string{"/usr/lib"})
		if result == "" && pkg != "boost" && pkg != "qt5-base" && pkg != "qt6-base" {
			fmt.Fprintf(os.Stderr, "WARNING: No pkg-config files for: %s\n", pkg)
		}
		return result
	}
	return pcFilesToFlags(pcFiles)
}

// debIncludePathToFlags resolves an include path to flags on Debian/Ubuntu.
func debIncludePathToFlags(includePath, cxx string) string {
	if includePath == "" || !fileExists(includePath) {
		return ""
	}
	pkg := lookupPackageOwner("deb", includePath)
	if pkg == "" || skipPackages[pkg] {
		return ""
	}
	pcFiles := lookupPCFiles("deb", pkg)
	if len(pcFiles) == 0 {
		machineName := ""
		if cxx != "" {
			if out, err := exec.Command(cxx, "-dumpmachine").Output(); err == nil {
				machineName = strings.TrimSpace(string(out))
			}
		}
		libPaths := []string{"/usr/lib", "/usr/lib/x86_64-linux-gnu", "/usr/local/lib"}
		if machineName != "" {
			libPaths = append(libPaths, "/usr/lib/"+machineName)
		}
		result := tryLibFallback(includePath, pkg, libPaths)
		if result == "" && pkg != "boost" && pkg != "qt5-base" && pkg != "qt6-base" {
			fmt.Fprintf(os.Stderr, "WARNING: No pkg-config files for: %s\n", pkg)
		}
		return result
	}
	return pcFilesToFlags(pcFiles)
}

// freebsdIncludePathToFlags resolves an include path to flags on FreeBSD.
func freebsdIncludePathToFlags(includePath string) string {
	if includePath == "" || !fileExists(includePath) {
		return ""
	}
	pkg := lookupPackageOwner("freebsd", includePath)
	if pkg == "" || skipPackages[pkg] {
		return ""
	}
	pcFiles := lookupPCFiles("freebsd", pkg)
	if len(pcFiles) == 0 {
		result := tryLibFallback(includePath, pkg, []string{"/usr/local/lib"})
		if result == "" && pkg != "boost" && pkg != "qt5-base" && pkg != "qt6-base" {
			fmt.Fprintf(os.Stderr, "WARNING: No pkg-config files for: %s\n", pkg)
		}
		return result
	}
	return pcFilesToFlags(pcFiles)
}

// openbsdIncludePathToFlags resolves an include path to flags on OpenBSD.
func openbsdIncludePathToFlags(includePath string) string {
	if includePath == "" || !fileExists(includePath) {
		return ""
	}
	pkg := lookupPackageOwner("openbsd", includePath)
	if pkg == "" || skipPackages[pkg] {
		return ""
	}
	pcFiles := lookupPCFiles("openbsd", pkg)
	if len(pcFiles) == 0 {
		result := tryLibFallback(includePath, pkg, []string{"/usr/local/lib"})
		if result == "" && pkg != "boost" && pkg != "qt5-base" && pkg != "qt6-base" {
			fmt.Fprintf(os.Stderr, "WARNING: No pkg-config files for: %s\n", pkg)
		}
		return result
	}
	return pcFilesToFlags(pcFiles)
}

// brewIncludePathToFlags resolves an include path to flags on macOS with Homebrew.
func brewIncludePathToFlags(includePath string) string {
	if includePath == "" {
		return ""
	}
	normPath := filepath.Clean(includePath)
	if !fileExists(normPath) {
		return ""
	}
	realPath, err := filepath.EvalSymlinks(includePath)
	if err != nil {
		realPath = includePath
	}

	var pkg string
	if strings.HasPrefix(realPath, "/usr/local/Cellar/") && strings.Count(realPath, string(os.PathSeparator)) > 4 {
		pkg = strings.Split(realPath[18:], string(os.PathSeparator))[0]
	} else if strings.HasPrefix(realPath, "/opt/homebrew/Cellar/") && strings.Count(realPath, string(os.PathSeparator)) > 4 {
		pkg = strings.Split(realPath[20:], string(os.PathSeparator))[0]
	} else {
		pkg = strings.Replace(includePath, "/usr/local/include/", "", 1)
		pkg = strings.Replace(pkg, "/opt/homebrew/include/", "", 1)
	}
	if pkg == "" || skipPackages[pkg] {
		return ""
	}

	pcFiles := lookupPCFiles("brew", pkg)
	if len(pcFiles) == 0 {
		result := tryLibFallback(includePath, pkg, []string{"/usr/local/lib", "/opt/homebrew/lib", "/usr/lib"})
		if result == "" && pkg != "boost" && pkg != "qt5-base" && pkg != "qt6-base" {
			fmt.Fprintf(os.Stderr, "WARNING: No pkg-config files for: %s\n", pkg)
		}
		return result
	}
	return pcFilesToFlagsWithDir(pcFiles)
}

// genericIncludePathToFlags tries to resolve flags on an unknown Linux distro.
func genericIncludePathToFlags(includePath string) string {
	if includePath == "" || !fileExists(includePath) {
		return ""
	}
	parts := strings.Split(includePath, string(os.PathSeparator))
	var pkgGuess string
	if len(parts) > 3 {
		pkgGuess = parts[3]
	} else {
		return ""
	}

	booststyle := ""
	pathParts := strings.Split(includePath, "/")
	if len(pathParts) >= 2 {
		base := strings.TrimSuffix(pathParts[len(pathParts)-1], filepath.Ext(pathParts[len(pathParts)-1]))
		booststyle = pathParts[len(pathParts)-2] + "_" + base
	}

	candidates := []string{pkgGuess, booststyle, strings.ToLower(pkgGuess)}
	for _, pkg := range candidates {
		if skipPackages[pkg] {
			return ""
		}
	}

	libPaths := []string{"/usr/lib", "/usr/lib/x86_64-linux-gnu", "/usr/local/lib", "/usr/pkg/lib"}
	for _, pkg := range candidates {
		if pkg == "" {
			continue
		}
		for _, soName := range []string{"lib" + pkg + ".so", "lib" + strings.ToUpper(pkg) + ".so"} {
			for _, libPath := range libPaths {
				if fileExists(filepath.Join(libPath, soName)) {
					libName := soName[3 : len(soName)-3]
					ppSo := strings.Replace(soName, ".so", "++.so", 1)
					if fileExists(filepath.Join(libPath, ppSo)) {
						return "-l" + libName + " -l" + libName + "++"
					}
					return "-l" + libName
				}
			}
		}
	}
	return ""
}

// recommendPackage suggests a package to install for missing includes.
func recommendPackage(missingIncludes []string) {
	if len(missingIncludes) == 0 {
		return
	}
	platform := detectPlatformType()
	for _, inc := range missingIncludes {
		found := false
		for _, sysDir := range systemIncludeDirs() {
			if fileExists(filepath.Join(sysDir, inc)) {
				found = true
				break
			}
		}
		if found {
			continue
		}
		switch platform {
		case "arch":
			if files.WhichCached("pkgfile") != "" {
				out, err := exec.Command("sh", "-c", "LC_ALL=C pkgfile "+inc+" 2>/dev/null").Output()
				if err == nil {
					for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
						pkg := line
						if strings.Contains(pkg, "/") {
							parts := strings.SplitN(pkg, "/", 2)
							pkg = parts[1]
						}
						if pkg != "" && !skipPackages[pkg] {
							fmt.Fprintf(os.Stderr, "\nerror: Could not find \"%s\", install with: pacman -S %s\n\n", inc, pkg)
							os.Exit(1)
						}
					}
				}
			}
		case "deb":
			if files.WhichCached("apt-file") != "" {
				out, err := exec.Command("sh", "-c", "LC_ALL=C apt-file find -Fl "+inc+" 2>/dev/null").Output()
				if err == nil {
					pkg := strings.TrimSpace(string(out))
					if pkg != "" && !skipPackages[pkg] {
						fmt.Fprintf(os.Stderr, "\nerror: Could not find \"%s\", install with: apt install %s\n\n", inc, pkg)
						os.Exit(1)
					}
				}
			}
		}
	}
}
