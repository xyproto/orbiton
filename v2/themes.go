package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt"
)

// termHas256Colors reports whether the terminal actually supports >=256 colors.
// It checks TERM first: if TERM indicates a basic terminal (vt100, vt220, dumb, etc.)
// then COLORTERM is ignored, because some terminal emulators (e.g. Kitty) always set
// COLORTERM=truecolor even when TERM is overridden to a limited value.
func termHas256Colors() bool {
	term := strings.ToLower(env.Str("TERM"))
	switch {
	case term == "", term == "dumb":
		return false
	case strings.HasPrefix(term, "vt") && !strings.Contains(term, "256color"):
		// vt100, vt220, vt320, vt52, etc. — but not something like vt-256color
		return false
	}
	return vt.Has256Colors() || vt.HasTrueColor()
}

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
	CurlyBracket                string
	IncludeSystem               string
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
	DebugLineIndicator          vt.AttributeColor
	TableBackground             vt.AttributeColor
	JumpToLetterColor           vt.AttributeColor
	NanoHelpForeground          vt.AttributeColor
	NanoHelpBackground          vt.AttributeColor
	HighlightForeground         vt.AttributeColor
	HighlightBackground         vt.AttributeColor
	MultiCursorBackground       vt.AttributeColor
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
		CurlyBracket:                "lightblue",
		IncludeSystem:               "lightred",
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
		DebugLineIndicator:          vt.LightGreen,
		DebugInstructionsForeground: vt.LightYellow,
		DebugInstructionsBackground: vt.BackgroundMagenta,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.LightRed,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundYellow,
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
		CurlyBracket:                "lightgray",
		IncludeSystem:               "lightcyan",
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
		DebugLineIndicator:          vt.LightCyan,
		DebugInstructionsForeground: vt.LightGreen,
		DebugInstructionsBackground: vt.DarkGray,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.LightRed,
		NanoHelpForeground:          vt.White,
		NanoHelpBackground:          vt.DarkGray,
		MultiCursorBackground:       vt.BackgroundYellow,
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
		CurlyBracket:                "lightgray",
		IncludeSystem:               "lightred",
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
		DebugLineIndicator:          vt.LightGreen,
		DebugInstructionsForeground: vt.LightYellow,
		DebugInstructionsBackground: vt.BackgroundMagenta,
		BoxUpperEdge:                vt.LightGray,
		JumpToLetterColor:           vt.LightRed,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundYellow,
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
		CurlyBracket:                "lightgray",
		IncludeSystem:               "lightgreen",
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
		DebugLineIndicator:          vt.LightCyan,
		DebugInstructionsForeground: vt.LightYellow,
		DebugInstructionsBackground: vt.BackgroundBlack,
		BoxUpperEdge:                vt.LightGray,
		JumpToLetterColor:           vt.LightMagenta,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundYellow,
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
		CurlyBracket:                "black",
		IncludeSystem:               "lightred",
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
		DebugLineIndicator:          vt.Cyan,
		DebugInstructionsForeground: vt.Red,
		DebugInstructionsBackground: vt.BackgroundMagenta,
		BoxUpperEdge:                vt.DarkGray,
		JumpToLetterColor:           vt.LightRed,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundYellow,
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
		CurlyBracket:                "lightblue",
		IncludeSystem:               "magenta",
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
		DebugLineIndicator:          vt.LightMagenta,
		DebugInstructionsForeground: vt.LightGray,
		DebugInstructionsBackground: vt.BackgroundRed,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.LightMagenta,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundYellow,
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
		CurlyBracket:                "white",
		IncludeSystem:               "white",
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
		DebugLineIndicator:          vt.LightCyan,
		DebugInstructionsForeground: vt.White,
		DebugInstructionsBackground: vt.BackgroundBlack,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.White,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundYellow,
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
		CurlyBracket:                "darkred",
		IncludeSystem:               "darkred",
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
		ItalicsColor:                vt.White,
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
		DebugLineIndicator:          vt.Red,
		DebugInstructionsForeground: vt.Red,
		DebugInstructionsBackground: vt.BackgroundGray,
		BoxUpperEdge:                vt.Black,
		JumpToLetterColor:           vt.Red,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundRed,
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
		CurlyBracket:                "white",
		IncludeSystem:               "lightcyan",
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
		DebugLineIndicator:          vt.LightBlue,
		DebugInstructionsForeground: vt.LightYellow,
		DebugInstructionsBackground: vt.BackgroundCyan,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.LightBlue,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundYellow,
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
		CurlyBracket:                "white",
		IncludeSystem:               "lightyellow",
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
		DebugLineIndicator:          vt.LightYellow,
		DebugInstructionsForeground: vt.White,
		DebugInstructionsBackground: vt.BackgroundGray,
		BoxUpperEdge:                vt.LightYellow,
		JumpToLetterColor:           vt.White,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundYellow,
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
		CurlyBracket:                "black",
		IncludeSystem:               "blue",
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
		DebugLineIndicator:          vt.Blue,
		DebugInstructionsForeground: vt.Black,
		DebugInstructionsBackground: vt.BackgroundGray,
		BoxUpperEdge:                vt.Black,
		JumpToLetterColor:           vt.Red,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundYellow,
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
		CurlyBracket:                "black",
		IncludeSystem:               "blue",
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
		DebugLineIndicator:          vt.LightBlue,
		DebugInstructionsForeground: vt.Black,
		DebugInstructionsBackground: vt.BackgroundGray,
		BoxUpperEdge:                vt.Black,
		JumpToLetterColor:           vt.Red,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundYellow,
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
	t.MultiCursorBackground = vt.BackgroundYellow
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
	t.MultiCursorBackground = vt.BackgroundMagenta
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
	t.MultiCursorBackground = vt.BackgroundRed
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
	t.MultiCursorBackground = vt.BackgroundYellow
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
		CurlyBracket:                "",
		IncludeSystem:               "",
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
		DebugLineIndicator:          vt.White,
		DebugInstructionsForeground: vt.Black,
		DebugInstructionsBackground: vt.BackgroundGray,
		BoxUpperEdge:                vt.Black,
		JumpToLetterColor:           vt.White,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundWhite,
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
		CurlyBracket:                "",
		IncludeSystem:               "",
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
		DebugLineIndicator:          vt.Black,
		DebugInstructionsForeground: vt.White,
		DebugInstructionsBackground: vt.BackgroundGray,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundBlack,
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
		CurlyBracket:  t.CurlyBracket,
		IncludeSystem: t.IncludeSystem,
	}
}

