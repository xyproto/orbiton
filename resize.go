package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/xyproto/vt100"
)

// SetUpResizeHandler sets up a signal handler for when the terminal is resized
func SetUpResizeHandler(c *vt100.Canvas, e *Editor, tty *vt100.TTY) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	go func() {
		for range sigChan {
			*c = *(vt100.NewCanvas())
			e.redraw = true
			e.redrawCursor = true
		}
	}()
}
