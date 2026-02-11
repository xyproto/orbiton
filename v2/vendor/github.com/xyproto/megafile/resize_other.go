//go:build windows || plan9

package megafile

import "os"

// SetupResizeSignal is a no-op on Windows and Plan 9 (no SIGWINCH signal)
func SetupResizeSignal(sigChan chan os.Signal) {}

// ResetResizeSignal is a no-op on Windows and Plan 9
func ResetResizeSignal() {}
