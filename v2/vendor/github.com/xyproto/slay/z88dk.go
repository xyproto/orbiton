package slay

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xyproto/files"
)

// z88dkHeaders maps the C headers that select a z88dk target to that target
var z88dkHeaders = []struct{ header, target string }{
	{"arch/zx.h", "zx"},
	{"spectrum.h", "zx"},
	{"arch/zx/spectrum.h", "zx"},
}

// z88dkIncludeTarget returns the z88dk target that the given #include line selects, or an empty string
func z88dkIncludeTarget(line string) string {
	for _, h := range z88dkHeaders {
		if strings.Contains(line, "#include <"+h.header+">") || strings.Contains(line, `#include "`+h.header+`"`) {
			return h.target
		}
	}
	return ""
}

// makefileZ88DKTarget returns the z88dk target that the given Makefile contents build for, or an empty string
func makefileZ88DKTarget(s string) string {
	if strings.Contains(s, "zcc +zx ") || strings.Contains(s, "zcc +zx\n") {
		return "zx"
	}
	return ""
}

// z88dkExtension returns the file extension of the program that is built for the given z88dk target
func z88dkExtension(string) string {
	return ".tap"
}

// buildZ88DK builds the given C sources for the given target with zcc from z88dk
func buildZ88DK(target string, srcs []string, out string, opts BuildOptions) ([]byte, []string, error) {
	zcc, err := exec.LookPath("zcc")
	if err != nil {
		return nil, nil, fmt.Errorf("zcc was not found, please install z88dk")
	}
	base := strings.TrimSuffix(out, z88dkExtension(target))
	args := []string{"+" + target, "-create-app"}
	if opts.Debug {
		args = append(args, "-debug")
	}
	args = append(args, "-o", base)
	args = append(args, srcs...)
	cmd := exec.Command(zcc, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, []string{cmdToString(cmd)}, fmt.Errorf("compilation failed: %w", err)
	}
	if base != out && fileExists(out) {
		os.Remove(base)
	}
	return output, []string{cmdToString(cmd)}, nil
}

// isC64Tape returns true if the given .tap file is a Commodore 64 tape image instead of a ZX Spectrum tape
func isC64Tape(tap string) bool {
	f, err := os.Open(tap)
	if err != nil {
		return false
	}
	defer f.Close()
	magic := make([]byte, 12)
	n, _ := f.Read(magic)
	return n == len(magic) && string(magic) == "C64-TAPE-RAW"
}

// ZXEmulatorCommand returns a command that runs the given ZX Spectrum tape, snapshot or disk image in an
// installed emulator, or nil if none is found
func ZXEmulatorCommand(program string, args ...string) *exec.Cmd {
	emulators := [][]string{
		{"fuse"},
		{"fuse-sdl"},
		{"fuse-gtk"},
		{"tinyhat-emulate", "zx"},
	}
	for _, emulator := range emulators {
		if p := files.WhichCached(emulator[0]); p != "" {
			cmdArgs := append(append(emulator[1:], args...), program)
			return exec.Command(p, cmdArgs...)
		}
	}
	return nil
}
