package slay

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xyproto/files"
)

// gbaLibraries maps headers that mark a project as a Game Boy Advance program
// to the library that provides them.
var gbaLibraries = map[string]string{
	"tonc.h": "tonc",
	"gba.h":  "gba",
}

// isGBAInclude returns true if the given #include line pulls in a GBA library header.
func isGBAInclude(line string) bool {
	for header := range gbaLibraries {
		if strings.Contains(line, "#include <"+header+">") || strings.Contains(line, `#include "`+header+`"`) {
			return true
		}
	}
	return strings.Contains(line, "#include <gba_")
}

// makefileUsesGBA returns true if the Makefile contents look like they build a GBA ROM.
func makefileUsesGBA(s string) bool {
	return strings.Contains(s, "gba.specs") || strings.Contains(s, "gbafix") || strings.Contains(s, "gba_rules")
}

// gbaToolPath finds a GBA cross tool, first in $DEVKITARM/bin and then in PATH.
func gbaToolPath(name string) string {
	if devkitARM := os.Getenv("DEVKITARM"); devkitARM != "" {
		if p := filepath.Join(devkitARM, "bin", name); fileExists(p) {
			return p
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

// assembleGBAFlags returns the flags for building a GBA ROM with arm-none-eabi-gcc and gba.specs.
func assembleGBAFlags(proj Project, opts BuildOptions) BuildFlags {
	bf := BuildFlags{GBA: true, Compiler: gbaToolPath("arm-none-eabi-gcc"), Std: "gnu17"}
	if !proj.IsC {
		bf.Compiler = gbaToolPath("arm-none-eabi-g++")
		bf.Std = "gnu++17"
	}
	arch := []string{"-mcpu=arm7tdmi", "-mthumb", "-mthumb-interwork"}
	bf.CFlags = append(bf.CFlags, arch...)
	switch {
	case opts.Debug:
		bf.CFlags = append(bf.CFlags, "-O0", "-g")
	case opts.Small || opts.Tiny:
		bf.CFlags = append(bf.CFlags, "-Os")
	default:
		bf.CFlags = append(bf.CFlags, "-O2")
	}
	switch {
	case opts.Sloppy:
		bf.CFlags = append(bf.CFlags, "-w")
	case opts.Strict:
		bf.CFlags = append(bf.CFlags, "-Wall", "-Wextra", "-Wpedantic")
	default:
		bf.CFlags = append(bf.CFlags, "-Wall")
	}
	bf.LDFlags = append(bf.LDFlags, arch...)
	bf.LDFlags = append(bf.LDFlags, "-specs=gba.specs")
	for _, inc := range proj.Includes {
		if lib, ok := gbaLibraries[filepath.Base(inc)]; ok {
			bf.LDFlags = appendUnique(bf.LDFlags, "-l"+lib)
		}
	}
	bf.LDFlags = append(bf.LDFlags, "-lm")
	return bf
}

// gbaELFName returns the name of the intermediate ELF file for the given ROM name.
func gbaELFName(rom string) string {
	return strings.TrimSuffix(rom, ".gba") + ".elf"
}

// gbaFinish turns the linked ELF file into a GBA ROM and fixes the ROM header,
// so that it runs both in emulators and on real hardware.
func gbaFinish(elf, rom string) ([]byte, []string, error) {
	var output bytes.Buffer
	var commands []string
	objcopy := gbaToolPath("arm-none-eabi-objcopy")
	if objcopy == "" {
		return nil, nil, fmt.Errorf("arm-none-eabi-objcopy was not found")
	}
	steps := []*exec.Cmd{exec.Command(objcopy, "-O", "binary", elf, rom)}
	if gbafix := gbaToolPath("gbafix"); gbafix != "" {
		steps = append(steps, exec.Command(gbafix, "-t"+gbaTitle(rom), rom))
	}
	for _, cmd := range steps {
		commands = append(commands, cmdToString(cmd))
		out, err := cmd.CombinedOutput()
		output.Write(out)
		if err != nil {
			return output.Bytes(), commands, fmt.Errorf("%s failed: %w", filepath.Base(cmd.Path), err)
		}
	}
	return output.Bytes(), commands, nil
}

// gbaTitle returns the ROM header title for the given ROM file name,
// which is at most 12 uppercase characters.
func gbaTitle(rom string) string {
	title := strings.ToUpper(strings.TrimSuffix(filepath.Base(rom), ".gba"))
	if len(title) > 12 {
		title = title[:12]
	}
	return title
}

// GBAEmulatorCommand returns a command that runs the given GBA ROM in an installed emulator,
// or nil if no emulator was found.
func GBAEmulatorCommand(rom string, args ...string) *exec.Cmd {
	emulators := [][]string{
		{"mgba", "-3"},
		{"mgba-qt"},
		{"visualboyadvance-m"},
		{"mednafen"},
	}
	for _, emulator := range emulators {
		if p := files.WhichCached(emulator[0]); p != "" {
			cmdArgs := append(append(emulator[1:], args...), rom)
			return exec.Command(p, cmdArgs...)
		}
	}
	return nil
}
