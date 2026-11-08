package slay

import (
	"errors"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/xyproto/files"
)

// chip8Source returns the Octo (CHIP-8 assembly) source file in the current directory,
// preferring main.8o, or an empty string if there is none.
func chip8Source() string {
	matches, _ := filepath.Glob("*.8o")
	if len(matches) == 0 {
		return ""
	}
	if slices.Contains(matches, "main.8o") {
		return "main.8o"
	}
	slices.Sort(matches)
	return matches[0]
}

// buildChip8 assembles an Octo source file into a CHIP-8 ROM with octo-cli.
func buildChip8(src, rom string) ([]byte, []string, error) {
	octo := files.WhichCached("octo-cli")
	if octo == "" {
		return nil, nil, errors.New("octo-cli was not found, it is needed for assembling Octo (CHIP-8) programs")
	}
	out, err := exec.Command(octo, src, rom).CombinedOutput()
	return out, []string{"octo-cli " + src + " " + rom}, err
}

// Chip8EmulatorCommand returns a command that runs the given CHIP-8 ROM in an installed emulator,
// or nil if no emulator was found.
func Chip8EmulatorCommand(rom string, args ...string) *exec.Cmd {
	for _, emulator := range []string{"octo-run", "chip8"} {
		if p := files.WhichCached(emulator); p != "" {
			return exec.Command(p, append(args, rom)...)
		}
	}
	return nil
}
