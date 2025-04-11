package main

import (
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

// TODO: Restructure how themes are stored, so that it's easier to list all themes that works with a dark background or all that works with a light background, ref. initialLightBackground

var (
	initialLightBackground *bool
)

// Theme contains iformation about:
// * If the theme is light or dark
// * If syntax highlighting should be enabled
// * If no colors should be used
// * Colors for all the textual elements
type Theme struct {
	TextAttrValue               string
	Name                        string
	Decimal                     string
	Mut                         string
	AssemblyEnd                 string
	Whitespace                  string
	Public                      string
	Protected                   string
	Private                     string
	Class                       string
	Star                        string
	Static                      string
	Self                        string
	Tag                         string
	Dollar                      string
	String                      string
	Keyword                     string
	Comment                     string
	Type                        string
	Literal                     string
	Punctuation                 string
	Plaintext                   string
	AndOr                       string
	TextTag                     string
	TextAttrName                string
	HeaderBulletColor           vt100.AttributeColor
	MultiLineString             vt100.AttributeColor
	DebugInstructionsBackground vt100.AttributeColor
	Git                         vt100.AttributeColor
	MultiLineComment            vt100.AttributeColor
	SearchHighlight             vt100.AttributeColor
	StatusErrorBackground       vt100.AttributeColor
	StatusErrorForeground       vt100.AttributeColor
	StatusBackground            vt100.AttributeColor
	StatusForeground            vt100.AttributeColor
	Background                  vt100.AttributeColor
	Foreground                  vt100.AttributeColor
	RainbowParenColors          []vt100.AttributeColor
	MarkdownTextColor           vt100.AttributeColor
	BoxUpperEdge                vt100.AttributeColor
	HeaderTextColor             vt100.AttributeColor
	ListBulletColor             vt100.AttributeColor
	ListTextColor               vt100.AttributeColor
	ListCodeColor               vt100.AttributeColor
	CodeColor                   vt100.AttributeColor
	CodeBlockColor              vt100.AttributeColor
	ImageColor                  vt100.AttributeColor
	LinkColor                   vt100.AttributeColor
	QuoteColor                  vt100.AttributeColor
	QuoteTextColor              vt100.AttributeColor
	HTMLColor                   vt100.AttributeColor
	CommentColor                vt100.AttributeColor
	BoldColor                   vt100.AttributeColor
	ItalicsColor                vt100.AttributeColor
	StrikeColor                 vt100.AttributeColor
	TableColor                  vt100.AttributeColor
	CheckboxColor               vt100.AttributeColor
	XColor                      vt100.AttributeColor
	DebugInstructionsForeground vt100.AttributeColor
	UnmatchedParenColor         vt100.AttributeColor
	MenuTitleColor              vt100.AttributeColor
	MenuArrowColor              vt100.AttributeColor
	MenuTextColor               vt100.AttributeColor
	MenuHighlightColor          vt100.AttributeColor
	MenuSelectedColor           vt100.AttributeColor
	ManSectionColor             vt100.AttributeColor
	ManSynopsisColor            vt100.AttributeColor
	BoxTextColor                vt100.AttributeColor
	BoxBackground               vt100.AttributeColor
	BoxHighlight                vt100.AttributeColor
	DebugRunningBackground      vt100.AttributeColor
	DebugStoppedBackground      vt100.AttributeColor
	DebugRegistersBackground    vt100.AttributeColor
	DebugOutputBackground       vt100.AttributeColor
	TableBackground             vt100.AttributeColor
	JumpToLetterColor           vt100.AttributeColor
	NanoHelpForeground          vt100.AttributeColor
	NanoHelpBackground          vt100.AttributeColor
	HighlightForeground         vt100.AttributeColor
	HighlightBackground         vt100.AttributeColor
	StatusMode                  bool
	Light                       bool
}

// NewDefaultTheme creates a new default Theme struct
func NewDefaultTheme() Theme {
	return Theme{
		Name:                        "Default",
		Light:                       false,
		Foreground:                  vt100.LightBlue,
		Background:                  vt100.BackgroundDefault,
		StatusForeground:            vt100.White,
		StatusBackground:            vt100.BackgroundBlack,
		StatusErrorForeground:       vt100.LightRed,
		StatusErrorBackground:       vt100.BackgroundDefault,
		SearchHighlight:             vt100.LightMagenta,
		MultiLineComment:            vt100.Gray,
		MultiLineString:             vt100.Magenta,
		HighlightForeground:         vt100.White,
		HighlightBackground:         vt100.BackgroundDefault,
		Git:                         vt100.LightGreen,
		String:                      "lightyellow",
		Keyword:                     "lightred",
		Comment:                     "gray",
		Type:                        "lightblue",
		Literal:                     "lightgreen",
		Punctuation:                 "lightblue",
		Plaintext:                   "lightgreen",
		Tag:                         "lightgreen",
		TextTag:                     "lightgreen",
		TextAttrName:                "lightgreen",
		TextAttrValue:               "lightgreen",
		Decimal:                     "white",
		AndOr:                       "lightyellow",
		Dollar:                      "lightred",
		Star:                        "lightyellow",
		Static:                      "lightyellow",
		Self:                        "white",
		Class:                       "lightred",
		Private:                     "darkred",
		Protected:                   "darkyellow",
		Public:                      "darkgreen",
		Whitespace:                  "",
		AssemblyEnd:                 "cyan",
		Mut:                         "darkyellow",
		RainbowParenColors:          []vt100.AttributeColor{vt100.LightMagenta, vt100.LightRed, vt100.Yellow, vt100.LightYellow, vt100.LightGreen, vt100.LightBlue, vt100.Red},
		MarkdownTextColor:           vt100.LightBlue,
		HeaderBulletColor:           vt100.DarkGray,
		HeaderTextColor:             vt100.LightGreen,
		ListBulletColor:             vt100.Red,
		ListTextColor:               vt100.LightCyan,
		ListCodeColor:               vt100.Default,
		CodeColor:                   vt100.Default,
		CodeBlockColor:              vt100.Default,
		ImageColor:                  vt100.LightYellow,
		LinkColor:                   vt100.Magenta,
		QuoteColor:                  vt100.Yellow,
		QuoteTextColor:              vt100.LightCyan,
		HTMLColor:                   vt100.Default,
		CommentColor:                vt100.DarkGray,
		BoldColor:                   vt100.LightYellow,
		ItalicsColor:                vt100.White,
		StrikeColor:                 vt100.DarkGray,
		TableColor:                  vt100.Blue,
		CheckboxColor:               vt100.Default,
		XColor:                      vt100.LightYellow,
		TableBackground:             vt100.BackgroundDefault,
		UnmatchedParenColor:         vt100.White,
		MenuTitleColor:              vt100.LightYellow,
		MenuArrowColor:              vt100.Red,
		MenuTextColor:               vt100.Gray,
		MenuHighlightColor:          vt100.LightBlue,
		MenuSelectedColor:           vt100.LightCyan,
		ManSectionColor:             vt100.LightRed,
		ManSynopsisColor:            vt100.LightYellow,
		BoxTextColor:                vt100.Black,
		BoxBackground:               vt100.BackgroundBlue,
		BoxHighlight:                vt100.LightYellow,
		DebugRunningBackground:      vt100.BackgroundCyan,
		DebugStoppedBackground:      vt100.BackgroundMagenta,
		DebugRegistersBackground:    vt100.BackgroundBlue,
		DebugOutputBackground:       vt100.BackgroundYellow,
		DebugInstructionsForeground: vt100.LightYellow,
		DebugInstructionsBackground: vt100.BackgroundMagenta,
		BoxUpperEdge:                vt100.White,
		JumpToLetterColor:           vt100.LightRed,
		NanoHelpForeground:          vt100.Black,
		NanoHelpBackground:          vt100.BackgroundGray,
	}
}

// NewOrbTheme creates a new logical Theme struct with a refined color palette
func NewOrbTheme() Theme {
	return Theme{
		Name:                        "Orb",
		Light:                       false,
		Foreground:                  vt100.LightGray,
		Background:                  vt100.BackgroundBlack,
		StatusForeground:            vt100.LightGray,
		StatusBackground:            vt100.Gray,
		StatusErrorForeground:       vt100.LightRed,
		StatusErrorBackground:       vt100.BackgroundBlack,
		SearchHighlight:             vt100.LightMagenta,
		MultiLineComment:            vt100.Gray,
		MultiLineString:             vt100.LightCyan,
		HighlightForeground:         vt100.White,
		HighlightBackground:         vt100.BackgroundBlack,
		Git:                         vt100.LightCyan,
		String:                      "cyan",
		Keyword:                     "lightcyan",
		Comment:                     "gray",
		Type:                        "lightblue",
		Literal:                     "lightcyan",
		Punctuation:                 "lightgray",
		Plaintext:                   "white",
		Tag:                         "lightcyan",
		TextTag:                     "lightcyan",
		TextAttrName:                "lightblue",
		TextAttrValue:               "lightgreen",
		Decimal:                     "white",
		AndOr:                       "lightcyan",
		Dollar:                      "lightred",
		Star:                        "lightgreen",
		Static:                      "lightgreen",
		Self:                        "white",
		Class:                       "lightcyan",
		Private:                     "lightred",
		Protected:                   "lightyellow",
		Public:                      "lightgreen",
		Whitespace:                  "",
		AssemblyEnd:                 "lightblue",
		Mut:                         "lightgreen",
		RainbowParenColors:          []vt100.AttributeColor{vt100.LightRed, vt100.LightCyan, vt100.LightGreen, vt100.LightYellow, vt100.LightBlue, vt100.Gray, vt100.LightGray},
		MarkdownTextColor:           vt100.LightGray,
		HeaderBulletColor:           vt100.White,
		HeaderTextColor:             vt100.LightCyan,
		ListBulletColor:             vt100.LightRed,
		ListTextColor:               vt100.LightCyan,
		ListCodeColor:               vt100.White,
		CodeColor:                   vt100.White,
		CodeBlockColor:              vt100.White,
		ImageColor:                  vt100.LightGreen,
		LinkColor:                   vt100.LightCyan,
		QuoteColor:                  vt100.LightGreen,
		QuoteTextColor:              vt100.White,
		HTMLColor:                   vt100.White,
		CommentColor:                vt100.Gray,
		BoldColor:                   vt100.LightGreen,
		ItalicsColor:                vt100.LightGray,
		StrikeColor:                 vt100.White,
		TableColor:                  vt100.White,
		CheckboxColor:               vt100.White,
		XColor:                      vt100.LightGreen,
		TableBackground:             vt100.BackgroundBlack,
		UnmatchedParenColor:         vt100.LightRed,
		MenuTitleColor:              vt100.LightMagenta,
		MenuArrowColor:              vt100.White,
		MenuTextColor:               vt100.Blue,
		MenuHighlightColor:          vt100.LightCyan,
		MenuSelectedColor:           vt100.LightRed,
		ManSectionColor:             vt100.LightCyan,
		ManSynopsisColor:            vt100.LightGreen,
		BoxTextColor:                vt100.White,
		BoxBackground:               vt100.DarkGray,
		BoxHighlight:                vt100.LightYellow,
		DebugRunningBackground:      vt100.Cyan,
		DebugStoppedBackground:      vt100.LightRed,
		DebugRegistersBackground:    vt100.DarkGray,
		DebugOutputBackground:       vt100.LightGreen,
		DebugInstructionsForeground: vt100.LightGreen,
		DebugInstructionsBackground: vt100.DarkGray,
		BoxUpperEdge:                vt100.White,
		JumpToLetterColor:           vt100.LightRed,
		NanoHelpForeground:          vt100.White,
		NanoHelpBackground:          vt100.DarkGray,
	}
}

// NewPinetreeTheme creates a new Theme struct based on the base16-snazzy theme
func NewPinetreeTheme() Theme {
	return Theme{
		Name:                        "Pinetree",
		Light:                       false,
		Foreground:                  vt100.LightGray,
		Background:                  vt100.BackgroundBlack,
		StatusForeground:            vt100.LightGray,
		StatusBackground:            vt100.BackgroundBlack,
		StatusErrorForeground:       vt100.LightRed,
		StatusErrorBackground:       vt100.BackgroundBlack,
		SearchHighlight:             vt100.Yellow,
		MultiLineComment:            vt100.DarkGray,
		MultiLineString:             vt100.Magenta,
		HighlightForeground:         vt100.LightCyan,
		HighlightBackground:         vt100.BackgroundBlack,
		Git:                         vt100.LightGreen,
		String:                      "lightgreen",
		Keyword:                     "lightred",
		Comment:                     "darkgray",
		Type:                        "lightcyan",
		Literal:                     "lightgreen",
		Punctuation:                 "lightgray",
		Plaintext:                   "lightgray",
		Tag:                         "lightred",
		TextTag:                     "lightred",
		TextAttrName:                "lightyellow",
		TextAttrValue:               "lightgreen",
		Decimal:                     "lightgreen",
		AndOr:                       "lightred",
		Dollar:                      "lightgreen",
		Star:                        "lightyellow",
		Static:                      "lightblue",
		Self:                        "lightgray",
		Class:                       "lightblue",
		Private:                     "darkred",
		Protected:                   "darkyellow",
		Public:                      "lightgreen",
		Whitespace:                  "",
		AssemblyEnd:                 "cyan",
		Mut:                         "darkyellow",
		RainbowParenColors:          []vt100.AttributeColor{vt100.LightMagenta, vt100.LightRed, vt100.Yellow, vt100.LightYellow, vt100.LightGreen, vt100.LightBlue, vt100.Red},
		MarkdownTextColor:           vt100.LightGray,
		HeaderBulletColor:           vt100.DarkGray,
		HeaderTextColor:             vt100.LightBlue,
		ListBulletColor:             vt100.LightRed,
		ListTextColor:               vt100.LightGray,
		ListCodeColor:               vt100.LightGreen,
		CodeColor:                   vt100.LightGreen,
		CodeBlockColor:              vt100.BackgroundBlack,
		ImageColor:                  vt100.Yellow,
		LinkColor:                   vt100.LightBlue,
		QuoteColor:                  vt100.Yellow,
		QuoteTextColor:              vt100.LightGray,
		HTMLColor:                   vt100.LightRed,
		CommentColor:                vt100.DarkGray,
		BoldColor:                   vt100.White,
		ItalicsColor:                vt100.LightBlue,
		StrikeColor:                 vt100.DarkGray,
		TableColor:                  vt100.LightBlue,
		CheckboxColor:               vt100.LightGray,
		XColor:                      vt100.LightRed,
		TableBackground:             vt100.BackgroundBlack,
		UnmatchedParenColor:         vt100.LightRed,
		MenuTitleColor:              vt100.LightGreen,
		MenuArrowColor:              vt100.LightRed,
		MenuTextColor:               vt100.LightGray,
		MenuHighlightColor:          vt100.LightCyan,
		MenuSelectedColor:           vt100.White,
		ManSectionColor:             vt100.LightRed,
		ManSynopsisColor:            vt100.Yellow,
		BoxTextColor:                vt100.Black,
		BoxBackground:               vt100.BackgroundBlue,
		BoxHighlight:                vt100.LightYellow,
		DebugRunningBackground:      vt100.BackgroundGreen,
		DebugStoppedBackground:      vt100.BackgroundMagenta,
		DebugRegistersBackground:    vt100.BackgroundBlue,
		DebugOutputBackground:       vt100.BackgroundYellow,
		DebugInstructionsForeground: vt100.LightYellow,
		DebugInstructionsBackground: vt100.BackgroundMagenta,
		BoxUpperEdge:                vt100.LightGray,
		JumpToLetterColor:           vt100.LightRed,
		NanoHelpForeground:          vt100.Black,
		NanoHelpBackground:          vt100.BackgroundGray,
	}
}

// NewLitmusTheme creates a new default Theme struct
func NewLitmusTheme() Theme {
	return Theme{
		Name:                        "Litmus",
		Light:                       false,
		Foreground:                  vt100.Default,
		Background:                  vt100.BackgroundGray,
		StatusForeground:            vt100.Gray,
		StatusBackground:            vt100.BackgroundBlack,
		StatusErrorForeground:       vt100.LightRed,
		StatusErrorBackground:       vt100.BackgroundDefault,
		SearchHighlight:             vt100.LightMagenta,
		MultiLineComment:            vt100.Gray,
		MultiLineString:             vt100.Magenta,
		HighlightForeground:         vt100.LightRed,
		HighlightBackground:         vt100.BackgroundGray,
		Git:                         vt100.Black,
		String:                      "blue",
		Keyword:                     "lightred",
		Comment:                     "darkgray",
		Type:                        "cyan",
		Literal:                     "black",
		Punctuation:                 "black",
		Plaintext:                   "black",
		Tag:                         "black",
		TextTag:                     "black",
		TextAttrName:                "black",
		TextAttrValue:               "black",
		Decimal:                     "black",
		AndOr:                       "lightred",
		Dollar:                      "lightred",
		Star:                        "magenta",
		Static:                      "magenta",
		Self:                        "black",
		Class:                       "lightred",
		Private:                     "red",
		Protected:                   "yellow",
		Public:                      "green",
		Whitespace:                  "",
		AssemblyEnd:                 "magenta",
		Mut:                         "yellow",
		RainbowParenColors:          []vt100.AttributeColor{vt100.LightMagenta, vt100.LightRed, vt100.Yellow, vt100.Green, vt100.Blue, vt100.LightBlue, vt100.Red},
		MarkdownTextColor:           vt100.Black,
		HeaderBulletColor:           vt100.DarkGray,
		HeaderTextColor:             vt100.Black,
		ListBulletColor:             vt100.Red,
		ListTextColor:               vt100.LightBlue,
		ListCodeColor:               vt100.Black,
		CodeColor:                   vt100.Black,
		CodeBlockColor:              vt100.Black,
		ImageColor:                  vt100.Red,
		LinkColor:                   vt100.Magenta,
		QuoteColor:                  vt100.Red,
		QuoteTextColor:              vt100.LightBlue,
		HTMLColor:                   vt100.Black,
		CommentColor:                vt100.DarkGray,
		BoldColor:                   vt100.Red,
		ItalicsColor:                vt100.DarkGray,
		StrikeColor:                 vt100.DarkGray,
		TableColor:                  vt100.Black,
		CheckboxColor:               vt100.Black,
		XColor:                      vt100.Red,
		TableBackground:             vt100.BackgroundGray,
		UnmatchedParenColor:         vt100.Yellow,
		MenuTitleColor:              vt100.Black,
		MenuArrowColor:              vt100.Red,
		MenuTextColor:               vt100.Gray,
		MenuHighlightColor:          vt100.Cyan,
		MenuSelectedColor:           vt100.LightBlue,
		ManSectionColor:             vt100.LightRed,
		ManSynopsisColor:            vt100.Red,
		BoxTextColor:                vt100.Black,
		BoxBackground:               vt100.BackgroundCyan,
		BoxHighlight:                vt100.Red,
		DebugRunningBackground:      vt100.BackgroundBlue,
		DebugStoppedBackground:      vt100.BackgroundMagenta,
		DebugRegistersBackground:    vt100.BackgroundCyan,
		DebugOutputBackground:       vt100.BackgroundGray,
		DebugInstructionsForeground: vt100.Red,
		DebugInstructionsBackground: vt100.BackgroundMagenta,
		BoxUpperEdge:                vt100.DarkGray,
		JumpToLetterColor:           vt100.LightRed,
		NanoHelpForeground:          vt100.Black,
		NanoHelpBackground:          vt100.BackgroundGray,
	}
}

// NewSynthwaveTheme creates a new Theme struct
func NewSynthwaveTheme() Theme {
	return Theme{
		Name:                        "Synthwave",
		Light:                       false,
		Foreground:                  vt100.LightBlue,
		Background:                  vt100.BackgroundDefault,
		StatusForeground:            vt100.White,
		StatusBackground:            vt100.BackgroundBlack,
		StatusErrorForeground:       vt100.Magenta,
		StatusErrorBackground:       vt100.BackgroundDefault,
		SearchHighlight:             vt100.LightMagenta,
		MultiLineComment:            vt100.Gray,
		MultiLineString:             vt100.Magenta,
		HighlightForeground:         vt100.White,
		HighlightBackground:         vt100.BackgroundDefault,
		Git:                         vt100.Cyan,
		String:                      "lightgray",
		Keyword:                     "magenta",
		Comment:                     "gray",
		Type:                        "lightblue",
		Literal:                     "cyan",
		Punctuation:                 "lightblue",
		Plaintext:                   "cyan",
		Tag:                         "cyan",
		TextTag:                     "cyan",
		TextAttrName:                "cyan",
		TextAttrValue:               "cyan",
		Decimal:                     "white",
		AndOr:                       "lightgray",
		Dollar:                      "magenta",
		Star:                        "lightgray",
		Static:                      "lightgray",
		Self:                        "white",
		Class:                       "magenta",
		Private:                     "magenta",
		Protected:                   "blue", // also the word after the arrow in C/C++, for "object->property"
		Public:                      "green",
		Whitespace:                  "",
		AssemblyEnd:                 "cyan",
		Mut:                         "darkgray",
		RainbowParenColors:          []vt100.AttributeColor{vt100.LightRed, vt100.LightMagenta, vt100.Blue, vt100.LightCyan, vt100.LightBlue, vt100.Magenta, vt100.Cyan},
		MarkdownTextColor:           vt100.LightBlue,
		HeaderBulletColor:           vt100.DarkGray,
		HeaderTextColor:             vt100.Cyan,
		ListBulletColor:             vt100.Magenta,
		ListTextColor:               vt100.LightCyan,
		ListCodeColor:               vt100.Default,
		CodeColor:                   vt100.Default,
		CodeBlockColor:              vt100.Default,
		ImageColor:                  vt100.LightGray,
		LinkColor:                   vt100.LightMagenta,
		QuoteColor:                  vt100.Gray,
		QuoteTextColor:              vt100.LightCyan,
		HTMLColor:                   vt100.Default,
		CommentColor:                vt100.DarkGray,
		BoldColor:                   vt100.LightGray,
		ItalicsColor:                vt100.White,
		StrikeColor:                 vt100.DarkGray,
		TableColor:                  vt100.Blue,
		CheckboxColor:               vt100.Default,
		XColor:                      vt100.LightGray,
		TableBackground:             vt100.BackgroundDefault,
		UnmatchedParenColor:         vt100.LightRed, // to really stand out
		MenuTitleColor:              vt100.Cyan,
		MenuArrowColor:              vt100.Magenta,
		MenuTextColor:               vt100.Gray,
		MenuHighlightColor:          vt100.LightBlue,
		MenuSelectedColor:           vt100.LightCyan,
		ManSectionColor:             vt100.LightMagenta,
		ManSynopsisColor:            vt100.LightGray,
		BoxTextColor:                vt100.Black,
		BoxBackground:               vt100.BackgroundBlue,
		BoxHighlight:                vt100.LightGray,
		DebugRunningBackground:      vt100.BackgroundCyan,
		DebugStoppedBackground:      vt100.BackgroundRed,
		DebugRegistersBackground:    vt100.BackgroundBlue,
		DebugOutputBackground:       vt100.BackgroundGray,
		DebugInstructionsForeground: vt100.LightGray,
		DebugInstructionsBackground: vt100.BackgroundRed,
		BoxUpperEdge:                vt100.White,
		JumpToLetterColor:           vt100.LightMagenta,
		NanoHelpForeground:          vt100.Black,
		NanoHelpBackground:          vt100.BackgroundGray,
	}
}

// NewTealTheme creates a new Theme struct
func NewTealTheme() Theme {
	return Theme{
		Name:                        "Teal",
		Light:                       false,
		Foreground:                  vt100.Cyan,
		Background:                  vt100.BackgroundDefault,
		StatusForeground:            vt100.White,
		StatusBackground:            vt100.BackgroundBlack,
		StatusErrorForeground:       vt100.LightRed,
		StatusErrorBackground:       vt100.BackgroundGray,
		SearchHighlight:             vt100.Red,
		MultiLineComment:            vt100.Gray,
		MultiLineString:             vt100.Blue,
		HighlightForeground:         vt100.White,
		HighlightBackground:         vt100.BackgroundDefault,
		Git:                         vt100.Blue,
		String:                      "lightcyan",
		Keyword:                     "lightgray",
		Comment:                     "gray",
		Type:                        "lightblue",
		Literal:                     "lightcyan",
		Punctuation:                 "lightgray",
		Plaintext:                   "cyan",
		Tag:                         "cyan",
		TextTag:                     "cyan",
		TextAttrName:                "cyan",
		TextAttrValue:               "cyan",
		Decimal:                     "lightgray",
		AndOr:                       "lightgray",
		Dollar:                      "lightgray",
		Star:                        "lightgray",
		Static:                      "lightgray",
		Self:                        "lightyellow",
		Class:                       "lightblue",
		Private:                     "darkred",
		Protected:                   "darkyellow",
		Public:                      "darkgreen",
		Whitespace:                  "",
		AssemblyEnd:                 "lightgray",
		Mut:                         "blue",
		RainbowParenColors:          []vt100.AttributeColor{vt100.LightRed, vt100.LightGray, vt100.LightBlue, vt100.Blue},
		MarkdownTextColor:           vt100.Cyan,
		HeaderBulletColor:           vt100.LightGray,
		HeaderTextColor:             vt100.LightGray,
		ListBulletColor:             vt100.LightGray,
		ListTextColor:               vt100.Cyan,
		ListCodeColor:               vt100.Cyan,
		CodeColor:                   vt100.LightGray,
		CodeBlockColor:              vt100.LightGray,
		ImageColor:                  vt100.LightGray,
		LinkColor:                   vt100.LightGray,
		QuoteColor:                  vt100.LightGray,
		QuoteTextColor:              vt100.LightGray,
		HTMLColor:                   vt100.Cyan,
		CommentColor:                vt100.DarkGray,
		BoldColor:                   vt100.LightGray,
		ItalicsColor:                vt100.LightGray,
		StrikeColor:                 vt100.DarkGray,
		TableColor:                  vt100.LightGray,
		CheckboxColor:               vt100.Cyan,
		XColor:                      vt100.LightGray,
		TableBackground:             vt100.BackgroundDefault,
		UnmatchedParenColor:         vt100.LightGray,
		MenuTitleColor:              vt100.LightBlue,
		MenuArrowColor:              vt100.LightCyan,
		MenuTextColor:               vt100.Gray,
		MenuHighlightColor:          vt100.Cyan,
		MenuSelectedColor:           vt100.White,
		ManSectionColor:             vt100.LightGray,
		ManSynopsisColor:            vt100.LightGray,
		BoxTextColor:                vt100.Black,
		BoxBackground:               vt100.BackgroundBlue,
		BoxHighlight:                vt100.LightGray,
		DebugRunningBackground:      vt100.BackgroundBlue,
		DebugStoppedBackground:      vt100.BackgroundGray,
		DebugRegistersBackground:    vt100.BackgroundGreen,
		DebugOutputBackground:       vt100.BackgroundCyan,
		DebugInstructionsForeground: vt100.LightCyan,
		DebugInstructionsBackground: vt100.BackgroundBlue,
		BoxUpperEdge:                vt100.LightGray,
		JumpToLetterColor:           vt100.LightGreen,
		NanoHelpForeground:          vt100.Black,
		NanoHelpBackground:          vt100.BackgroundGray,
	}
}

// NewRedBlackTheme creates a new red/black/gray/white Theme struct
func NewRedBlackTheme() Theme {
	// NOTE: Dark gray may not be visible with light terminal emulator themes
	return Theme{
		Name:                        "Red & black",
		Light:                       false,
		Foreground:                  vt100.LightGray,
		Background:                  vt100.BackgroundBlack, // Dark gray background, as opposed to vt100.BackgroundDefault
		StatusForeground:            vt100.White,
		StatusBackground:            vt100.BackgroundBlack,
		StatusErrorForeground:       vt100.LightRed,
		StatusErrorBackground:       vt100.BackgroundDefault,
		SearchHighlight:             vt100.Red,
		MultiLineComment:            vt100.DarkGray,
		MultiLineString:             vt100.LightGray,
		HighlightForeground:         vt100.LightGray,
		HighlightBackground:         vt100.BackgroundBlack,
		Git:                         vt100.LightGreen,
		String:                      "white",
		Keyword:                     "darkred",
		Comment:                     "darkgray",
		Type:                        "white",
		Literal:                     "lightgray",
		Punctuation:                 "darkred",
		Plaintext:                   "lightgray",
		Tag:                         "darkred",
		TextTag:                     "darkred",
		TextAttrName:                "darkred",
		TextAttrValue:               "darkred",
		Decimal:                     "white",
		AndOr:                       "darkred",
		Dollar:                      "white",
		Star:                        "white",
		Static:                      "white",
		Self:                        "white",
		Class:                       "darkred",
		Private:                     "lightgray",
		Protected:                   "lightgray",
		Public:                      "white",
		Whitespace:                  "",
		AssemblyEnd:                 "darkred",
		Mut:                         "lightgray",
		RainbowParenColors:          []vt100.AttributeColor{vt100.LightGray, vt100.White, vt100.Red},
		MarkdownTextColor:           vt100.LightGray,
		HeaderBulletColor:           vt100.DarkGray,
		HeaderTextColor:             vt100.Red,
		ListBulletColor:             vt100.Red,
		ListTextColor:               vt100.LightGray,
		ListCodeColor:               vt100.Default,
		CodeColor:                   vt100.White,
		CodeBlockColor:              vt100.White,
		ImageColor:                  vt100.Red,
		LinkColor:                   vt100.DarkGray,
		QuoteColor:                  vt100.White,
		QuoteTextColor:              vt100.LightGray,
		HTMLColor:                   vt100.LightGray,
		CommentColor:                vt100.DarkGray,
		BoldColor:                   vt100.Red,
		ItalicsColor:                vt100.White,
		StrikeColor:                 vt100.DarkGray,
		TableColor:                  vt100.White,
		CheckboxColor:               vt100.Default,
		XColor:                      vt100.Red,
		TableBackground:             vt100.BackgroundBlack, // Dark gray background, as opposed to vt100.BackgroundDefault
		UnmatchedParenColor:         vt100.LightCyan,       // To really stand out
		MenuTitleColor:              vt100.LightRed,
		MenuArrowColor:              vt100.Red,
		MenuTextColor:               vt100.Gray,
		MenuHighlightColor:          vt100.LightGray,
		MenuSelectedColor:           vt100.DarkGray,
		ManSectionColor:             vt100.Red,
		ManSynopsisColor:            vt100.White,
		BoxTextColor:                vt100.Black,
		BoxBackground:               vt100.BackgroundGray,
		BoxHighlight:                vt100.Red,
		DebugRunningBackground:      vt100.BackgroundGray,
		DebugStoppedBackground:      vt100.BackgroundGray,
		DebugRegistersBackground:    vt100.BackgroundGray,
		DebugOutputBackground:       vt100.BackgroundGray,
		DebugInstructionsForeground: vt100.Red,
		DebugInstructionsBackground: vt100.BackgroundGray,
		BoxUpperEdge:                vt100.Black,
		JumpToLetterColor:           vt100.Red,
		NanoHelpForeground:          vt100.Black,
		NanoHelpBackground:          vt100.BackgroundGray,
	}
}

// NewLightBlueEditTheme creates a new blue/gray/yellow Theme struct, for light backgrounds
func NewLightBlueEditTheme() Theme {
	return Theme{
		Name:                        "Blue Edit Light",
		Light:                       true,
		StatusMode:                  false,
		Foreground:                  vt100.White,
		Background:                  vt100.BackgroundBlue,
		StatusForeground:            vt100.Black,
		StatusBackground:            vt100.BackgroundCyan,
		StatusErrorForeground:       vt100.Black,
		StatusErrorBackground:       vt100.BackgroundRed,
		SearchHighlight:             vt100.LightRed,
		MultiLineComment:            vt100.Gray,
		MultiLineString:             vt100.LightYellow,
		HighlightForeground:         vt100.LightYellow,
		HighlightBackground:         vt100.BackgroundBlue,
		Git:                         vt100.White,
		String:                      "lightyellow",
		Keyword:                     "lightcyan",
		Comment:                     "lightgray",
		Type:                        "white",
		Literal:                     "white",
		Punctuation:                 "white",
		Plaintext:                   "white",
		Tag:                         "white",
		TextTag:                     "white",
		TextAttrName:                "white",
		TextAttrValue:               "white",
		Decimal:                     "white",
		AndOr:                       "lightyellow",
		Dollar:                      "lightred",
		Star:                        "lightred",
		Static:                      "lightred",
		Self:                        "lightyellow",
		Class:                       "lightcyan",
		Private:                     "lightcyan",
		Protected:                   "lightyellow",
		Public:                      "white",
		Whitespace:                  "",
		AssemblyEnd:                 "lightcyan",
		Mut:                         "lightyellow",
		RainbowParenColors:          []vt100.AttributeColor{vt100.LightCyan, vt100.LightYellow, vt100.LightGreen, vt100.White},
		MarkdownTextColor:           vt100.White,
		HeaderBulletColor:           vt100.LightGray,
		HeaderTextColor:             vt100.White,
		ListBulletColor:             vt100.LightCyan,
		ListTextColor:               vt100.LightCyan,
		ListCodeColor:               vt100.White,
		CodeColor:                   vt100.White,
		CodeBlockColor:              vt100.White,
		ImageColor:                  vt100.LightYellow,
		LinkColor:                   vt100.LightYellow,
		QuoteColor:                  vt100.LightYellow,
		QuoteTextColor:              vt100.LightCyan,
		HTMLColor:                   vt100.White,
		CommentColor:                vt100.LightGray,
		BoldColor:                   vt100.LightYellow,
		ItalicsColor:                vt100.White,
		StrikeColor:                 vt100.LightGray,
		TableColor:                  vt100.White,
		CheckboxColor:               vt100.White,
		XColor:                      vt100.LightYellow,
		TableBackground:             vt100.BackgroundBlue,
		UnmatchedParenColor:         vt100.White,
		MenuTitleColor:              vt100.LightYellow,
		MenuArrowColor:              vt100.LightRed,
		MenuTextColor:               vt100.LightYellow,
		MenuHighlightColor:          vt100.LightRed,
		MenuSelectedColor:           vt100.White,
		ManSectionColor:             vt100.LightBlue,
		ManSynopsisColor:            vt100.LightBlue,
		BoxTextColor:                vt100.Black,
		BoxBackground:               vt100.BackgroundGray,
		BoxHighlight:                vt100.LightYellow,
		DebugRunningBackground:      vt100.BackgroundGray,
		DebugStoppedBackground:      vt100.BackgroundMagenta,
		DebugRegistersBackground:    vt100.BackgroundMagenta,
		DebugOutputBackground:       vt100.BackgroundYellow,
		DebugInstructionsForeground: vt100.LightYellow,
		DebugInstructionsBackground: vt100.BackgroundCyan,
		BoxUpperEdge:                vt100.White,
		JumpToLetterColor:           vt100.LightBlue,
		NanoHelpForeground:          vt100.Black,
		NanoHelpBackground:          vt100.BackgroundGray,
	}
}

// NewDarkBlueEditTheme creates a new blue/gray/yellow Theme struct, for dark backgrounds
func NewDarkBlueEditTheme() Theme {
	return Theme{
		Name:                        "Blue Edit Dark",
		Light:                       false,
		StatusMode:                  false,
		Foreground:                  vt100.LightYellow,
		Background:                  vt100.BackgroundBlue,
		StatusForeground:            vt100.White,
		StatusBackground:            vt100.BackgroundCyan,
		StatusErrorForeground:       vt100.Red,
		StatusErrorBackground:       vt100.BackgroundCyan,
		SearchHighlight:             vt100.Red,
		MultiLineComment:            vt100.White,
		MultiLineString:             vt100.White,
		HighlightForeground:         vt100.LightYellow,
		HighlightBackground:         vt100.BackgroundBlue,
		Git:                         vt100.White,
		String:                      "lightyellow",
		Keyword:                     "lightyellow",
		Comment:                     "white",
		Type:                        "white",
		Literal:                     "white",
		Punctuation:                 "white",
		Plaintext:                   "white",
		Tag:                         "white",
		TextTag:                     "white",
		TextAttrName:                "white",
		TextAttrValue:               "white",
		Decimal:                     "lightgreen",
		AndOr:                       "white",
		Dollar:                      "lightyellow",
		Star:                        "lightyellow",
		Static:                      "lightyellow",
		Self:                        "lightgreen",
		Class:                       "white",
		Private:                     "white",
		Protected:                   "white",
		Public:                      "white",
		Whitespace:                  "",
		AssemblyEnd:                 "white",
		Mut:                         "lightyellow",
		RainbowParenColors:          []vt100.AttributeColor{vt100.White, vt100.LightYellow},
		MarkdownTextColor:           vt100.White,
		HeaderBulletColor:           vt100.LightRed,
		HeaderTextColor:             vt100.White,
		ListBulletColor:             vt100.LightRed,
		ListTextColor:               vt100.White,
		ListCodeColor:               vt100.White,
		CodeColor:                   vt100.LightYellow,
		CodeBlockColor:              vt100.LightYellow,
		ImageColor:                  vt100.White,
		LinkColor:                   vt100.White,
		QuoteColor:                  vt100.LightYellow,
		QuoteTextColor:              vt100.LightYellow,
		HTMLColor:                   vt100.White,
		CommentColor:                vt100.LightYellow,
		BoldColor:                   vt100.White,
		ItalicsColor:                vt100.LightYellow,
		StrikeColor:                 vt100.LightYellow,
		TableColor:                  vt100.LightYellow,
		CheckboxColor:               vt100.White,
		XColor:                      vt100.White,
		TableBackground:             vt100.BackgroundBlue,
		UnmatchedParenColor:         vt100.LightRed,
		MenuTitleColor:              vt100.LightYellow,
		MenuArrowColor:              vt100.White,
		MenuTextColor:               vt100.LightGray,
		MenuHighlightColor:          vt100.LightYellow,
		MenuSelectedColor:           vt100.LightGreen,
		ManSectionColor:             vt100.White,
		ManSynopsisColor:            vt100.LightYellow,
		BoxTextColor:                vt100.LightYellow,
		BoxBackground:               vt100.LightYellow,
		BoxHighlight:                vt100.LightYellow,
		DebugRunningBackground:      vt100.BackgroundGray,
		DebugStoppedBackground:      vt100.BackgroundGray,
		DebugRegistersBackground:    vt100.BackgroundGray,
		DebugOutputBackground:       vt100.BackgroundGray,
		DebugInstructionsForeground: vt100.White,
		DebugInstructionsBackground: vt100.BackgroundGray,
		BoxUpperEdge:                vt100.LightYellow,
		JumpToLetterColor:           vt100.White,
		NanoHelpForeground:          vt100.Black,
		NanoHelpBackground:          vt100.BackgroundGray,
	}
}

// NewLightVSTheme creates a theme that is suitable for light xterm terminal emulator sessions
func NewLightVSTheme() Theme {
	return Theme{
		Name:                        "VS Light",
		Light:                       true,
		Foreground:                  vt100.Black,
		Background:                  vt100.BackgroundDefault,
		StatusForeground:            vt100.White,
		StatusBackground:            vt100.BackgroundBlack,
		StatusErrorForeground:       vt100.LightRed,
		StatusErrorBackground:       vt100.BackgroundDefault,
		SearchHighlight:             vt100.Red,
		MultiLineComment:            vt100.Gray,
		MultiLineString:             vt100.Red,
		HighlightForeground:         vt100.Red,
		HighlightBackground:         vt100.BackgroundDefault,
		Git:                         vt100.Blue,
		String:                      "red",
		Keyword:                     "blue",
		Comment:                     "gray",
		Type:                        "blue",
		Literal:                     "darkcyan",
		Punctuation:                 "black",
		Plaintext:                   "black",
		Tag:                         "black",
		TextTag:                     "black",
		TextAttrName:                "black",
		TextAttrValue:               "black",
		Decimal:                     "darkcyan",
		AndOr:                       "black",
		Dollar:                      "red",
		Star:                        "black",
		Static:                      "black",
		Self:                        "darkcyan",
		Class:                       "blue",
		Private:                     "black",
		Protected:                   "black",
		Public:                      "black",
		Whitespace:                  "",
		AssemblyEnd:                 "red",
		Mut:                         "black",
		RainbowParenColors:          []vt100.AttributeColor{vt100.Magenta, vt100.Black, vt100.Blue, vt100.Green},
		MarkdownTextColor:           vt100.Default,
		HeaderBulletColor:           vt100.DarkGray,
		HeaderTextColor:             vt100.Blue,
		ListBulletColor:             vt100.Red,
		ListTextColor:               vt100.Default,
		ListCodeColor:               vt100.Red,
		CodeColor:                   vt100.Red,
		CodeBlockColor:              vt100.Red,
		ImageColor:                  vt100.Green,
		LinkColor:                   vt100.Magenta,
		QuoteColor:                  vt100.Yellow,
		QuoteTextColor:              vt100.LightCyan,
		HTMLColor:                   vt100.Default,
		CommentColor:                vt100.DarkGray,
		BoldColor:                   vt100.Blue,
		ItalicsColor:                vt100.Blue,
		StrikeColor:                 vt100.DarkGray,
		TableColor:                  vt100.Blue,
		CheckboxColor:               vt100.Default,
		XColor:                      vt100.Blue,
		TableBackground:             vt100.BackgroundDefault,
		UnmatchedParenColor:         vt100.Red,
		MenuTitleColor:              vt100.Blue,
		MenuArrowColor:              vt100.Red,
		MenuTextColor:               vt100.Black,
		MenuHighlightColor:          vt100.Red,
		MenuSelectedColor:           vt100.LightRed,
		ManSectionColor:             vt100.Red,
		ManSynopsisColor:            vt100.Blue,
		BoxTextColor:                vt100.Black,
		BoxBackground:               vt100.BackgroundGray,
		BoxHighlight:                vt100.Red,
		DebugRunningBackground:      vt100.BackgroundCyan,
		DebugStoppedBackground:      vt100.BackgroundDefault,
		DebugRegistersBackground:    vt100.BackgroundGray,
		DebugOutputBackground:       vt100.BackgroundGray,
		DebugInstructionsForeground: vt100.Black,
		DebugInstructionsBackground: vt100.BackgroundGray,
		BoxUpperEdge:                vt100.Black,
		JumpToLetterColor:           vt100.Red,
		NanoHelpForeground:          vt100.Black,
		NanoHelpBackground:          vt100.BackgroundGray,
	}
}

// NewDarkVSTheme creates a theme that is suitable for dark terminal emulator sessions
func NewDarkVSTheme() Theme {
	return Theme{
		Name:                        "VS Dark",
		Light:                       false,
		Foreground:                  vt100.Black,
		Background:                  vt100.BackgroundWhite,
		StatusForeground:            vt100.White,
		StatusBackground:            vt100.BackgroundBlue,
		StatusErrorForeground:       vt100.Red,
		StatusErrorBackground:       vt100.BackgroundCyan,
		SearchHighlight:             vt100.Red,
		MultiLineComment:            vt100.Gray,
		MultiLineString:             vt100.Red,
		HighlightForeground:         vt100.Black,
		HighlightBackground:         vt100.BackgroundWhite,
		Git:                         vt100.Blue,
		String:                      "red",
		Keyword:                     "blue",
		Comment:                     "gray",
		Type:                        "blue",
		Literal:                     "darkcyan",
		Punctuation:                 "black",
		Plaintext:                   "black",
		Tag:                         "black",
		TextTag:                     "black",
		TextAttrName:                "black",
		TextAttrValue:               "black",
		Decimal:                     "darkcyan",
		AndOr:                       "black",
		Dollar:                      "red",
		Star:                        "red",
		Static:                      "red",
		Self:                        "darkcyan",
		Class:                       "blue",
		Private:                     "black",
		Protected:                   "black",
		Public:                      "black",
		Whitespace:                  "",
		AssemblyEnd:                 "red",
		Mut:                         "black",
		RainbowParenColors:          []vt100.AttributeColor{vt100.Magenta, vt100.Black, vt100.Blue, vt100.Green},
		MarkdownTextColor:           vt100.Black,
		HeaderBulletColor:           vt100.DarkGray,
		HeaderTextColor:             vt100.Blue,
		ListBulletColor:             vt100.Red,
		ListTextColor:               vt100.Black,
		ListCodeColor:               vt100.Red,
		CodeColor:                   vt100.Red,
		CodeBlockColor:              vt100.Red,
		ImageColor:                  vt100.DarkGray,
		LinkColor:                   vt100.Magenta,
		QuoteColor:                  vt100.Yellow,
		QuoteTextColor:              vt100.LightCyan,
		HTMLColor:                   vt100.Black,
		CommentColor:                vt100.DarkGray,
		BoldColor:                   vt100.Blue,
		ItalicsColor:                vt100.Blue,
		StrikeColor:                 vt100.DarkGray,
		TableColor:                  vt100.Blue,
		CheckboxColor:               vt100.Black,
		XColor:                      vt100.Blue,
		TableBackground:             vt100.DarkGray,
		UnmatchedParenColor:         vt100.Red,
		MenuTitleColor:              vt100.Blue,
		MenuArrowColor:              vt100.Red,
		MenuTextColor:               vt100.Black,
		MenuHighlightColor:          vt100.Red,
		MenuSelectedColor:           vt100.LightRed,
		ManSectionColor:             vt100.Red,
		ManSynopsisColor:            vt100.Blue,
		BoxTextColor:                vt100.Black,
		BoxBackground:               vt100.BackgroundGray,
		BoxHighlight:                vt100.Red,
		DebugRunningBackground:      vt100.BackgroundCyan,
		DebugStoppedBackground:      vt100.Gray,
		DebugRegistersBackground:    vt100.BackgroundGray,
		DebugOutputBackground:       vt100.BackgroundGray,
		DebugInstructionsForeground: vt100.Black,
		DebugInstructionsBackground: vt100.BackgroundGray,
		BoxUpperEdge:                vt100.Black,
		JumpToLetterColor:           vt100.Red,
		NanoHelpForeground:          vt100.Black,
		NanoHelpBackground:          vt100.BackgroundGray,
	}
}

// NewGrayTheme returns a theme where all text is light gray
func NewGrayTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Gray Mono"
	t.Foreground = vt100.LightGray
	t.Background = vt100.BackgroundDefault // black background
	//t.StatusBackground = vt100.BackgroundDefault
	//t.StatusErrorBackground = vt100.BackgroundDefault
	t.JumpToLetterColor = vt100.White // for jumping to a letter with ctrl-l
	return t
}