func (e *Editor) makeLightAdjustments() {
	if e.HighlightForeground == vt.White && e.Background != vt.BackgroundBlack && e.Light {
		e.HighlightForeground = vt.Black
	}
}

// registerXoria256Colors adds the Xoria256 palette entries to the vt color maps
// and rebuilds the tag replacers so the custom color names are recognized.
func registerXoria256Colors() {
	vt.DarkColorMap["xcomment"] = vt.Color256(244)
	vt.DarkColorMap["xstring"] = vt.Color256(229)
	vt.DarkColorMap["xident"] = vt.Color256(182)
	vt.DarkColorMap["xnumber"] = vt.Color256(180)
	vt.DarkColorMap["xpreproc"] = vt.Color256(150)
	vt.DarkColorMap["xkeyword"] = vt.Color256(110)
	vt.DarkColorMap["xtype"] = vt.Color256(146)
	vt.DarkColorMap["xplain"] = vt.Color256(252)
	vt.DarkColorMap["xescape"] = vt.Color256(174)
	vt.DarkColorMap["xpunct"] = vt.Color256(247)
	vt.DarkColorMap["xincsys"] = vt.Color256(110) // same as keyword
	vt.DarkColorMap["xcurly"] = vt.Color256(247)
	vt.RebuildTagReplacers()
	tout = vt.New()
}

