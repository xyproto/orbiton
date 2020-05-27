package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/xyproto/vt100"
)

var resizeMutex sync.RWMutex

// SetUpResizeHandler sets up a signal handler for when the terminal is resized
func (e *Editor) SetUpResizeHandler(c *vt100.Canvas, status *StatusBar, tty *vt100.TTY) {
	resizeMutex.Lock()
	sigChan := make(chan os.Signal, 1)

	// Clear any previous resize handlers
	signal.Reset(syscall.SIGWINCH)

	signal.Notify(sigChan, syscall.SIGWINCH)
	go func() {
		for range sigChan {
			newCanvas := e.FullResetRedraw(c, status)
			*c = *newCanvas
			e.DrawLines(c, true, false)
		}
	}()

	resizeMutex.Unlock()
}
