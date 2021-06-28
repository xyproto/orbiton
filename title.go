package main

import (
	"path/filepath"
)

// generateTitle tries to find a suitable terminal emulator title text for a given filename,
// that is not too long (ideally <30 characters)
func generateTitle(filename string) string {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return filepath.Base(filename)
	}
	// First try to find the relative path to the home directory
	relPath, err := filepath.Rel(homeDir, absPath)
	if err != nil {
		// If the relative directory to $HOME could not be found, then just use the base filename
		return filepath.Base(filename)
	}
	title := filepath.Join("~", relPath)
	// If the relative directory path is short enough, use that
	if len(title) < 30 {
		return title
	}
	// Just use the base filename
	return filepath.Base(filename)
}
