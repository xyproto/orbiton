package main

import (
	"fmt"
	"os"

	"github.com/xyproto/clip"
)

// SetClipboardFromFile can copy the given file to the clipboard.
// The returned int is the number of bytes written.
// The returned string is the last 7 characters written to the file.
func SetClipboardFromFile(filename string, primaryClipboard bool) (int, string, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return 0, "", err
	}

	// Write to the clipboard
	if err := clip.WriteAllBytes(data, primaryClipboard); err != nil {
		return 0, "", err
	}

	contents := string(data)
	tailString := ""
	if l := len(contents); l > 7 {
		tailString = string(contents[l-8:])
	}

	return len(data), tailString, nil
}

// WriteClipboardToFile can write the contents of the clipboard to a file.
// If overwrite is true, the original file will be removed first, if it exists.
// The returned int is the number of bytes written.
// The fist returned string is the first 7 characters written to the file.
// The second returned string is the last 7 characters written to the file.
func WriteClipboardToFile(filename string, overwrite, primaryClipboard bool) (int, string, string, error) {
	// Check if the file exists first
	if exists(filename) {
		if overwrite {
			if err := os.Remove(filename); err != nil {
				return 0, "", "", err
			}
		} else {
			return 0, "", "", fmt.Errorf("%s already exists", filename)
		}
	}

	// Read the clipboard
	contents, err := clip.ReadAllBytes(primaryClipboard)
	if err != nil {
		return 0, "", "", err
	}

	// Write to file
	f, err := os.Create(filename)
	if err != nil {
		return 0, "", "", err
	}
	defer f.Close()

	lenContents := len(contents)

	headString := ""
	if lenContents > 7 {
		headString = string(contents[:8])
	}

	tailString := ""
	if lenContents > 7 {
		tailString = string(contents[lenContents-8:])
	}

	n, err := f.Write(contents)
	if err != nil {
		return 0, "", "", err
	}
	return n, headString, tailString, nil
}