// NewXoria256Theme creates the Xoria256 theme, ported from the Vim color scheme
// by Dmitriy Y. Zotikov. Requires a terminal with 256-color or true-color support.
func NewXoria256Theme() Theme {
	return Theme{
		Name:                  "Xoria256",
		Light:                 false,
		Foreground:            vt.Color256(252),
		Background:            vt.Background256(234),
		StatusForeground:      vt.Color256(231),
		StatusBackground:      vt.Background256(239),
		TopRightForeground:    vt.Color256(252),
		TopRightBackground:    vt.Background256(234),
		StatusErrorForeground: vt.Color256(231),
		StatusErrorBackground: vt.Background256(160),
		SearchHighlight:       vt.Color256(214),
		MultiLineComment:      vt.Color256(244),
		MultiLineString:       vt.Color256(229),
		HighlightForeground:   vt.Color256(255),
		HighlightBackground:   vt.BackgroundDefault,
		Git:                   vt.Color256(150),
		String:                "xstring",
		Keyword:               "xkeyword",
		Comment:               "xcomment",
		Type:                  "xtype",
		Literal:               "xstring",
		Punctuation:           "xpunct",
		Plaintext:             "xplain",
		Tag:                   "xkeyword",
		TextTag:               "xkeyword",
		TextAttrName:          "xtype",
		TextAttrValue:         "xstring",
		Decimal:               "xnumber",
		AndOr:                 "xpunct",
		AngleBracket:          "xkeyword",
		Dollar:                "xpreproc",
		Star:                  "xplain",
		Static:                "xkeyword",
		Self:                  "xident",
		Class:                 "xident",
		Private:               "xescape",
		Protected:             "xtype",
		Public:                "xpreproc",
		Whitespace:            "",
		AssemblyEnd:           "xescape",
		Mut:                   "xpreproc",
		CurlyBracket:          "xcurly",
		IncludeSystem:         "xincsys",
		RainbowParenColors: []vt.AttributeColor{
			vt.Color256(110), vt.Color256(150), vt.Color256(174),
			vt.Color256(229), vt.Color256(182), vt.Color256(146), vt.Color256(180),
		},
		MarkdownTextColor:           vt.Color256(252),
		HeaderBulletColor:           vt.Color256(244),
		HeaderTextColor:             vt.Color256(229),
		ListBulletColor:             vt.Color256(174),
		ListTextColor:               vt.Color256(252),
		ListCodeColor:               vt.Color256(150),
		CodeColor:                   vt.Color256(150),
		CodeBlockColor:              vt.Color256(150),
		ImageColor:                  vt.Color256(229),
		LinkColor:                   vt.Color256(110),
		QuoteColor:                  vt.Color256(229),
		QuoteTextColor:              vt.Color256(252),
		HTMLColor:                   vt.Color256(110),
		CommentColor:                vt.Color256(244),
		BoldColor:                   vt.Color256(229),
		ItalicsColor:                vt.Color256(252),
		StrikeColor:                 vt.Color256(244),
		TableColor:                  vt.Color256(110),
		CheckboxColor:               vt.Color256(150),
		XColor:                      vt.Color256(229),
		TableBackground:             vt.Background256(234),
		UnmatchedParenColor:         vt.Color256(231),
		MenuTitleColor:              vt.Color256(229),
		MenuArrowColor:              vt.Color256(174),
		MenuTextColor:               vt.Color256(250),
		MenuHighlightColor:          vt.Color256(110),
		MenuSelectedColor:           vt.Color256(255),
		ManSectionColor:             vt.Color256(174),
		ManSynopsisColor:            vt.Color256(229),
		BoxTextColor:                vt.Color256(16),
		BoxBackground:               vt.Background256(250),
		ProgressIndicatorBackground: vt.Background256(136),
		BoxHighlight:                vt.Color256(229),
		DebugRunningBackground:      vt.Background256(150),
		DebugStoppedBackground:      vt.Background256(174),
		DebugRegistersBackground:    vt.Background256(110),
		DebugOutputBackground:       vt.Background256(229),
		DebugLineIndicator:          vt.Color256(150),
		DebugInstructionsForeground: vt.Color256(229),
		DebugInstructionsBackground: vt.Background256(174),
		BoxUpperEdge:                vt.Color256(252),
		JumpToLetterColor:           vt.Color256(214),
		NanoHelpForeground:          vt.Color256(16),
		NanoHelpBackground:          vt.Background256(250),
		MultiCursorBackground:       vt.Background256(96),
	}
}

