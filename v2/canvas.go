package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type ColorRune struct {
	fg    AttributeColor // Foreground color
	bg    AttributeColor // Background color
	r     rune           // The character to draw
	drawn bool           // Has been drawn to screen yet?
}

// for API stability
type Char ColorRune

type Canvas struct {
	mut           *sync.RWMutex
	chars         []ColorRune
	oldchars      []ColorRune
	w             uint
	h             uint
	cursorVisible bool
	lineWrap      bool
	runewise      bool
}

// canvasCopy is a Canvas without the mutex
type canvasCopy struct {
	chars         []ColorRune
	oldchars      []ColorRune
	w             uint
	h             uint
	cursorVisible bool
	lineWrap      bool
	runewise      bool
}

func NewCanvas() *Canvas {
	c := &Canvas{}
	c.w, c.h = MustTermSize()
	c.chars = make([]ColorRune, c.w*c.h)
	for i := 0; i < len(c.chars); i++ {
		c.chars[i].fg = Default
		c.chars[i].bg = DefaultBackground
	}
	c.oldchars = make([]ColorRune, 0)
	c.mut = &sync.RWMutex{}
	c.cursorVisible = false
	c.lineWrap = false
	c.SetShowCursor(c.cursorVisible)
	c.SetLineWrap(c.lineWrap)
	return c
}

// Copy creates a new Canvas struct that is a copy of this one.
// The mutex is initialized as a new mutex.
func (c *Canvas) Copy() Canvas {
	c.mut.RLock()
	defer c.mut.RUnlock()

	cc := canvasCopy{
		chars:         make([]ColorRune, len(c.chars)),
		oldchars:      make([]ColorRune, len(c.oldchars)),
		w:             c.w,
		h:             c.h,
		cursorVisible: c.cursorVisible,
		lineWrap:      c.lineWrap,
		runewise:      c.runewise,
	}
	copy(cc.chars, c.chars)
	copy(cc.oldchars, c.oldchars)

	return Canvas{
		chars:         cc.chars,
		oldchars:      cc.oldchars,
		w:             cc.w,
		h:             cc.h,
		cursorVisible: cc.cursorVisible,
		lineWrap:      cc.lineWrap,
		runewise:      cc.runewise,
		mut:           &sync.RWMutex{},
	}
}

// Change the background color for each character
func (c *Canvas) FillBackground(bg AttributeColor) {
	converted := bg.Background()
	c.mut.Lock()
	for i := range c.chars {
		c.chars[i].bg = converted
		c.chars[i].drawn = false
	}
	c.mut.Unlock()
}

// Change the foreground color for each character
func (c *Canvas) Fill(fg AttributeColor) {
	c.mut.Lock()
	for i := range c.chars {
		c.chars[i].fg = fg
	}
	c.mut.Unlock()
}

// String returns only the characters, as a long string with a newline after each row
func (c *Canvas) String() string {
	var sb strings.Builder
	c.mut.RLock()
	for y := uint(0); y < c.h; y++ {
		for x := uint(0); x < c.w; x++ {
			cr := &((*c).chars[y*c.w+x])
			if cr.r == rune(0) {
				sb.WriteRune(' ')
			} else {
				sb.WriteRune(cr.r)
			}
		}
		sb.WriteRune('\n')
	}
	c.mut.RUnlock()
	return sb.String()
}

// PlotAll tries to plot each individual rune.
// It's very inefficient and meant to be used as a robust fallback.
func (c *Canvas) PlotAll() {
	w := c.w
	h := c.h
	c.mut.Lock()
	for y := uint(0); y < h; y++ {
		for x := int(w - 1); x >= 0; x-- {
			cr := &((*c).chars[y*w+uint(x)])
			r := cr.r
			if cr.r == rune(0) {
				r = ' '
				//continue
			}
			SetXY(uint(x), y)
			fmt.Print(cr.fg.Combine(cr.bg).String() + string(r) + NoColor())
		}
	}
	c.mut.Unlock()
}

// Return the size of the current canvas
func (c *Canvas) Size() (uint, uint) {
	return c.w, c.h
}

