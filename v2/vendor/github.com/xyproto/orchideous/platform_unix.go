//go:build !windows

package orchideous

import (
	"os/exec"
	"strings"
)

// systemIncludeDirs returns the system include directories on Unix-like systems.
func systemIncludeDirs() []string {
	var dirs []string
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
func compilerSupportsStd(compiler, std string) bool {
	cmd := exec.Command("sh", "-c",
		"echo 'int main(){}' | "+compiler+" -std="+std+" -x c++ -fsyntax-only - 2>/dev/null")
	return cmd.Run() == nil
}

// msys2IncludePathToFlags is a no-op on non-Windows platforms.
func msys2IncludePathToFlags(_ string) string { return "" }

// vcpkgIncludePathToFlags is a no-op on non-Windows platforms.
func vcpkgIncludePathToFlags(_ string) string { return "" }
