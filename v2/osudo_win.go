//go:build windows

package main

import (
	"fmt"
	"os"
)

// Sudoers represents a stub for Windows (sudo not applicable)
type Sudoers struct{}

// NewSudoers creates a stub for Windows - sudo functionality not available
func NewSudoers(sudoersPath string) (*Sudoers, error) {
	return nil, fmt.Errorf("sudo functionality not available on Windows")
}

// TempPath returns empty string on Windows
func (s *Sudoers) TempPath() string {
	return ""
}

// Finalize does nothing on Windows
func (s *Sudoers) Finalize() error {
	return nil
}

func validateSudoersSyntax(filepath string) bool {
	return false
}

func askWhatNow() rune {
	return 'x'
}

func (s *Sudoers) commitChanges() error {
	return nil
}

func (s *Sudoers) setupSignalHandlers() {
	// No-op on Windows
}

func (s *Sudoers) cleanup() {
	// No-op on Windows
}

func isQuietMode() bool {
	stat, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}
