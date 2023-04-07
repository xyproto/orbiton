package main

import (
	"fmt"
	"os"

	"github.com/xyproto/clip"
)

// WriteClipboardToFile can write the contents of the clipboard to a file
func WriteClipboardToFile(filename string) (int, error) {
	// Check if the file exists first
	if exists(filename) {
		return 0, fmt.Errorf("%s already exists", filename)
	}
	// Read the clipboard
	contents, err := clip.ReadAllBytes()
	if err != nil {
		return 0, err
	}
	// Write to file
	f, err := os.Create(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.Write(contents)
}
