//go:build !windows && !plan9

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

// terminalCellPixels returns the terminal's cell dimensions in pixels by querying
// the kernel via TIOCGWINSZ. Falls back to (8, 16) when pixel info is unavailable.
func terminalCellPixels() (cellW, cellH uint) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 || ws.Xpixel == 0 || ws.Ypixel == 0 {
		return 8, 16
	}
	return uint(ws.Xpixel) / uint(ws.Col), uint(ws.Ypixel) / uint(ws.Row)
}
