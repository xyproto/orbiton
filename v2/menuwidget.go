package main

import (
	"unicode"

	"github.com/xyproto/vt"
)

// MenuWidget represents a TUI widget for presenting a menu with choices for the user
type MenuWidget struct {
	selectionLetterMap map[string]*RuneAndPosition // used for knowing which accelerator letter of each choice should be drawn in a different color (not all choices may have a suitable letter)
	title              string                      // title
	choices            []string                    // a slice of menu items
	marginLeft         int                         // margin, may be negative?
	marginTop          int                         // margin, may be negative?
	titleHeight        int                         // rows reserved for title block (1 title + 1 blank); 0 when skipped due to overflow
	selected           int                         // the index of the currently selected item
	w                  uint                        // width
	h                  uint                        // height (number of menu items)
	y                  uint                        // current position
	oldy               uint                        // previous position
	titleColor         vt.AttributeColor           // title color (above the choices)
	arrowColor         vt.AttributeColor           // arrow color (before each menu choice)
	textColor          vt.AttributeColor           // text color (the choices that are not highlighted)
	highlightColor     vt.AttributeColor           // highlight color (the choice that will be selected if return is pressed)
	selectedColor      vt.AttributeColor           // selected color (the choice that has been selected after return has been pressed)
	extraDashes        bool                        // add "---" after each menu item?
}

// NewMenuWidget creates a new MenuWidget
func NewMenuWidget(title string, choices []string, titleColor, arrowColor, textColor, highlightColor, selectedColor vt.AttributeColor, canvasWidth, canvasHeight uint, extraDashes bool, selectionLetterMap map[string]*RuneAndPosition) *MenuWidget {
	maxlen := uint(0)
	for _, choice := range choices {
		if ulen(choice) > uint(maxlen) {
			maxlen = ulen(choice)
		}
	}
	marginLeft := 10
	if int(canvasWidth)-(int(maxlen)+marginLeft) <= 0 {
		marginLeft = 0
	}
	marginTop := 8
	titleHeight := 2 // 1 row for the title + 1 blank row below it
	// If the full layout overflows the canvas, drop the top margin first;
	// if it still overflows, drop the title block (title + blank row) as well.
	if len(choices)+titleHeight+marginTop > int(canvasHeight) {
		marginTop = 0
		if len(choices)+titleHeight > int(canvasHeight) {
			titleHeight = 0
		}
	}
	return &MenuWidget{
		title:              title,
		w:                  uint(marginLeft + int(maxlen)),
		h:                  ulen(choices),
		y:                  0,
		oldy:               0,
		marginLeft:         marginLeft,
		marginTop:          marginTop,
		titleHeight:        titleHeight,
		choices:            choices,
		selected:           -1,
		extraDashes:        extraDashes,
		titleColor:         titleColor,
		arrowColor:         arrowColor,
		textColor:          textColor,
		highlightColor:     highlightColor,
		selectedColor:      selectedColor,
		selectionLetterMap: selectionLetterMap,
	}
}

// Resize recalculates the layout margins for a new canvas size, preserving the current selection.
func (m *MenuWidget) Resize(canvasWidth, canvasHeight uint) {
	maxlen := uint(0)
	for _, choice := range m.choices {
		if ulen(choice) > maxlen {
			maxlen = ulen(choice)
		}
	}
	marginLeft := 10
	if int(canvasWidth)-(int(maxlen)+marginLeft) <= 0 {
		marginLeft = 0
	}
	marginTop := 8
	titleHeight := 2
	if len(m.choices)+titleHeight+marginTop > int(canvasHeight) {
		marginTop = 0
		if len(m.choices)+titleHeight > int(canvasHeight) {
			titleHeight = 0
		}
	}
	m.marginLeft = marginLeft
	m.w = uint(marginLeft + int(maxlen))
	m.marginTop = marginTop
	m.titleHeight = titleHeight
}

