package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/xyproto/env"
	"github.com/xyproto/vt100"
)

// SetUpSignalHandlers sets up a signal handler for when ctrl-c is pressed (SIGTERM),
// and also for when SIGUSR1 or SIGWINCH is received.
func (e *Editor) SetUpSignalHandlers(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar) {

	sigChan := make(chan os.Signal, 1)

	// Clear any previous terminate or USR handlers
	signal.Reset(syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGWINCH)

	// Set up notifications
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGWINCH)

	// Send a SIGWINCH signal to the parent process if "KO" is set,
	// to signal that "o is ready" to resize. The "ko" GUI will then
	// send SIGWINCH back, which will trigger FullResetRedraw in the case below.
	if env.Bool("KO") {
		syscall.Kill(os.Getppid(), syscall.SIGWINCH)
	}

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
				if absFilename, err := filepath.Abs(e.filename); err != nil {
					// Just unlock the non-absolute filename
					fileLock.Unlock(e.filename)
				} else {
					fileLock.Unlock(absFilename)
				}
				fileLock.Save()
			case syscall.SIGWINCH:
				// Full redraw, like if Esc was pressed
				drawLines := true
				resized := true
				e.FullResetRedraw(c, status, drawLines, resized)
			}
		}
	}()
}
