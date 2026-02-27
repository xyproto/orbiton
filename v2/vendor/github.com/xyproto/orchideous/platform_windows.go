//go:build windows

package orchideous

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func isLinux() bool  { return false }
func isDarwin() bool { return false }

// Windows doesn't need a POSIX source define; use an empty string so it's harmless if appended.
const platformCDefine = "-D_WIN32_WINNT=0x0601"

func extraLDLibPaths() []string { return nil }

// Windows does not use --as-needed.
func prependAsNeededFlag(ldflags []string) []string { return ldflags }

// detectPlatformType detects the Windows development environment.
// Returns "msys2" if running inside MSYS2/MinGW, "vcpkg" if vcpkg is
// available, or "generic" as a fallback.
func detectPlatformType() string {
	// MSYS2 sets MSYSTEM (MINGW64, UCRT64, CLANG64, etc.)
	if msystem := os.Getenv("MSYSTEM"); msystem != "" {
		if _, err := exec.LookPath("pacman"); err == nil {
			return "msys2"
		}
	}
	if vcpkgRoot() != "" {
		return "vcpkg"
	}
	return "generic"
}

// vcpkgRoot returns the vcpkg root directory, or "" if not found.
func vcpkgRoot() string {
	if root := os.Getenv("VCPKG_ROOT"); root != "" && fileExists(root) {
		return root
	}
	if p, err := exec.LookPath("vcpkg"); err == nil {
		return filepath.Dir(p)
	}
	return ""
}

// vcpkgTriplet returns the active vcpkg triplet for the current platform.
func vcpkgTriplet() string {
	if t := os.Getenv("VCPKG_DEFAULT_TRIPLET"); t != "" {
		return t
	}
	return "x64-windows"
}

// vcpkgInstalledDir returns the vcpkg installed directory for the active triplet.
func vcpkgInstalledDir() string {
	root := vcpkgRoot()
	if root == "" {
		return ""
	}
	triplet := vcpkgTriplet()
	dir := filepath.Join(root, "installed", triplet)
	if fileExists(dir) {
		return dir
	}
	return ""
}

// msys2Prefix returns the MSYS2 environment prefix (e.g. C:\msys64\mingw64).
func msys2Prefix() string {
	// MINGW_PREFIX is set by MSYS2 shells (e.g. /mingw64)
	if prefix := os.Getenv("MINGW_PREFIX"); prefix != "" {
		// Convert MSYS2 path to Windows path if needed
		if msysRoot := os.Getenv("MSYSTEM_PREFIX"); msysRoot != "" && fileExists(msysRoot) {
			return msysRoot
		}
		// Try common MSYS2 install locations
		for _, root := range []string{`C:\msys64`, `C:\msys2`, `D:\msys64`} {
			candidate := filepath.Join(root, filepath.FromSlash(prefix))
			if fileExists(candidate) {
				return candidate
			}
		}
	}
	// Fallback: infer from pacman location
	if p, err := exec.LookPath("pacman"); err == nil {
		// pacman is in <root>/usr/bin/pacman, and the mingw prefix is <root>/mingw64 etc.
		root := filepath.Dir(filepath.Dir(filepath.Dir(p)))
		msystem := strings.ToLower(os.Getenv("MSYSTEM"))
		switch {
		case strings.Contains(msystem, "clang64"):
			return filepath.Join(root, "clang64")
		case strings.Contains(msystem, "ucrt64"):
			return filepath.Join(root, "ucrt64")
		case strings.Contains(msystem, "clang32"):
			return filepath.Join(root, "clang32")
		case strings.Contains(msystem, "mingw32"):
			return filepath.Join(root, "mingw32")
		default: // MINGW64 or fallback
			return filepath.Join(root, "mingw64")
		}
	}
	return ""
}

// extraWindowsIncludeDirs returns additional include directories for the
// detected Windows development environment.
func extraWindowsIncludeDirs() []string {
	var dirs []string
	platform := detectPlatformType()
	switch platform {
	case "msys2":
		prefix := msys2Prefix()
		if prefix != "" {
			inc := filepath.Join(prefix, "include")
			if fileExists(inc) {
				dirs = append(dirs, inc)
			}
		}
	case "vcpkg":
		installed := vcpkgInstalledDir()
		if installed != "" {
			inc := filepath.Join(installed, "include")
			if fileExists(inc) {
				dirs = append(dirs, inc)
			}
		}
	}
	return dirs
}

