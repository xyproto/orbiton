//go:build windows || plan9

package megafile

// terminalCellPixels returns a safe default on platforms without TIOCGWINSZ support.
func terminalCellPixels() (cellW, cellH uint) {
	return 8, 16
}
