// go:build unix

package main

import (
	"github.com/atotto/clipboard"
)

func getOtherClipboardContents() (string, error) {
	// Try using the other clipboard
	clipboard.Primary = !clipboard.Primary
	// Fetch the contents
	s, err := clipboard.ReadAll()
	// Switch back
	clipboard.Primary = !clipboard.Primary
	return s, err
}