// NewAmberTheme returns a theme where all text is amber / yellow
func NewAmberTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Amber Mono"
	t.Foreground = vt100.Yellow
	t.Background = vt100.BackgroundDefault // black background
	t.JumpToLetterColor = t.Foreground     // for jumping to a letter with ctrl-l
	return t
}

// NewGreenTheme returns a theme where all text is green
func NewGreenTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Green Mono"
	t.Foreground = vt100.LightGreen
	t.Background = vt100.BackgroundDefault // black background
	t.JumpToLetterColor = t.Foreground     // for jumping to a letter with ctrl-l
	return t
}

// NewBlueTheme returns a theme where all text is blue
func NewBlueTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Blue Mono"
	t.Foreground = vt100.LightBlue
	t.Background = vt100.BackgroundDefault // black background
	t.JumpToLetterColor = t.Foreground     // for jumping to a letter with ctrl-l
	return t
}

// NewNoColorDarkBackgroundTheme creates a new theme without colors or syntax highlighting
func NewNoColorDarkBackgroundTheme() Theme {
	return Theme{
		Name:                        "No color",
		Light:                       false,
		Foreground:                  vt100.Default,
		Background:                  vt100.BackgroundDefault,
		StatusForeground:            vt100.White,
		StatusBackground:            vt100.BackgroundBlack,
		StatusErrorForeground:       vt100.White,
		StatusErrorBackground:       vt100.BackgroundDefault,
		SearchHighlight:             vt100.Default,
		MultiLineComment:            vt100.Default,
		MultiLineString:             vt100.Default,
		HighlightForeground:         vt100.White,
		HighlightBackground:         vt100.BackgroundDefault,
		Git:                         vt100.White,
		String:                      "",
		Keyword:                     "",
		Comment:                     "",
		Type:                        "",
		Literal:                     "",
		Punctuation:                 "",
		Plaintext:                   "",
		Tag:                         "",
		TextTag:                     "",
		TextAttrName:                "",
		TextAttrValue:               "",
		Decimal:                     "",
		AndOr:                       "",
		Dollar:                      "",
		Star:                        "",
		Static:                      "",
		Self:                        "",
		Class:                       "",
		Private:                     "",
		Protected:                   "",
		Public:                      "",
		Whitespace:                  "",
		AssemblyEnd:                 "",
		Mut:                         "",
		RainbowParenColors:          []vt100.AttributeColor{vt100.Gray},
		MarkdownTextColor:           vt100.Default,
		HeaderBulletColor:           vt100.Default,
		HeaderTextColor:             vt100.Default,
		ListBulletColor:             vt100.Default,
		ListTextColor:               vt100.Default,
		ListCodeColor:               vt100.Default,
		CodeColor:                   vt100.Default,
		CodeBlockColor:              vt100.Default,
		ImageColor:                  vt100.Default,
		LinkColor:                   vt100.Default,
		QuoteColor:                  vt100.Default,
		QuoteTextColor:              vt100.Default,
		HTMLColor:                   vt100.Default,
		CommentColor:                vt100.Default,
		BoldColor:                   vt100.Default,
		ItalicsColor:                vt100.Default,
		StrikeColor:                 vt100.Default,
		TableColor:                  vt100.Default,
		CheckboxColor:               vt100.Default,
		XColor:                      vt100.White,
		TableBackground:             vt100.BackgroundDefault,
		UnmatchedParenColor:         vt100.White,
		MenuTitleColor:              vt100.White,
		MenuArrowColor:              vt100.White,
		MenuTextColor:               vt100.Gray,
		MenuHighlightColor:          vt100.White,
		MenuSelectedColor:           vt100.Black,
		ManSectionColor:             vt100.White,
		ManSynopsisColor:            vt100.White,
		BoxTextColor:                vt100.Black,
		BoxBackground:               vt100.BackgroundGray,
		BoxHighlight:                vt100.Black,
		DebugRunningBackground:      vt100.BackgroundGray,
		DebugStoppedBackground:      vt100.BackgroundGray,
		DebugRegistersBackground:    vt100.BackgroundGray,
		DebugOutputBackground:       vt100.BackgroundGray,
		DebugInstructionsForeground: vt100.Black,
		DebugInstructionsBackground: vt100.BackgroundGray,
		BoxUpperEdge:                vt100.Black,
		JumpToLetterColor:           vt100.White,
		NanoHelpForeground:          vt100.Black,
		NanoHelpBackground:          vt100.BackgroundGray,
	}
}

