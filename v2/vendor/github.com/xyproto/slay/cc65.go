package slay

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/xyproto/files"
)

// cc65Headers maps the C headers that select a cc65 target to that target.
// The more specific headers come first, since cbm.h is included by all of the Commodore headers.
var cc65Headers = []struct{ header, target string }{
	{"c128.h", "c128"},
	{"vic20.h", "vic20"},
	{"nes.h", "nes"},
	{"c64.h", "c64"},
	{"cbm.h", "c64"},
}

// cc65IncludeTarget returns the cc65 target that the given #include line selects, or an empty string
func cc65IncludeTarget(line string) string {
	for _, h := range cc65Headers {
		if strings.Contains(line, "#include <"+h.header+">") || strings.Contains(line, `#include "`+h.header+`"`) {
			return h.target
		}
	}
	return ""
}

// makefileCC65Target returns the cc65 target that the given Makefile contents build for, or an empty string
func makefileCC65Target(s string) string {
	for _, target := range []string{"c128", "vic20", "nes", "c64"} {
		if strings.Contains(s, "-t "+target) || strings.Contains(s, "-t"+target) {
			return target
		}
	}
	return ""
}

var (
	asm6502Instruction  = regexp.MustCompile(`(?im)^\s*(?:[a-z_@][a-z0-9_@]*:)?\s*(lda|ldx|ldy|sta|stx|sty|jsr|jmp|rts|rti|inx|iny|dex|dey|bne|beq|bpl|bmi|bcc|bcs|cmp|cpx|cpy|sei|cli|sec|clc)\b`)
	snesDirective       = regexp.MustCompile(`(?im)^\s*\.p816\b`)
	snesModeInstruction = regexp.MustCompile(`(?im)^\s*(?:[a-z_@][a-z0-9_@]*:)?\s*(rep|sep)\s+#`)
	snesRegister        = regexp.MustCompile(`(?i)\$21[0-3][0-9a-f]\b`)
	c128Address         = regexp.MustCompile(`(?i)\$(d50[0-9ab]|d60[01]|ff00)\b`)
	vic20Address        = regexp.MustCompile(`(?i)\$(900[0-9a-f]|9400)\b`)
	c64Address          = regexp.MustCompile(`(?i)\$(ffd2|ffe4|ffcf|ffe1|d0[12][0-9a-f]|d4[01][0-9a-f])\b`)
	nesPPURegister      = regexp.MustCompile(`(?i)\$200[0-7]\b`)
	nesAPURegister      = regexp.MustCompile(`(?i)\$401[4-7]\b`)
)

// AssemblyTarget returns the system that the given assembly source code looks like it is written for,
// with the ca65 assembler from cc65: "c64", "c128", "vic20", "nes" or "snes" (65816 assembly for a LoROM cartridge).
// Returns an empty string if it does not look like 6502 or 65816 assembly for any of these.
func AssemblyTarget(source string) string {
	if len(asm6502Instruction.FindAllStringIndex(source, 3)) < 3 {
		return ""
	}
	switch {
	case snesDirective.MatchString(source) || (snesModeInstruction.MatchString(source) && snesRegister.MatchString(source)):
		return "snes"
	case c128Address.MatchString(source):
		return "c128"
	case vic20Address.MatchString(source):
		return "vic20"
	case c64Address.MatchString(source):
		return "c64"
	case nesPPURegister.MatchString(source) && (nesAPURegister.MatchString(source) || strings.Contains(source, `"NES"`)):
		return "nes"
	}
	return ""
}

// isAssemblySource returns true if the filename has an assembly file extension
func isAssemblySource(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".s", ".asm", ".a65":
		return true
	}
	return false
}

// assemblyProject returns the assembly files in the current directory and the system they are written for,
// if one of them looks like assembly for a system that AssemblyTarget recognizes. That file comes first.
func assemblyProject() ([]string, string) {
	var srcs []string
	for _, pattern := range []string{"*.s", "*.asm", "*.a65"} {
		matches, _ := filepath.Glob(pattern)
		srcs = append(srcs, matches...)
	}
	for i, src := range srcs {
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if target := AssemblyTarget(string(data)); target != "" {
			return append([]string{src}, slices.Delete(slices.Clone(srcs), i, i+1)...), target
		}
	}
	return nil, ""
}

