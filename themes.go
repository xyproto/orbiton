package main

import (
	"os"

	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

var (
	// Color scheme for the "text edit" mode
	defaultEditorForeground       = vt100.LightGreen // for when syntax highlighting is not in use
	defaultEditorBackground       = vt100.BackgroundDefault
	defaultStatusForeground       = vt100.White
	defaultStatusBackground       = vt100.BackgroundBlack
	defaultStatusErrorForeground  = vt100.LightRed
	defaultStatusErrorBackground  = vt100.BackgroundDefault
	defaultEditorSearchHighlight  = vt100.LightMagenta
	defaultEditorMultilineComment = vt100.Gray
	defaultEditorMultilineString  = vt100.Magenta
	defaultEditorHighlightTheme   = syntax.TextConfig{
		String:        "lightyellow",
		Keyword:       "lightred",
		Comment:       "gray",
		Type:          "lightblue",
		Literal:       "lightgreen",
		Punctuation:   "lightblue",
		Plaintext:     "lightgreen",
		Tag:           "lightgreen",
		TextTag:       "lightgreen",
		TextAttrName:  "lightgreen",
		TextAttrValue: "lightgreen",
		Decimal:       "white",
		AndOr:         "lightyellow",
		Dollar:        "lightred",
		Star:          "lightyellow",
		Class:         "lightred",
		Private:       "darkred",
		Protected:     "darkyellow",
		Public:        "darkgreen",
		Whitespace:    "",
	}
)

// setLightTheme sets a theme suitable for white backgrounds
func (e *Editor) setLightTheme() {
	e.lightTheme = true

	e.fg = vt100.Black
	e.bg = vt100.BackgroundDefault // BackgroundWhite
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

// setRedBlackTheme sets a red/black/gray theme
func (e *Editor) setRedBlackTheme() {
	e.lightTheme = false

	e.fg = vt100.White
	e.bg = vt100.BackgroundBlack // Dark gray background, as opposed to BackgroundDefault
	e.searchFg = vt100.Red
	e.gitColor = vt100.Red
	e.multiLineComment = vt100.DarkGray
	e.multiLineString = vt100.Red

	syntax.DefaultTextConfig.String = "white"
	syntax.DefaultTextConfig.Keyword = "darkred"
	syntax.DefaultTextConfig.Comment = "gray"
	syntax.DefaultTextConfig.Type = "darkred"
	syntax.DefaultTextConfig.Literal = "white"
	syntax.DefaultTextConfig.Punctuation = "darkred"
	syntax.DefaultTextConfig.Plaintext = "lightgray"
	syntax.DefaultTextConfig.Tag = "darkred"
	syntax.DefaultTextConfig.TextTag = "darkred"
	syntax.DefaultTextConfig.TextAttrName = "darkred"
	syntax.DefaultTextConfig.TextAttrValue = "darkred"
	syntax.DefaultTextConfig.Decimal = "white"
	syntax.DefaultTextConfig.AndOr = "darkred"
	syntax.DefaultTextConfig.Dollar = "white"
	syntax.DefaultTextConfig.Star = "darkred"
	syntax.DefaultTextConfig.Class = "darkred"
	syntax.DefaultTextConfig.Private = "white"
	syntax.DefaultTextConfig.Protected = "white"
	syntax.DefaultTextConfig.Public = "lightgray"
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
	rainbowParenColors = []vt100.AttributeColor{vt100.Gray, vt100.White, vt100.Red}
	unmatchedParenColor = vt100.White

	// Command menu
	menuTitleColor = vt100.Red
	menuArrowColor = vt100.White
	menuTextColor = vt100.White
	menuHighlightColor = vt100.Yellow
	menuSelectedColor = vt100.LightYellow
}

func (e *Editor) respectNoColorEnvironmentVariable() {
	if os.Getenv("NO_COLOR") != "" {
		e.fg = vt100.Default
		e.bg = vt100.BackgroundDefault

		// The following is not strictly needed, since the text will be black and white just by setting e.fg and e.bg above
		e.searchFg = vt100.Default
		e.gitColor = vt100.Default
		e.multiLineComment = vt100.Default
		e.multiLineString = vt100.Default

		syntax.DefaultTextConfig.String = ""
		syntax.DefaultTextConfig.Keyword = ""
		syntax.DefaultTextConfig.Comment = ""
		syntax.DefaultTextConfig.Type = ""
		syntax.DefaultTextConfig.Literal = ""
		syntax.DefaultTextConfig.Punctuation = ""
		syntax.DefaultTextConfig.Plaintext = ""
		syntax.DefaultTextConfig.Tag = ""
		syntax.DefaultTextConfig.TextTag = ""
		syntax.DefaultTextConfig.TextAttrName = ""
		syntax.DefaultTextConfig.TextAttrValue = ""
		syntax.DefaultTextConfig.Decimal = ""
		syntax.DefaultTextConfig.AndOr = ""
		syntax.DefaultTextConfig.Dollar = ""
		syntax.DefaultTextConfig.Star = ""
		syntax.DefaultTextConfig.Class = ""
		syntax.DefaultTextConfig.Private = ""
		syntax.DefaultTextConfig.Protected = ""
		syntax.DefaultTextConfig.Public = ""
		syntax.DefaultTextConfig.Whitespace = ""

		// Rainbow parentheses
		rainbowParenColors = []vt100.AttributeColor{vt100.Gray}
		unmatchedParenColor = vt100.White

		// Command menu
		menuTitleColor = vt100.White
		menuArrowColor = vt100.White
		menuTextColor = vt100.Gray
		menuHighlightColor = vt100.White
		menuSelectedColor = vt100.Black
	}
}

func (status *StatusBar) respectNoColorEnvironmentVariable() {
	if os.Getenv("NO_COLOR") != "" {
		status.fg = vt100.Default
		status.bg = vt100.BackgroundDefault
	}
}