func (c *Canvas) Width() uint {
	return c.w
}

func (c *Canvas) Height() uint {
	return c.h
}

// Move cursor to the given position (0,0 is top left)
func SetXY(x, y uint) {
	// Add 1 to y to make the position correct
	Set("Cursor Home", map[string]string{"{ROW}": strconv.Itoa(int(y + 1)), "{COLUMN}": strconv.Itoa(int(x + 1))})
}

// Move the cursor down
func Down(n uint) {
	Set("Cursor Down", map[string]string{"{COUNT}": strconv.Itoa(int(n))})
}

// Move the cursor up
func Up(n uint) {
	Set("Cursor Up", map[string]string{"{COUNT}": strconv.Itoa(int(n))})
}

// Move the cursor to the right
func Right(n uint) {
	Set("Cursor Forward", map[string]string{"{COUNT}": strconv.Itoa(int(n))})
}

// Move the cursor to the left
func Left(n uint) {
	Set("Cursor Backward", map[string]string{"{COUNT}": strconv.Itoa(int(n))})
}

func Home() {
	Set("Cursor Home", map[string]string{"{ROW};{COLUMN}": ""})
}

func Reset() {
	Do("Reset Device")
}

// Clear screen
func Clear() {
	Do("Erase Screen")
}

// Clear canvas
func (c *Canvas) Clear() {
	c.mut.Lock()
	defer c.mut.Unlock()
	for _, cr := range c.chars {
		cr.r = rune(0)
		cr.drawn = false
	}
}

func (c *Canvas) SetLineWrap(enable bool) {
	c.mut.Lock()
	defer c.mut.Unlock()
	SetLineWrap(enable)
}

func (c *Canvas) SetShowCursor(enable bool) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.cursorVisible = enable
	ShowCursor(enable)
}

func (c *Canvas) W() uint {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.w
}

func (c *Canvas) H() uint {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.h
}

func (c *Canvas) HideCursor() {
	c.SetShowCursor(false)
}

func (c *Canvas) ShowCursor() {
	c.SetShowCursor(true)
}

func (c *Canvas) SetRunewise(b bool) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.runewise = b
}

// DrawAndSetCursor draws the entire canvas and then places the cursor at x,y
func (c *Canvas) DrawAndSetCursor(x, y uint) {
	c.Draw()
	// Reposition the cursor
	SetXY(x, y)
}

// HideCursorAndDraw will hide the cursor and then draw the entire canvas
func (c *Canvas) HideCursorAndDraw() {

	c.cursorVisible = false
	c.SetShowCursor(false)

	var (
		lastfg = Default // AttributeColor
		lastbg = Default // AttributeColor
		cr     ColorRune
		oldcr  ColorRune
		sb     strings.Builder
	)

	cr.fg = Default
	cr.bg = Default
	oldcr.fg = Default
	oldcr.bg = Default

	// NOTE: If too many runes are written to the screen, the contents will scroll up,
	// and it will appear like the first line(s) are lost!

	c.mut.RLock()

	if len((*c).chars) == 0 {
		c.mut.RUnlock()
		return
	}

	firstRun := len(c.oldchars) == 0
	skipAll := !firstRun // true by default, except for the first run

	size := c.w*c.h - 1
	sb.Grow(int(size))

	if !firstRun {
		for index := uint(0); index < size; index++ {
			cr = (*c).chars[index]
			oldcr = (*c).oldchars[index]
			if cr.fg.Equal(lastfg) && cr.fg.Equal(oldcr.fg) && cr.bg.Equal(lastbg) && cr.bg.Equal(oldcr.bg) && cr.r == oldcr.r {
				// One is not skippable, can not skip all
				skipAll = false
			}
			// Only output a color code if it's different from the last character, or it's the first one
			if (index == 0) || !lastfg.Equal(cr.fg) || !lastbg.Equal(cr.bg) {
				// Write to the string builder
				sb.WriteString(cr.fg.Combine(cr.bg).String())
			}
			// Write the character
			if cr.r != 0 {
				sb.WriteRune(cr.r)
			} else {
				sb.WriteRune(' ')
			}
			lastfg = cr.fg
			lastbg = cr.bg
		}
	} else {
		for index := uint(0); index < size; index++ {
			cr = (*c).chars[index]
			// Only output a color code if it's different from the last character, or it's the first one
			if (index == 0) || !lastfg.Equal(cr.fg) || !lastbg.Equal(cr.bg) {
				// Write to the string builder
				sb.WriteString(cr.fg.Combine(cr.bg).String())
			}
			// Write the character
			if cr.r != 0 {
				sb.WriteRune(cr.r)
			} else {
				sb.WriteRune(' ')
			}
			lastfg = cr.fg
			lastbg = cr.bg
		}
	}

	c.mut.RUnlock()

	// The screenfull so far is correct (sb.String())

	if skipAll {
		return
	}

	// Enable line wrap, temporarily, if it's disabled
	reDisableLineWrap := false
	if !c.lineWrap {
		c.SetLineWrap(true)
		reDisableLineWrap = true
	}

	// Draw each and every line, or push one large string to screen?
	if c.runewise {

		Clear()
		c.PlotAll()

	} else {
		c.mut.Lock()
		SetXY(0, 0)
		os.Stdout.Write([]byte(sb.String()))
		c.mut.Unlock()
	}

	// Restore the line wrap, if it was temporarily enabled
	if reDisableLineWrap {
		c.SetLineWrap(false)
	}

	// Save the current state to oldchars
	c.mut.Lock()
	if lc := len(c.chars); len(c.oldchars) != lc {
		c.oldchars = make([]ColorRune, lc)
	}
	copy(c.oldchars, c.chars)
	c.mut.Unlock()
}

