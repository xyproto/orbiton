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
			// Create a new canvas
			c = vt100.NewCanvas()
			c.ShowCursor()
			// Then write to that
			e.DrawLines(c, true, true)
			//h := int(c.Height())
			//e.WriteLines(c, e.pos.Offset(), h+e.pos.Offset(), 0, 0)
			//c.Redraw()
			// Redraw cursor
			x := e.pos.ScreenX()
			y := e.pos.ScreenY()
			vt100.SetXY(uint(x), uint(y))
		}
	}()
}
