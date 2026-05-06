package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

const (
	fourSpaces = "    "
	fiveSpaces = "     "
)

var (
	mut     *sync.RWMutex
	mutOnce sync.Once
)

// StatusBar represents the little status field that can appear at the bottom of the screen
type StatusBar struct {
	editor             *Editor           // an editor struct (for getting the colors when clearing the status)
	msg                string            // status message
	messageAfterRedraw string            // a message to be drawn and cleared AFTER the redraw
	fg                 vt.AttributeColor // draw foreground color
	bg                 vt.AttributeColor // draw background color
	errfg              vt.AttributeColor // error foreground color
	errbg              vt.AttributeColor // error background color
	show               time.Duration     // show the message for how long before clearing
	offsetY            int               // scroll offset
	isError            bool              // is this an error message that should be shown after redraw?
	nanoMode           bool              // Nano emulation?
}

// Used for keeping track of how many status messages are lined up to be cleared
var statusBeingShown int

// statusMsgGen is incremented whenever sb.msg changes,
// so that auto-clear goroutines only clear the message they were spawned for.
var statusMsgGen atomic.Int64

// stickyBarsJustDrawn is set by Show/ShowNoTimeout when they draw the
// sticky bars directly. The redraw loop checks this flag to avoid
// overwriting the bars with a stale message.
var stickyBarsJustDrawn atomic.Bool

// NewStatusBar takes a foreground color, background color, foreground color for clearing,
// background color for clearing and a duration for how long to display status messages.
func (e *Editor) NewStatusBar(statusDuration time.Duration, initialMessageAfterRedraw string) *StatusBar {
	mutOnce.Do(func() { mut = &sync.RWMutex{} })
	return &StatusBar{e, "", initialMessageAfterRedraw, e.StatusForeground, e.StatusBackground, e.StatusErrorForeground, e.StatusErrorBackground, statusDuration, 0, false, e.nanoMode.Load()}
}

// Draw will draw the status bar to the canvas
func (sb *StatusBar) Draw(c *vt.Canvas, offsetY int) {
	w := int(c.W())

	// Shorten the status message if it's longer than the terminal width
	if len(sb.msg) >= w && w > 4 {
		sb.msg = sb.msg[:w-4] + "..."
	}

	h := c.H() - 1
	if sb.nanoMode {
		h -= 2
	}

	msgX := max((w-len(sb.msg))/2, 0)

	// In text book mode the bottom bar uses stickyBottomBarFormat (or
	// the book text mode default). The status message, if any, is passed
	// through [[...]] fields in the format string. When the bars are
	// toggled off, skip the formatted bar and fall through to the normal
	// centered-message path.
	if sb.editor.bookTextMode() && sb.editor.stickyStatusBars {
		e := sb.editor
		format := e.stickyBottomBarFormat
		if format == "" {
			_, format = e.defaultStickyBarFormats()
		}
		text := e.expandStatusBarFormat(format, sb.msg)
		e.drawBar(c, h, w, text, e.bookBarStyle())
		mut.Lock()
		sb.offsetY = offsetY
		mut.Unlock()
		return
	}

	if sb.IsError() {
		mut.RLock()
		c.Write(uint(msgX), h, sb.errfg, sb.errbg, sb.msg)
		mut.RUnlock()
	} else {
		mut.RLock()
		c.Write(uint(msgX), h, sb.fg, sb.bg, sb.msg)
		mut.RUnlock()
	}

	if sb.nanoMode {
		mut.RLock()
		// x-align
		helpX := max((w-len(nanoHelpString1))/2, 0)
		x := uint(helpX)
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
	msg = asciiFallback(msg)

	if len(msg)%2 == 0 {
		sb.msg = "     "
	} else {
		sb.msg = "    "
	}
	sb.msg += msg + "    "

	sb.isError = false
	mut.Unlock()
	statusMsgGen.Add(1)
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
	msg = asciiFallback(msg)

	if len(msg)%2 == 0 {
		sb.msg = fiveSpaces
	} else {
		sb.msg = fourSpaces
	}
	sb.msg += msg + fourSpaces

	sb.isError = true
	mut.Unlock()
	statusMsgGen.Add(1)
}

// SetError is for setting the error message
func (sb *StatusBar) SetError(err error) {
	sb.SetErrorMessage(err.Error())
}

// Clear will set the message to nothing and then use the editor contents
// to remove the status bar field at the bottom of the editor.
func (sb *StatusBar) Clear(c *vt.Canvas, repositionCursorAfterDrawing bool) {
	mut.Lock()
	defer mut.Unlock()

	// Clear the message
	sb.msg = ""
	// Not an error message
	sb.isError = false

	if c == nil {
		return
	}

	// In book mode with graphics the image renderer owns the terminal.
	// Just clear the state; bookModeRenderAll at the end of the key loop
	// will render the updated status bar as part of the full frame.
	if sb.editor.bookGraphicalMode() {
		bookSetTemporaryStatusMsg("")
		return
	}

	// In text book mode the end-of-keyloop always fully redraws the canvas.
	// Skipping the intermediate draw avoids a flash of stale content.
	if sb.editor.bookTextMode() {
		return
	}

	// Then clear/redraw the bottom line
	h := int(c.H())
	if sb.nanoMode {
		h -= 2
	}
	barRows := sb.editor.stickyBarRows()
	h -= barRows
	offsetY := sb.editor.pos.OffsetY()
	sb.editor.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, uint(sb.editor.stickyTopBarHeight()), false, true)

	c.HideCursorAndDraw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		sb.editor.EnableAndPlaceCursor(c)
	}
}

