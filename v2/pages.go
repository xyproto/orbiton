package main

import (
	"github.com/xyproto/vt100"
)

// Page represents a single page of text.
type Page struct {
	Lines []string
}

// ScrollableText holds pages of text and keeps track of the current page.
type ScrollableText struct {
	Pages       []Page
	CurrentPage int
}

// NewScrollableText creates a new instance of ScrollableText.
func NewScrollableText(pages []Page) *ScrollableText {
	return &ScrollableText{
		Pages:       pages,
		CurrentPage: 0,
	}
}

// DrawScrollableText will draw a scrollable text widget.
// Takes a Box struct for the size and position.
// Uses bt.Foreground and bt.Background.
func (e *Editor) DrawScrollableText(bt *BoxTheme, c *vt100.Canvas, r *Box, st *ScrollableText) {
	if st.CurrentPage >= len(st.Pages) || st.CurrentPage < 0 {
		// Invalid page number, do nothing or log an error
		return
	}

	page := st.Pages[st.CurrentPage]
	x := uint(r.X)
	for i, s := range page.Lines {
		y := uint(r.Y + i)
		if int(y) < r.Y+r.H { // Ensure we're within the box height
			c.Write(x, y, *bt.Foreground, *bt.Background, s)
		}
	}
}

// NextPage advances to the next page if there is one.
func (st *ScrollableText) NextPage() {
	if st.CurrentPage < len(st.Pages)-1 {
		st.CurrentPage++
	}
}

// PrevPage goes back to the previous page if there is one.
func (st *ScrollableText) PrevPage() {
	if st.CurrentPage > 0 {
		st.CurrentPage--
	}
}
