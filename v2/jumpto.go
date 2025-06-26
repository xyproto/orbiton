package main

import (
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/xyproto/vt"
)

// PositionIndex has a ColIndex and LineIndex
type PositionIndex struct {
	X ColIndex
	Y LineIndex
}

var jumpLetters map[rune]PositionIndex

// RegisterJumpLetter will register a jump-letter together with a location that is visible on screen
func (e *Editor) RegisterJumpLetter(r rune, x ColIndex, y LineIndex) bool {
	const skipThese = "0123456789%.,btc?!/" // used by the ctrl-l functionality for other things
	if strings.ContainsRune(skipThese, r) || unicode.IsSymbol(r) {
		return false
	}
	if jumpLetters == nil {
		jumpLetters = make(map[rune]PositionIndex)
	}
	jumpLetters[r] = PositionIndex{x, y}
	return true
}

// HasJumpLetter checks if this jump letter has been registered yet
func (e *Editor) HasJumpLetter(r rune) bool {
	if jumpLetters == nil {
		return false
	}
	_, found := jumpLetters[r]
	return found
}

// GetJumpX returns the X position for the given jump letter, or -1 if not found
func (e *Editor) GetJumpX(r rune) ColIndex {
	if jumpLetters == nil {
		return -1
	}
	xy, found := jumpLetters[r]
	if !found {
		return -1
	}
	return xy.X
}

// GetJumpY returns the Y position for the given jump letter, or -1 if not found
func (e *Editor) GetJumpY(r rune) LineIndex {
	if jumpLetters == nil {
		return -1
	}
	xy, found := jumpLetters[r]
	if !found {
		return -1
	}
	return xy.Y
}

// ClearJumpLetters clears all jump letters (typically after the ctrl-l screen is done)
func (e *Editor) ClearJumpLetters() {
	jumpLetters = nil
}

// GoTo will go to a given line index, counting from 0
// status is used for clearing status bar messages and can be nil
// Returns true if the editor should be redrawn
// The second returned bool is if the end has been reached
func (e *Editor) GoTo(dataY LineIndex, c *vt.Canvas, status *StatusBar) (bool, bool) {
	if dataY == e.DataY() {
		// Already at the correct line, but still trigger a redraw
		return true, false
	}
	reachedTheEnd := false
	// Out of bounds checking for y
	if dataY < 0 {
		dataY = 0
	} else if dataY >= LineIndex(e.Len()) {
		dataY = LineIndex(e.Len() - 1)
		reachedTheEnd = true
	}

	h := 25
	if c != nil {
		// Get the current terminal height
		h = int(c.Height())
	}

	// Is the place we want to go within the current scroll window?
	topY := LineIndex(e.pos.offsetY)
	botY := LineIndex(e.pos.offsetY + h)

	if dataY >= topY && dataY < botY {
		// No scrolling is needed, just move the screen y position
		e.pos.sy = int(dataY) - e.pos.offsetY
		if e.pos.sy < 0 {
			e.pos.sy = 0
		}
	} else if int(dataY) < h {
		// No scrolling is needed, just move the screen y position
		e.pos.offsetY = 0
		e.pos.sy = int(dataY)
		if e.pos.sy < 0 {
			e.pos.sy = 0
		}
	} else if reachedTheEnd {
		// To the end of the text
		e.pos.offsetY = e.Len() - h
		e.pos.sy = h - 1
	} else {
		prevY := e.pos.sy
		// Scrolling is needed
		e.pos.sy = 0
		e.pos.offsetY = int(dataY)
		lessJumpY := prevY
		lessJumpOffset := int(dataY) - prevY
		if (lessJumpY + lessJumpOffset) < e.Len() {
			e.pos.sy = lessJumpY
			e.pos.offsetY = lessJumpOffset
		}
	}

	// The Y scrolling is done, move the X position according to the contents of the line
	e.pos.SetX(c, int(e.FirstScreenPosition(e.DataY())))

	// Clear all status messages
	if status != nil {
		status.ClearAll(c, true)
	}

	// Trigger cursor redraw
	e.redrawCursor.Store(true)

	// Should also redraw the text, and has the end been reached?
	return true, reachedTheEnd
}

// GoToLineNumber will go to a given line number, but counting from 1, not from 0!
func (e *Editor) GoToLineNumber(lineNumber LineNumber, c *vt.Canvas, status *StatusBar, center bool) bool {
	if lineNumber < 1 {
		lineNumber = 1
	}
	redraw, _ := e.GoTo(lineNumber.LineIndex(), c, status)
	if redraw && center {
		e.Center(c)
	}
	return redraw
}

