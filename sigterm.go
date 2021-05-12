package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/xyproto/vt100"
)

// SetUpTerminateHandler sets up a signal handler for when ctrl-c is pressed
func (e *Editor) SetUpTerminateHandler(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar) {
	sigChan := make(chan os.Signal, 1)

	// Clear any previous terminate handlers
	signal.Reset(syscall.SIGTERM)

	signal.Notify(sigChan, syscall.SIGTERM)
	go func() {
		for {
			// Block until the signal is received
			<-sigChan

			// Quickly save the file
			e.UserSave(c, tty, status)

			status.SetMessage("ctrl-c")
			status.Show(c, e)
		}
	}()
}
