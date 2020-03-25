package main

import (
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

// lightTheme sets a theme suitable for white backgrounds
func (e *Editor) lightTheme() {
	e.fg = vt100.Black
	e.bg = vt100.Gray
	e.searchFg = vt100.Red
	e.gitColor = vt100.Blue
	syntax.DefaultTextConfig.String = "red"
	syntax.DefaultTextConfig.Keyword = "blue"
	syntax.DefaultTextConfig.Comment = "darkgreen"
	syntax.DefaultTextConfig.Type = "blue"
	syntax.DefaultTextConfig.Literal = "cyan"
	syntax.DefaultTextConfig.Punctuation = "black"
	syntax.DefaultTextConfig.Plaintext = "black"
	syntax.DefaultTextConfig.Tag = "black"
	syntax.DefaultTextConfig.TextTag = "black"
	syntax.DefaultTextConfig.TextAttrName = "black"
	syntax.DefaultTextConfig.TextAttrValue = "black"
	syntax.DefaultTextConfig.Decimal = "cyan"
	syntax.DefaultTextConfig.AndOr = "red"
	syntax.DefaultTextConfig.Whitespace = ""
}
