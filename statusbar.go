package main

import (
	"strconv"
	"time"

	"github.com/xyproto/vt100"
)

// StatusBar represents the little status field that can appear at the bottom of the screen
type StatusBar struct {
	msg    string               // status message
	fg     vt100.AttributeColor // draw foreground color
	bg     vt100.AttributeColor // draw background color
	editor *Editor              // an editor struct (for getting the colors when clearing the status)
	show   time.Duration        // show the message for how long before clearing
	offset int                  // scroll offset
}

// Used for keeping track of how many status messages are lined up to be cleared
var statusBeingShown int

// NewStatusBar takes a foreground color, background color, foreground color for clearing,
// background color for clearing and a duration for how long to display status messages.
func NewStatusBar(fg, bg vt100.AttributeColor, editor *Editor, show time.Duration) *StatusBar {
	return &StatusBar{"", fg, bg, editor, show, 0}
}

// Draw will draw the status bar to the canvas
func (sb *StatusBar) Draw(c *vt100.Canvas, offset int) {
	w := int(c.W())
	c.Write(uint((w-len(sb.msg))/2), c.H()-1, sb.fg, sb.bg, sb.msg)
	sb.offset = offset
}

// SetMessage will change the status bar message.
// A couple of spaces are added as padding.
func (sb *StatusBar) SetMessage(msg string) {
	sb.msg = "    " + msg + "    "
}

// Clear will set the message to nothing and then use the editor contents
// to remove the status bar field at the bottom of the editor.
func (sb *StatusBar) Clear(c *vt100.Canvas) {
	sb.msg = ""
	// place all characters back in the canvas, but only for the last line
	h := int(c.H())
	sb.editor.WriteLines(c, (h-1)+sb.offset, h+sb.offset, 0, h-1)
	c.Draw()
}

// ClearAll will clear all status messages
func (sb *StatusBar) ClearAll(c *vt100.Canvas) {
	sb.Clear(c)
	statusBeingShown = 0
}

// Show will draw a status message, then clear it after a certain delay
func (sb *StatusBar) Show(c *vt100.Canvas, e *Editor) {
	if sb.msg == "" {
		return
	}
	sb.Draw(c, e.pos.Offset())
	statusBeingShown++
	go func() {
		time.Sleep(sb.show)
		statusBeingShown--
		if statusBeingShown == 0 {
			sb.Clear(c)
		}
	}()
	c.Draw()
}

// ShowNoTimeout will draw a status message that will not be cleared after a certain timeout
func (sb *StatusBar) ShowNoTimeout(c *vt100.Canvas, e *Editor) {
	if sb.msg == "" {
		return
	}
	sb.Draw(c, e.pos.Offset())
	statusBeingShown++
	c.Draw()
}

// SetColors can be used for setting a color theme for the status bar field
// bg should be a background attribute, like vt100.BackgroundBlue.
func (sb *StatusBar) SetColors(fg, bg vt100.AttributeColor) {
	sb.fg = fg
	sb.bg = bg
}

// ShowWordCount displays a status message with only the current word count
func (sb *StatusBar) ShowWordCount(c *vt100.Canvas, e *Editor) {
	wordCountString := strconv.Itoa(e.WordCount())
	sb.SetMessage(wordCountString)
	sb.ShowNoTimeout(c, e)
}

// ShowLineColWordCount shows a status message with the current filename, line, column and word count
func (sb *StatusBar) ShowLineColWordCount(c *vt100.Canvas, e *Editor, filename string) {
	statusString := filename + ": " + e.StatusMessage()
	sb.SetMessage(statusString)
	sb.ShowNoTimeout(c, e)
}
