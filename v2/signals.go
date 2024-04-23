package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xyproto/vt100"
)

// SetUpSignalHandlers sets up a signal handler for when ctrl-c is pressed (SIGTERM),
// and also for when SIGUSR1 or SIGWINCH is received.
func (e *Editor) SetUpSignalHandlers(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar) {

	// For the drawing and the statusbar
	resizeMut.Lock()
	mut.Lock()

	sigChan := make(chan os.Signal, 1)

	// Send a SIGWINCH signal to the parent process if "OG" is set,
	// to signal that "o is ready" to resize. The "og" GUI will then
	// send SIGWINCH back, which will trigger FullResetRedraw in the case below.
	if inVTEGUI {
		// Clear any previous terminate or USR handlers
		signal.Reset(syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGWINCH)
		// Set up notifications
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGWINCH)
		// Send a SIGWINCH signal to the "og" GUI, which is catched there
		syscall.Kill(os.Getppid(), syscall.SIGWINCH)
	} else {
		// Start these in the background, since the "og" GUI isn't waiting
		defer func() {
			// Clear any previous terminate or USR handlers
			signal.Reset(syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGWINCH)
			// Set up notifications
			signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGWINCH)
		}()
	}

	// For the drawing and the statusbar
	resizeMut.Unlock()
	mut.Unlock()

	go func() {
		for {
			// Block until the signal is received
			sig := <-sigChan

			switch sig {
			case syscall.SIGTERM:
				// Save the file
				mut.Lock()
				e.UserSave(c, tty, status)
				mut.Unlock()
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
				e.FullResetRedraw(c, status, drawLines)
				// Try twice
				time.Sleep(300 * time.Millisecond)
				e.FullResetRedraw(c, status, drawLines)
			}
		}
	}()
}
