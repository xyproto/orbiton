package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/xyproto/vt100"
)

// SetUpResizeHandler sets up a signal handler for when the terminal is resized
func SetUpResizeHandler(c *vt100.Canvas, e *Editor) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	go func() {
		for range sigChan {
			// Create a new canvas, with the new size
			nc := c.Resized()
			if nc != nil {
				c.Clear()
				vt100.Clear()
				c.Draw()
				c = nc
			}
			h := int(c.Height())
			e.WriteLines(c, e.pos.Offset(), h+e.pos.Offset(), 0, 0)
			c.Draw()
		}
	}()
}
