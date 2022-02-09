package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/xyproto/vt100"
)

// SetUpSignalHandlers sets up a signal handler for when ctrl-c is pressed (SIGTERM),
// and also for when SIGUSR1 is received.
func (e *Editor) SetUpSignalHandlers(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, absFilename string) {

	sigChan := make(chan os.Signal, 1)

	// Clear any previous terminate or USR1 handlers
	signal.Reset(syscall.SIGTERM, syscall.SIGUSR1)

	// Set up notifications
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGUSR1)

	go func() {
		for {
			// Block until the signal is received
			sig := <-sigChan

			switch sig {
			case syscall.SIGTERM:
				// Save the file
				e.UserSave(c, tty, status)
				status.SetMessage("ctrl-c")
				status.Show(c, e)
			case syscall.SIGUSR1:
				// Unlock the file
				fileLock.Unlock(absFilename)
				fileLock.Save()
			}
		}
	}()
}
