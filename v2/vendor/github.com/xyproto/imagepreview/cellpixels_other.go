//go:build windows || plan9

package imagepreview

// TerminalCellPixels returns a safe default on platforms without TIOCGWINSZ support.
func TerminalCellPixels() (cellW, cellH uint) {
	return 8, 16
}