// systemIncludeDirs returns the system include directories on Windows.
func systemIncludeDirs() []string {
	dirs := extraWindowsIncludeDirs()
	if fileExists("/usr/include") {
		dirs = append(dirs, "/usr/include")
	}
	cxx := findCompiler(false, false)
	if cxx != "" {
		out, err := exec.Command(cxx, "-dumpmachine").Output()
		if err == nil {
			machine := strings.TrimSpace(string(out))
			machineDir := "/usr/include/" + machine
			if fileExists(machineDir) {
				dirs = append(dirs, machineDir)
			}
		}
	}
	if fileExists("/usr/local/include") {
		dirs = append(dirs, "/usr/local/include")
	}
	if fileExists("/usr/pkg/include") {
		dirs = append(dirs, "/usr/pkg/include")
	}
	return dirs
}

// compilerSupportsStd checks if the compiler supports a given -std= flag.
// On Windows, uses a temp file instead of piping via sh -c.
func compilerSupportsStd(compiler, std string) bool {
	tmpFile := filepath.Join(os.TempDir(), "oh_stdcheck.cpp")
	os.WriteFile(tmpFile, []byte("int main(){}"), 0o644)
	defer os.Remove(tmpFile)
	cmd := exec.Command(compiler, "-std="+std, "-fsyntax-only", tmpFile)
	cmd.Stderr = nil
	cmd.Stdout = nil
	return cmd.Run() == nil
}

// msys2IncludePathToFlags resolves an include path to compiler/linker flags
// using MSYS2's pacman package manager.
func msys2IncludePathToFlags(includePath string) string {
	if includePath == "" || !fileExists(includePath) {
		return ""
	}
	pkg := lookupPackageOwnerMSYS2(includePath)
	if pkg == "" || skipPackages[pkg] {
		return ""
	}
	pcFiles := lookupPCFilesMSYS2(pkg)
	if len(pcFiles) == 0 {
		prefix := msys2Prefix()
		libDir := filepath.Join(prefix, "lib")
		result := tryLibFallbackWindows(includePath, pkg, []string{libDir})
		if result == "" && pkg != "boost" && pkg != "qt5-base" && pkg != "qt6-base" {
			fmt.Fprintf(os.Stderr, "WARNING: No pkg-config files for: %s\n", pkg)
		}
		return result
	}
	return pcFilesToFlags(pcFiles)
}

// lookupPackageOwnerMSYS2 queries pacman for the package owning a file.
func lookupPackageOwnerMSYS2(filePath string) string {
	// Convert Windows path to MSYS2-style path for pacman
	msysPath := windowsToMSYS2Path(filePath)
	out, err := exec.Command("pacman", "-Qo", "--quiet", msysPath).Output()
	if err != nil {
		// Try with the original Windows path
		out, err = exec.Command("pacman", "-Qo", "--quiet", filePath).Output()
		if err != nil {
			return ""
		}
	}
	return strings.TrimSpace(string(out))
}

// lookupPCFilesMSYS2 queries pacman for .pc files belonging to a package.
func lookupPCFilesMSYS2(pkg string) []string {
	if cached, ok := cachedPCFiles[pkg]; ok {
		return cached
	}
	out, err := exec.Command("pacman", "-Ql", pkg).Output()
	if err != nil {
		cachedPCFiles[pkg] = nil
		return nil
	}
	var pcFiles []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		// pacman -Ql output: "pkgname /path/to/file"
		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		if len(parts) == 2 && strings.HasSuffix(parts[1], ".pc") {
			pcFile := parts[1]
			// Convert MSYS2 path to Windows path
			if winPath := msys2ToWindowsPath(pcFile); winPath != "" {
				pcFiles = append(pcFiles, winPath)
			} else {
				pcFiles = append(pcFiles, pcFile)
			}
		}
	}
	cachedPCFiles[pkg] = pcFiles
	return pcFiles
}

// vcpkgIncludePathToFlags resolves an include path to flags using vcpkg's
// installed package tree.
func vcpkgIncludePathToFlags(includePath string) string {
	if includePath == "" || !fileExists(includePath) {
		return ""
	}
	installed := vcpkgInstalledDir()
	if installed == "" {
		return ""
	}
	// vcpkg provides pkg-config files in <installed>/lib/pkgconfig
	pkgconfigDir := filepath.Join(installed, "lib", "pkgconfig")
	if !fileExists(pkgconfigDir) {
		return ""
	}

	// Try to guess the package name from the include path
	pkgGuess := vcpkgGuessPackage(includePath, installed)
	if pkgGuess == "" {
		return ""
	}

	// Look for a .pc file matching the package guess
	pcFile := filepath.Join(pkgconfigDir, pkgGuess+".pc")
	if fileExists(pcFile) {
		return vcpkgPkgConfigFlags(pkgGuess, pkgconfigDir)
	}

	// Try to find any .pc file in the pkgconfig dir that matches
	matches, _ := filepath.Glob(filepath.Join(pkgconfigDir, "*.pc"))
	for _, m := range matches {
		pcName := strings.TrimSuffix(filepath.Base(m), ".pc")
		if strings.EqualFold(pcName, pkgGuess) {
			return vcpkgPkgConfigFlags(pcName, pkgconfigDir)
		}
	}

	// Fallback: try to link with -l<name> and -I/-L paths
	libDir := filepath.Join(installed, "lib")
	incDir := filepath.Join(installed, "include")
	return vcpkgLibFallback(pkgGuess, libDir, incDir)
}

