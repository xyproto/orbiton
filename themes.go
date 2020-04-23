package main

import (
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

// setLightTheme sets a theme suitable for white backgrounds
func (e *Editor) setLightTheme() {
	e.fg = vt100.Black
	e.bg = vt100.Gray
	e.searchFg = vt100.Red
	e.gitColor = vt100.Blue
	e.multilineComment = vt100.Green

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
	syntax.DefaultTextConfig.AndOr = "black"
	syntax.DefaultTextConfig.Star = "black"
	syntax.DefaultTextConfig.Class = "blue"
	syntax.DefaultTextConfig.Private = "black"
	syntax.DefaultTextConfig.Protected = "black"
	syntax.DefaultTextConfig.Public = "black"
	syntax.DefaultTextConfig.Whitespace = ""

	// Markdown, switch light colors to darker ones
	headerTextColor = vt100.Blue
	textColor = vt100.Default
	listTextColor = vt100.Default
	imageColor = vt100.Green
	boldColor = vt100.Blue
	xColor = vt100.Blue
	listCodeColor = vt100.Red
	codeColor = vt100.Red
	codeBlockColor = vt100.Red
}
