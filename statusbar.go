package main

import (
	"strconv"
	"sync"
	"time"

	"github.com/xyproto/vt100"
)

var mut sync.RWMutex

// StatusBar represents the little status field that can appear at the bottom of the screen
type StatusBar struct {
	msg     string               // status message
	fg      vt100.AttributeColor // draw foreground color
	bg      vt100.AttributeColor // draw background color
	errfg   vt100.AttributeColor // error foreground color
	errbg   vt100.AttributeColor // error background color
	editor  *Editor              // an editor struct (for getting the colors when clearing the status)
	show    time.Duration        // show the message for how long before clearing
	offset  int                  // scroll offset
	isError bool                 // is this an error message that should be shown after redraw?
}

// Used for keeping track of how many status messages are lined up to be cleared
var statusBeingShown int

// NewStatusBar takes a foreground color, background color, foreground color for clearing,
// background color for clearing and a duration for how long to display status messages.
func NewStatusBar(fg, bg, errfg, errbg vt100.AttributeColor, editor *Editor, show time.Duration) *StatusBar {
	return &StatusBar{"", fg, bg, errfg, errbg, editor, show, 0, false}
}

// Draw will draw the status bar to the canvas
func (sb *StatusBar) Draw(c *vt100.Canvas, offset int) {
	w := int(c.W())

	if sb.IsError() {
		mut.RLock()
		c.Write(uint((w-len(sb.msg))/2), c.H()-1, sb.errfg, sb.errbg, sb.msg)
		mut.RUnlock()
	} else {
		mut.RLock()
		c.Write(uint((w-len(sb.msg))/2), c.H()-1, sb.fg, sb.bg, sb.msg)
		mut.RUnlock()
	}

	mut.Lock()
	sb.offset = offset
	mut.Unlock()
}

// SetMessage will change the status bar message.
// A couple of spaces are added as padding.
func (sb *StatusBar) SetMessage(msg string) {
	mut.Lock()
	sb.msg = "    " + msg + "    "
	sb.isError = false
	mut.Unlock()
}

// IsError returns true if the error message to be shown is an error message
// (it's being displayed a bit longer)
func (sb *StatusBar) IsError() bool {
	var isError bool

	mut.RLock()
	isError = sb.isError
	mut.RUnlock()

	return isError
}

// SetErrorMessage is for setting a message that will be shown after a full editor redraw,
// to make the message appear also after jumping around in the text.
func (sb *StatusBar) SetErrorMessage(msg string) {
	mut.Lock()
	sb.msg = "    " + msg + "    "
	sb.isError = true
	mut.Unlock()
}

// Clear will set the message to nothing and then use the editor contents
// to remove the status bar field at the bottom of the editor.
func (sb *StatusBar) Clear(c *vt100.Canvas) {
	mut.Lock()
	sb.msg = ""
	mut.Unlock()

	h := int(c.H())

	mut.RLock()
	e := sb.editor
	mut.RUnlock()

	// Write all lines to the buffer
	mut.RLock()
	offset := e.pos.Offset()
	mut.RUnlock()

	e.WriteLines(c, offset, h+offset, 0, 0)
	c.Draw()

	// Not an error message
	mut.Lock()
	sb.isError = false
	mut.Unlock()
}

// ClearAll will clear all status messages
func (sb *StatusBar) ClearAll(c *vt100.Canvas) {
	sb.Clear(c)

	mut.Lock()
	statusBeingShown = 0
	mut.Unlock()
}

// Show will draw a status message, then clear it after a certain delay
func (sb *StatusBar) Show(c *vt100.Canvas, e *Editor) {
	mut.RLock()
	if sb.msg == "" {
		mut.RUnlock()
		return
	}
	mut.RUnlock()

	sb.Draw(c, e.pos.Offset())

	mut.Lock()
	statusBeingShown++
	mut.Unlock()

	go func() {
		mut.RLock()
		sleepDuration := sb.show
		mut.RUnlock()

		if sb.IsError() {
			// Show error messages for 3x as long
			sleepDuration *= 3
		}
		time.Sleep(sleepDuration)

		mut.Lock()
		statusBeingShown--
		mut.Unlock()

		mut.RLock()
		if statusBeingShown == 0 {
			mut.RUnlock()
			sb.Clear(c)
		} else {
			mut.RUnlock()
		}
	}()
	c.Draw()
}

// ShowNoTimeout will draw a status message that will not be cleared after a certain timeout
func (sb *StatusBar) ShowNoTimeout(c *vt100.Canvas, e *Editor) {
	mut.RLock()
	if sb.msg == "" {
		mut.RUnlock()
		return
	}
	mut.RUnlock()

	mut.RLock()
	offset := e.pos.Offset()
	mut.RUnlock()

	sb.Draw(c, offset)

	mut.Lock()
	statusBeingShown++
	mut.Unlock()

	c.Draw()
}

// SetColors can be used for setting a color theme for the status bar field
// bg should be a background attribute, like vt100.BackgroundBlue.
func (sb *StatusBar) SetColors(fg, bg vt100.AttributeColor) {
	mut.Lock()
	sb.fg = fg
	sb.bg = bg
	mut.Unlock()
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
