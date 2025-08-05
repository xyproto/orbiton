//go:build !windows

package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xyproto/vt"
)

var cancelPreviousSignalHandler context.CancelFunc

// SetUpSignalHandlers sets up signal handlers for SIGTERM, SIGUSR1, and SIGWINCH.
func (e *Editor) SetUpSignalHandlers(c *vt.Canvas, tty *vt.TTY, status *StatusBar, justClear bool) {

	// Cancel the previous signal handler if it exists
	if cancelPreviousSignalHandler != nil {
		cancelPreviousSignalHandler()
	}

	resizeMut.Lock()
	defer resizeMut.Unlock()

	sigChan := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancelPreviousSignalHandler = cancel

	// Handle signals differently for VTEGUI
	if inVTEGUI {
		signal.Reset(syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGWINCH)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGWINCH)
		syscall.Kill(os.Getppid(), syscall.SIGWINCH)
	} else {
		defer func() {
			signal.Reset(syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGWINCH)
			signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGWINCH)
		}()
	}

	if justClear {
		return
	}

	go func() {
		defer cancel()
		for {
			select {
			case sig := <-sigChan:
				switch sig {
				case syscall.SIGTERM:
					e.UserSave(c, tty, status)
				case syscall.SIGUSR1:
					if absFilename, err := filepath.Abs(e.filename); err != nil {
						fileLock.Unlock(e.filename)
					} else {
						fileLock.Unlock(absFilename)
					}
					fileLock.Save()
				case syscall.SIGWINCH:
					noDrawUntilResize.Store(false)
					e.FullResetRedraw(c, status, true, false)
					time.Sleep(300 * time.Millisecond)
					e.FullResetRedraw(c, status, true, false)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
