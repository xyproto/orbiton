//go:build windows || plan9

package main

// terminalCellPixels returns a fallback cell size on platforms without TIOCGWINSZ.
func terminalCellPixels() (cellW, cellH uint) {
	return 8, 16
}
