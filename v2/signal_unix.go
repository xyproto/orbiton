//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// Unix implementation for setting up window resize signal handling
func setupResizeSignal(sigChan chan os.Signal) {
	signal.Notify(sigChan, syscall.SIGWINCH)
}

// Unix implementation for resetting window resize signals
func resetResizeSignal() {
	signal.Reset(syscall.SIGWINCH)
}

// Unix implementation for sending parent process quit signal
func sendParentQuitSignal() {
	syscall.Kill(os.Getppid(), syscall.SIGQUIT)
}
