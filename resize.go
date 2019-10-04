package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/xyproto/vt100"
)

func SetUpResizeHandler(c *vt100.Canvas, e *Editor, p *Position) {
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
			e.WriteLines(c, 0+p.Offset(), h+p.Offset(), 0, 0)
			c.Draw()
		}
	}()
}