// NewNoColorLightBackgroundTheme creates a new theme without colors or syntax highlighting
func NewNoColorLightBackgroundTheme() Theme {
	return Theme{
		Name:                        "No color",
		Light:                       true,
		Foreground:                  vt100.Default,
		Background:                  vt100.BackgroundDefault,
		StatusForeground:            vt100.Black,
		StatusBackground:            vt100.BackgroundWhite,
		StatusErrorForeground:       vt100.Black,
		StatusErrorBackground:       vt100.BackgroundDefault,
		SearchHighlight:             vt100.Default,
		MultiLineComment:            vt100.Default,
		MultiLineString:             vt100.Default,
		HighlightForeground:         vt100.Default,
		HighlightBackground:         vt100.BackgroundDefault,
		Git:                         vt100.Black,
		String:                      "",
		Keyword:                     "",
		Comment:                     "",
		Type:                        "",
		Literal:                     "",
		Punctuation:                 "",
		Plaintext:                   "",
		Tag:                         "",
		TextTag:                     "",
		TextAttrName:                "",
		TextAttrValue:               "",
		Decimal:                     "",
		AndOr:                       "",
		Dollar:                      "",
		Star:                        "",
		Static:                      "",
		Self:                        "",
		Class:                       "",
		Private:                     "",
		Protected:                   "",
		Public:                      "",
		Whitespace:                  "",
		AssemblyEnd:                 "",
		Mut:                         "",
		RainbowParenColors:          []vt100.AttributeColor{vt100.Gray},
		MarkdownTextColor:           vt100.Default,
		HeaderBulletColor:           vt100.Default,
		HeaderTextColor:             vt100.Default,
		ListBulletColor:             vt100.Default,
		ListTextColor:               vt100.Default,
		ListCodeColor:               vt100.Default,
		CodeColor:                   vt100.Default,
		CodeBlockColor:              vt100.Default,
		ImageColor:                  vt100.Default,
		LinkColor:                   vt100.Default,
		QuoteColor:                  vt100.Default,
		QuoteTextColor:              vt100.Default,
		HTMLColor:                   vt100.Default,
		CommentColor:                vt100.Default,
		BoldColor:                   vt100.Default,
		ItalicsColor:                vt100.Default,
		StrikeColor:                 vt100.Default,
		TableColor:                  vt100.Default,
		CheckboxColor:               vt100.Default,
		XColor:                      vt100.Black,
		TableBackground:             vt100.BackgroundDefault,
		UnmatchedParenColor:         vt100.Black,
		MenuTitleColor:              vt100.Black,
		MenuArrowColor:              vt100.Black,
		MenuTextColor:               vt100.Gray,
		MenuHighlightColor:          vt100.Black,
		MenuSelectedColor:           vt100.White,
		ManSectionColor:             vt100.Black,
		ManSynopsisColor:            vt100.Black,
		BoxTextColor:                vt100.White,
		BoxBackground:               vt100.BackgroundGray,
		BoxHighlight:                vt100.White,
		DebugRunningBackground:      vt100.BackgroundGray,
		DebugStoppedBackground:      vt100.BackgroundGray,
		DebugRegistersBackground:    vt100.BackgroundGray,
		DebugOutputBackground:       vt100.BackgroundGray,
		DebugInstructionsForeground: vt100.White,
		DebugInstructionsBackground: vt100.BackgroundGray,
		BoxUpperEdge:                vt100.White,
		JumpToLetterColor:           vt100.Black,
		NanoHelpBackground:          vt100.BackgroundGray,
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
		Static:        t.Static,
		Self:          t.Self,
		Class:         t.Class,
		Private:       t.Private,
		Protected:     t.Protected,
		Public:        t.Public,
		Whitespace:    t.Whitespace,
		AssemblyEnd:   t.AssemblyEnd,
		Mut:           t.Mut,
	}
}