// GoToLineNumberAndCol will go to a given line number (counting from 1) and column number (counting from 1)
func (e *Editor) GoToLineNumberAndCol(lineNumber LineNumber, colNumber ColNumber, c *vt.Canvas, status *StatusBar, center, handleTabExpansion bool) bool {
	if colNumber < 1 {
		colNumber = 1
	}
	if lineNumber < 1 {
		lineNumber = 1
	}
	xIndex := colNumber.ColIndex()
	yIndex := lineNumber.LineIndex()
	// Go to the correct line
	redraw, _ := e.GoTo(yIndex, c, status)
	// Go to the correct column as well
	if handleTabExpansion {
		tabs := strings.Count(e.Line(yIndex), "\t")
		newScreenX := int(xIndex) + (tabs * (e.indentation.PerTab - 1))
		if e.pos.sx != newScreenX {
			redraw = true
		}
		e.pos.sx = newScreenX
	} else {
		if e.pos.sx != int(xIndex) {
			redraw = true
		}
		e.pos.sx = int(xIndex)
	}
	if redraw && center {
		e.Center(c)
	}
	return redraw

}

// GoToLineIndexAndColIndex will go to a given line index (counting from 0) and column index (counting from 0)
func (e *Editor) GoToLineIndexAndColIndex(yIndex LineIndex, xIndex ColIndex, c *vt.Canvas, status *StatusBar, center, handleTabExpansion bool) bool {
	if xIndex < 0 {
		xIndex = 0
	}
	if yIndex < 0 {
		yIndex = 0
	}
	// Go to the correct line
	redraw, _ := e.GoTo(yIndex, c, status)
	// Go to the correct column as well
	if handleTabExpansion {
		tabs := strings.Count(e.Line(yIndex), "\t")
		newScreenX := int(xIndex) + (tabs * (e.indentation.PerTab - 1))
		if e.pos.sx != newScreenX {
			redraw = true
		}
		e.pos.sx = newScreenX
	} else {
		if e.pos.sx != int(xIndex) {
			redraw = true
		}
		e.pos.sx = int(xIndex)
	}
	if redraw && center {
		e.Center(c)
	}
	return redraw
}

const (
	noAction = iota
	showHotkeyOverviewAction
	launchTutorialAction
	scrollUpAction
	scrollDownAction
	displayQuickHelpAction
)