// Draw the entire canvas
func (c *Canvas) Draw() {
	var (
		lastfg = Default // AttributeColor
		lastbg = Default // AttributeColor
		cr     ColorRune
		oldcr  ColorRune
		sb     strings.Builder
	)

	cr.fg = Default
	cr.bg = Default
	oldcr.fg = Default
	oldcr.bg = Default

	// NOTE: If too many runes are written to the screen, the contents will scroll up,
	// and it will appear like the first line(s) are lost!

	c.mut.RLock()

	if len((*c).chars) == 0 {
		c.mut.RUnlock()
		return
	}

	firstRun := len(c.oldchars) == 0
	skipAll := !firstRun // true by default, except for the first run

	size := c.w*c.h - 1
	sb.Grow(int(size))

	if !firstRun {
		for index := uint(0); index < size; index++ {
			cr = (*c).chars[index]
			oldcr = (*c).oldchars[index]
			if cr.fg.Equal(lastfg) && cr.fg.Equal(oldcr.fg) && cr.bg.Equal(lastbg) && cr.bg.Equal(oldcr.bg) && cr.r == oldcr.r {
				// One is not skippable, can not skip all
				skipAll = false
			}
			// Only output a color code if it's different from the last character, or it's the first one
			if (index == 0) || !lastfg.Equal(cr.fg) || !lastbg.Equal(cr.bg) {
				// Write to the string builder
				sb.WriteString(cr.fg.Combine(cr.bg).String())
			}
			// Write the character
			if cr.r != 0 {
				sb.WriteRune(cr.r)
			} else {
				sb.WriteRune(' ')
			}
			lastfg = cr.fg
			lastbg = cr.bg
		}
	} else {
		for index := uint(0); index < size; index++ {
			cr = (*c).chars[index]
			// Only output a color code if it's different from the last character, or it's the first one
			if (index == 0) || !lastfg.Equal(cr.fg) || !lastbg.Equal(cr.bg) {
				// Write to the string builder
				sb.WriteString(cr.fg.Combine(cr.bg).String())
			}
			// Write the character
			if cr.r != 0 {
				sb.WriteRune(cr.r)
			} else {
				sb.WriteRune(' ')
			}
			lastfg = cr.fg
			lastbg = cr.bg
		}
	}

	c.mut.RUnlock()

	// The screenfull so far is correct (sb.String())

	if skipAll {
		return
	}

	// Output the combined string, also disable the color codes

	// Hide the cursor, temporarily, if it's visible
	reEnableCursor := false
	if c.cursorVisible {
		c.SetShowCursor(false)
		reEnableCursor = true
	}

	// Enable line wrap, temporarily, if it's disabled
	reDisableLineWrap := false
	if !c.lineWrap {
		c.SetLineWrap(true)
		reDisableLineWrap = true
	}

	// Draw each and every line, or push one large string to screen?
	if c.runewise {

		Clear()
		c.PlotAll()

	} else {
		c.mut.Lock()
		SetXY(0, 0)
		os.Stdout.Write([]byte(sb.String()))
		c.mut.Unlock()
	}

	// Restore the cursor, if it was temporarily hidden
	if reEnableCursor {
		c.SetShowCursor(true)
	}

	// Restore the line wrap, if it was temporarily enabled
	if reDisableLineWrap {
		c.SetLineWrap(false)
	}

	// Save the current state to oldchars
	c.mut.Lock()
	c.oldchars = make([]ColorRune, len(c.chars))
	copy(c.oldchars, c.chars)
	c.mut.Unlock()
}