// Selected returns the currently selected item
func (m *MenuWidget) Selected() int {
	return m.selected
}

// Draw will draw this menu widget on the given canvas
func (m *MenuWidget) Draw(c *vt.Canvas) {
	titleHeight := m.titleHeight
	if titleHeight > 0 {
		for x, r := range m.title {
			c.PlotColor(uint(m.marginLeft+x), uint(m.marginTop), m.titleColor, r)
		}
	}
	// Draw the menu entries, with various colors
	ulenChoices := ulen(m.choices)
	for y := uint(0); y < m.h; y++ {
		var itemString string
		var selectionLetter rune
		if y < ulenChoices {
			for choiceString, runeAndPosition := range m.selectionLetterMap {
				if m.choices[y] == choiceString && y == runeAndPosition.pos {
					selectionLetter = runeAndPosition.r
				}
			}
			prefix := "   "
			if y == m.y {
				prefix = "-> "
			}
			itemString = prefix + m.choices[y] + " "
			if m.extraDashes {
				itemString += "---"
			}
		}
		highlightedAccelerator := false
		afterLeftBracket := false
		beforeRightBracket := true
		for x := uint(0); x < m.w; x++ {
			r := '-'
			if x < ulen([]rune(itemString)) {
				r = []rune(itemString)[x]
			} else if !m.extraDashes {
				break
			}
			if r == ']' {
				beforeRightBracket = false
			}
			if x < 2 {
				c.PlotColor(uint(m.marginLeft+int(x)), uint(m.marginTop+int(y)+titleHeight), m.arrowColor, r)
			} else if y < 10 && afterLeftBracket && beforeRightBracket {
				// color the 0-9 number differrently (in the title color)
				c.PlotColor(uint(m.marginLeft+int(x)), uint(m.marginTop+int(y)+titleHeight), m.titleColor, r)
			} else if y == m.y {
				c.PlotColor(uint(m.marginLeft+int(x)), uint(m.marginTop+int(y)+titleHeight), m.highlightColor, r)
			} else if !highlightedAccelerator && unicode.ToLower(r) == selectionLetter {
				// color the accelerator letter differently (in the arrow color)
				c.PlotColor(uint(m.marginLeft+int(x)), uint(m.marginTop+int(y)+titleHeight), m.arrowColor, r)
				highlightedAccelerator = true
			} else {
				c.PlotColor(uint(m.marginLeft+int(x)), uint(m.marginTop+int(y)+titleHeight), m.textColor, r)
			}
			if r == '[' {
				afterLeftBracket = true
			}
		}
	}
}

// SelectDraw will draw the currently highlighted menu choices with the selected color.
// This is used after a menu item has been selected.
func (m *MenuWidget) SelectDraw(c *vt.Canvas) {
	old := m.highlightColor
	m.highlightColor = m.selectedColor
	m.Draw(c)
	m.highlightColor = old
}

// Select will select the currently highlighted menu option
func (m *MenuWidget) Select() {
	m.selected = int(m.y)
}

// Up will move the highlight up (with wrap-around)
func (m *MenuWidget) Up() {
	m.oldy = m.y
	if m.y <= 0 {
		m.y = m.h - 1
	} else {
		m.y--
	}
}

// Down will move the highlight down (with wrap-around)
func (m *MenuWidget) Down() {
	m.oldy = m.y
	m.y++
	if m.y >= m.h {
		m.y = 0
	}
}

// SelectIndex will select a specific index. Returns false if it was not possible.
func (m *MenuWidget) SelectIndex(n uint) bool {
	if n >= m.h {
		return false
	}
	m.oldy = m.y
	m.y = n
	return true
}

// SelectFirst will select the first menu choice
func (m *MenuWidget) SelectFirst() bool {
	return m.SelectIndex(0)
}

// SelectLast will select the last menu choice
func (m *MenuWidget) SelectLast() bool {
	m.oldy = m.y
	m.y = m.h - 1
	return true
}
