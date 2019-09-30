package main

import (
	"github.com/xyproto/vt100"
	"time"
)

type StatusBar struct {
	msg    string               // status message
	fg     vt100.AttributeColor // draw foreground color
	bg     vt100.AttributeColor // draw background color
	editor *Editor              // an editor struct (for getting the colors when clearing the status)
	show   time.Duration        // show the message for how long before clearing
	offset int
}

// Takes a foreground color, background color, foreground color for clearing,
// background color for clearing and a duration for how long to display status
// messages.
func NewStatusBar(fg, bg vt100.AttributeColor, editor *Editor, show time.Duration) *StatusBar {
	return &StatusBar{"", fg, bg, editor, show, 0}
}

func (sb *StatusBar) Draw(c *vt100.Canvas, offset int) {
	w := int(c.W())
	c.Write(uint((w-len(sb.msg))/2), c.H()-1, sb.fg, sb.bg, sb.msg)
	sb.offset = offset
}

func (sb *StatusBar) SetMessage(msg string) {
	sb.msg = "    " + msg + "    "
}

func (sb *StatusBar) Clear(c *vt100.Canvas) {
	sb.msg = ""
	// place all characters back in the canvas, but only for the last line
	h := int(c.H())
	sb.editor.WriteLines(c, (h-1)+sb.offset, h+sb.offset, 0, h-1)
	//c.Draw()
}

// Draw a status message, then clear it after a configurable delay
func (sb *StatusBar) Show(c *vt100.Canvas, p *Position) {
	if sb.msg == "" {
		return
	}
	sb.Draw(c, p.Offset())
	go func() {
		time.Sleep(sb.show)
		sb.Clear(c)
	}()
}

func (sb *StatusBar) SetColors(fg, bg vt100.AttributeColor) {
	sb.fg = fg
	sb.bg = bg
}