// cc65Extension returns the file extension of the program or ROM that is built for the given target
func cc65Extension(target string) string {
	switch target {
	case "nes":
		return ".nes"
	case "snes":
		return ".sfc"
	}
	return ".prg"
}

// projectLinkerConfig returns the ld65 configuration file in the current directory, if there is exactly one
func projectLinkerConfig() string {
	matches, _ := filepath.Glob("*.cfg")
	if len(matches) == 1 {
		return matches[0]
	}
	return ""
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

// buildCC65 builds the given C or assembly sources for the given target with cc65
func buildCC65(target string, srcs []string, out string, opts BuildOptions) ([]byte, []string, error) {
	if target == "snes" {
		return buildSNESAssembly(srcs, out, opts)
	}
	cl65 := cc65ToolPath("cl65")
	if cl65 == "" {
		return nil, nil, fmt.Errorf("cl65 was not found, please install cc65")
	}
	args := []string{"-t", target}
	if !slices.ContainsFunc(srcs, func(src string) bool { return !isAssemblySource(src) }) {
		if cfg := projectLinkerConfig(); cfg != "" {
			args = append(args, "-C", cfg)
		} else if target != "nes" {
			// Pure assembly has no main function for the C startup code, so link it as a plain program with a BASIC header
			args = append(args, "-C", target+"-asm.cfg", "-u", "__EXEHDR__")
		}
		if opts.Debug {
			args = append(args, "-g")
		}
	} else if opts.Debug {
		args = append(args, "-g")
	} else {
		args = append(args, "-Oirs")
	}
	args = append(args, "-o", out)
	args = append(args, srcs...)
	cmd := exec.Command(cl65, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, []string{cmdToString(cmd)}, fmt.Errorf("compilation failed: %w", err)
	}
	return output, []string{cmdToString(cmd)}, nil
}

// C64EmulatorCommand returns a command that runs the given C64 program in VICE,
// or nil if VICE is not installed
func C64EmulatorCommand(prg string, args ...string) *exec.Cmd {
	return viceCommand([]string{"x64sc", "x64"}, prg, args...)
}

// CBMEmulatorCommand returns a command that runs the given Commodore program in the VICE emulator
// for the machine that the load address of the program is for (C64, C128 or VIC-20),
// or nil if that emulator is not installed
func CBMEmulatorCommand(prg string, args ...string) *exec.Cmd {
	emulators := []string{"x64sc", "x64"}
	if data, err := os.ReadFile(prg); err == nil && len(data) >= 2 {
		switch binary.LittleEndian.Uint16(data) {
		case 0x1C01:
			emulators = []string{"x128"}
		case 0x1001, 0x1201, 0x0401:
			emulators = []string{"xvic"}
		}
	}
	return viceCommand(emulators, prg, args...)
}

func viceCommand(emulators []string, prg string, args ...string) *exec.Cmd {
	for _, emulator := range emulators {
		if p := files.WhichCached(emulator); p != "" {
			cmdArgs := append(append([]string{}, args...), "-autostart", prg)
			return exec.Command(p, cmdArgs...)
		}
	}
	return nil
}

// NESEmulatorCommand returns a command that runs the given NES ROM in an installed emulator,
// or nil if none is found
func NESEmulatorCommand(rom string, args ...string) *exec.Cmd {
	emulators := [][]string{
		{"fceux"},
		{"nestopia"},
		{"mesen"},
		{"mednafen"},
		{"tinyhat-emulate", "nes"},
	}
	for _, emulator := range emulators {
		if p := files.WhichCached(emulator[0]); p != "" {
			cmdArgs := append(append(emulator[1:], args...), rom)
			return exec.Command(p, cmdArgs...)
		}
	}
	return nil
}

// EmulatorCommand returns a command that runs the given ROM or program in an installed emulator,
// chosen by the file extension (and the load address, for Commodore programs).
// Returns nil if the file type is unknown or if no suitable emulator is installed.
func EmulatorCommand(rom string, args ...string) *exec.Cmd {
	switch strings.ToLower(filepath.Ext(rom)) {
	case ".prg":
		return CBMEmulatorCommand(rom, args...)
	case ".nes":
		return NESEmulatorCommand(rom, args...)
	case ".sfc", ".smc":
		return SNESEmulatorCommand(rom, args...)
	case ".gba":
		return GBAEmulatorCommand(rom, args...)
	}
	return nil
}