// NewXoria16Theme creates a 16-color approximation of the Xoria256 theme,
// for terminals that do not support 256 colors.
func NewXoria16Theme() Theme {
	return Theme{
		Name:                        "Xoria",
		Light:                       false,
		Foreground:                  vt.White,
		Background:                  vt.BackgroundBlack,
		StatusForeground:            vt.White,
		StatusBackground:            vt.BackgroundDefault,
		TopRightForeground:          vt.White,
		TopRightBackground:          vt.BackgroundBlack,
		StatusErrorForeground:       vt.White,
		StatusErrorBackground:       vt.BackgroundRed,
		SearchHighlight:             vt.Yellow,
		MultiLineComment:            vt.Gray,
		MultiLineString:             vt.LightYellow,
		HighlightForeground:         vt.White,
		HighlightBackground:         vt.BackgroundDefault,
		Git:                         vt.LightGreen,
		String:                      "lightyellow",
		Keyword:                     "lightblue",
		Comment:                     "gray",
		Type:                        "white",
		Literal:                     "lightyellow",
		Punctuation:                 "lightgray",
		Plaintext:                   "white",
		Tag:                         "lightblue",
		TextTag:                     "lightblue",
		TextAttrName:                "white",
		TextAttrValue:               "lightyellow",
		Decimal:                     "darkyellow",
		AndOr:                       "lightgray",
		AngleBracket:                "lightblue",
		Dollar:                      "lightgreen",
		Star:                        "white",
		Static:                      "lightblue",
		Self:                        "white",
		Class:                       "white",
		Private:                     "lightred",
		Protected:                   "white",
		Public:                      "lightgreen",
		Whitespace:                  "",
		AssemblyEnd:                 "lightred",
		Mut:                         "lightgreen",
		CurlyBracket:                "lightgray",
		IncludeSystem:               "lightblue",
		RainbowParenColors:          []vt.AttributeColor{vt.LightBlue, vt.LightGreen, vt.LightRed, vt.LightYellow, vt.White, vt.LightCyan, vt.Yellow},
		MarkdownTextColor:           vt.White,
		HeaderBulletColor:           vt.Gray,
		HeaderTextColor:             vt.LightYellow,
		ListBulletColor:             vt.LightRed,
		ListTextColor:               vt.White,
		ListCodeColor:               vt.LightGreen,
		CodeColor:                   vt.LightGreen,
		CodeBlockColor:              vt.LightGreen,
		ImageColor:                  vt.LightYellow,
		LinkColor:                   vt.LightBlue,
		QuoteColor:                  vt.LightYellow,
		QuoteTextColor:              vt.White,
		HTMLColor:                   vt.LightBlue,
		CommentColor:                vt.Gray,
		BoldColor:                   vt.LightYellow,
		ItalicsColor:                vt.White,
		StrikeColor:                 vt.Gray,
		TableColor:                  vt.LightBlue,
		CheckboxColor:               vt.LightGreen,
		XColor:                      vt.LightYellow,
		TableBackground:             vt.BackgroundBlack,
		UnmatchedParenColor:         vt.White,
		MenuTitleColor:              vt.LightYellow,
		MenuArrowColor:              vt.LightRed,
		MenuTextColor:               vt.Gray,
		MenuHighlightColor:          vt.LightBlue,
		MenuSelectedColor:           vt.White,
		ManSectionColor:             vt.LightRed,
		ManSynopsisColor:            vt.LightYellow,
		BoxTextColor:                vt.Black,
		BoxBackground:               vt.BackgroundBlue,
		ProgressIndicatorBackground: vt.BackgroundYellow,
		BoxHighlight:                vt.LightYellow,
		DebugRunningBackground:      vt.BackgroundGreen,
		DebugStoppedBackground:      vt.BackgroundMagenta,
		DebugRegistersBackground:    vt.BackgroundBlue,
		DebugOutputBackground:       vt.BackgroundYellow,
		DebugLineIndicator:          vt.LightGreen,
		DebugInstructionsForeground: vt.LightYellow,
		DebugInstructionsBackground: vt.BackgroundMagenta,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.Yellow,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundGray,
		MultiCursorBackground:       vt.BackgroundYellow,
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
	syntax.DefaultTextConfig = *(theme.TextConfig())
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
	syntax.DefaultTextConfig = *(e.TextConfig())
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

// registerGruvboxColors adds the Gruvbox palette entries to the vt color maps
// and rebuilds the tag replacers so the custom color names are recognized.
func registerGruvboxColors() {
	vt.DarkColorMap["gbcomment"] = vt.TrueColor(0x66, 0x5c, 0x54)
	vt.DarkColorMap["gbstring"] = vt.TrueColor(0xb8, 0xbb, 0x26)
	vt.DarkColorMap["gbident"] = vt.TrueColor(0xd3, 0x86, 0x9b)
	vt.DarkColorMap["gbnumber"] = vt.TrueColor(0xfe, 0x80, 0x19)
	vt.DarkColorMap["gbpreproc"] = vt.TrueColor(0x8e, 0xc0, 0x7c)
	vt.DarkColorMap["gbkeyword"] = vt.TrueColor(0xfb, 0x49, 0x34)
	vt.DarkColorMap["gbtype"] = vt.TrueColor(0xfa, 0xbd, 0x2f)
	vt.DarkColorMap["gbplain"] = vt.TrueColor(0xd5, 0xc4, 0xa1)
	vt.DarkColorMap["gbescape"] = vt.TrueColor(0xd6, 0x5d, 0x0e)
	vt.DarkColorMap["gbpunct"] = vt.TrueColor(0xbd, 0xae, 0x93)
	vt.DarkColorMap["gbfunc"] = vt.TrueColor(0x83, 0xa5, 0x98)
	vt.DarkColorMap["gbcurly"] = vt.TrueColor(0xbd, 0xae, 0x93)
	vt.RebuildTagReplacers()
	tout = vt.New()
}

// NewGruvboxTheme creates the Gruvbox theme, based on the base16 "Gruvbox dark, hard"
// scheme by Dawid Kurek / morhetz. Requires true-color or 256-color support.
func NewGruvboxTheme() Theme {
	return Theme{
		Name:                  "Gruvbox",
		Light:                 false,
		Foreground:            vt.TrueColor(0xd5, 0xc4, 0xa1),
		Background:            vt.TrueBackground(0x1d, 0x20, 0x21),
		StatusForeground:      vt.TrueColor(0xfb, 0xf1, 0xc7),
		StatusBackground:      vt.TrueBackground(0x3c, 0x38, 0x36),
		TopRightForeground:    vt.TrueColor(0xd5, 0xc4, 0xa1),
		TopRightBackground:    vt.TrueBackground(0x1d, 0x20, 0x21),
		StatusErrorForeground: vt.TrueColor(0xfb, 0xf1, 0xc7),
		StatusErrorBackground: vt.TrueBackground(0xfb, 0x49, 0x34),
		SearchHighlight:       vt.TrueColor(0xfa, 0xbd, 0x2f),
		MultiLineComment:      vt.TrueColor(0x66, 0x5c, 0x54),
		MultiLineString:       vt.TrueColor(0xb8, 0xbb, 0x26),
		HighlightForeground:   vt.TrueColor(0xfb, 0xf1, 0xc7),
		HighlightBackground:   vt.BackgroundDefault,
		Git:                   vt.TrueColor(0x8e, 0xc0, 0x7c),
		String:                "gbstring",
		Keyword:               "gbkeyword",
		Comment:               "gbcomment",
		Type:                  "gbtype",
		Literal:               "gbstring",
		Punctuation:           "gbpunct",
		Plaintext:             "gbplain",
		Tag:                   "gbkeyword",
		TextTag:               "gbkeyword",
		TextAttrName:          "gbtype",
		TextAttrValue:         "gbstring",
		Decimal:               "gbnumber",
		AndOr:                 "gbpunct",
		AngleBracket:          "gbkeyword",
		Dollar:                "gbpreproc",
		Star:                  "gbplain",
		Static:                "gbkeyword",
		Self:                  "gbident",
		Class:                 "gbident",
		Private:               "gbescape",
		Protected:             "gbtype",
		Public:                "gbpreproc",
		Whitespace:            "",
		AssemblyEnd:           "gbescape",
		Mut:                   "gbpreproc",
		CurlyBracket:          "gbcurly",
		IncludeSystem:         "gbfunc",
		RainbowParenColors: []vt.AttributeColor{
			vt.TrueColor(0xfb, 0x49, 0x34), vt.TrueColor(0x8e, 0xc0, 0x7c), vt.TrueColor(0xd6, 0x5d, 0x0e),
			vt.TrueColor(0xb8, 0xbb, 0x26), vt.TrueColor(0xd3, 0x86, 0x9b), vt.TrueColor(0xfa, 0xbd, 0x2f), vt.TrueColor(0xfe, 0x80, 0x19),
		},
		MarkdownTextColor:           vt.TrueColor(0xd5, 0xc4, 0xa1),
		HeaderBulletColor:           vt.TrueColor(0x66, 0x5c, 0x54),
		HeaderTextColor:             vt.TrueColor(0xfa, 0xbd, 0x2f),
		ListBulletColor:             vt.TrueColor(0xfb, 0x49, 0x34),
		ListTextColor:               vt.TrueColor(0xd5, 0xc4, 0xa1),
		ListCodeColor:               vt.TrueColor(0x8e, 0xc0, 0x7c),
		CodeColor:                   vt.TrueColor(0x8e, 0xc0, 0x7c),
		CodeBlockColor:              vt.TrueColor(0x8e, 0xc0, 0x7c),
		ImageColor:                  vt.TrueColor(0xfa, 0xbd, 0x2f),
		LinkColor:                   vt.TrueColor(0x83, 0xa5, 0x98),
		QuoteColor:                  vt.TrueColor(0xfa, 0xbd, 0x2f),
		QuoteTextColor:              vt.TrueColor(0xd5, 0xc4, 0xa1),
		HTMLColor:                   vt.TrueColor(0x83, 0xa5, 0x98),
		CommentColor:                vt.TrueColor(0x66, 0x5c, 0x54),
		BoldColor:                   vt.TrueColor(0xfa, 0xbd, 0x2f),
		ItalicsColor:                vt.TrueColor(0xd5, 0xc4, 0xa1),
		StrikeColor:                 vt.TrueColor(0x66, 0x5c, 0x54),
		TableColor:                  vt.TrueColor(0x83, 0xa5, 0x98),
		CheckboxColor:               vt.TrueColor(0x8e, 0xc0, 0x7c),
		XColor:                      vt.TrueColor(0xfa, 0xbd, 0x2f),
		TableBackground:             vt.TrueBackground(0x1d, 0x20, 0x21),
		UnmatchedParenColor:         vt.TrueColor(0xfb, 0xf1, 0xc7),
		MenuTitleColor:              vt.TrueColor(0xfa, 0xbd, 0x2f),
		MenuArrowColor:              vt.TrueColor(0xfb, 0x49, 0x34),
		MenuTextColor:               vt.TrueColor(0xbd, 0xae, 0x93),
		MenuHighlightColor:          vt.TrueColor(0x83, 0xa5, 0x98),
		MenuSelectedColor:           vt.TrueColor(0xfb, 0xf1, 0xc7),
		ManSectionColor:             vt.TrueColor(0xfb, 0x49, 0x34),
		ManSynopsisColor:            vt.TrueColor(0xfa, 0xbd, 0x2f),
		BoxTextColor:                vt.TrueColor(0x1d, 0x20, 0x21),
		BoxBackground:               vt.TrueBackground(0xbd, 0xae, 0x93),
		ProgressIndicatorBackground: vt.TrueBackground(0xfe, 0x80, 0x19),
		BoxHighlight:                vt.TrueColor(0xfa, 0xbd, 0x2f),
		DebugRunningBackground:      vt.TrueBackground(0xb8, 0xbb, 0x26),
		DebugStoppedBackground:      vt.TrueBackground(0xfb, 0x49, 0x34),
		DebugRegistersBackground:    vt.TrueBackground(0x83, 0xa5, 0x98),
		DebugOutputBackground:       vt.TrueBackground(0xfa, 0xbd, 0x2f),
		DebugLineIndicator:          vt.TrueColor(0xb8, 0xbb, 0x26),
		DebugInstructionsForeground: vt.TrueColor(0xfa, 0xbd, 0x2f),
		DebugInstructionsBackground: vt.TrueBackground(0xfb, 0x49, 0x34),
		BoxUpperEdge:                vt.TrueColor(0xd5, 0xc4, 0xa1),
		JumpToLetterColor:           vt.TrueColor(0xfe, 0x80, 0x19),
		NanoHelpForeground:          vt.TrueColor(0x1d, 0x20, 0x21),
		NanoHelpBackground:          vt.TrueBackground(0xbd, 0xae, 0x93),
		MultiCursorBackground:       vt.TrueBackground(0x50, 0x49, 0x45),
	}
}

// NewGruvbox16Theme creates a 16-color approximation of the Gruvbox theme,
// for terminals that do not support 256 colors.
func NewGruvbox16Theme() Theme {
	return Theme{
		Name:                        "Gruvbox",
		Light:                       false,
		Foreground:                  vt.White,
		Background:                  vt.BackgroundBlack,
		StatusForeground:            vt.White,
		StatusBackground:            vt.BackgroundDefault,
		TopRightForeground:          vt.White,
		TopRightBackground:          vt.BackgroundBlack,
		StatusErrorForeground:       vt.White,
		StatusErrorBackground:       vt.BackgroundRed,
		SearchHighlight:             vt.Yellow,
		MultiLineComment:            vt.Gray,
		MultiLineString:             vt.LightGreen,
		HighlightForeground:         vt.White,
		HighlightBackground:         vt.BackgroundDefault,
		Git:                         vt.LightCyan,
		String:                      "lightgreen",
		Keyword:                     "lightred",
		Comment:                     "gray",
		Type:                        "yellow",
		Literal:                     "lightgreen",
		Punctuation:                 "lightgray",
		Plaintext:                   "white",
		Tag:                         "lightred",
		TextTag:                     "lightred",
		TextAttrName:                "yellow",
		TextAttrValue:               "lightgreen",
		Decimal:                     "darkyellow",
		AndOr:                       "lightgray",
		AngleBracket:                "lightred",
		Dollar:                      "lightcyan",
		Star:                        "white",
		Static:                      "lightred",
		Self:                        "lightmagenta",
		Class:                       "lightmagenta",
		Private:                     "darkyellow",
		Protected:                   "yellow",
		Public:                      "lightcyan",
		Whitespace:                  "",
		AssemblyEnd:                 "darkyellow",
		Mut:                         "lightcyan",
		CurlyBracket:                "lightgray",
		IncludeSystem:               "lightblue",
		RainbowParenColors:          []vt.AttributeColor{vt.LightRed, vt.LightCyan, vt.Yellow, vt.LightGreen, vt.LightMagenta, vt.LightYellow, vt.White},
		MarkdownTextColor:           vt.White,
		HeaderBulletColor:           vt.Gray,
		HeaderTextColor:             vt.Yellow,
		ListBulletColor:             vt.LightRed,
		ListTextColor:               vt.White,
		ListCodeColor:               vt.LightCyan,
		CodeColor:                   vt.LightCyan,
		CodeBlockColor:              vt.LightCyan,
		ImageColor:                  vt.Yellow,
		LinkColor:                   vt.LightBlue,
		QuoteColor:                  vt.Yellow,
		QuoteTextColor:              vt.White,
		HTMLColor:                   vt.LightBlue,
		CommentColor:                vt.Gray,
		BoldColor:                   vt.Yellow,
		ItalicsColor:                vt.White,
		StrikeColor:                 vt.Gray,
		TableColor:                  vt.LightBlue,
		CheckboxColor:               vt.LightCyan,
		XColor:                      vt.Yellow,
		TableBackground:             vt.BackgroundBlack,
		UnmatchedParenColor:         vt.White,
		MenuTitleColor:              vt.Yellow,
		MenuArrowColor:              vt.LightRed,
		MenuTextColor:               vt.Gray,
		MenuHighlightColor:          vt.LightBlue,
		MenuSelectedColor:           vt.White,
		ManSectionColor:             vt.LightRed,
		ManSynopsisColor:            vt.Yellow,
		BoxTextColor:                vt.Black,
		BoxBackground:               vt.BackgroundYellow,
		ProgressIndicatorBackground: vt.BackgroundYellow,
		BoxHighlight:                vt.Yellow,
		DebugRunningBackground:      vt.BackgroundGreen,
		DebugStoppedBackground:      vt.BackgroundRed,
		DebugRegistersBackground:    vt.BackgroundBlue,
		DebugOutputBackground:       vt.BackgroundYellow,
		DebugLineIndicator:          vt.LightGreen,
		DebugInstructionsForeground: vt.Yellow,
		DebugInstructionsBackground: vt.BackgroundRed,
		BoxUpperEdge:                vt.White,
		JumpToLetterColor:           vt.Yellow,
		NanoHelpForeground:          vt.Black,
		NanoHelpBackground:          vt.BackgroundYellow,
		MultiCursorBackground:       vt.BackgroundMagenta,
	}
}

// parseBase16File reads a base16 YAML scheme file and returns the theme name
// and the 16 palette colors. It handles both the spec-0.11 format (with a
// "palette:" block) and the older flat format where base00..base0F are at the
// top level.
func parseBase16File(filename string) (name string, colors [16]vt.AttributeColor, bgs [16]vt.AttributeColor, err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", colors, bgs, err
	}
	found := 0
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if after, ok := strings.CutPrefix(trimmed, "name:"); ok {
			name = strings.Trim(strings.TrimSpace(after), "\"'")
			continue
		}
		// Match base00..base0F
		if !strings.HasPrefix(trimmed, "base0") {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if len(key) != 6 { // "base0X"
			continue
		}
		hexDigit := key[5]
		var idx int
		switch {
		case hexDigit >= '0' && hexDigit <= '9':
			idx = int(hexDigit - '0')
		case hexDigit >= 'A' && hexDigit <= 'F':
			idx = int(hexDigit-'A') + 10
		case hexDigit >= 'a' && hexDigit <= 'f':
			idx = int(hexDigit-'a') + 10
		default:
			continue
		}
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		c, cErr := vt.ColorFromHex(val)
		if cErr != nil {
			return "", colors, bgs, fmt.Errorf("%s: %w", key, cErr)
		}
		b, bErr := vt.BackgroundFromHex(val)
		if bErr != nil {
			return "", colors, bgs, fmt.Errorf("%s background: %w", key, bErr)
		}
		colors[idx] = c
		bgs[idx] = b
		found++
	}
	if found == 0 {
		return "", colors, bgs, fmt.Errorf("%s: no base16 colors found", filename)
	}
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	}
	return name, colors, bgs, nil
}