// JumpMode initiates the mode where the user can enter where to jump to
// (line number, percentage, fraction or highlighted letter).
// Returns ShowHotkeyOverviewAction if a hotkey overview should be shown after this function.
// Returns LaunchTutorialAction if the tutorial should be launched after this function.
func (e *Editor) JumpMode(c *vt.Canvas, status *StatusBar, tty *vt.TTY) int {
	e.jumpToLetterMode = true
	prevCommentColor := e.CommentColor
	prevSyntaxHighlighting := e.syntaxHighlight
	e.syntaxHighlight = true
	prompt := "Go to line number, letter or percentage:"
	if envNoColor {
		// TODO: NO_COLOR=1 does not have the "jump to letter" feature, this could be implemented
		prompt = "Go to line number or percentage:"
	}
	// Minor adjustments for some of the themes used in the VTE/GTK frontend
	if inVTEGUI {
		if !e.Light && e.Name == "Default" {
			e.CommentColor = vt.White
		} else if strings.HasPrefix(e.Name, "Blue") {
			e.CommentColor = vt.Gray
		}
	}

	// TODO: Figure out why this call is needed for the letters to be highlighted
	status.ClearAll(c, true)

	status.SetMessage(prompt)
	status.ShowNoTimeout(c, e)
	lns := ""
	cancel := false
	doneCollectingDigits := false
	goToEnd := false
	goToTop := false
	goToCenter := false
	goToLetter := rune(0)
	toggleQuickHelpScreen := false

	// Which action should be taken after this function returns?
	postAction := noAction

	for !doneCollectingDigits {
		numkey := tty.String()
		switch numkey {
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "%", ".", ",": // 0..9 + %,.
			lns += numkey // string('0' + (numkey - 48))
			status.SetMessage(prompt + " " + lns)
			status.ShowNoTimeout(c, e)
		case "c:8", "c:127": // ctrl-h or backspace
			if len(lns) > 0 {
				lns = lns[:len(lns)-1]
				status.SetMessage(prompt + " " + lns)
				status.ShowNoTimeout(c, e)
			}
		case "t", homeKey: // top of file
			doneCollectingDigits = true
			goToTop = true
		case "b", endKey: // end of file
			doneCollectingDigits = true
			goToEnd = true
		case "c": // center of file
			doneCollectingDigits = true
			goToCenter = true
		case "?": // display tutorial
			doneCollectingDigits = true
			postAction = launchTutorialAction
		case "!": // disable splash screen
			doneCollectingDigits = true
			toggleQuickHelpScreen = true
		case "/": // display hotkey overview
			doneCollectingDigits = true
			postAction = showHotkeyOverviewAction
		case upArrow, downArrow, leftArrow, rightArrow: // one of the arrow keys
			fallthrough // cancel
		case "c:16": // ctrl-p, scroll up
			doneCollectingDigits = true
			postAction = scrollUpAction
		case "c:14": // ctrl-n, scroll down
			doneCollectingDigits = true
			postAction = scrollDownAction
		case "c:17", "c:27", "c:11", "c:15": // ctrl-q, esc, ctrl-k or ctrl-o (keys near ctrl-l)
			cancel = true
			lns = ""
			e.redraw.Store(true)
			e.redrawCursor.Store(true)
			postAction = noAction
			fallthrough // done
		case "c:13", "c:12": // return or ctrl-l
			doneCollectingDigits = true
		default:
			if numkey != "" {
				r := []rune(numkey)[0]
				// check the "jump to" keys
				if e.HasJumpLetter(r) {
					goToLetter = r
					doneCollectingDigits = true
				}
			}
		}
	}
	if !cancel {
		e.ClearSearch()
	}
	status.ClearAll(c, true)
	if goToLetter != rune(0) {
		colIndex := e.GetJumpX(goToLetter)
		lineIndex := e.GetJumpY(goToLetter)
		const center = false
		const handleTabsAsWell = false
		redraw := e.GoToLineIndexAndColIndex(lineIndex, colIndex, c, status, center, handleTabsAsWell)
		e.redraw.Store(redraw)
	} else if goToTop {
		e.GoToTop(c, status)
		e.drawProgress.Store(true)
	} else if goToCenter {
		// Go to the center line
		e.GoToMiddle(c, status)
		e.drawProgress.Store(true)
	} else if goToEnd {
		e.GoToEnd(c, status)
		e.drawProgress.Store(true)
	} else if toggleQuickHelpScreen {
		ok := false
		if QuickHelpScreenIsDisabled() {
			ok = EnableQuickHelpScreen(status)
			e.displayQuickHelp = true
			postAction = displayQuickHelpAction
		} else {
			ok = DisableQuickHelpScreen(status)
		}
		e.redraw.Store(ok)
		e.redrawCursor.Store(ok)
	} else if lns == "" && !cancel && postAction == noAction {
		if e.DataY() > 0 {
			// If not already at the top, go there
			e.GoToTop(c, status)
			e.drawProgress.Store(true)
		} else {
			// Go to the last line
			e.GoToEnd(c, status)
			e.drawProgress.Store(true)
		}
	} else if strings.HasSuffix(lns, "%") {
		// Go to the specified percentage
		if percentageInt, err := strconv.Atoi(lns[:len(lns)-1]); err == nil { // no error {
			lineIndex := int(math.Round(float64(e.Len()) * float64(percentageInt) * 0.01))
			e.redraw.Store(e.GoToLineNumber(LineNumber(lineIndex), c, status, true))
			e.drawProgress.Store(true)
		}
	} else if strings.Count(lns, ".") == 1 || strings.Count(lns, ",") == 1 {
		if percentageFloat, err := strconv.ParseFloat(strings.ReplaceAll(lns, ",", "."), 64); err == nil { // no error
			lineIndex := int(math.Round(float64(e.Len()) * percentageFloat))
			e.redraw.Store(e.GoToLineNumber(LineNumber(lineIndex), c, status, true))
			e.drawProgress.Store(true)
		}
	} else if postAction == noAction {
		// Go to the specified line
		if ln, err := strconv.Atoi(lns); err == nil { // no error
			e.redraw.Store(e.GoToLineNumber(LineNumber(ln), c, status, true))
			e.drawProgress.Store(true)
		}
	}
	e.jumpToLetterMode = false
	e.syntaxHighlight = prevSyntaxHighlighting
	e.CommentColor = prevCommentColor
	e.ClearJumpLetters()

	return postAction
}

// JumpToMatching can jump to a to matching parenthesis or bracket ([{.
// Return true if a jump was possible and happened.
func (e *Editor) JumpToMatching(c *vt.Canvas) bool {
	const maxSearchLength = 256000
	var r = e.Rune()
	// Find which opening and closing parenthesis/curly brackets to look for
	opening, closing := rune(0), rune(0)
	onparen := true
	switch r {
	case '(', ')':
		opening = '('
		closing = ')'
	case '{', '}':
		opening = '{'
		closing = '}'
	case '[', ']':
		opening = '['
		closing = ']'
	default:
		onparen = false
	}
	if onparen {
		// Search either forwards or backwards to find a matching rune
		switch r {
		case '(', '{', '[':
			parcount := 0
			counter := 0
			found := false
			for counter < maxSearchLength {
				counter++
				if r := e.Rune(); r == closing {
					if parcount == 1 {
						found = true
						// FOUND, STOP
						break
					}
					parcount--
				} else if r == opening {
					parcount++
				}
				if err := e.Next(c); err != nil {
					break
				}
			}
			if !found {
				return false
			}
		case ')', '}', ']':
			parcount := 0
			counter := 0
			found := false
			for counter < maxSearchLength {
				counter++
				if r := e.Rune(); r == opening {
					if parcount == 1 {
						found = true
						// FOUND, STOP
						break
					}
					parcount--
				} else if r == closing {
					parcount++
				}
				if err := e.Prev(c); err != nil {
					break
				}
			}
			if !found {
				return false
			}
		}
		e.redrawCursor.Store(true)
		e.redraw.Store(true)
		return true
	}
	return false
}
