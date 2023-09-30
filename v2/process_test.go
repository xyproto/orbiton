package main

import (
	"testing"
)

func TestParentIsMan(t *testing.T) {
	// This test will only pass if run from the 'man' command, which is not typical.
	isMan := parentIsMan()
	if isMan {
		t.Logf("Parent process is 'man'.")
	} else {
		t.Logf("Parent process is not 'man'.")
	}
}

func TestParentCommand(t *testing.T) {
	cmd := parentCommand()
	if cmd == "" {
		t.Errorf("Failed to get parent command")
	} else {
		t.Logf("Parent command: %s", cmd)
	}
}

func TestTerminalEmulator(t *testing.T) {
	term, err := terminalEmulator()
	if err != nil {
		t.Errorf("Error: %v", err)
	} else if term == "" {
		t.Logf("Terminal emulator not found")
	} else {
		t.Logf("Terminal emulator: %s", term)
	}
}
