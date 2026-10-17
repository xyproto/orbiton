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

func isSNESInclude(line string) bool {
	return strings.Contains(line, "#include <snes.h>") || strings.Contains(line, `#include "snes.h"`)
}

func makefileUsesSNES(s string) bool {
	return strings.Contains(s, "snes_rules")
}

func pvsneslibHome() string {
	candidates := []string{os.Getenv("PVSNESLIB_HOME"), "/usr/share/pvsneslib", "/opt/pvsneslib"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, "pvsneslib"))
	}
	for _, dir := range candidates {
		if dir != "" && fileExists(filepath.Join(dir, "devkitsnes", "bin", "816-tcc")) {
			return dir
		}
	}
	return ""
}

func snesTitle(rom string) string {
	var sb strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSuffix(filepath.Base(rom), ".sfc")) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == ' ':
			sb.WriteRune(r)
		case r == '-' || r == '_':
			sb.WriteRune(' ')
		}
	}
	title := sb.String()
	if len(title) > 21 {
		title = title[:21]
	}
	return title
}

func writeSNESHeader(home, rom string) error {
	template, err := os.ReadFile(filepath.Join(home, "devkitsnes", "include", "hdr.asm.in"))
	if err != nil {
		return err
	}
	header := strings.NewReplacer(
		"@HIROMDEF@", "",
		"@FASTROMDEF@", "",
		"@ROMTITLE@", snesTitle(rom),
		"@CARTRIDGETYPE@", "00",
		"@ROMSIZE@", "08",
		"@SRAMSIZE@", "00",
		"@COUNTRY@", "01",
		"@LICENSEECODE@", "00",
		"@VERSION@", "00",
		"@ROMBANKS@", "8",
		"@ROMBANKSIZE@", "8000",
		"@ROMMODE@", "LOROM",
		"@ROMSPEED@", "SLOWROM",
	).Replace(string(template))
	return os.WriteFile("hdr.asm", []byte(header), 0o644)
}

func buildSNES(srcs []string, rom string) ([]byte, []string, error) {
	var output bytes.Buffer
	var commands []string
	home := pvsneslibHome()
	if home == "" {
		return nil, nil, fmt.Errorf("PVSnesLib was not found, please set PVSNESLIB_HOME")
	}
	for _, src := range srcs {
		if filepath.Ext(src) != ".c" {
			return nil, nil, fmt.Errorf("%s: SNES programs are built with 816-tcc, which only compiles C", src)
		}
	}
	if err := writeSNESHeader(home, rom); err != nil {
		return nil, nil, err
	}
	bin := filepath.Join(home, "devkitsnes", "bin")
	tools := filepath.Join(home, "devkitsnes", "tools")
	libDir := filepath.Join(home, "pvsneslib", "lib", "LoROM_SlowROM")

	run := func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		commands = append(commands, cmdToString(cmd))
		out, err := cmd.CombinedOutput()
		output.Write(out)
		if err != nil {
			return fmt.Errorf("%s failed: %w", filepath.Base(name), err)
		}
		return nil
	}

	generated := map[string]bool{}
	asmFiles := []string{}
	for _, src := range srcs {
		base := strings.TrimSuffix(src, ".c")
		if err := run(filepath.Join(bin, "816-tcc"),
			"-I"+filepath.Join(home, "pvsneslib", "include"), "-I"+filepath.Join(home, "devkitsnes", "include"), "-I.",
			"-Wall", "-c", src, "-o", base+".ps"); err != nil {
			return output.Bytes(), commands, err
		}
		if err := run(filepath.Join(tools, "816-opt"), "-i", base+".ps", "-o", base+".asm"); err != nil {
			return output.Bytes(), commands, err
		}
		generated[filepath.Clean(base+".asm")] = true
		asmFiles = append(asmFiles, base+".asm")
	}
	asmFiles = append(asmFiles, "hdr.asm")
	if matches, err := filepath.Glob("*.asm"); err == nil {
		for _, m := range matches {
			if m != "hdr.asm" && !generated[filepath.Clean(m)] {
				asmFiles = append(asmFiles, m)
			}
		}
	}

	var link strings.Builder
	link.WriteString("[objects]\n")
	for _, asm := range asmFiles {
		obj := strings.TrimSuffix(asm, ".asm") + ".obj"
		if err := run(filepath.Join(bin, "wla-65816"), "-d", "-s", "-x", "-o", obj, asm); err != nil {
			return output.Bytes(), commands, err
		}
		link.WriteString(obj + "\n")
	}
	libObjs, err := filepath.Glob(filepath.Join(libDir, "*.obj"))
	if err != nil || len(libObjs) == 0 {
		return output.Bytes(), commands, fmt.Errorf("no PVSnesLib objects were found in %s", libDir)
	}
	for _, obj := range libObjs {
		link.WriteString(obj + "\n")
	}
	if err := os.WriteFile("linkfile", []byte(link.String()), 0o644); err != nil {
		return output.Bytes(), commands, err
	}
	if err := run(filepath.Join(bin, "wlalink"), "-d", "-s", "-A", "-c", "-L", libDir, "linkfile", rom); err != nil {
		return output.Bytes(), commands, err
	}
	return output.Bytes(), commands, nil
}

func SNESEmulatorCommand(rom string, args ...string) *exec.Cmd {
	emulators := [][]string{
		{"zsnes"},
		{"snes9x-gtk"},
		{"snes9x"},
		{"bsnes"},
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
