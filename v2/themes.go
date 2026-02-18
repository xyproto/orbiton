package main

import "github.com/xyproto/vt"

// TODO: Restructure how themes are stored, so that it's easier to list all themes that
// works with a dark background or all that works with a light background, ref. initialLightBackground

var initialLightBackground *bool

// Theme contains information about:
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
	AngleBracket                string
	TextTag                     string
	TextAttrName                string
	RainbowParenColors          []vt.AttributeColor
	HeaderBulletColor           vt.AttributeColor
	MultiLineString             vt.AttributeColor
	DebugInstructionsBackground vt.AttributeColor
	Git                         vt.AttributeColor
	MultiLineComment            vt.AttributeColor
	SearchHighlight             vt.AttributeColor
	StatusErrorForeground       vt.AttributeColor
	StatusErrorBackground       vt.AttributeColor
	StatusForeground            vt.AttributeColor
	StatusBackground            vt.AttributeColor
	TopRightForeground          vt.AttributeColor
	TopRightBackground          vt.AttributeColor
	Foreground                  vt.AttributeColor
	Background                  vt.AttributeColor
	MarkdownTextColor           vt.AttributeColor
	BoxUpperEdge                vt.AttributeColor
	HeaderTextColor             vt.AttributeColor
	ListBulletColor             vt.AttributeColor
	ListTextColor               vt.AttributeColor
	ListCodeColor               vt.AttributeColor
	CodeColor                   vt.AttributeColor
	CodeBlockColor              vt.AttributeColor
	ImageColor                  vt.AttributeColor
	LinkColor                   vt.AttributeColor
	QuoteColor                  vt.AttributeColor
	QuoteTextColor              vt.AttributeColor
	HTMLColor                   vt.AttributeColor
	CommentColor                vt.AttributeColor
	BoldColor                   vt.AttributeColor
	ItalicsColor                vt.AttributeColor
	StrikeColor                 vt.AttributeColor
	TableColor                  vt.AttributeColor
	CheckboxColor               vt.AttributeColor
	XColor                      vt.AttributeColor
	DebugInstructionsForeground vt.AttributeColor
	UnmatchedParenColor         vt.AttributeColor
	MenuTitleColor              vt.AttributeColor
	MenuArrowColor              vt.AttributeColor
	MenuTextColor               vt.AttributeColor
	MenuHighlightColor          vt.AttributeColor
	MenuSelectedColor           vt.AttributeColor
	ManSectionColor             vt.AttributeColor
	ManSynopsisColor            vt.AttributeColor
	BoxTextColor                vt.AttributeColor
	BoxBackground               vt.AttributeColor
	ProgressIndicatorBackground vt.AttributeColor
	BoxHighlight                vt.AttributeColor
	DebugRunningBackground      vt.AttributeColor
	DebugStoppedBackground      vt.AttributeColor
	DebugRegistersBackground    vt.AttributeColor
	DebugOutputBackground       vt.AttributeColor
	TableBackground             vt.AttributeColor
	JumpToLetterColor           vt.AttributeColor
	NanoHelpForeground          vt.AttributeColor
	NanoHelpBackground          vt.AttributeColor
	HighlightForeground         vt.AttributeColor
	HighlightBackground         vt.AttributeColor
	StatusMode                  bool
	Light                       bool
}