func (c *Canvas) Redraw() {
	c.mut.Lock()
	for _, cr := range c.chars {
		cr.drawn = false
	}
	c.mut.Unlock()
	c.Draw()
}

func (c *Canvas) HideCursorAndRedraw() {
	c.mut.Lock()
	for _, cr := range c.chars {
		cr.drawn = false
	}
	c.mut.Unlock()
	c.HideCursorAndDraw()
}

// At returns the rune at the given coordinates, or an error if out of bounds
func (c *Canvas) At(x, y uint) (rune, error) {
	c.mut.RLock()
	defer c.mut.RUnlock()
	chars := (*c).chars
	index := y*c.w + x
	if index < uint(0) || index >= uint(len(chars)) {
		return rune(0), errors.New("out of bounds")
	}
	return chars[index].r, nil
}

func (c *Canvas) Plot(x, y uint, r rune) {
	if x >= c.w || y >= c.h {
		return
	}
	index := y*c.w + x
	c.mut.Lock()
	chars := (*c).chars
	chars[index].r = r
	chars[index].drawn = false
	c.mut.Unlock()
}

func (c *Canvas) PlotColor(x, y uint, fg AttributeColor, r rune) {
	if x >= c.w || y >= c.h {
		return
	}
	index := y*c.w + x
	c.mut.Lock()
	chars := (*c).chars
	chars[index].r = r
	chars[index].fg = fg
	chars[index].drawn = false
	c.mut.Unlock()
}

// WriteString will write a string to the canvas.
func (c *Canvas) WriteString(x, y uint, fg, bg AttributeColor, s string) {
	if x >= c.w || y >= c.h {
		return
	}
	c.mut.RLock()
	chars := (*c).chars
	counter := uint(0)
	startpos := y*c.w + x
	lchars := uint(len(chars))
	c.mut.RUnlock()
	bgb := bg.Background()
	for _, r := range s {
		i := startpos + counter
		if i >= lchars {
			break
		}
		c.mut.Lock()
		chars[i].r = r
		chars[i].fg = fg
		chars[i].bg = bgb
		chars[i].drawn = false
		c.mut.Unlock()
		counter++
	}
}

func (c *Canvas) Write(x, y uint, fg, bg AttributeColor, s string) {
	c.WriteString(x, y, fg, bg, s)
}

// WriteRune will write a colored rune to the canvas
func (c *Canvas) WriteRune(x, y uint, fg, bg AttributeColor, r rune) {
	if x >= c.w || y >= c.h {
		return
	}
	index := y*c.w + x
	c.mut.Lock()
	defer c.mut.Unlock()
	chars := (*c).chars
	chars[index].r = r
	chars[index].fg = fg
	chars[index].bg = bg.Background()
	chars[index].drawn = false
}