// registerBase16Colors adds base16 palette entries to the vt color maps
// and rebuilds the tag replacers so the custom color names are recognized.
func registerBase16Colors(colors [16]vt.AttributeColor) {
	vt.DarkColorMap["b16comment"] = colors[0x3]
	vt.DarkColorMap["b16string"] = colors[0xB]
	vt.DarkColorMap["b16ident"] = colors[0xE]
	vt.DarkColorMap["b16number"] = colors[0x9]
	vt.DarkColorMap["b16preproc"] = colors[0xC]
	vt.DarkColorMap["b16keyword"] = colors[0x8]
	vt.DarkColorMap["b16type"] = colors[0xA]
	vt.DarkColorMap["b16plain"] = colors[0x5]
	vt.DarkColorMap["b16escape"] = colors[0xF]
	vt.DarkColorMap["b16punct"] = colors[0x4]
	vt.DarkColorMap["b16func"] = colors[0xD]
	vt.DarkColorMap["b16curly"] = colors[0x4]
	vt.RebuildTagReplacers()
	tout = vt.New()
}

// newBase16Theme creates a Theme from pre-parsed base16 foreground and background
// color arrays, using the same base16 slot mapping as the built-in Gruvbox theme.
func newBase16Theme(name string, c [16]vt.AttributeColor, bg [16]vt.AttributeColor) Theme {
	return Theme{
		Name:                  name,
		Light:                 false,
		Foreground:            c[0x5],
		Background:            bg[0x0],
		StatusForeground:      c[0x7],
		StatusBackground:      bg[0x1],
		TopRightForeground:    c[0x5],
		TopRightBackground:    bg[0x0],
		StatusErrorForeground: c[0x7],
		StatusErrorBackground: bg[0x8],
		SearchHighlight:       c[0xA],
		MultiLineComment:      c[0x3],
		MultiLineString:       c[0xB],
		HighlightForeground:   c[0x7],
		HighlightBackground:   vt.BackgroundDefault,
		Git:                   c[0xC],
		String:                "b16string",
		Keyword:               "b16keyword",
		Comment:               "b16comment",
		Type:                  "b16type",
		Literal:               "b16string",
		Punctuation:           "b16punct",
		Plaintext:             "b16plain",
		Tag:                   "b16keyword",
		TextTag:               "b16keyword",
		TextAttrName:          "b16type",
		TextAttrValue:         "b16string",
		Decimal:               "b16number",
		AndOr:                 "b16punct",
		AngleBracket:          "b16keyword",
		Dollar:                "b16preproc",
		Star:                  "b16plain",
		Static:                "b16keyword",
		Self:                  "b16ident",
		Class:                 "b16ident",
		Private:               "b16escape",
		Protected:             "b16type",
		Public:                "b16preproc",
		Whitespace:            "",
		AssemblyEnd:           "b16escape",
		Mut:                   "b16preproc",
		CurlyBracket:          "b16curly",
		IncludeSystem:         "b16func",
		RainbowParenColors: []vt.AttributeColor{
			c[0x8], c[0xC], c[0xF], c[0xB], c[0xE], c[0xA], c[0x9],
		},
		MarkdownTextColor:           c[0x5],
		HeaderBulletColor:           c[0x3],
		HeaderTextColor:             c[0xA],
		ListBulletColor:             c[0x8],
		ListTextColor:               c[0x5],
		ListCodeColor:               c[0xC],
		CodeColor:                   c[0xC],
		CodeBlockColor:              c[0xC],
		ImageColor:                  c[0xA],
		LinkColor:                   c[0xD],
		QuoteColor:                  c[0xA],
		QuoteTextColor:              c[0x5],
		HTMLColor:                   c[0xD],
		CommentColor:                c[0x3],
		BoldColor:                   c[0xA],
		ItalicsColor:                c[0x5],
		StrikeColor:                 c[0x3],
		TableColor:                  c[0xD],
		CheckboxColor:               c[0xC],
		XColor:                      c[0xA],
		TableBackground:             bg[0x0],
		UnmatchedParenColor:         c[0x7],
		MenuTitleColor:              c[0xA],
		MenuArrowColor:              c[0x8],
		MenuTextColor:               c[0x4],
		MenuHighlightColor:          c[0xD],
		MenuSelectedColor:           c[0x7],
		ManSectionColor:             c[0x8],
		ManSynopsisColor:            c[0xA],
		BoxTextColor:                c[0x0],
		BoxBackground:               bg[0x4],
		ProgressIndicatorBackground: bg[0x9],
		BoxHighlight:                c[0xA],
		DebugRunningBackground:      bg[0xB],
		DebugStoppedBackground:      bg[0x8],
		DebugRegistersBackground:    bg[0xD],
		DebugOutputBackground:       bg[0xA],
		DebugLineIndicator:          c[0xB],
		DebugInstructionsForeground: c[0xA],
		DebugInstructionsBackground: bg[0x8],
		BoxUpperEdge:                c[0x5],
		JumpToLetterColor:           c[0x9],
		NanoHelpForeground:          c[0x0],
		NanoHelpBackground:          bg[0x4],
		MultiCursorBackground:       bg[0x2],
	}
}

// NewThemeFromBase16File reads a base16 YAML scheme file and returns a fully
// populated Theme. It also registers the palette colors for syntax highlighting.
// Both the spec-0.11 format (with a "palette:" block) and the older flat format
// are supported.
func NewThemeFromBase16File(filename string) (Theme, error) {
	name, colors, bgs, err := parseBase16File(filename)
	if err != nil {
		return Theme{}, err
	}
	registerBase16Colors(colors)
	return newBase16Theme(name, colors, bgs), nil
}
