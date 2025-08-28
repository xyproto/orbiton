package main

// Page represents a single page of text.
type Page struct {
	Lines []string
}

// ScrollableTextBox holds pages of text and keeps track of the current page.
type ScrollableTextBox struct {
	*Box
	Pages       []Page
	CurrentPage int
}

// NewScrollableTextBox creates a new instance of ScrollableText which also encapsulates a Box
func NewScrollableTextBox(pages []Page) *ScrollableTextBox {
	return &ScrollableTextBox{
		Box:         &Box{0, 0, 0, 0},
		Pages:       pages,
		CurrentPage: 0,
	}
}

// DrawScrollableText will draw a scrollable text widget.
// Takes a Box struct for the size and position.
// Uses bt.Foreground and bt.Background.
func (e *Editor) DrawScrollableText(bt *BoxTheme, c *Canvas, stb *ScrollableTextBox) {
	if stb.CurrentPage >= len(stb.Pages) || stb.CurrentPage < 0 {
		// Invalid page number, do nothing or log an error
		return
	}

	page := stb.Pages[stb.CurrentPage]
	x := uint(stb.X)
	for i, s := range page.Lines {
		y := uint(stb.Y + i)
		if int(y) < stb.Y+stb.H { // Ensure we're within the box height
			c.Write(x, y, *bt.Foreground, *bt.Background, s)
		}
	}
}

// NextPage advances to the next page if there is one.
func (stb *ScrollableTextBox) NextPage() {
	if stb.CurrentPage < len(stb.Pages)-1 {
		stb.CurrentPage++
	}
}

// PrevPage goes back to the previous page if there is one.
func (stb *ScrollableTextBox) PrevPage() {
	if stb.CurrentPage > 0 {
		stb.CurrentPage--
	}
}
