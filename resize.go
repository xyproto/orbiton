package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/xyproto/vt100"
)

// SetUpResizeHandler sets up a signal handler for when the terminal is resized
func (e *Editor) SetUpResizeHandler(c *vt100.Canvas, status *StatusBar, tty *vt100.TTY) {
	sigChan := make(chan os.Signal, 1)

	// Clear any previous resize handlers
	signal.Reset(syscall.SIGWINCH)
	signal.Notify(sigChan, syscall.SIGWINCH)

	go func() {
		for {
			// Block until SIGWINCH signal is received
			<-sigChan

			e.FullResetRedraw(c, status, true)
		}
	}()
}