// vcpkgGuessPackage guesses the vcpkg package name from an include path.
func vcpkgGuessPackage(includePath, installedDir string) string {
	incDir := filepath.Join(installedDir, "include")
	// Strip the include directory prefix to get the relative path
	rel, err := filepath.Rel(incDir, includePath)
	if err != nil {
		return ""
	}
	// The first path component is typically the package name
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) > 0 && parts[0] != "." && parts[0] != ".." {
		return strings.ToLower(parts[0])
	}
	return ""
}

// vcpkgPkgConfigFlags runs pkg-config with the vcpkg pkgconfig directory.
func vcpkgPkgConfigFlags(pkgName, pkgconfigDir string) string {
	cmd := exec.Command("pkg-config", "--cflags", "--libs", pkgName)
	cmd.Env = append(os.Environ(), "PKG_CONFIG_PATH="+pkgconfigDir)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// vcpkgLibFallback tries direct -l/-I/-L flags for a vcpkg package.
func vcpkgLibFallback(name, libDir, incDir string) string {
	// Check for .lib (MSVC) or .a (MinGW) files
	for _, ext := range []string{".lib", ".a"} {
		candidates := []string{
			filepath.Join(libDir, name+ext),
			filepath.Join(libDir, "lib"+name+ext),
		}
		for _, c := range candidates {
			if fileExists(c) {
				flags := "-l" + name
				if fileExists(incDir) {
					flags = "-I" + incDir + " " + flags
				}
				if fileExists(libDir) {
					flags += " -L" + libDir
				}
				return flags
			}
		}
	}
	return ""
}

// tryLibFallbackWindows is like tryLibFallback but checks for Windows library extensions.
func tryLibFallbackWindows(includePath, packageName string, libPaths []string) string {
	baseName := strings.TrimSuffix(filepath.Base(includePath), filepath.Ext(filepath.Base(includePath)))
	candidates := []string{packageName, baseName}

	for _, name := range candidates {
		if name == "" {
			continue
		}
		for _, libPath := range libPaths {
			for _, pattern := range []string{
				filepath.Join(libPath, "lib"+name+".a"),
				filepath.Join(libPath, "lib"+name+".dll.a"),
				filepath.Join(libPath, name+".lib"),
			} {
				if fileExists(pattern) {
					result := "-l" + name
					incDir := filepath.Dir(includePath)
					if fileExists(incDir) {
						result = "-I" + incDir + " " + result
					}
					result += " -L" + libPath
					return result
				}
			}
		}
	}
	return ""
}

// windowsToMSYS2Path converts a Windows path to an MSYS2 path.
func windowsToMSYS2Path(winPath string) string {
	// C:\msys64\mingw64\include\SDL2 -> /mingw64/include/SDL2
	winPath = filepath.ToSlash(winPath)
	// Strip common MSYS2 root prefixes
	for _, root := range []string{"C:/msys64", "C:/msys2", "D:/msys64"} {
		if strings.HasPrefix(strings.ToLower(winPath), strings.ToLower(root)) {
			return winPath[len(root):]
		}
	}
	return winPath
}

// msys2ToWindowsPath converts an MSYS2 path to a Windows path.
func msys2ToWindowsPath(msysPath string) string {
	if !strings.HasPrefix(msysPath, "/") {
		return msysPath
	}
	// Try to resolve via MSYS2 root
	for _, root := range []string{`C:\msys64`, `C:\msys2`, `D:\msys64`} {
		candidate := filepath.Join(root, filepath.FromSlash(msysPath))
		if fileExists(candidate) {
			return candidate
		}
	}
	// If MSYSTEM_PREFIX is set, use that as the base
	if prefix := os.Getenv("MSYSTEM_PREFIX"); prefix != "" {
		candidate := filepath.Join(filepath.Dir(filepath.Dir(prefix)), filepath.FromSlash(msysPath))
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}
