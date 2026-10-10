package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsDOSExecutable(t *testing.T) {
	dir := t.TempDir()
	dos := make([]byte, 128)
	copy(dos, "MZ")
	dos[0x3c] = 0x40
	pe := make([]byte, 128)
	copy(pe, "MZ")
	pe[0x3c] = 0x40
	copy(pe[0x40:], "PE\x00\x00")
	for name, data := range map[string][]byte{"hello.exe": dos, "win.exe": pe, "text.exe": []byte("hello")} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if !isDOSExecutable(filepath.Join(dir, "hello.exe")) {
		t.Error("hello.exe should be a DOS executable")
	}
	if isDOSExecutable(filepath.Join(dir, "win.exe")) || isDOSExecutable(filepath.Join(dir, "text.exe")) {
		t.Error("win.exe and text.exe should not be DOS executables")
	}
	if got := dosExecutable(dir, filepath.Join(dir, "main.c")); got != filepath.Join(dir, "hello.exe") {
		t.Errorf("dosExecutable returned %q", got)
	}
}

func TestIsDOSAssembly(t *testing.T) {
	if !isDOSAssembly("; intro\norg 100h ; COM file\nmov al, 13h\n") || !isDOSAssembly("[ORG 0x100]\n") {
		t.Error("org 100h should be detected as DOS assembly")
	}
	if isDOSAssembly("section .text\nglobal _start\n; org 100h\n") {
		t.Error("Linux assembly should not be detected as DOS assembly")
	}
}