// setDefaultTheme sets the default colors
func (e *Editor) setDefaultTheme() {
	e.SetTheme(NewDefaultTheme())
}

// setVSTheme sets the VS theme
func (e *Editor) setVSTheme(bs ...bool) {
	if len(bs) == 1 {
		initialLightBackground = &(bs[0])
	}
	if initialLightBackground != nil && *initialLightBackground {
		e.SetTheme(NewLightVSTheme())
	} else {
		e.SetTheme(NewDarkVSTheme())
	}
}

// SetTheme assigns the given theme to the Editor,
// and also configures syntax highlighting by setting syntax.DefaultTextConfig.
// Light/dark, syntax highlighting and no color information is also set.
// Respect the NO_COLOR environment variable. May set e.NoSyntaxHighlight to true.
func (e *Editor) SetTheme(theme Theme, bs ...bool) {
	if envNoColor {
		if initialLightBackground != nil && *initialLightBackground {
			theme = NewNoColorLightBackgroundTheme()
		} else {
			theme = NewNoColorDarkBackgroundTheme()
		}
		e.syntaxHighlight = false
	} else if len(bs) == 1 {
		initialLightBackground = &(bs[0])
	}
	e.Theme = theme
	e.statusMode = theme.StatusMode
	syntax.DefaultTextConfig = *(theme.TextConfig())
}

