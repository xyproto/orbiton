package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/xyproto/vt100"
)

// SetUpTerminateHandler sets up a signal handler for when ctrl-c is pressed
func (e *Editor) SetUpTerminateHandler(c *vt100.Canvas, status *StatusBar, tty *vt100.TTY) {
	sigChan := make(chan os.Signal, 1)

	// Clear any previous terminate handlers
	signal.Reset(syscall.SIGTERM)

	signal.Notify(sigChan, syscall.SIGTERM)
	go func() {
		for range sigChan {
			status.SetMessage("ctrl-c")
			status.Show(c, e)
		}
	}()
}
