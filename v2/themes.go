package main

import (
	"github.com/xyproto/env"
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

var envNoColor = env.Bool("NO_COLOR")

// Theme contains iformation about:
// * If the theme is light or dark
// * If syntax highlighting should be enabled
// * If no colors should be used
// * Colors for all the textual elements
type Theme struct {
	Light bool
	Foreground, Background,
	StatusForeground, StatusBackground,
	StatusErrorForeground, StatusErrorBackground,
	SearchHighlight, MultiLineComment, MultiLineString,
	Git vt100.AttributeColor
	String, Keyword, Comment, Type, Literal, Punctuation, Plaintext, Tag, TextTag, TextAttrName, TextAttrValue,
	Decimal, AndOr, Dollar, Star, Class, Private, Protected, Public, Whitespace, AssemblyEnd, Mut string
	RainbowParenColors []vt100.AttributeColor
	MarkdownTextColor, HeaderBulletColor, HeaderTextColor, ListBulletColor, ListTextColor,
	ListCodeColor, CodeColor, CodeBlockColor, ImageColor, LinkColor, QuoteColor, QuoteTextColor,
	HTMLColor, CommentColor, BoldColor, ItalicsColor, StrikeColor, TableColor, CheckboxColor,
	XColor, TableBackground, UnmatchedParenColor, MenuTitleColor, MenuArrowColor, MenuTextColor,
	MenuHighlightColor, MenuSelectedColor, ManSectionColor, ManSynopsisColor,
	BoxTextColor, BoxBackground, BoxHighlight vt100.AttributeColor
}

// NewDefaultTheme creates a new default Theme struct
func NewDefaultTheme() Theme {
	return Theme{
		Light:                 false,
		Foreground:            vt100.LightBlue,
		Background:            vt100.BackgroundDefault,
		StatusForeground:      vt100.White,
		StatusBackground:      vt100.BackgroundBlack,
		StatusErrorForeground: vt100.LightRed,
		StatusErrorBackground: vt100.BackgroundDefault,
		SearchHighlight:       vt100.LightMagenta,
		MultiLineComment:      vt100.Gray,
		MultiLineString:       vt100.Magenta,
		Git:                   vt100.LightGreen,
		String:                "lightyellow",
		Keyword:               "lightred",
		Comment:               "gray",
		Type:                  "lightblue",
		Literal:               "lightgreen",
		Punctuation:           "lightblue",
		Plaintext:             "lightgreen",
		Tag:                   "lightgreen",
		TextTag:               "lightgreen",
		TextAttrName:          "lightgreen",
		TextAttrValue:         "lightgreen",
		Decimal:               "white",
		AndOr:                 "lightyellow",
		Dollar:                "lightred",
		Star:                  "lightyellow",
		Class:                 "lightred",
		Private:               "darkred",
		Protected:             "darkyellow",
		Public:                "darkgreen",
		Whitespace:            "",
		AssemblyEnd:           "cyan",
		Mut:                   "darkyellow",
		RainbowParenColors:    []vt100.AttributeColor{vt100.LightMagenta, vt100.LightRed, vt100.Yellow, vt100.LightYellow, vt100.LightGreen, vt100.LightBlue, vt100.Red},
		MarkdownTextColor:     vt100.LightBlue,
		HeaderBulletColor:     vt100.DarkGray,
		HeaderTextColor:       vt100.LightGreen,
		ListBulletColor:       vt100.Red,
		ListTextColor:         vt100.LightCyan,
		ListCodeColor:         vt100.Default,
		CodeColor:             vt100.Default,
		CodeBlockColor:        vt100.Default,
		ImageColor:            vt100.LightYellow,
		LinkColor:             vt100.Magenta,
		QuoteColor:            vt100.Yellow,
		QuoteTextColor:        vt100.LightCyan,
		HTMLColor:             vt100.Default,
		CommentColor:          vt100.DarkGray,
		BoldColor:             vt100.LightYellow,
		ItalicsColor:          vt100.White,
		StrikeColor:           vt100.DarkGray,
		TableColor:            vt100.Blue,
		CheckboxColor:         vt100.Default,
		XColor:                vt100.LightYellow,
		TableBackground:       vt100.BackgroundDefault,
		UnmatchedParenColor:   vt100.White,
		MenuTitleColor:        vt100.LightYellow,
		MenuArrowColor:        vt100.Red,
		MenuTextColor:         vt100.Gray,
		MenuHighlightColor:    vt100.LightBlue,
		MenuSelectedColor:     vt100.LightCyan,
		ManSectionColor:       vt100.LightRed,
		ManSynopsisColor:      vt100.LightYellow,
		BoxTextColor:          vt100.Black,
		BoxBackground:         vt100.BackgroundBlue,
		BoxHighlight:          vt100.LightYellow,
	}
}

// NewRedBlackTheme creates a new red/black/gray/white Theme struct
func NewRedBlackTheme() Theme {
	// NOTE: Dark gray may not be visible with light terminal emulator themes
	return Theme{
		Light:                 false,
		Foreground:            vt100.LightGray,
		Background:            vt100.BackgroundBlack, // Dark gray background, as opposed to vt100.BackgroundDefault
		StatusForeground:      vt100.White,
		StatusBackground:      vt100.BackgroundBlack,
		StatusErrorForeground: vt100.LightRed,
		StatusErrorBackground: vt100.BackgroundDefault,
		SearchHighlight:       vt100.Red,
		MultiLineComment:      vt100.DarkGray,
		MultiLineString:       vt100.LightGray,
		Git:                   vt100.LightGreen,
		String:                "lightwhite",
		Keyword:               "darkred",
		Comment:               "darkgray",
		Type:                  "white",
		Literal:               "lightgray",
		Punctuation:           "darkred",
		Plaintext:             "lightgray",
		Tag:                   "darkred",
		TextTag:               "darkred",
		TextAttrName:          "darkred",
		TextAttrValue:         "darkred",
		Decimal:               "lightwhite",
		AndOr:                 "darkred",
		Dollar:                "lightwhite",
		Star:                  "lightwhite",
		Class:                 "darkred",
		Private:               "lightgray",
		Protected:             "lightgray",
		Public:                "lightwhite",
		Whitespace:            "",
		AssemblyEnd:           "darkred",
		Mut:                   "lightgray",
		RainbowParenColors:    []vt100.AttributeColor{vt100.LightGray, vt100.White, vt100.Red},
		MarkdownTextColor:     vt100.LightGray,
		HeaderBulletColor:     vt100.DarkGray,
		HeaderTextColor:       vt100.Red,
		ListBulletColor:       vt100.Red,
		ListTextColor:         vt100.LightGray,
		ListCodeColor:         vt100.Default,
		CodeColor:             vt100.White,
		CodeBlockColor:        vt100.White,
		ImageColor:            vt100.Red,
		LinkColor:             vt100.Magenta,
		QuoteColor:            vt100.White,
		QuoteTextColor:        vt100.LightGray,
		HTMLColor:             vt100.Default,
		CommentColor:          vt100.DarkGray,
		BoldColor:             vt100.Red,
		ItalicsColor:          vt100.White,
		StrikeColor:           vt100.DarkGray,
		TableColor:            vt100.White,
		CheckboxColor:         vt100.Default,
		XColor:                vt100.Red,
		TableBackground:       vt100.BackgroundBlack, // Dark gray background, as opposed to vt100.BackgroundDefault
		UnmatchedParenColor:   vt100.LightCyan,       // To really stand out
		MenuTitleColor:        vt100.Red,
		MenuArrowColor:        vt100.White,
		MenuTextColor:         vt100.White,
		MenuHighlightColor:    vt100.Yellow,
		MenuSelectedColor:     vt100.LightYellow,
		ManSectionColor:       vt100.Red,
		ManSynopsisColor:      vt100.White,
		BoxTextColor:          vt100.Black,
		BoxBackground:         vt100.BackgroundGray,
		BoxHighlight:          vt100.Red,
	}
}

// NewLightTheme creates a theme that is suitable for light xterm terminal emulator sessions
func NewLightTheme() Theme {
	return Theme{
		Light:                 true,
		Foreground:            vt100.Black,
		Background:            vt100.BackgroundDefault,
		StatusForeground:      vt100.White,
		StatusBackground:      vt100.BackgroundBlack,
		StatusErrorForeground: vt100.LightRed,
		StatusErrorBackground: vt100.BackgroundDefault,
		SearchHighlight:       vt100.Red,
		MultiLineComment:      vt100.Gray,
		MultiLineString:       vt100.Red,
		Git:                   vt100.Blue,
		String:                "red",
		Keyword:               "blue",
		Comment:               "gray",
		Type:                  "blue",
		Literal:               "darkcyan",
		Punctuation:           "black",
		Plaintext:             "black",
		Tag:                   "black",
		TextTag:               "black",
		TextAttrName:          "black",
		TextAttrValue:         "black",
		Decimal:               "darkcyan",
		AndOr:                 "black",
		Dollar:                "red",
		Star:                  "black",
		Class:                 "blue",
		Private:               "black",
		Protected:             "black",
		Public:                "black",
		Whitespace:            "",
		AssemblyEnd:           "red",
		Mut:                   "black",
		RainbowParenColors:    []vt100.AttributeColor{vt100.Magenta, vt100.Black, vt100.Blue, vt100.Green},
		MarkdownTextColor:     vt100.Default,
		HeaderBulletColor:     vt100.DarkGray,
		HeaderTextColor:       vt100.Blue,
		ListBulletColor:       vt100.Red,
		ListTextColor:         vt100.Default,
		ListCodeColor:         vt100.Red,
		CodeColor:             vt100.Red,
		CodeBlockColor:        vt100.Red,
		ImageColor:            vt100.Green,
		LinkColor:             vt100.Magenta,
		QuoteColor:            vt100.Yellow,
		QuoteTextColor:        vt100.LightCyan,
		HTMLColor:             vt100.Default,
		CommentColor:          vt100.DarkGray,
		BoldColor:             vt100.Blue,
		ItalicsColor:          vt100.Blue,
		StrikeColor:           vt100.DarkGray,
		TableColor:            vt100.Blue,
		CheckboxColor:         vt100.Default,
		XColor:                vt100.Blue,
		TableBackground:       vt100.BackgroundDefault,
		UnmatchedParenColor:   vt100.Red,
		MenuTitleColor:        vt100.Blue,
		MenuArrowColor:        vt100.Red,
		MenuTextColor:         vt100.Black,
		MenuHighlightColor:    vt100.Red,
		MenuSelectedColor:     vt100.LightRed,
		ManSectionColor:       vt100.Red,
		ManSynopsisColor:      vt100.Blue,
		BoxTextColor:          vt100.Black,
		BoxBackground:         vt100.BackgroundGray,
		BoxHighlight:          vt100.Red,
	}
}

// NewNoColorTheme creates a new theme without colors or syntax highlighting
func NewNoColorTheme() Theme {
	return Theme{
		Light:                 false,
		Foreground:            vt100.Default,
		Background:            vt100.BackgroundDefault,
		StatusForeground:      vt100.White,
		StatusBackground:      vt100.BackgroundBlack,
		StatusErrorForeground: vt100.White,
		StatusErrorBackground: vt100.BackgroundDefault,
		SearchHighlight:       vt100.Default,
		MultiLineComment:      vt100.Default,
		MultiLineString:       vt100.Default,
		Git:                   vt100.White,
		String:                "",
		Keyword:               "",
		Comment:               "",
		Type:                  "",
		Literal:               "",
		Punctuation:           "",
		Plaintext:             "",
		Tag:                   "",
		TextTag:               "",
		TextAttrName:          "",
		TextAttrValue:         "",
		Decimal:               "",
		AndOr:                 "",
		Dollar:                "",
		Star:                  "",
		Class:                 "",
		Private:               "",
		Protected:             "",
		Public:                "",
		Whitespace:            "",
		AssemblyEnd:           "",
		Mut:                   "",
		RainbowParenColors:    []vt100.AttributeColor{vt100.Gray},
		MarkdownTextColor:     vt100.Default,
		HeaderBulletColor:     vt100.Default,
		HeaderTextColor:       vt100.Default,
		ListBulletColor:       vt100.Default,
		ListTextColor:         vt100.Default,
		ListCodeColor:         vt100.Default,
		CodeColor:             vt100.Default,
		CodeBlockColor:        vt100.Default,
		ImageColor:            vt100.Default,
		LinkColor:             vt100.Default,
		QuoteColor:            vt100.Default,
		QuoteTextColor:        vt100.Default,
		HTMLColor:             vt100.Default,
		CommentColor:          vt100.Default,
		BoldColor:             vt100.Default,
		ItalicsColor:          vt100.Default,
		StrikeColor:           vt100.Default,
		TableColor:            vt100.Default,
		CheckboxColor:         vt100.Default,
		XColor:                vt100.White,
		TableBackground:       vt100.BackgroundDefault,
		UnmatchedParenColor:   vt100.White,
		MenuTitleColor:        vt100.White,
		MenuArrowColor:        vt100.White,
		MenuTextColor:         vt100.Gray,
		MenuHighlightColor:    vt100.White,
		MenuSelectedColor:     vt100.Black,
		ManSectionColor:       vt100.White,
		ManSynopsisColor:      vt100.White,
		BoxTextColor:          vt100.Black,
		BoxBackground:         vt100.BackgroundGray,
		BoxHighlight:          vt100.Black,
	}
}

// TextConfig returns a TextConfig struct that can be used for settings
// the syntax highlighting colors in the public TextConfig variable that is
// exported from the syntax package.
func (t Theme) TextConfig() *syntax.TextConfig {
	return &syntax.TextConfig{
		String:        t.String,
		Keyword:       t.Keyword,
		Comment:       t.Comment,
		Type:          t.Type,
		Literal:       t.Literal,
		Punctuation:   t.Punctuation,
		Plaintext:     t.Plaintext,
		Tag:           t.Tag,
		TextTag:       t.TextTag,
		TextAttrName:  t.TextAttrName,
		TextAttrValue: t.TextAttrValue,
		Decimal:       t.Decimal,
		AndOr:         t.AndOr,
		Dollar:        t.Dollar,
		Star:          t.Star,
		Class:         t.Class,
		Private:       t.Private,
		Protected:     t.Protected,
		Public:        t.Public,
		Whitespace:    t.Whitespace,
		AssemblyEnd:   t.AssemblyEnd,
		Mut:           t.Mut,
	}
}

// SetTheme assigns the given theme to the Editor,
// and also configures syntax highlighting by setting syntax.DefaultTextConfig.
// Light/dark, syntax highlighting and no color information is also set.
// Respect the NO_COLOR environment variable. May set e.NoSyntaxHighlight to true.
func (e *Editor) SetTheme(t Theme) {
	if envNoColor {
		t = NewNoColorTheme()
		e.syntaxHighlight = false
	}
	e.Theme = t
	syntax.DefaultTextConfig = *(t.TextConfig())
}

// setDefaultTheme sets the default colors
func (e *Editor) setDefaultTheme() {
	e.SetTheme(NewDefaultTheme())
}

// setLightTheme sets the light theme suitable for xterm
func (e *Editor) setLightTheme() {
	e.SetTheme(NewLightTheme())
}

// setRedBlackTheme sets a red/black/gray theme
func (e *Editor) setRedBlackTheme() {
	e.SetTheme(NewRedBlackTheme())
}
