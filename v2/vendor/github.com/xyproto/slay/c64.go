package slay

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xyproto/files"
)

var c64Headers = []string{"c64.h", "cbm.h"}

func isC64Include(line string) bool {
	for _, header := range c64Headers {
		if strings.Contains(line, "#include <"+header+">") || strings.Contains(line, `#include "`+header+`"`) {
			return true
		}
	}
	return false
}

func makefileUsesC64(s string) bool {
	return strings.Contains(s, "-t c64") || strings.Contains(s, "-tc64")
}

func cc65ToolPath(name string) string {
	if home := os.Getenv("CC65_HOME"); home != "" {
		if p := filepath.Join(home, "bin", name); fileExists(p) {
			return p
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func buildC64(srcs []string, prg string, opts BuildOptions) ([]byte, []string, error) {
	cl65 := cc65ToolPath("cl65")
	if cl65 == "" {
		return nil, nil, fmt.Errorf("cl65 was not found, please install cc65")
	}
	args := []string{"-t", "c64"}
	if opts.Debug {
		args = append(args, "-g")
	} else {
		args = append(args, "-Oirs")
	}
	args = append(args, "-o", prg)
	args = append(args, srcs...)
	cmd := exec.Command(cl65, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, []string{cmdToString(cmd)}, fmt.Errorf("compilation failed: %w", err)
	}
	return out, []string{cmdToString(cmd)}, nil
}

func C64EmulatorCommand(prg string, args ...string) *exec.Cmd {
	for _, emulator := range []string{"x64sc", "x64"} {
		if p := files.WhichCached(emulator); p != "" {
			cmdArgs := append(append([]string{}, args...), "-autostart", prg)
			return exec.Command(p, cmdArgs...)
		}
	}
	return nil
}