// WriteRuneB will write a colored rune to the canvas
// The x and y must be within range (x < c.w and y < c.h)
func (c *Canvas) WriteRuneB(x, y uint, fg, bgb AttributeColor, r rune) {
	index := y*c.w + x
	c.mut.Lock()
	defer c.mut.Unlock()
	(*c).chars[index] = ColorRune{fg, bgb, r, false}
}

// WriteRuneBNoLock will write a colored rune to the canvas
// The x and y must be within range (x < c.w and y < c.h)
// The canvas mutex is not locked
func (c *Canvas) WriteRuneBNoLock(x, y uint, fg, bgb AttributeColor, r rune) {
	(*c).chars[y*c.w+x] = ColorRune{fg, bgb, r, false}
}

// WriteBackground will write a background color to the canvas
// The x and y must be within range (x < c.w and y < c.h)
func (c *Canvas) WriteBackground(x, y uint, bg AttributeColor) {
	index := y*c.w + x
	c.mut.Lock()
	defer c.mut.Unlock()
	(*c).chars[index].bg = bg
	(*c).chars[index].drawn = false
}

// WriteBackgroundAddRuneIfEmpty will write a background color to the canvas
// The x and y must be within range (x < c.w and y < c.h)
func (c *Canvas) WriteBackgroundAddRuneIfEmpty(x, y uint, bg AttributeColor, r rune) {
	index := y*c.w + x
	c.mut.Lock()
	defer c.mut.Unlock()
	(*c).chars[index].bg = bg
	if (*c).chars[index].r == 0 {
		(*c).chars[index].r = r
	}
	(*c).chars[index].drawn = false
}

// WriteBackgroundNoLock will write a background color to the canvas
// The x and y must be within range (x < c.w and y < c.h)
// The canvas mutex is not locked
func (c *Canvas) WriteBackgroundNoLock(x, y uint, bg AttributeColor) {
	index := y*c.w + x
	(*c).chars[index].bg = bg
	(*c).chars[index].drawn = false
}

func (c *Canvas) Lock() {
	c.mut.Lock()
}

func (c *Canvas) Unlock() {
	c.mut.Unlock()
}

// WriteRunesB will write repeated colored runes to the canvas.
// This is the same as WriteRuneB, but bg.Background() has already been called on
// the background attribute.
// The x and y must be within range (x < c.w and y < c.h). x + count must be within range too.
func (c *Canvas) WriteRunesB(x, y uint, fg, bgb AttributeColor, r rune, count uint) {
	startIndex := y*c.w + x
	afterLastIndex := startIndex + count
	c.mut.Lock()
	chars := (*c).chars
	for i := startIndex; i < afterLastIndex; i++ {
		chars[i] = ColorRune{fg, bgb, r, false}
	}
	c.mut.Unlock()
}

func (c *Canvas) Resize() {
	w, h := MustTermSize()
	c.mut.Lock()
	if (w != c.w) || (h != c.h) {
		// Resize to the new size
		c.w = w
		c.h = h
		c.chars = make([]ColorRune, w*h)
		c.mut = &sync.RWMutex{}
	}
	c.mut.Unlock()
}

// Check if the canvas was resized, and adjust values accordingly.
// Returns a new canvas, or nil.
func (c *Canvas) Resized() *Canvas {
	w, h := MustTermSize()
	if (w != c.w) || (h != c.h) {
		// The terminal was resized!
		oldc := c

		nc := &Canvas{}
		nc.w = w
		nc.h = h
		nc.chars = make([]ColorRune, w*h)
		nc.mut = &sync.RWMutex{}

		nc.mut.Lock()
		c.mut.Lock()
		defer c.mut.Unlock()
		defer nc.mut.Unlock()
	OUT:
		// Plot in the old characters
		for y := uint(0); y < umin(oldc.h, h); y++ {
			for x := uint(0); x < umin(oldc.w, w); x++ {
				oldIndex := y*oldc.w + x
				index := y*nc.w + x
				if oldIndex > index {
					break OUT
				}
				// Copy over old characters, and mark them as not drawn
				cr := oldc.chars[oldIndex]
				cr.drawn = false
				nc.chars[index] = cr
			}
		}
		// Return the new canvas
		return nc
	}
	return nil
}