// NewDefaultTheme creates a new default Theme struct
func NewDefaultTheme() Theme {
	return Theme{
		Name:                        "Default",
		Light:                       false,
		Foreground:                  vt.LightBlue,
		Background:                  vt.BackgroundDefault,
		StatusForeground:            vt.White,
		StatusBackground:            vt.BackgroundBlack,
		TopRightForeground:          vt.White,
		TopRightBackground:          vt.BackgroundDefault,
		StatusErrorForeground:       vt.LightRed,
		StatusErrorBackground:       vt.BackgroundDefault,
		SearchHighlight:             vt.LightMagenta,
		MultiLineComment:            vt.Gray,
		MultiLineString:             vt.Magenta,
		HighlightForeground:         vt.White,
		HighlightBackground:         vt.BackgroundDefault,
		Git:                         vt.LightGreen,
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
		AngleBracket:                "lightyellow",
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
		RainbowParenColors:          []vt.AttributeColor{vt.LightMagenta, vt.LightRed, vt.Yellow, vt.LightYellow, vt.LightGreen, vt.LightBlue, vt.Red},
		MarkdownTextColor:           vt.LightBlue,
		HeaderBulletColor:           vt.DarkGray,
		HeaderTextColor:             vt.LightGreen,
		ListBulletColor:             vt.Red,
		ListTextColor:               vt.LightCyan,
		ListCodeColor:               vt.Default,
		CodeColor:                   vt.Default,
		CodeBlockColor:              vt.Default,
		ImageColor:                  vt.LightYellow,
		LinkColor:                   vt.Magenta,
		QuoteColor:                  vt.Yellow,
		QuoteTextColor:              vt.LightCyan,
		HTMLColor:                   vt.Default,
		CommentColor:                vt.DarkGray,
		BoldColor:                   vt.LightYellow,
		ItalicsColor:                vt.White,
		StrikeColor:                 vt.DarkGray,
		TableColor:                  vt.Blue,
		CheckboxColor:               vt.Default,
		XColor:                      vt.LightYellow,
		TableBackground:             vt.BackgroundDefault,
		UnmatchedParenColor:         vt.White,
		MenuTitleColor:              vt.LightYellow,
		MenuArrowColor:              vt.Red,
		MenuTextColor:               vt.Gray,
		MenuHighlightColor:          vt.LightBlue,
		MenuSelectedColor:           vt.LightCyan,
		ManSectionColor:             vt.LightRed,
		ManSynopsisColor:            vt.LightYellow,
		BoxTextColor:                vt.Black,
		BoxBackground:               vt.BackgroundBlue,
		ProgressIndicatorBackground: vt.BackgroundBlue,
		BoxHighlight:                vt.LightYellow,
		DebugRunningBackground:      vt.BackgroundCyan,
		DebugStoppedBackground:      vt.BackgroundMagenta,
		DebugRegistersBackground:    vt.BackgroundBlue,
		DebugOutputBackground:       vt.BackgroundYellow,
		DebugInstructionsForeground: vt.LightYellow,
		DebugInstructionsBackground: vt.BackgroundMagenta,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.LightRed,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// NewOrbTheme creates a new "logical looking" theme
func NewOrbTheme() Theme {
	return Theme{
		Name:                        "Orb",
		Light:                       false,
		Foreground:                  vt.LightGray,
		Background:                  vt.BackgroundBlack,
		StatusForeground:            vt.LightGray,
		StatusBackground:            vt.Gray,
		TopRightForeground:          vt.LightGray,
		TopRightBackground:          vt.BackgroundBlack,
		StatusErrorForeground:       vt.LightRed,
		StatusErrorBackground:       vt.BackgroundBlack,
		SearchHighlight:             vt.LightMagenta,
		MultiLineComment:            vt.Gray,
		MultiLineString:             vt.LightCyan,
		HighlightForeground:         vt.White,
		HighlightBackground:         vt.BackgroundBlack,
		Git:                         vt.LightCyan,
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
		AngleBracket:                "lightcyan",
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
		RainbowParenColors:          []vt.AttributeColor{vt.LightRed, vt.LightCyan, vt.LightGreen, vt.LightYellow, vt.LightBlue, vt.Gray, vt.LightGray},
		MarkdownTextColor:           vt.LightGray,
		HeaderBulletColor:           vt.White,
		HeaderTextColor:             vt.LightCyan,
		ListBulletColor:             vt.LightRed,
		ListTextColor:               vt.LightCyan,
		ListCodeColor:               vt.White,
		CodeColor:                   vt.White,
		CodeBlockColor:              vt.White,
		ImageColor:                  vt.LightGreen,
		LinkColor:                   vt.LightCyan,
		QuoteColor:                  vt.LightGreen,
		QuoteTextColor:              vt.White,
		HTMLColor:                   vt.White,
		CommentColor:                vt.Gray,
		BoldColor:                   vt.LightGreen,
		ItalicsColor:                vt.LightGray,
		StrikeColor:                 vt.White,
		TableColor:                  vt.White,
		CheckboxColor:               vt.White,
		XColor:                      vt.LightGreen,
		TableBackground:             vt.BackgroundBlack,
		UnmatchedParenColor:         vt.LightRed,
		MenuTitleColor:              vt.Blue,
		MenuArrowColor:              vt.LightMagenta,
		MenuTextColor:               vt.LightCyan,
		MenuHighlightColor:          vt.White,
		MenuSelectedColor:           vt.LightRed,
		ManSectionColor:             vt.LightCyan,
		ManSynopsisColor:            vt.LightGreen,
		BoxTextColor:                vt.White,
		BoxBackground:               vt.Black,
		ProgressIndicatorBackground: vt.BackgroundGreen,
		BoxHighlight:                vt.LightYellow,
		DebugRunningBackground:      vt.Cyan,
		DebugStoppedBackground:      vt.BackgroundRed,
		DebugRegistersBackground:    vt.DarkGray,
		DebugOutputBackground:       vt.LightGreen,
		DebugInstructionsForeground: vt.LightGreen,
		DebugInstructionsBackground: vt.DarkGray,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.LightRed,
		NanoHelpForeground:          vt.White,
		NanoHelpBackground:          vt.DarkGray,
	}
}

// NewPinetreeTheme creates a new Theme struct based on the base16-snazzy theme
func NewPinetreeTheme() Theme {
	return Theme{
		Name:                        "Pinetree",
		Light:                       false,
		Foreground:                  vt.LightGray,
		Background:                  vt.BackgroundBlack,
		StatusForeground:            vt.LightGray,
		StatusBackground:            vt.BackgroundBlack,
		TopRightForeground:          vt.LightGray,
		TopRightBackground:          vt.BackgroundBlack,
		StatusErrorForeground:       vt.LightRed,
		StatusErrorBackground:       vt.BackgroundBlack,
		SearchHighlight:             vt.Yellow,
		MultiLineComment:            vt.DarkGray,
		MultiLineString:             vt.Magenta,
		HighlightForeground:         vt.LightCyan,
		HighlightBackground:         vt.BackgroundBlack,
		Git:                         vt.LightGreen,
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
		AngleBracket:                "lightred",
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
		RainbowParenColors:          []vt.AttributeColor{vt.LightMagenta, vt.LightRed, vt.Yellow, vt.LightYellow, vt.LightGreen, vt.LightBlue, vt.Red},
		MarkdownTextColor:           vt.LightGray,
		HeaderBulletColor:           vt.DarkGray,
		HeaderTextColor:             vt.LightBlue,
		ListBulletColor:             vt.LightRed,
		ListTextColor:               vt.LightGray,
		ListCodeColor:               vt.LightGreen,
		CodeColor:                   vt.LightGreen,
		CodeBlockColor:              vt.BackgroundBlack,
		ImageColor:                  vt.Yellow,
		LinkColor:                   vt.LightBlue,
		QuoteColor:                  vt.Yellow,
		QuoteTextColor:              vt.LightGray,
		HTMLColor:                   vt.LightRed,
		CommentColor:                vt.DarkGray,
		BoldColor:                   vt.White,
		ItalicsColor:                vt.LightBlue,
		StrikeColor:                 vt.DarkGray,
		TableColor:                  vt.LightBlue,
		CheckboxColor:               vt.LightGray,
		XColor:                      vt.LightRed,
		TableBackground:             vt.BackgroundBlack,
		UnmatchedParenColor:         vt.LightRed,
		MenuTitleColor:              vt.LightGreen,
		MenuArrowColor:              vt.LightRed,
		MenuTextColor:               vt.LightGray,
		MenuHighlightColor:          vt.LightCyan,
		MenuSelectedColor:           vt.White,
		ManSectionColor:             vt.LightRed,
		ManSynopsisColor:            vt.Yellow,
		BoxTextColor:                vt.Black,
		BoxBackground:               vt.BackgroundBlue,
		ProgressIndicatorBackground: vt.BackgroundBlue,
		BoxHighlight:                vt.LightYellow,
		DebugRunningBackground:      vt.BackgroundGreen,
		DebugStoppedBackground:      vt.BackgroundMagenta,
		DebugRegistersBackground:    vt.BackgroundBlue,
		DebugOutputBackground:       vt.BackgroundYellow,
		DebugInstructionsForeground: vt.LightYellow,
		DebugInstructionsBackground: vt.BackgroundMagenta,
		BoxUpperEdge:                vt.LightGray,
		JumpToLetterColor:           vt.LightRed,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// NewZuluTheme creates a unique semantic Theme with nature-inspired colors
func NewZuluTheme() Theme {
	return Theme{
		Name:                        "Zulu",
		Light:                       false,
		Foreground:                  vt.Default,
		Background:                  vt.BackgroundDefault,
		StatusForeground:            vt.White,
		StatusBackground:            vt.BackgroundBlack,
		TopRightForeground:          vt.White,
		TopRightBackground:          vt.BackgroundDefault,
		StatusErrorForeground:       vt.LightRed,
		StatusErrorBackground:       vt.BackgroundDefault,
		SearchHighlight:             vt.Yellow,
		MultiLineComment:            vt.Gray,
		MultiLineString:             vt.LightYellow,
		HighlightForeground:         vt.LightCyan,
		HighlightBackground:         vt.BackgroundDefault,
		Git:                         vt.LightGreen,
		String:                      "lightyellow",
		Keyword:                     "lightgreen",
		Comment:                     "gray",
		Type:                        "lightcyan",
		Literal:                     "lightmagenta",
		Punctuation:                 "lightgray",
		Plaintext:                   "lightgray",
		Tag:                         "lightcyan",
		TextTag:                     "lightcyan",
		TextAttrName:                "lightcyan",
		TextAttrValue:               "lightyellow",
		Decimal:                     "lightmagenta",
		AndOr:                       "lightgreen",
		AngleBracket:                "lightgreen",
		Dollar:                      "lightmagenta",
		Star:                        "lightyellow",
		Static:                      "lightgreen",
		Self:                        "white",
		Class:                       "lightcyan",
		Private:                     "darkyellow",
		Protected:                   "yellow",
		Public:                      "lightgreen",
		Whitespace:                  "",
		AssemblyEnd:                 "lightcyan",
		Mut:                         "lightgreen",
		RainbowParenColors:          []vt.AttributeColor{vt.LightYellow, vt.LightGreen, vt.LightCyan, vt.LightMagenta, vt.White},
		MarkdownTextColor:           vt.LightGray,
		HeaderBulletColor:           vt.Gray,
		HeaderTextColor:             vt.LightCyan,
		ListBulletColor:             vt.LightMagenta,
		ListTextColor:               vt.LightGray,
		ListCodeColor:               vt.LightYellow,
		CodeColor:                   vt.LightYellow,
		CodeBlockColor:              vt.BackgroundDefault,
		ImageColor:                  vt.LightMagenta,
		LinkColor:                   vt.LightCyan,
		QuoteColor:                  vt.Yellow,
		QuoteTextColor:              vt.LightGray,
		HTMLColor:                   vt.LightCyan,
		CommentColor:                vt.Gray,
		BoldColor:                   vt.White,
		ItalicsColor:                vt.LightCyan,
		StrikeColor:                 vt.Gray,
		TableColor:                  vt.LightCyan,
		CheckboxColor:               vt.LightGray,
		XColor:                      vt.LightMagenta,
		TableBackground:             vt.BackgroundDefault,
		UnmatchedParenColor:         vt.LightRed,
		MenuTitleColor:              vt.LightCyan,
		MenuArrowColor:              vt.LightGreen,
		MenuTextColor:               vt.Default,
		MenuHighlightColor:          vt.LightYellow,
		MenuSelectedColor:           vt.White,
		ManSectionColor:             vt.LightMagenta,
		ManSynopsisColor:            vt.LightCyan,
		BoxTextColor:                vt.White,
		BoxBackground:               vt.BackgroundBlack,
		ProgressIndicatorBackground: vt.BackgroundGray,
		BoxHighlight:                vt.LightYellow,
		DebugRunningBackground:      vt.BackgroundGreen,
		DebugStoppedBackground:      vt.BackgroundMagenta,
		DebugRegistersBackground:    vt.BackgroundCyan,
		DebugOutputBackground:       vt.BackgroundYellow,
		DebugInstructionsForeground: vt.LightYellow,
		DebugInstructionsBackground: vt.BackgroundBlack,
		BoxUpperEdge:                vt.LightGray,
		JumpToLetterColor:           vt.LightMagenta,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// NewLitmusTheme creates a new default Theme struct
func NewLitmusTheme() Theme {
	return Theme{
		Name:                        "Litmus",
		Light:                       false,
		Foreground:                  vt.Default,
		Background:                  vt.BackgroundGray,
		StatusForeground:            vt.Gray,
		StatusBackground:            vt.BackgroundBlack,
		TopRightForeground:          vt.Black,
		TopRightBackground:          vt.BackgroundGray,
		StatusErrorForeground:       vt.LightRed,
		StatusErrorBackground:       vt.BackgroundDefault,
		SearchHighlight:             vt.LightMagenta,
		MultiLineComment:            vt.Gray,
		MultiLineString:             vt.Magenta,
		HighlightForeground:         vt.LightRed,
		HighlightBackground:         vt.BackgroundGray,
		Git:                         vt.Black,
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
		AngleBracket:                "lightred",
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
		RainbowParenColors:          []vt.AttributeColor{vt.LightMagenta, vt.LightRed, vt.Yellow, vt.Green, vt.Blue, vt.LightBlue, vt.Red},
		MarkdownTextColor:           vt.Black,
		HeaderBulletColor:           vt.DarkGray,
		HeaderTextColor:             vt.Black,
		ListBulletColor:             vt.Red,
		ListTextColor:               vt.LightBlue,
		ListCodeColor:               vt.Black,
		CodeColor:                   vt.Black,
		CodeBlockColor:              vt.Black,
		ImageColor:                  vt.Red,
		LinkColor:                   vt.Magenta,
		QuoteColor:                  vt.Red,
		QuoteTextColor:              vt.LightBlue,
		HTMLColor:                   vt.Black,
		CommentColor:                vt.DarkGray,
		BoldColor:                   vt.Red,
		ItalicsColor:                vt.DarkGray,
		StrikeColor:                 vt.DarkGray,
		TableColor:                  vt.Black,
		CheckboxColor:               vt.Black,
		XColor:                      vt.Red,
		TableBackground:             vt.BackgroundGray,
		UnmatchedParenColor:         vt.Yellow,
		MenuTitleColor:              vt.Black,
		MenuArrowColor:              vt.Red,
		MenuTextColor:               vt.Gray,
		MenuHighlightColor:          vt.Cyan,
		MenuSelectedColor:           vt.LightBlue,
		ManSectionColor:             vt.LightRed,
		ManSynopsisColor:            vt.Red,
		BoxTextColor:                vt.Black,
		BoxBackground:               vt.BackgroundCyan,
		ProgressIndicatorBackground: vt.BackgroundCyan,
		BoxHighlight:                vt.Red,
		DebugRunningBackground:      vt.BackgroundBlue,
		DebugStoppedBackground:      vt.BackgroundMagenta,
		DebugRegistersBackground:    vt.BackgroundCyan,
		DebugOutputBackground:       vt.BackgroundGray,
		DebugInstructionsForeground: vt.Red,
		DebugInstructionsBackground: vt.BackgroundMagenta,
		BoxUpperEdge:                vt.DarkGray,
		JumpToLetterColor:           vt.LightRed,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// NewSynthwaveTheme creates a new Theme struct
func NewSynthwaveTheme() Theme {
	return Theme{
		Name:                        "Synthwave",
		Light:                       false,
		Foreground:                  vt.LightBlue,
		Background:                  vt.BackgroundDefault,
		StatusForeground:            vt.White,
		StatusBackground:            vt.BackgroundBlack,
		TopRightForeground:          vt.Cyan,
		TopRightBackground:          vt.BackgroundDefault,
		StatusErrorForeground:       vt.Magenta,
		StatusErrorBackground:       vt.BackgroundDefault,
		SearchHighlight:             vt.LightMagenta,
		MultiLineComment:            vt.Gray,
		MultiLineString:             vt.Magenta,
		HighlightForeground:         vt.White,
		HighlightBackground:         vt.BackgroundDefault,
		Git:                         vt.Cyan,
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
		AngleBracket:                "lightgray",
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
		RainbowParenColors:          []vt.AttributeColor{vt.LightRed, vt.LightMagenta, vt.Blue, vt.LightCyan, vt.LightBlue, vt.Magenta, vt.Cyan},
		MarkdownTextColor:           vt.LightBlue,
		HeaderBulletColor:           vt.DarkGray,
		HeaderTextColor:             vt.Cyan,
		ListBulletColor:             vt.Magenta,
		ListTextColor:               vt.LightCyan,
		ListCodeColor:               vt.Default,
		CodeColor:                   vt.Default,
		CodeBlockColor:              vt.Default,
		ImageColor:                  vt.LightGray,
		LinkColor:                   vt.LightMagenta,
		QuoteColor:                  vt.Gray,
		QuoteTextColor:              vt.LightCyan,
		HTMLColor:                   vt.Default,
		CommentColor:                vt.DarkGray,
		BoldColor:                   vt.LightGray,
		ItalicsColor:                vt.White,
		StrikeColor:                 vt.DarkGray,
		TableColor:                  vt.Blue,
		CheckboxColor:               vt.Default,
		XColor:                      vt.LightGray,
		TableBackground:             vt.BackgroundDefault,
		UnmatchedParenColor:         vt.LightRed, // to really stand out
		MenuTitleColor:              vt.Cyan,
		MenuArrowColor:              vt.Magenta,
		MenuTextColor:               vt.Gray,
		MenuHighlightColor:          vt.LightBlue,
		MenuSelectedColor:           vt.LightCyan,
		ManSectionColor:             vt.LightMagenta,
		ManSynopsisColor:            vt.LightGray,
		BoxTextColor:                vt.Black,
		BoxBackground:               vt.BackgroundBlue,
		ProgressIndicatorBackground: vt.BackgroundBlue,
		BoxHighlight:                vt.LightGray,
		DebugRunningBackground:      vt.BackgroundCyan,
		DebugStoppedBackground:      vt.BackgroundRed,
		DebugRegistersBackground:    vt.BackgroundBlue,
		DebugOutputBackground:       vt.BackgroundGray,
		DebugInstructionsForeground: vt.LightGray,
		DebugInstructionsBackground: vt.BackgroundRed,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.LightMagenta,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// NewTealTheme creates a blue, white, gray and teal theme
func NewTealTheme() Theme {
	return Theme{
		Name:                        "Teal",
		Light:                       false,
		Foreground:                  vt.White,
		Background:                  vt.BackgroundBlack,
		StatusForeground:            vt.White,
		StatusBackground:            vt.BackgroundBlack,
		TopRightForeground:          vt.White,
		TopRightBackground:          vt.BackgroundBlack,
		StatusErrorForeground:       vt.LightRed,
		StatusErrorBackground:       vt.BackgroundDefault,
		SearchHighlight:             vt.Yellow,
		MultiLineComment:            vt.Gray,
		MultiLineString:             vt.LightBlue,
		HighlightForeground:         vt.LightCyan,
		HighlightBackground:         vt.BackgroundBlack,
		Git:                         vt.LightBlue,
		String:                      "lightblue",
		Keyword:                     "white",
		Comment:                     "gray",
		Type:                        "lightcyan",
		Literal:                     "white",
		Punctuation:                 "white",
		Plaintext:                   "white",
		Tag:                         "white",
		TextTag:                     "white",
		TextAttrName:                "white",
		TextAttrValue:               "lightblue",
		Decimal:                     "white",
		AndOr:                       "white",
		AngleBracket:                "white",
		Dollar:                      "white",
		Star:                        "white",
		Static:                      "white",
		Self:                        "white",
		Class:                       "lightcyan",
		Private:                     "lightgray",
		Protected:                   "lightgray",
		Public:                      "white",
		Whitespace:                  "",
		AssemblyEnd:                 "white",
		Mut:                         "white",
		RainbowParenColors:          []vt.AttributeColor{vt.White, vt.LightCyan, vt.Gray, vt.LightBlue, vt.Blue},
		MarkdownTextColor:           vt.White,
		HeaderBulletColor:           vt.Gray,
		HeaderTextColor:             vt.White,
		ListBulletColor:             vt.White,
		ListTextColor:               vt.White,
		ListCodeColor:               vt.LightBlue,
		CodeColor:                   vt.LightBlue,
		CodeBlockColor:              vt.BackgroundDefault,
		ImageColor:                  vt.White,
		LinkColor:                   vt.LightBlue,
		QuoteColor:                  vt.LightGray,
		QuoteTextColor:              vt.White,
		HTMLColor:                   vt.White,
		CommentColor:                vt.Gray,
		BoldColor:                   vt.White,
		ItalicsColor:                vt.LightBlue,
		StrikeColor:                 vt.Gray,
		TableColor:                  vt.White,
		CheckboxColor:               vt.White,
		XColor:                      vt.White,
		TableBackground:             vt.BackgroundDefault,
		UnmatchedParenColor:         vt.LightRed,
		MenuTitleColor:              vt.Blue,
		MenuArrowColor:              vt.LightCyan,
		MenuTextColor:               vt.White,
		MenuHighlightColor:          vt.LightCyan,
		MenuSelectedColor:           vt.LightRed,
		ManSectionColor:             vt.White,
		ManSynopsisColor:            vt.White,
		BoxTextColor:                vt.White,
		BoxBackground:               vt.BackgroundBlack,
		ProgressIndicatorBackground: vt.BackgroundBlue,
		BoxHighlight:                vt.LightBlue,
		DebugRunningBackground:      vt.BackgroundBlue,
		DebugStoppedBackground:      vt.BackgroundRed,
		DebugRegistersBackground:    vt.BackgroundCyan,
		DebugOutputBackground:       vt.BackgroundGray,
		DebugInstructionsForeground: vt.White,
		DebugInstructionsBackground: vt.BackgroundBlack,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.White,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// NewRedBlackTheme creates a new red/black/gray/white Theme struct
func NewRedBlackTheme() Theme {
	// NOTE: Dark gray may not be visible with light terminal emulator themes
	return Theme{
		Name:                        "Red & black",
		Light:                       false,
		Foreground:                  vt.LightGray,
		Background:                  vt.BackgroundBlack, // Dark gray background, as opposed tovt.BackgroundDefault
		StatusForeground:            vt.White,
		StatusBackground:            vt.BackgroundBlack,
		TopRightForeground:          vt.White,
		TopRightBackground:          vt.BackgroundBlack,
		StatusErrorForeground:       vt.LightRed,
		StatusErrorBackground:       vt.BackgroundDefault,
		SearchHighlight:             vt.Red,
		MultiLineComment:            vt.DarkGray,
		MultiLineString:             vt.LightGray,
		HighlightForeground:         vt.LightGray,
		HighlightBackground:         vt.BackgroundBlack,
		Git:                         vt.LightGreen,
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
		AngleBracket:                "darkred",
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
		RainbowParenColors:          []vt.AttributeColor{vt.LightGray, vt.White, vt.Red},
		MarkdownTextColor:           vt.LightGray,
		HeaderBulletColor:           vt.DarkGray,
		HeaderTextColor:             vt.Red,
		ListBulletColor:             vt.Red,
		ListTextColor:               vt.LightGray,
		ListCodeColor:               vt.Default,
		CodeColor:                   vt.White,
		CodeBlockColor:              vt.White,
		ImageColor:                  vt.Red,
		LinkColor:                   vt.DarkGray,
		QuoteColor:                  vt.White,
		QuoteTextColor:              vt.LightGray,
		HTMLColor:                   vt.LightGray,
		CommentColor:                vt.DarkGray,
		BoldColor:                   vt.Red,
		ItalicsColor:                vt.Black,
		StrikeColor:                 vt.DarkGray,
		TableColor:                  vt.White,
		CheckboxColor:               vt.Default,
		XColor:                      vt.Red,
		TableBackground:             vt.BackgroundBlack, // Dark gray background, as opposed tovt.BackgroundDefault
		UnmatchedParenColor:         vt.White,           // Should perhaps stand out more, but cases in bash scripts looks wrong with this light cyan
		MenuTitleColor:              vt.LightRed,
		MenuArrowColor:              vt.Red,
		MenuTextColor:               vt.Gray,
		MenuHighlightColor:          vt.LightGray,
		MenuSelectedColor:           vt.DarkGray,
		ManSectionColor:             vt.Red,
		ManSynopsisColor:            vt.White,
		BoxTextColor:                vt.Black,
		BoxBackground:               vt.BackgroundGray,
		ProgressIndicatorBackground: vt.BackgroundGray,
		BoxHighlight:                vt.Red,
		DebugRunningBackground:      vt.BackgroundGray,
		DebugStoppedBackground:      vt.BackgroundGray,
		DebugRegistersBackground:    vt.BackgroundGray,
		DebugOutputBackground:       vt.BackgroundGray,
		DebugInstructionsForeground: vt.Red,
		DebugInstructionsBackground: vt.BackgroundGray,
		BoxUpperEdge:                vt.Black,
		JumpToLetterColor:           vt.Red,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// NewLightBlueEditTheme creates a new blue/gray/yellow Theme struct, for light backgrounds
func NewLightBlueEditTheme() Theme {
	return Theme{
		Name:                        "Blue Edit Light",
		Light:                       true,
		StatusMode:                  false,
		Foreground:                  vt.White,
		Background:                  vt.BackgroundBlue,
		StatusForeground:            vt.Black,
		StatusBackground:            vt.BackgroundCyan,
		TopRightForeground:          vt.White,
		TopRightBackground:          vt.BackgroundBlue,
		StatusErrorForeground:       vt.Black,
		StatusErrorBackground:       vt.BackgroundRed,
		SearchHighlight:             vt.LightRed,
		MultiLineComment:            vt.Gray,
		MultiLineString:             vt.LightYellow,
		HighlightForeground:         vt.LightYellow,
		HighlightBackground:         vt.BackgroundBlue,
		Git:                         vt.White,
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
		AngleBracket:                "lightyellow",
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
		RainbowParenColors:          []vt.AttributeColor{vt.LightCyan, vt.LightYellow, vt.LightGreen, vt.White},
		MarkdownTextColor:           vt.White,
		HeaderBulletColor:           vt.LightGray,
		HeaderTextColor:             vt.White,
		ListBulletColor:             vt.LightCyan,
		ListTextColor:               vt.LightCyan,
		ListCodeColor:               vt.White,
		CodeColor:                   vt.White,
		CodeBlockColor:              vt.White,
		ImageColor:                  vt.LightYellow,
		LinkColor:                   vt.LightYellow,
		QuoteColor:                  vt.LightYellow,
		QuoteTextColor:              vt.LightCyan,
		HTMLColor:                   vt.White,
		CommentColor:                vt.LightGray,
		BoldColor:                   vt.LightYellow,
		ItalicsColor:                vt.White,
		StrikeColor:                 vt.LightGray,
		TableColor:                  vt.White,
		CheckboxColor:               vt.White,
		XColor:                      vt.LightYellow,
		TableBackground:             vt.BackgroundBlue,
		UnmatchedParenColor:         vt.White,
		MenuTitleColor:              vt.LightYellow,
		MenuArrowColor:              vt.LightRed,
		MenuTextColor:               vt.LightYellow,
		MenuHighlightColor:          vt.LightRed,
		MenuSelectedColor:           vt.White,
		ManSectionColor:             vt.LightBlue,
		ManSynopsisColor:            vt.LightBlue,
		BoxTextColor:                vt.Black,
		BoxBackground:               vt.BackgroundGray,
		ProgressIndicatorBackground: vt.BackgroundGray,
		BoxHighlight:                vt.LightYellow,
		DebugRunningBackground:      vt.BackgroundGray,
		DebugStoppedBackground:      vt.BackgroundMagenta,
		DebugRegistersBackground:    vt.BackgroundMagenta,
		DebugOutputBackground:       vt.BackgroundYellow,
		DebugInstructionsForeground: vt.LightYellow,
		DebugInstructionsBackground: vt.BackgroundCyan,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.LightBlue,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// NewDarkBlueEditTheme creates a new blue/gray/yellow Theme struct, for dark backgrounds
func NewDarkBlueEditTheme() Theme {
	return Theme{
		Name:                        "Blue Edit Dark",
		Light:                       false,
		StatusMode:                  false,
		Foreground:                  vt.LightYellow,
		Background:                  vt.BackgroundBlue,
		StatusForeground:            vt.White,
		StatusBackground:            vt.BackgroundCyan,
		TopRightForeground:          vt.LightYellow,
		TopRightBackground:          vt.BackgroundBlue,
		StatusErrorForeground:       vt.Red,
		StatusErrorBackground:       vt.BackgroundCyan,
		SearchHighlight:             vt.Red,
		MultiLineComment:            vt.White,
		MultiLineString:             vt.White,
		HighlightForeground:         vt.LightYellow,
		HighlightBackground:         vt.BackgroundBlue,
		Git:                         vt.White,
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
		AngleBracket:                "white",
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
		RainbowParenColors:          []vt.AttributeColor{vt.White, vt.LightYellow},
		MarkdownTextColor:           vt.White,
		HeaderBulletColor:           vt.LightRed,
		HeaderTextColor:             vt.White,
		ListBulletColor:             vt.LightRed,
		ListTextColor:               vt.White,
		ListCodeColor:               vt.White,
		CodeColor:                   vt.LightYellow,
		CodeBlockColor:              vt.LightYellow,
		ImageColor:                  vt.White,
		LinkColor:                   vt.White,
		QuoteColor:                  vt.LightYellow,
		QuoteTextColor:              vt.LightYellow,
		HTMLColor:                   vt.White,
		CommentColor:                vt.LightYellow,
		BoldColor:                   vt.White,
		ItalicsColor:                vt.LightYellow,
		StrikeColor:                 vt.LightYellow,
		TableColor:                  vt.LightYellow,
		CheckboxColor:               vt.White,
		XColor:                      vt.White,
		TableBackground:             vt.BackgroundBlue,
		UnmatchedParenColor:         vt.LightRed,
		MenuTitleColor:              vt.LightYellow,
		MenuArrowColor:              vt.White,
		MenuTextColor:               vt.LightGray,
		MenuHighlightColor:          vt.LightYellow,
		MenuSelectedColor:           vt.LightGreen,
		ManSectionColor:             vt.White,
		ManSynopsisColor:            vt.LightYellow,
		BoxTextColor:                vt.Black,
		BoxBackground:               vt.BackgroundYellow,
		ProgressIndicatorBackground: vt.BackgroundYellow,
		BoxHighlight:                vt.LightYellow,
		DebugRunningBackground:      vt.BackgroundGray,
		DebugStoppedBackground:      vt.BackgroundGray,
		DebugRegistersBackground:    vt.BackgroundGray,
		DebugOutputBackground:       vt.BackgroundGray,
		DebugInstructionsForeground: vt.White,
		DebugInstructionsBackground: vt.BackgroundGray,
		BoxUpperEdge:                vt.LightYellow,
		JumpToLetterColor:           vt.White,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// NewLightVSTheme creates a theme that is suitable for light xterm terminal emulator sessions
func NewLightVSTheme() Theme {
	return Theme{
		Name:                        "VS Light",
		Light:                       true,
		Foreground:                  vt.Black,
		Background:                  vt.BackgroundDefault,
		StatusForeground:            vt.White,
		StatusBackground:            vt.BackgroundBlack,
		TopRightForeground:          vt.White,
		TopRightBackground:          vt.BackgroundBlack,
		StatusErrorForeground:       vt.LightRed,
		StatusErrorBackground:       vt.BackgroundDefault,
		SearchHighlight:             vt.Red,
		MultiLineComment:            vt.Gray,
		MultiLineString:             vt.Red,
		HighlightForeground:         vt.Red,
		HighlightBackground:         vt.BackgroundDefault,
		Git:                         vt.Blue,
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
		AngleBracket:                "black",
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
		RainbowParenColors:          []vt.AttributeColor{vt.Magenta, vt.Black, vt.Blue, vt.Green},
		MarkdownTextColor:           vt.Default,
		HeaderBulletColor:           vt.DarkGray,
		HeaderTextColor:             vt.Blue,
		ListBulletColor:             vt.Red,
		ListTextColor:               vt.Default,
		ListCodeColor:               vt.Red,
		CodeColor:                   vt.Red,
		CodeBlockColor:              vt.Red,
		ImageColor:                  vt.Green,
		LinkColor:                   vt.Magenta,
		QuoteColor:                  vt.Yellow,
		QuoteTextColor:              vt.LightCyan,
		HTMLColor:                   vt.Default,
		CommentColor:                vt.DarkGray,
		BoldColor:                   vt.Blue,
		ItalicsColor:                vt.Blue,
		StrikeColor:                 vt.DarkGray,
		TableColor:                  vt.Blue,
		CheckboxColor:               vt.Default,
		XColor:                      vt.Blue,
		TableBackground:             vt.BackgroundDefault,
		UnmatchedParenColor:         vt.Red,
		MenuTitleColor:              vt.Blue,
		MenuArrowColor:              vt.Red,
		MenuTextColor:               vt.Black,
		MenuHighlightColor:          vt.Red,
		MenuSelectedColor:           vt.LightRed,
		ManSectionColor:             vt.Red,
		ManSynopsisColor:            vt.Blue,
		BoxTextColor:                vt.Black,
		BoxBackground:               vt.BackgroundGray,
		ProgressIndicatorBackground: vt.BackgroundRed,
		BoxHighlight:                vt.Red,
		DebugRunningBackground:      vt.BackgroundCyan,
		DebugStoppedBackground:      vt.BackgroundBlack,
		DebugRegistersBackground:    vt.BackgroundGray,
		DebugOutputBackground:       vt.BackgroundGray,
		DebugInstructionsForeground: vt.Black,
		DebugInstructionsBackground: vt.BackgroundGray,
		BoxUpperEdge:                vt.Black,
		JumpToLetterColor:           vt.Red,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// NewDarkVSTheme creates a theme that is suitable for dark terminal emulator sessions
func NewDarkVSTheme() Theme {
	return Theme{
		Name:                        "VS Dark",
		Light:                       false,
		Foreground:                  vt.Black,
		Background:                  vt.BackgroundWhite,
		StatusForeground:            vt.White,
		StatusBackground:            vt.BackgroundBlue,
		TopRightForeground:          vt.White,
		TopRightBackground:          vt.BackgroundBlue,
		StatusErrorForeground:       vt.Red,
		StatusErrorBackground:       vt.BackgroundCyan,
		SearchHighlight:             vt.Red,
		MultiLineComment:            vt.Gray,
		MultiLineString:             vt.Red,
		HighlightForeground:         vt.Black,
		HighlightBackground:         vt.BackgroundWhite,
		Git:                         vt.Blue,
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
		AngleBracket:                "black",
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
		RainbowParenColors:          []vt.AttributeColor{vt.Magenta, vt.Black, vt.Blue, vt.Green},
		MarkdownTextColor:           vt.Black,
		HeaderBulletColor:           vt.DarkGray,
		HeaderTextColor:             vt.Blue,
		ListBulletColor:             vt.Red,
		ListTextColor:               vt.Black,
		ListCodeColor:               vt.Red,
		CodeColor:                   vt.Red,
		CodeBlockColor:              vt.Red,
		ImageColor:                  vt.DarkGray,
		LinkColor:                   vt.Magenta,
		QuoteColor:                  vt.Yellow,
		QuoteTextColor:              vt.LightCyan,
		HTMLColor:                   vt.Black,
		CommentColor:                vt.DarkGray,
		BoldColor:                   vt.Blue,
		ItalicsColor:                vt.Blue,
		StrikeColor:                 vt.DarkGray,
		TableColor:                  vt.Blue,
		CheckboxColor:               vt.Black,
		XColor:                      vt.Blue,
		TableBackground:             vt.DarkGray,
		UnmatchedParenColor:         vt.Red,
		MenuTitleColor:              vt.Blue,
		MenuArrowColor:              vt.Red,
		MenuTextColor:               vt.Black,
		MenuHighlightColor:          vt.Red,
		MenuSelectedColor:           vt.LightRed,
		ManSectionColor:             vt.Red,
		ManSynopsisColor:            vt.Blue,
		BoxTextColor:                vt.Black,
		BoxBackground:               vt.BackgroundGray,
		ProgressIndicatorBackground: vt.BackgroundBlack,
		BoxHighlight:                vt.Red,
		DebugRunningBackground:      vt.BackgroundCyan,
		DebugStoppedBackground:      vt.BackgroundBlack,
		DebugRegistersBackground:    vt.BackgroundGray,
		DebugOutputBackground:       vt.BackgroundGray,
		DebugInstructionsForeground: vt.Black,
		DebugInstructionsBackground: vt.BackgroundGray,
		BoxUpperEdge:                vt.Black,
		JumpToLetterColor:           vt.Red,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// NewGrayTheme returns a theme where all text is light gray
func NewGrayTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Gray Mono"
	t.Foreground = vt.LightGray
	t.Background = vt.BackgroundDefault // black background
	t.JumpToLetterColor = vt.White      // for jumping to a letter with ctrl-l
	t.ProgressIndicatorBackground = vt.BackgroundGray
	return t
}

// NewAmberTheme returns a theme where all text is amber / yellow
func NewAmberTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Amber Mono"
	t.Foreground = vt.Yellow
	t.Background = vt.BackgroundDefault // black background
	t.JumpToLetterColor = t.Foreground  // for jumping to a letter with ctrl-l
	t.ProgressIndicatorBackground = vt.BackgroundYellow
	t.TopRightForeground = t.Foreground
	return t
}

// NewGreenTheme returns a theme where all text is green
func NewGreenTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Green Mono"
	t.Foreground = vt.LightGreen
	t.Background = vt.BackgroundDefault // black background
	t.JumpToLetterColor = t.Foreground  // for jumping to a letter with ctrl-l
	t.ProgressIndicatorBackground = vt.BackgroundGreen
	t.TopRightForeground = t.Foreground
	return t
}

// NewBlueTheme returns a theme where all text is blue
func NewBlueTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Blue Mono"
	t.Foreground = vt.LightBlue
	t.Background = vt.BackgroundDefault // black background
	t.JumpToLetterColor = t.Foreground  // for jumping to a letter with ctrl-l
	t.ProgressIndicatorBackground = vt.BackgroundBlue
	t.TopRightForeground = t.Foreground
	return t
}

// NewNoColorDarkBackgroundTheme creates a new theme without colors or syntax highlighting
func NewNoColorDarkBackgroundTheme() Theme {
	return Theme{
		Name:                        "No color",
		Light:                       false,
		Foreground:                  vt.Default,
		Background:                  vt.BackgroundDefault,
		StatusForeground:            vt.White,
		StatusBackground:            vt.BackgroundBlack,
		TopRightForeground:          vt.White,
		TopRightBackground:          vt.BackgroundBlack,
		StatusErrorForeground:       vt.White,
		StatusErrorBackground:       vt.BackgroundDefault,
		SearchHighlight:             vt.Default,
		MultiLineComment:            vt.Default,
		MultiLineString:             vt.Default,
		HighlightForeground:         vt.White,
		HighlightBackground:         vt.BackgroundDefault,
		Git:                         vt.White,
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
		AngleBracket:                "",
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
		RainbowParenColors:          []vt.AttributeColor{vt.Gray},
		MarkdownTextColor:           vt.Default,
		HeaderBulletColor:           vt.Default,
		HeaderTextColor:             vt.Default,
		ListBulletColor:             vt.Default,
		ListTextColor:               vt.Default,
		ListCodeColor:               vt.Default,
		CodeColor:                   vt.Default,
		CodeBlockColor:              vt.Default,
		ImageColor:                  vt.Default,
		LinkColor:                   vt.Default,
		QuoteColor:                  vt.Default,
		QuoteTextColor:              vt.Default,
		HTMLColor:                   vt.Default,
		CommentColor:                vt.Default,
		BoldColor:                   vt.Default,
		ItalicsColor:                vt.Default,
		StrikeColor:                 vt.Default,
		TableColor:                  vt.Default,
		CheckboxColor:               vt.Default,
		XColor:                      vt.White,
		TableBackground:             vt.BackgroundDefault,
		UnmatchedParenColor:         vt.White,
		MenuTitleColor:              vt.White,
		MenuArrowColor:              vt.White,
		MenuTextColor:               vt.Gray,
		MenuHighlightColor:          vt.White,
		MenuSelectedColor:           vt.Black,
		ManSectionColor:             vt.White,
		ManSynopsisColor:            vt.White,
		BoxTextColor:                vt.Black,
		BoxBackground:               vt.BackgroundGray,
		ProgressIndicatorBackground: vt.BackgroundGray,
		BoxHighlight:                vt.Black,
		DebugRunningBackground:      vt.BackgroundGray,
		DebugStoppedBackground:      vt.BackgroundGray,
		DebugRegistersBackground:    vt.BackgroundGray,
		DebugOutputBackground:       vt.BackgroundGray,
		DebugInstructionsForeground: vt.Black,
		DebugInstructionsBackground: vt.BackgroundGray,
		BoxUpperEdge:                vt.Black,
		JumpToLetterColor:           vt.White,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// NewNoColorLightBackgroundTheme creates a new theme without colors or syntax highlighting
func NewNoColorLightBackgroundTheme() Theme {
	return Theme{
		Name:                        "No color",
		Light:                       true,
		Foreground:                  vt.Default,
		Background:                  vt.BackgroundDefault,
		StatusForeground:            vt.Black,
		StatusBackground:            vt.BackgroundWhite,
		TopRightForeground:          vt.Black,
		TopRightBackground:          vt.BackgroundWhite,
		StatusErrorForeground:       vt.Black,
		StatusErrorBackground:       vt.BackgroundDefault,
		SearchHighlight:             vt.Default,
		MultiLineComment:            vt.Default,
		MultiLineString:             vt.Default,
		HighlightForeground:         vt.Default,
		HighlightBackground:         vt.BackgroundDefault,
		Git:                         vt.Black,
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
		AngleBracket:                "",
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
		RainbowParenColors:          []vt.AttributeColor{vt.Gray},
		MarkdownTextColor:           vt.Default,
		HeaderBulletColor:           vt.Default,
		HeaderTextColor:             vt.Default,
		ListBulletColor:             vt.Default,
		ListTextColor:               vt.Default,
		ListCodeColor:               vt.Default,
		CodeColor:                   vt.Default,
		CodeBlockColor:              vt.Default,
		ImageColor:                  vt.Default,
		LinkColor:                   vt.Default,
		QuoteColor:                  vt.Default,
		QuoteTextColor:              vt.Default,
		HTMLColor:                   vt.Default,
		CommentColor:                vt.Default,
		BoldColor:                   vt.Default,
		ItalicsColor:                vt.Default,
		StrikeColor:                 vt.Default,
		TableColor:                  vt.Default,
		CheckboxColor:               vt.Default,
		XColor:                      vt.Black,
		TableBackground:             vt.BackgroundDefault,
		UnmatchedParenColor:         vt.Black,
		MenuTitleColor:              vt.Black,
		MenuArrowColor:              vt.Black,
		MenuTextColor:               vt.Gray,
		MenuHighlightColor:          vt.Black,
		MenuSelectedColor:           vt.White,
		ManSectionColor:             vt.Black,
		ManSynopsisColor:            vt.Black,
		BoxTextColor:                vt.White,
		BoxBackground:               vt.BackgroundGray,
		ProgressIndicatorBackground: vt.BackgroundGray,
		BoxHighlight:                vt.White,
		DebugRunningBackground:      vt.BackgroundGray,
		DebugStoppedBackground:      vt.BackgroundGray,
		DebugRegistersBackground:    vt.BackgroundGray,
		DebugOutputBackground:       vt.BackgroundGray,
		DebugInstructionsForeground: vt.White,
		DebugInstructionsBackground: vt.BackgroundGray,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
	}
}

// TextConfig returns a TextConfig struct that can be used for settings
// the syntax highlighting colors in the public TextConfig variable that is
// exported from the syntax package.
func (t Theme) TextConfig() *TextConfig {
	return &TextConfig{
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
		AngleBracket:  t.AngleBracket,
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

func (e *Editor) makeLightAdjustments() {
	if e.HighlightForeground == vt.White && e.Background != vt.BackgroundBlack && e.Light {
		e.HighlightForeground = vt.Black
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
	if initialLightBackground != nil && *initialLightBackground { // light
		e.SetTheme(NewLightVSTheme())
	} else { // dark
		e.SetTheme(NewDarkVSTheme())
	}
}

// SetTheme assigns the given theme to the Editor,
// and also configures syntax highlighting by setting vt.DefaultTextConfig.
// Light/dark, syntax highlighting and no color information is also set.
// Respect the NO_COLOR environment variable. May set e.NoSyntaxHighlight to true.
func (e *Editor) SetTheme(theme Theme, bs ...bool) {
	if envNoColor {
		if initialLightBackground != nil && *initialLightBackground { // light
			theme = NewNoColorLightBackgroundTheme()
		} else { // dark
			theme = NewNoColorDarkBackgroundTheme()
		}
		e.syntaxHighlight = false
	} else if len(bs) == 1 {
		initialLightBackground = &(bs[0])
	}
	e.Theme = theme
	e.statusMode = theme.StatusMode
	DefaultTextConfig = *(theme.TextConfig())
	if initialLightBackground != nil && *initialLightBackground { // light
		e.makeLightAdjustments()
	}
}

// setNoColorTheme sets the NoColor theme, and considers the background color
func (e *Editor) setNoColorTheme() {
	if initialLightBackground != nil && *initialLightBackground { // light
		e.Theme = NewNoColorLightBackgroundTheme()
	} else { //dark
		e.Theme = NewNoColorDarkBackgroundTheme()
	}
	e.statusMode = e.StatusMode
	DefaultTextConfig = *(e.TextConfig())
	if initialLightBackground != nil && *initialLightBackground { // light
		e.makeLightAdjustments()
	}
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
	if initialLightBackground != nil && *initialLightBackground { // light
		e.SetTheme(NewLightBlueEditTheme())
	} else { // dark
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
