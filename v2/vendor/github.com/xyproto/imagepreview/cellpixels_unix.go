//go:build !windows && !plan9

package imagepreview

import (
	"os"

	"golang.org/x/sys/unix"
)

// TerminalCellPixels returns the terminal's cell dimensions in pixels by
// querying the kernel via TIOCGWINSZ. Falls back to (8, 16) when pixel info
// is unavailable or when the resulting aspect ratio is unrealistic.
func TerminalCellPixels() (cellW, cellH uint) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 || ws.Xpixel == 0 || ws.Ypixel == 0 {
		return 8, 16
	}
	cellW = uint(ws.Xpixel) / uint(ws.Col)
	cellH = uint(ws.Ypixel) / uint(ws.Row)

	// Sanity check: most terminal cells are between 1:1 and 1:3.
	// Fall back to a default ratio if we get something extreme (like 1:10 or 10:1).
	if cellW == 0 || cellH == 0 || cellH > cellW*5 || cellW > cellH*5 {
		return 8, 16
	}
	return cellW, cellH
}