// ClearAll will clear all status messages
func (sb *StatusBar) ClearAll(c *vt.Canvas, repositionCursorAfterDrawing bool) {
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

	// In book mode with graphics the image renderer owns the terminal.
	// Just clear the state; bookModeRenderAll at the end of the key loop
	// will render the updated status bar as part of the full frame.
	if sb.editor.bookGraphicalMode() {
		bookSetTemporaryStatusMsg("")
		return
	}

	// In text book mode the end-of-keyloop always fully redraws the canvas.
	// Skipping the intermediate draw avoids a flash of stale content.
	if sb.editor.bookTextMode() {
		return
	}

	// Then clear/redraw the bottom line
	h := int(c.H())
	if sb.nanoMode {
		h -= 2
	}
	barRows := sb.editor.stickyBarRows()
	h -= barRows
	offsetY := sb.editor.pos.OffsetY()
	sb.editor.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, uint(sb.editor.stickyTopBarHeight()), false, true)

	c.HideCursorAndDraw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		sb.editor.EnableAndPlaceCursor(c)
	}
}

// Show will draw a status message, then clear it after a certain delay
func (sb *StatusBar) Show(c *vt.Canvas, e *Editor) {
	if c == nil {
		return
	}

	// In book mode with graphics, show the message via the graphical status bar.
	if sb.editor.bookGraphicalMode() {
		mut.RLock()
		msg := sb.msg
		dur := sb.show
		isErr := sb.isError
		mut.RUnlock()
		if msg != "" {
			if isErr {
				dur *= 3
			}
			bookSetTemporaryStatusMsg(msg)
			redrawMutex.Lock()
			sb.editor.bookModeFullFrame(c)
			redrawMutex.Unlock()
			// Schedule a single auto-clear after dur. The generation
			// counter ensures that rapid repeated Show() calls don't
			// pile up stale goroutines all re-acquiring redrawMutex and
			// issuing full-frame re-renders — only the freshest one
			// actually fires and clears.
			myGen := bookStatusClearGen.Add(1)
			go func() {
				time.Sleep(dur)
				if bookStatusClearGen.Load() != myGen {
					return
				}
				bookSetTemporaryStatusMsg("")
				mut.Lock()
				sb.msg = ""
				sb.isError = false
				mut.Unlock()
				redrawMutex.Lock()
				sb.editor.bookModeFullFrame(c)
				redrawMutex.Unlock()
			}()
		}
		return
	}

	// When the sticky bars show status messages via [[...]], draw them
	// directly with the current message and schedule an auto-clear so
	// the bar reverts to showing the field value after the normal timeout.
	if e.stickyBarsHandleStatus() {
		msg := sb.Message()
		e.drawBothBars(c, msg)
		c.Draw()
		stickyBarsJustDrawn.Store(true)
		mut.RLock()
		sleepDuration := sb.show
		isErr := sb.isError
		mut.RUnlock()
		if isErr {
			sleepDuration *= 3
		}
		myGen := statusMsgGen.Load()
		go func() {
			time.Sleep(sleepDuration)
			if statusMsgGen.Load() != myGen {
				return
			}
			mut.Lock()
			sb.msg = ""
			sb.isError = false
			mut.Unlock()
			e.redraw.Store(true)
		}()
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

	myGen := statusMsgGen.Load()
	go func() {
		mut.RLock()
		sleepDuration := sb.show
		mut.RUnlock()

		if sb.IsError() {
			// Show error messages for 3x as long
			sleepDuration *= 3
		}
		time.Sleep(sleepDuration)

		// Don't clear if a newer message has been set while sleeping
		if statusMsgGen.Load() != myGen {
			return
		}

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
func (sb *StatusBar) ShowNoTimeout(c *vt.Canvas, e *Editor) {
	if c == nil {
		return
	}

	// In book mode (graphical or text), the "no timeout" semantic of the
	// regular terminal path doesn't really hold: the full-frame renderer
	// owns the bottom row, so a stale message like "EOF" would otherwise
	// linger on the page until something else happens to redraw. Route
	// these calls through the same auto-clearing code path that Show uses
	// so transient status messages disappear after sb.show, matching the
	// user-visible behaviour of the regular (non-book) mode.
	if sb.editor.bookGraphicalMode() || sb.editor.bookTextMode() {
		sb.Show(c, e)
		return
	}

	// When the sticky bars show status messages via [[...]], draw them
	// directly so there is no window for goroutines to clear sb.msg
	// before the redraw loop picks it up.
	if e.stickyBarsHandleStatus() {
		msg := sb.Message()
		e.drawBothBars(c, msg)
		c.Draw()
		stickyBarsJustDrawn.Store(true)
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
		p := int(100.0 * (float64(lineNumber) / float64(lastLineNumber)))
		if p > 100 {
			return 100
		}
		return p
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

// defaultStickyBarFormats returns the default top and bottom sticky bar
// format strings for the current editor mode.
func (e *Editor) defaultStickyBarFormats() (top, bottom string) {
	switch {
	case e.bookTextMode():
		return "<->{{filename}}<->{{word_count}} words   {{est_reading_time}}",
			"{{funcname}}<->[[line {{linenr}} of {{total_lines}}]]<->{{book_percentage}}"
	case e.bookGraphicalMode():
		return "", // the graphical book mode has no top bar
			"Line {{linenr:*}} of {{total_lines}}  Col {{col:*}}<->[[{{filename}}]]<->{{word_count:*}} words{|}{{est_reading_time}}{|}{{book_percentage:4}}"
	default: // regular mode
		return "<-><->{{funcname}}",
			"{{filename}}<->[[line {{linenr}} of {{total_lines}}]]<->{{mode}} [{{indentation}}]"
	}
}

// expandStatusBarFormat replaces template placeholders in a format string
// with the corresponding editor values. Supported placeholders:
//
//	{{filename}}          - basename of the current file
//	{{mode}}              - file mode / language (e.g. "Go", "Markdown")
//	{{linenr}}            - current line number (1-based)
//	{{colnr}}             - current column number
//	{{col}}               - alias for {{colnr}}
//	{{total_lines}}       - total number of lines in the document
//	{{scroll_percentage}} - vertical scroll position as "NN%"
//	{{book_percentage}}   - reading progress as "NN%" (0% at top, 100% at bottom)
//	{{indentation}}       - "tabs" or "spaces"
//	{{funcname}}          - current function name or heading
//	{{word_count}}        - total word count of the document
//	{{est_reading_time}}  - estimated reading time (e.g. "~3 min")
//
// Fields support an optional width specifier: {{field:width}}. A positive
// width right-aligns (pads with leading spaces), a negative width
// left-aligns (pads with trailing spaces). For example {{linenr:5}} pads
// the line number to 5 characters, right-aligned. The special width "*"
// auto-sizes: {{linenr:*}} pads to the digit count of total_lines,
// {{word_count:*}} pads to its own digit count, and so on. In
// proportional-font mode, padding uses U+2007 FIGURE SPACE for stable
// column widths.
//
// Using [[...]] instead of {{...}} shows the current status message
// when one is active, falling back to the enclosed content otherwise.
// The content inside [[...]] may itself contain {{field}} placeholders.
//
// The token {|} acts as a column separator within a segment. In terminal
// mode it becomes spaces. In graphical (proportional font) mode the
// pixel-based renderer positions each column independently so that
// variable-width digits in one column do not shift adjacent columns.
//
// The special token <-> separates the result into left, center and right
// segments (1 separator = left | right, 2 = left | center | right).
//
// When proportional is true, width padding uses U+2007 FIGURE SPACE instead
// of regular spaces so that columns stay aligned in proportional fonts.
func (e *Editor) expandStatusBarFormat(format, statusMsg string, proportional ...bool) string {
	useFigureSpace := len(proportional) > 0 && proportional[0]
	// Resolve {{funcname}}
	funcName := ""
	if ProgrammingLanguage(e.mode) || e.mode == mode.GoAssembly || e.mode == mode.Assembly {
		funcName = e.FindCurrentFunctionName()
	} else {
		funcName = e.bookCurrentHeading(e.DataY())
	}

	percentage, lineNumber, lastLineNumber := e.PLA()
	bookPct := bookReadingPercent(lineNumber, lastLineNumber)
	colNr := fmt.Sprintf("%d", e.ColNumber())
	words := e.WordCount()
	readingTime := ""
	if minutes := words / 200; minutes > 0 {
		readingTime = fmt.Sprintf("~%d min", minutes)
	}
	fields := map[string]string{
		"filename":          e.filename,
		"mode":              e.mode.String(),
		"linenr":            fmt.Sprintf("%d", lineNumber),
		"colnr":             colNr,
		"col":               colNr,
		"total_lines":       fmt.Sprintf("%d", lastLineNumber),
		"scroll_percentage": fmt.Sprintf("%d%%", percentage),
		"book_percentage":   fmt.Sprintf("%d%%", bookPct),
		"indentation":       e.IndentationDescription(),
		"funcname":          funcName,
		"word_count":        fmt.Sprintf("%d", words),
		"est_reading_time":  readingTime,
	}

	// Auto-width map: for {{field:*}}, look up how wide the field should
	// be based on a related maximum value. Fields not listed here use
	// their own string length.
	totalDigits := len(fields["total_lines"])
	autoWidth := map[string]int{
		"linenr":      totalDigits,
		"total_lines": totalDigits,
		"col":         totalDigits,
		"colnr":       totalDigits,
	}

	// Replace {{field}} and {{field:width}} with the corresponding value.
	// A width specifier pads the value: positive = right-align, negative =
	// left-align (matching fmt.Sprintf behavior). The width "*" auto-sizes.
	for name, value := range fields {
		// Plain {{field}} replacement (most common path).
		format = strings.ReplaceAll(format, "{{"+name+"}}", value)

		// {{field:width}} replacement — scan for the prefix and parse the
		// width from the characters between ":" and "}}".
		prefix := "{{" + name + ":"
		for {
			idx := strings.Index(format, prefix)
			if idx < 0 {
				break
			}
			rest := format[idx+len(prefix):]
			before, _, ok := strings.Cut(rest, "}}")
			if !ok {
				break
			}
			widthStr := before
			var width int
			if widthStr == "*" {
				if w, ok := autoWidth[name]; ok {
					width = w
				} else {
					width = len(value)
				}
			} else {
				var err error
				width, err = strconv.Atoi(widthStr)
				if err != nil {
					break
				}
			}
			padded := fmt.Sprintf("%*s", width, value)
			if useFigureSpace {
				padded = strings.ReplaceAll(padded, " ", "\u2007")
			}
			tag := prefix + widthStr + "}}"
			format = format[:idx] + padded + format[idx+len(tag):]
		}
	}

	// Replace [[...]] blocks: if a status message is active, the entire
	// block is replaced with the status message. Otherwise the content
	// between the brackets is kept as-is (its {{}} placeholders have
	// already been expanded above).
	msg := strings.TrimSpace(statusMsg)
	searchFrom := 0
	for {
		start := strings.Index(format[searchFrom:], "[[")
		if start < 0 {
			break
		}
		start += searchFrom
		end := strings.Index(format[start:], "]]")
		if end < 0 {
			break
		}
		end += start + len("]]")
		inner := format[start+len("[[") : end-len("]]")]
		var replacement string
		if msg != "" {
			replacement = msg
		} else {
			replacement = inner
		}
		format = format[:start] + replacement + format[end:]
		searchFrom = start + len(replacement)
	}

	// In terminal mode, column separators become regular spaces. In
	// graphical (proportional font) mode they are left intact so the
	// pixel-based renderer can position each column independently.
	if !useFigureSpace {
		format = strings.ReplaceAll(format, "{|}", "   ")
	}

	return format
}

// stickyTopBarHeight returns 1 when a sticky top bar is active in regular
// (non-book, non-nano) mode, 0 otherwise.
func (e *Editor) stickyTopBarHeight() uint {
	if e.stickyStatusBars && !e.bookMode.Load() && !e.nanoMode.Load() {
		return 1
	}
	return 0
}

// stickyBottomBarHeight returns 1 when a sticky bottom bar is active in
// regular (non-book, non-nano) mode, 0 otherwise.
func (e *Editor) stickyBottomBarHeight() uint {
	if e.stickyStatusBars && !e.bookMode.Load() && !e.nanoMode.Load() {
		return 1
	}
	return 0
}

// stickyBarRows returns the total number of rows reserved by sticky bars
// (top + bottom).
func (e *Editor) stickyBarRows() int {
	return int(e.stickyTopBarHeight() + e.stickyBottomBarHeight())
}

// stickyBarsHandleStatus returns true if either sticky bar format contains
// a [[field]] placeholder, meaning status messages are shown inline in the
// bars and should not be drawn separately at the bottom.
func (e *Editor) stickyBarsHandleStatus() bool {
	if !e.stickyStatusBars {
		return false
	}
	defaultTop, defaultBottom := e.defaultStickyBarFormats()
	top := e.stickyTopBarFormat
	if top == "" {
		top = defaultTop
	}
	bottom := e.stickyBottomBarFormat
	if bottom == "" {
		bottom = defaultBottom
	}
	return strings.Contains(top, "[[") || strings.Contains(bottom, "[[")
}

// barStyle controls the appearance of a terminal status bar drawn by drawBar.
type barStyle struct {
	fg      vt.AttributeColor // foreground for center text and single-segment bars
	dimFg   vt.AttributeColor // foreground for side slots (left/right); if zero, fg is used
	bg      vt.AttributeColor // background color
	clearBg vt.AttributeColor // background used to clear the row; if zero, bg is used
	clearFg vt.AttributeColor // foreground used to clear the row; if zero, fg is used
	pad     int               // horizontal padding in columns (typically 1 or 2)
}

// drawBar paints a single-row status bar at row y on canvas c. The
// expanded text is split on <-> into up to three segments (left, center,
// right). Side slots are capped at roughly a third of the bar width and
// drawn in dimFg; the center slot uses fg and is truncated with an
// ellipsis when it would overlap a side slot.
func (e *Editor) drawBar(c *vt.Canvas, y uint, w int, text string, s barStyle) {
	if w <= 0 {
		return
	}

	clearFg := s.clearFg
	if clearFg == 0 {
		clearFg = s.fg
	}
	clearBg := s.clearBg
	if clearBg == 0 {
		clearBg = s.bg
	}
	c.Write(0, y, clearFg, clearBg, strings.Repeat(" ", w))

	dimFg := s.dimFg
	if dimFg == 0 {
		dimFg = s.fg
	}
	pad := s.pad
	if pad <= 0 {
		pad = 1
	}

	ellipsis := "…"
	ellipsisW := 1
	if useASCII {
		ellipsis = "..."
		ellipsisW = 3
	}
	truncate := func(str string, maxW int) string {
		sw := runewidth.StringWidth(str)
		if maxW <= 0 || (sw > maxW && maxW <= ellipsisW) {
			return ""
		}
		if sw <= maxW {
			return str
		}
		// Truncate rune-by-rune to fit within maxW - ellipsisW columns.
		target := maxW - ellipsisW
		w := 0
		for i, r := range str {
			rw := runewidth.RuneWidth(r)
			if w+rw > target {
				return str[:i] + ellipsis
			}
			w += rw
		}
		return str + ellipsis
	}

	// Split on <-> to get left / center / right segments.
	parts := strings.Split(text, "<->")

	var left, center, right string
	switch len(parts) {
	case 1:
		center = strings.TrimSpace(parts[0])
	case 2:
		left = strings.TrimSpace(parts[0])
		right = strings.TrimSpace(parts[1])
	default:
		left = strings.TrimSpace(parts[0])
		center = strings.TrimSpace(parts[1])
		right = strings.TrimSpace(parts[2])
	}

	sideMax := w/3 - pad
	left = truncate(left, sideMax)
	right = truncate(right, sideMax)

	leftLen := runewidth.StringWidth(left)
	rightLen := runewidth.StringWidth(right)
	rightStart := w - pad - rightLen

	if leftLen > 0 {
		c.Write(uint(pad), y, dimFg, s.bg, left)
	}
	if rightLen > 0 {
		c.Write(uint(rightStart), y, dimFg, s.bg, right)
	}

	if center == "" {
		return
	}
	leftBound := pad
	if leftLen > 0 {
		leftBound = pad + leftLen + 1
	}
	rightBound := w
	if rightLen > 0 {
		rightBound = rightStart - 1
	}
	center = truncate(center, rightBound-leftBound)
	if center == "" {
		return
	}
	centerLen := runewidth.StringWidth(center)
	cx := max((w-centerLen)/2, leftBound)
	if cx+centerLen > rightBound {
		cx = max(rightBound-centerLen, leftBound)
	}
	c.Write(uint(cx), y, s.fg, s.bg, center)
}

// stickyBarStyle returns the barStyle used by the regular-mode sticky bars.
func (e *Editor) stickyBarStyle() barStyle {
	return barStyle{
		fg:      e.TopRightForeground,
		bg:      e.TopRightBackground,
		clearFg: e.Foreground,
		clearBg: e.Background,
		pad:     1,
	}
}

// drawStickyTopBar paints the sticky status bar at row 0 using the current
// stickyTopBarFormat and the regular-mode color scheme.
func (e *Editor) drawStickyTopBar(c *vt.Canvas, statusMsg string) {
	w := int(c.W())
	format := e.stickyTopBarFormat
	if format == "" {
		format, _ = e.defaultStickyBarFormats()
	}
	text := e.expandStatusBarFormat(format, statusMsg)
	e.drawBar(c, 0, w, text, e.stickyBarStyle())
}

// drawStickyBottomBar paints the sticky bottom bar at the last row using
// the stickyBottomBarFormat and the regular-mode color scheme.
func (e *Editor) drawStickyBottomBar(c *vt.Canvas, statusMsg string) {
	w := int(c.W())
	format := e.stickyBottomBarFormat
	if format == "" {
		_, format = e.defaultStickyBarFormats()
	}
	text := e.expandStatusBarFormat(format, statusMsg)
	e.drawBar(c, c.H()-1, w, text, e.stickyBarStyle())
}

// drawBothBars draws the top and bottom sticky bars using the correct
// style for the current editor mode (regular or book text). The status
// message is passed through so [[...]] fields can display it.
func (e *Editor) drawBothBars(c *vt.Canvas, statusMsg string) {
	w := int(c.W())
	if w <= 0 {
		return
	}
	defaultTop, defaultBottom := e.defaultStickyBarFormats()
	topFmt := e.stickyTopBarFormat
	if topFmt == "" {
		topFmt = defaultTop
	}
	bottomFmt := e.stickyBottomBarFormat
	if bottomFmt == "" {
		bottomFmt = defaultBottom
	}

	style := e.stickyBarStyle()
	if e.bookTextMode() {
		style = e.bookBarStyle()
	}

	if topFmt != "" {
		text := e.expandStatusBarFormat(topFmt, statusMsg)
		e.drawBar(c, 0, w, text, style)
	}
	if bottomFmt != "" {
		text := e.expandStatusBarFormat(bottomFmt, statusMsg)
		e.drawBar(c, c.H()-1, w, text, style)
	}
}

// ShowBlockModeStatusLine shows a status message for when block mode is enabled
func (sb *StatusBar) ShowBlockModeStatusLine(c *vt.Canvas, e *Editor) {
	sb.SetMessage("Block Edit")
	sb.ShowNoTimeout(c, e)
}

// NanoInfo shows info about the current position, for the Nano emulation mode
func (sb *StatusBar) NanoInfo(c *vt.Canvas, e *Editor) {
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
func (sb *StatusBar) HoldMessage(c *vt.Canvas, dur time.Duration) {
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
	sb.messageAfterRedraw = asciiFallback(message)
}

// SetErrorAfterRedraw prepares a status bar message that will be shown after redraw
func (sb *StatusBar) SetErrorAfterRedraw(err error) {
	sb.messageAfterRedraw = asciiFallback(err.Error())
}

// SetErrorMessageAfterRedraw prepares a status bar message that will be shown after redraw
func (sb *StatusBar) SetErrorMessageAfterRedraw(errorMessage string) {
	sb.messageAfterRedraw = asciiFallback(errorMessage)
}
