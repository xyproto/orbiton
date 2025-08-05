//go:build windows

package main

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"

	"github.com/xyproto/vt"
)

var (
	runPID                      atomic.Int64
	cancelPreviousSignalHandler context.CancelFunc
)

// Windows stub for setting up window resize signal handling
func setupResizeSignal(sigChan chan os.Signal) {
	// On Windows, terminal resize events work differently
	// We use os.Interrupt as a basic signal handler
	signal.Notify(sigChan, os.Interrupt)
}

// Windows stub for resetting window resize signals
func resetResizeSignal() {
	// On Windows, reset interrupt signal
	signal.Reset(os.Interrupt)
}

// Windows stub for sending parent process quit signal
func sendParentQuitSignal() {
	// On Windows, we can't send SIGQUIT to parent process
	// This is a no-op
}

// Windows implementation for SetUpSignalHandlers
func (e *Editor) SetUpSignalHandlers(c *vt.Canvas, tty *vt.TTY, status *StatusBar, justClear bool) {
	// Cancel the previous signal handler if it exists
	if cancelPreviousSignalHandler != nil {
		cancelPreviousSignalHandler()
	}

	resizeMut.Lock()
	mut.Lock()

	sigChan := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancelPreviousSignalHandler = cancel

	// Windows signal handling - only handle interrupt signals
	signal.Reset(os.Interrupt)
	signal.Notify(sigChan, os.Interrupt)

	resizeMut.Unlock()
	mut.Unlock()

	if justClear {
		return
	}

	go func() {
		defer cancel()
		for {
			select {
			case sig := <-sigChan:
				switch sig {
				case os.Interrupt:
					mut.Lock()
					e.UserSave(c, tty, status)
					mut.Unlock()
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Windows stub for stopBackgroundProcesses
func stopBackgroundProcesses() bool {
	if runPID.Load() <= 0 {
		return false
	}
	runPID.Store(-1)
	return true
}

// Windows stub for getProcPath
func getProcPath(pid int, suffix string) (string, error) {
	return "", os.ErrNotExist
}

// Windows stub for parentProcessIs
func parentProcessIs(name string) bool {
	return false
}

// Windows stub for parentCommand
func parentCommand() string {
	return ""
}

// Windows stub for getPID
func getPID(name string) (int64, error) {
	return 0, os.ErrNotExist
}

// Windows stub for foundProcess
func foundProcess(name string) bool {
	return false
}

// Windows stub for pkill
func pkill(name string) (int, error) {
	return 0, os.ErrNotExist
}
