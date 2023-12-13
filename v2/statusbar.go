package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xyproto/vt100"
)

const (
	fourSpaces = "    "
	fiveSpaces = "     "
)

var mut *sync.RWMutex

// StatusBar represents the little status field that can appear at the bottom of the screen
type StatusBar struct {
	editor             *Editor              // an editor struct (for getting the colors when clearing the status)
	msg                string               // status message
	messageAfterRedraw string               // a message to be drawn and cleared AFTER the redraw
	fg                 vt100.AttributeColor // draw foreground color
	bg                 vt100.AttributeColor // draw background color
	errfg              vt100.AttributeColor // error foreground color
	errbg              vt100.AttributeColor // error background color
	show               time.Duration        // show the message for how long before clearing
	offsetY            int                  // scroll offset
	isError            bool                 // is this an error message that should be shown after redraw?
	nanoMode           bool                 // Nano emulation?
}

// Used for keeping track of how many status messages are lined up to be cleared
var statusBeingShown int

// NewStatusBar takes a foreground color, background color, foreground color for clearing,
// background color for clearing and a duration for how long to display status messages.
func NewStatusBar(fg, bg, errfg, errbg vt100.AttributeColor, editor *Editor, show time.Duration, initialMessageAfterRedraw string, nanoMode bool) *StatusBar {
	mut = &sync.RWMutex{}
	return &StatusBar{editor, "", initialMessageAfterRedraw, fg, bg, errfg, errbg, show, 0, false, nanoMode}
}

// Draw will draw the status bar to the canvas
func (sb *StatusBar) Draw(c *vt100.Canvas, offsetY int) {
	w := int(c.W())

	// Shorten the status message if it's longer than the terminal width
	if len(sb.msg) >= w && w > 4 {
		sb.msg = sb.msg[:w-4] + "..."
	}

	h := c.H() - 1
	if sb.nanoMode {
		h -= 2
	}

	if sb.IsError() {
		mut.RLock()
		c.Write(uint((w-len(sb.msg))/2), h, sb.errfg, sb.errbg, sb.msg)
		mut.RUnlock()
	} else {
		mut.RLock()
		c.Write(uint((w-len(sb.msg))/2), h, sb.fg, sb.bg, sb.msg)
		mut.RUnlock()
	}

	if sb.nanoMode {
		mut.RLock()
		// x-align
		x := uint((w - len(nanoHelpString1)) / 2)
		c.Write(x, h+1, sb.editor.NanoHelpForeground, sb.editor.NanoHelpBackground, nanoHelpString1)
		c.Write(x, h+2, sb.editor.NanoHelpForeground, sb.editor.NanoHelpBackground, nanoHelpString2)
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
		sb.msg = fiveSpaces
	} else {
		sb.msg = fourSpaces
	}
	sb.msg += msg + fourSpaces

	sb.isError = true
	mut.Unlock()
}

// SetError is for setting the error message
func (sb *StatusBar) SetError(err error) {
	sb.SetErrorMessage(err.Error())
}

// Clear will set the message to nothing and then use the editor contents
// to remove the status bar field at the bottom of the editor.
func (sb *StatusBar) Clear(c *vt100.Canvas) {
	mut.Lock()
	defer mut.Unlock()

	// Clear the message
	sb.msg = ""
	// Not an error message
	sb.isError = false

	if c == nil {
		return
	}

	// Then clear/redraw the bottom line
	h := int(c.H())
	if sb.nanoMode {
		h -= 2
	}
	offsetY := sb.editor.pos.OffsetY()
	sb.editor.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, 0)

	c.Draw()
}

// ClearAll will clear all status messages
func (sb *StatusBar) ClearAll(c *vt100.Canvas) {
	mut.Lock()
	defer mut.Unlock()

	statusBeingShown = 0

	// Clear the message
	sb.msg = ""
	// Not an error message
	sb.isError = false

	if c == nil {
		return
	}

	// Then clear/redraw the bottom line
	h := int(c.H())
	if sb.nanoMode {
		h -= 2
	}
	offsetY := sb.editor.pos.OffsetY()
	sb.editor.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, 0)

	c.Draw()
}

// Show will draw a status message, then clear it after a certain delay
func (sb *StatusBar) Show(c *vt100.Canvas, e *Editor) {
	mut.Lock()
	statusBeingShown++
	mut.Unlock()

	mut.RLock()
	if sb.msg == "" && !sb.nanoMode {
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
	if sb.msg == "" && !sb.nanoMode {
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

// ShowLineColWordCount shows a status message with the current filename, line, column and word count
func (sb *StatusBar) ShowLineColWordCount(c *vt100.Canvas, e *Editor, filename string) {
	statusString := filename + ": " + e.PositionPercentageAndModeInfo()
	sb.SetMessage(statusString)
	sb.ShowNoTimeout(c, e)
}

// NanoInfo shows info about the current position, for the Nano emulation mode
func (sb *StatusBar) NanoInfo(c *vt100.Canvas, e *Editor) {
	l := e.LineNumber()
	ls := e.LastLineNumber()
	lp := 0
	if ls > 0 {
		lp = int(100.0 * (float64(l) / float64(ls)))
	}

	// TODO: implement char/byte number, like: [ line 2/2 (100%), col 1/1 (100%), char 8/8 (100%) ]
	//statusString := fmt.Sprintf("[ line %d/%d (%d%), col 1/1 (100%), char 8/8 (100%) ]", l, ls, int(lp*100.0), e.ColNumber(), 999, ?/?)
	// also available: e.indentation.Spaces and e.mode

	statusString := fmt.Sprintf("[ line %d/%d (%d%%), col %d, word count %d ]", l, ls, lp, e.ColNumber(), e.WordCount())

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

// SetMessageAfterRedraw prepares a status bar message that will be shown after redraw
func (sb *StatusBar) SetMessageAfterRedraw(message string) {
	sb.messageAfterRedraw = message
}

// SetErrorAfterRedraw prepares a status bar message that will be shown after redraw
func (sb *StatusBar) SetErrorAfterRedraw(err error) {
	sb.messageAfterRedraw = err.Error()
}

// SetErrorMessageAfterRedraw prepares a status bar message that will be shown after redraw
func (sb *StatusBar) SetErrorMessageAfterRedraw(errorMessage string) {
	sb.messageAfterRedraw = errorMessage
}
