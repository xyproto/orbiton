package main

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xyproto/vt100"
)

var mut *sync.RWMutex

// StatusBar represents the little status field that can appear at the bottom of the screen
type StatusBar struct {
	msg                string               // status message
	fg                 vt100.AttributeColor // draw foreground color
	bg                 vt100.AttributeColor // draw background color
	errfg              vt100.AttributeColor // error foreground color
	errbg              vt100.AttributeColor // error background color
	editor             *Editor              // an editor struct (for getting the colors when clearing the status)
	show               time.Duration        // show the message for how long before clearing
	offsetY            int                  // scroll offset
	isError            bool                 // is this an error message that should be shown after redraw?
	messageAfterRedraw string               // a message to be drawn and cleared AFTER the redraw
}

// Used for keeping track of how many status messages are lined up to be cleared
var statusBeingShown int

// NewStatusBar takes a foreground color, background color, foreground color for clearing,
// background color for clearing and a duration for how long to display status messages.
func NewStatusBar(fg, bg, errfg, errbg vt100.AttributeColor, editor *Editor, show time.Duration, initialMessageAfterRedraw string) *StatusBar {
	mut = &sync.RWMutex{}
	return &StatusBar{"", fg, bg, errfg, errbg, editor, show, 0, false, initialMessageAfterRedraw}
}

// Draw will draw the status bar to the canvas
func (sb *StatusBar) Draw(c *vt100.Canvas, offsetY int) {
	w := int(c.W())

	// Shorten the status message if it's longer than the terminal width
	if len(sb.msg) >= w {
		sb.msg = sb.msg[:w-4] + "..."
	}

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
	sb.offsetY = offsetY
	mut.Unlock()
}

// SetMessage will change the status bar message.
// A couple of spaces are added as padding.
func (sb *StatusBar) SetMessage(msg string) {
	mut.Lock()

	if len(msg)%2 == 0 {
		sb.msg = "     "
	} else {
		sb.msg = "    "
	}
	sb.msg += msg + "    "

	sb.isError = false
	mut.Unlock()
}

// Message trims and returns the currently set status bar message
func (sb *StatusBar) Message() string {
	mut.RLock()
	s := strings.TrimSpace(sb.msg)
	mut.RUnlock()
	return s
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

	if len(msg)%2 == 0 {
		sb.msg = "     "
	} else {
		sb.msg = "    "
	}
	sb.msg += msg + "    "

	sb.isError = true
	mut.Unlock()
}

// Clear will set the message to nothing and then use the editor contents
// to remove the status bar field at the bottom of the editor.
func (sb *StatusBar) Clear(c *vt100.Canvas) error {
	var err error
	// Write all lines to the buffer
	mut.Lock()

	// Clear the message
	sb.msg = ""
	// Not an error message
	sb.isError = false

	mut.Unlock()

	if c == nil {
		return nil
	}

	// Then clear/redraw the bottom line
	h := int(c.H())
	mut.RLock()
	offsetY := sb.editor.pos.OffsetY()
	sb.editor.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, 0)
	mut.RUnlock()
	c.Draw()
	return err
}

// ClearAll will clear all status messages
func (sb *StatusBar) ClearAll(c *vt100.Canvas) {
	mut.Lock()
	statusBeingShown = 0
	// Clear the message
	sb.msg = ""
	// Not an error message
	sb.isError = false
	mut.Unlock()

	if c == nil {
		return
	}

	// Then clear/redraw the bottom line
	h := int(c.H())
	mut.RLock()
	offsetY := sb.editor.pos.OffsetY()
	sb.editor.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, 0)
	mut.RUnlock()
	c.Draw()
}

// Show will draw a status message, then clear it after a certain delay
func (sb *StatusBar) Show(c *vt100.Canvas, e *Editor) {
	mut.Lock()
	statusBeingShown++
	mut.Unlock()

	mut.RLock()
	if sb.msg == "" {
		mut.RUnlock()
		return
	}
	offsetY := e.pos.OffsetY()
	mut.RUnlock()

	sb.Draw(c, offsetY)

	go func() {
		mut.RLock()
		sleepDuration := sb.show
		mut.RUnlock()

		if sb.IsError() {
			// Show error messages for 3x as long
			sleepDuration *= 3
		}
		time.Sleep(sleepDuration)

		mut.RLock()
		// Has everyhing been cleared while sleeping?
		if statusBeingShown <= 0 {
			// Yes, so just quit
			mut.RUnlock()
			return
		}
		mut.RUnlock()

		mut.Lock()
		statusBeingShown--
		mut.Unlock()

		mut.RLock()
		if statusBeingShown == 0 {
			mut.RUnlock()
			mut.Lock()
			// Clear the message
			sb.msg = ""
			// Not an error message
			sb.isError = false
			mut.Unlock()
		} else {
			mut.RUnlock()
		}
	}()
	c.Draw()
}

// ShowNoTimeout will draw a status message that will not be
// cleared after a certain timeout.
func (sb *StatusBar) ShowNoTimeout(c *vt100.Canvas, e *Editor) {
	mut.RLock()
	if sb.msg == "" {
		mut.RUnlock()
		return
	}
	mut.RUnlock()

	mut.RLock()
	offsetY := e.pos.OffsetY()
	mut.RUnlock()

	sb.Draw(c, offsetY)

	mut.Lock()
	statusBeingShown++
	mut.Unlock()

	c.Draw()
}

// SetColors can be used for setting a color theme for the status bar field
// bg should be a background attribute, like vt100.BackgroundBlue.
func (sb *StatusBar) SetColors(fg, bg vt100.AttributeColor) {
	if envNoColor {
		fg = vt100.Default
		bg = vt100.BackgroundDefault
	}
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

// HoldMessage can be used to let a status message survive on screen for N seconds,
// even if e.redraw has been set. statusMessageAfterRedraw is a pointer to the one-off
// variable that will be used in keyloop.go, after redrawing.
func (sb *StatusBar) HoldMessage(c *vt100.Canvas, dur time.Duration) {
	if strings.TrimSpace(sb.msg) != "" {
		sb.messageAfterRedraw = sb.msg
		go func() {
			time.Sleep(dur)
			sb.ClearAll(c)
		}()
	}
}

// ShowAfterRedraw prepares a status bar message that will be shown after redraw
func (sb *StatusBar) ShowAfterRedraw(message string) {
	sb.messageAfterRedraw = message
}
