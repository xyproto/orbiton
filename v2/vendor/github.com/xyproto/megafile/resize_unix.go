//go:build !windows && !plan9

package megafile

import (
	"os"
	"os/signal"
	"syscall"
)

// SetupResizeSignal sets up window resize signal handling (Unix implementation)
func SetupResizeSignal(sigChan chan os.Signal) {
	signal.Notify(sigChan, syscall.SIGWINCH)
}

// ResetResizeSignal resets window resize signals (Unix implementation)
func ResetResizeSignal() {
	signal.Reset(syscall.SIGWINCH)
}