// setNoColorTheme sets the NoColor theme, and considers the background color
func (e *Editor) setNoColorTheme() {
	if initialLightBackground != nil && *initialLightBackground {
		e.Theme = NewNoColorLightBackgroundTheme()
	} else {
		e.Theme = NewNoColorDarkBackgroundTheme()
	}
	e.statusMode = e.Theme.StatusMode
	syntax.DefaultTextConfig = *(e.Theme.TextConfig())
}

// setLightVSTheme sets the light theme suitable for xterm
func (e *Editor) setLightVSTheme() {
	e.SetTheme(NewLightVSTheme())
}

// setBlueEditTheme sets a blue/yellow/gray theme, for light or dark backgrounds
// if given "true" as an argument, then a light background is assumed
func (e *Editor) setBlueEditTheme(bs ...bool) {
	if len(bs) == 1 {
		initialLightBackground = &(bs[0])
	}
	if initialLightBackground != nil && *initialLightBackground {
		e.SetTheme(NewLightBlueEditTheme())
	} else {
		e.SetTheme(NewDarkBlueEditTheme())
	}
}

// setGratTheme sets a gray theme
func (e *Editor) setGrayTheme() {
	e.SetTheme(NewGrayTheme())
}

// setAmberTheme sets an amber theme
func (e *Editor) setAmberTheme() {
	e.SetTheme(NewAmberTheme())
}

// setGreenTheme sets a green theme
func (e *Editor) setGreenTheme() {
	e.SetTheme(NewGreenTheme())
}

// setBlueTheme sets a blue theme
func (e *Editor) setBlueTheme() {
	e.SetTheme(NewBlueTheme())
}
