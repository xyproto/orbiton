package main

import (
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

// setLightTheme sets a theme suitable for white backgrounds
func (e *Editor) setLightTheme() {
	e.lightTheme = true

	e.fg = vt100.Black
	e.bg = vt100.Gray
	e.searchFg = vt100.Red
	e.gitColor = vt100.Blue
	e.multiLineComment = vt100.Green
	e.multiLineString = vt100.Red

	syntax.DefaultTextConfig.String = "red"
	syntax.DefaultTextConfig.Keyword = "blue"
	syntax.DefaultTextConfig.Comment = "darkgreen"
	syntax.DefaultTextConfig.Type = "blue"
	syntax.DefaultTextConfig.Literal = "darkcyan"
	syntax.DefaultTextConfig.Punctuation = "black"
	syntax.DefaultTextConfig.Plaintext = "black"
	syntax.DefaultTextConfig.Tag = "black"
	syntax.DefaultTextConfig.TextTag = "black"
	syntax.DefaultTextConfig.TextAttrName = "black"
	syntax.DefaultTextConfig.TextAttrValue = "black"
	syntax.DefaultTextConfig.Decimal = "darkcyan"
	syntax.DefaultTextConfig.AndOr = "black"
	syntax.DefaultTextConfig.Dollar = "red"
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

	// Rainbow parentheses
	rainbowParenColors = []vt100.AttributeColor{vt100.Magenta, vt100.Black, vt100.Blue, vt100.Green}
	unmatchedParenColor = vt100.Red

	// Command menu
	menuTitleColor = vt100.Blue
	menuArrowColor = vt100.Red
	menuTextColor = vt100.Black
	menuHighlightColor = vt100.Red
	menuSelectedColor = vt100.LightRed
}

// setFlameTheme sets a gray/red/orange/black/white theme, suitable for dark backgrounds
func (e *Editor) setFlameTheme() {
	e.lightTheme = false

	e.fg = vt100.White
	e.bg = vt100.Black
	e.searchFg = vt100.Red
	e.gitColor = vt100.Red
	e.multiLineComment = vt100.DarkGray
	e.multiLineString = vt100.Red

	syntax.DefaultTextConfig.String = "white"
	syntax.DefaultTextConfig.Keyword = "darkred"
	syntax.DefaultTextConfig.Comment = "darkgray"
	syntax.DefaultTextConfig.Type = "gray"
	syntax.DefaultTextConfig.Literal = "darkred"
	syntax.DefaultTextConfig.Punctuation = "white"
	syntax.DefaultTextConfig.Plaintext = "gray"
	syntax.DefaultTextConfig.Tag = "darkred"
	syntax.DefaultTextConfig.TextTag = "darkred"
	syntax.DefaultTextConfig.TextAttrName = "darkred"
	syntax.DefaultTextConfig.TextAttrValue = "darkred"
	syntax.DefaultTextConfig.Decimal = "darkred"
	syntax.DefaultTextConfig.AndOr = "darkred"
	syntax.DefaultTextConfig.Dollar = "white"
	syntax.DefaultTextConfig.Star = "white"
	syntax.DefaultTextConfig.Class = "darkred"
	syntax.DefaultTextConfig.Private = "white"
	syntax.DefaultTextConfig.Protected = "white"
	syntax.DefaultTextConfig.Public = "gray"
	syntax.DefaultTextConfig.Whitespace = ""

	// Markdown, switch light colors to darker ones
	headerTextColor = vt100.Red
	textColor = vt100.LightGray
	listTextColor = vt100.LightGray
	imageColor = vt100.Red
	boldColor = vt100.Red
	xColor = vt100.Red
	listCodeColor = vt100.White
	codeColor = vt100.White
	codeBlockColor = vt100.White

	// Rainbow parentheses
	rainbowParenColors = []vt100.AttributeColor{vt100.White, vt100.Red}
	unmatchedParenColor = vt100.Gray

	// Command menu
	menuTitleColor = vt100.Red
	menuArrowColor = vt100.White
	menuTextColor = vt100.White
	menuHighlightColor = vt100.Yellow
	menuSelectedColor = vt100.LightYellow
}
