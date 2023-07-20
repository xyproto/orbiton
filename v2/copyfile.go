package main

import (
	"github.com/xyproto/clip"
)

// SetClipboardFromFile can copy the given file to the clipboard.
// The returned int is the number of bytes written.
// The returned string is the last 7 characters written to the file.
func SetClipboardFromFile(filename string) (int, string, error) {
	// Read the file
	data, err := ReadFileNoStat(filename)
	if err != nil {
		return 0, "", err
	}

	// Write to the clipboard
	if err := clip.WriteAllBytes(data); err != nil {
		return 0, "", err
	}

	contents := string(data)
	tailString := ""
	if l := len(contents); l > 7 {
		tailString = string(contents[l-8:])
	}

	return len(data), tailString, nil
}
