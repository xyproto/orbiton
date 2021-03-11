package main

import (
	"os"
	"path/filepath"
)

func generateTitle(filename string) string {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return filepath.Base(filename)
	}
	// First try to find the relative path to the home directory
	if relPath, err := filepath.Rel(os.Getenv("HOME"), absPath); err != nil {
		// If the relative directory to $HOME could not be found, then just use the base filename
		return filepath.Base(filename)
	} else {
		title := filepath.Join("~", relPath)
		// If the relative directory path is too long, just use the base filename
		if len(title) >= 30 {
			title = filepath.Base(filename)
		}
		return title
	}
	return "o"
}
