package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	fourSpaces = "    "
	fiveSpaces = "     "
)

var mut *sync.RWMutex

// StatusBar represents the little status field that can appear at the bottom of the screen
type StatusBar struct {
	editor             *Editor        // an editor struct (for getting the colors when clearing the status)
	msg                string         // status message
	messageAfterRedraw string         // a message to be drawn and cleared AFTER the redraw
	fg                 AttributeColor // draw foreground color
	bg                 AttributeColor // draw background color
	errfg              AttributeColor // error foreground color
	errbg              AttributeColor // error background color
	show               time.Duration  // show the message for how long before clearing
	offsetY            int            // scroll offset
	isError            bool           // is this an error message that should be shown after redraw?
	nanoMode           bool           // Nano emulation?
}

// Used for keeping track of how many status messages are lined up to be cleared
var statusBeingShown int

// NewStatusBar takes a foreground color, background color, foreground color for clearing,
// background color for clearing and a duration for how long to display status messages.
func (e *Editor) NewStatusBar(statusDuration time.Duration, initialMessageAfterRedraw string) *StatusBar {
	mut = &sync.RWMutex{}
	return &StatusBar{e, "", initialMessageAfterRedraw, e.StatusForeground, e.StatusBackground, e.StatusErrorForeground, e.StatusErrorBackground, statusDuration, 0, false, e.nanoMode.Load()}
}

// Draw will draw the status bar to the canvas
func (sb *StatusBar) Draw(c *Canvas, offsetY int) {
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
func (sb *StatusBar) Clear(c *Canvas, repositionCursorAfterDrawing bool) {
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
	sb.editor.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, 0, false, true)

	c.HideCursorAndDraw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		sb.editor.EnableAndPlaceCursor(c)
	}
}

// ClearAll will clear all status messages
func (sb *StatusBar) ClearAll(c *Canvas, repositionCursorAfterDrawing bool) {
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
	sb.editor.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, 0, false, true)

	c.HideCursorAndDraw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		sb.editor.EnableAndPlaceCursor(c)
	}
}

// Show will draw a status message, then clear it after a certain delay
func (sb *StatusBar) Show(c *Canvas, e *Editor) {
	if c == nil {
		return
	}

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

	c.HideCursorAndDraw()
}

// ShowNoTimeout will draw a status message that will not be
// cleared after a certain timeout.
func (sb *StatusBar) ShowNoTimeout(c *Canvas, e *Editor) {
	if c == nil {
		return
	}

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

	c.HideCursorAndDraw()
}

func getPercentage(lineNumber, lastLineNumber LineNumber) int {
	if lastLineNumber > 0 {
		return int(100.0 * (float64(lineNumber) / float64(lastLineNumber)))
	}
	return 0
}

// PLA returns the linewise percentage, the current line number and the total number of lines
func (e *Editor) PLA() (int, LineNumber, LineNumber) {
	lineNumber := e.LineNumber()
	lastLineNumber := e.LastLineNumber()
	percentage := getPercentage(lineNumber, lastLineNumber)
	return percentage, lineNumber, lastLineNumber
}

// IndentationDescription returns "tabs" or "spaces", depending on the current setting
func (e *Editor) IndentationDescription() string {
	if e.indentation.Spaces {
		return "spaces"
	}
	return "tabs"
}

// ShowFilenameLineColWordCount sets a status message at the bottom, containing:
// * the current filename
// * the current line number (counting from 1)
// * the current number of lines
// * the current line percentage
// * the current column number (counting from 1)
// * the current rune unicode value
// * the current word count
// * the currently detected file mode
// * the current indentation mode (tabs or spaces)
// func FilenamePositionPercentageAndModeInfo(e *Editor) string {
func (sb *StatusBar) ShowFilenameLineColWordCount(c *Canvas, e *Editor) {
	indentation := e.IndentationDescription()
	percentage, lineNumber, lastLineNumber := e.PLA()
	statusLine := fmt.Sprintf("%s: line %d/%d (%d%%) col %d rune %U words %d, [%s] %s", e.filename, lineNumber, lastLineNumber, percentage, e.ColNumber(), e.Rune(), e.WordCount(), e.mode, indentation)
	sb.SetMessage(statusLine)
	sb.ShowNoTimeout(c, e)
}

// ShowBlockModeStatusLine shows a status message for when block mode is enabled
func (sb *StatusBar) ShowBlockModeStatusLine(c *Canvas, e *Editor) {
	indentation := e.IndentationDescription()
	percentage, lineNumber, lastLineNumber := e.PLA()
	statusLine := fmt.Sprintf("%s: line %d/%d (%d%%) col %d rune %U words %d, [Block Edit Mode, %s] %s", e.filename, lineNumber, lastLineNumber, percentage, e.ColNumber(), e.Rune(), e.WordCount(), e.mode, indentation)
	sb.SetMessage(statusLine)
	sb.ShowNoTimeout(c, e)
}

// NanoInfo shows info about the current position, for the Nano emulation mode
func (sb *StatusBar) NanoInfo(c *Canvas, e *Editor) {
	percentage, lineNumber, lastLineNumber := e.PLA()

	// TODO: implement char/byte number, like: [ line 2/2 (100%), col 1/1 (100%), char 8/8 (100%) ]
	//statusString := fmt.Sprintf("[ line %d/%d (%d%), col 1/1 (100%), char 8/8 (100%) ]", l, ls, int(lp*100.0), e.ColNumber(), 999, ?/?)
	// also available: e.indentation.Spaces and e.mode

	sb.SetMessage(fmt.Sprintf("[ line %d/%d (%d%%), col %d, word count %d ]", lineNumber, lastLineNumber, percentage, e.ColNumber(), e.WordCount()))
	sb.ShowNoTimeout(c, e)
}

// HoldMessage can be used to let a status message survive on screen for N seconds,
// even if e.redraw has been set. statusMessageAfterRedraw is a pointer to the one-off
// variable that will be used in keyloop.go, after redrawing.
func (sb *StatusBar) HoldMessage(c *Canvas, dur time.Duration) {
	if strings.TrimSpace(sb.msg) != "" {
		sb.messageAfterRedraw = sb.msg
		go func() {
			time.Sleep(dur)
			sb.ClearAll(c, true)
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
