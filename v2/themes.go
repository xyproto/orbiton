package main

import (
	"github.com/xyproto/env"
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

var (
	envNoColor             = env.Bool("NO_COLOR")
	allThemes              = []string{"Default", "Light background", "Red/black", "Amber", "Green", "Blue", "No color"}
	initialLightBackground *bool
)

// Theme contains iformation about:
// * If the theme is light or dark
// * If syntax highlighting should be enabled
// * If no colors should be used
// * Colors for all the textual elements
type Theme struct {
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
	TextAttrValue               string
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
	HeaderBulletColor           vt100.AttributeColor
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
	TableBackground             vt100.AttributeColor
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
	DebugInstructionsForeground vt100.AttributeColor
	BoxUpperEdge                vt100.AttributeColor
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
		DebugOutputBackground:       vt100.BackgroundGray,
		DebugInstructionsForeground: vt100.LightYellow,
		DebugInstructionsBackground: vt100.BackgroundMagenta,
		BoxUpperEdge:                vt100.White,
	}
}

// NewRedBlackTheme creates a new red/black/gray/white Theme struct
func NewRedBlackTheme() Theme {
	// NOTE: Dark gray may not be visible with light terminal emulator themes
	return Theme{
		Name:                        "Red/black",
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
		Decimal:                     "lightwhite",
		AndOr:                       "darkred",
		Dollar:                      "lightwhite",
		Star:                        "lightwhite",
		Class:                       "darkred",
		Private:                     "lightgray",
		Protected:                   "lightgray",
		Public:                      "lightwhite",
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
		LinkColor:                   vt100.Magenta,
		QuoteColor:                  vt100.White,
		QuoteTextColor:              vt100.LightGray,
		HTMLColor:                   vt100.Default,
		CommentColor:                vt100.DarkGray,
		BoldColor:                   vt100.Red,
		ItalicsColor:                vt100.White,
		StrikeColor:                 vt100.DarkGray,
		TableColor:                  vt100.White,
		CheckboxColor:               vt100.Default,
		XColor:                      vt100.Red,
		TableBackground:             vt100.BackgroundBlack, // Dark gray background, as opposed to vt100.BackgroundDefault
		UnmatchedParenColor:         vt100.LightCyan,       // To really stand out
		MenuTitleColor:              vt100.Red,
		MenuArrowColor:              vt100.White,
		MenuTextColor:               vt100.White,
		MenuHighlightColor:          vt100.Red,
		MenuSelectedColor:           vt100.White,
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
	}
}

// NewLightTheme creates a theme that is suitable for light xterm terminal emulator sessions
func NewLightTheme() Theme {
	return Theme{
		Name:                        "Light",
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
	}
}

// NewAmberTheme returns a theme where all text is amber / yellow
func NewAmberTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Amber"
	t.Foreground = vt100.Yellow
	t.Background = vt100.BackgroundDefault // black background
	return t
}

// NewGreenTheme returns a theme where all text is green
func NewGreenTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Green"
	t.Foreground = vt100.LightGreen
	t.Background = vt100.BackgroundDefault // black background
	return t
}

// NewBlueTheme returns a theme where all text is blue
func NewBlueTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Blue"
	t.Foreground = vt100.LightBlue
	t.Background = vt100.BackgroundDefault // black background
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
		if initialLightBackground != nil && *initialLightBackground {
			t = NewNoColorLightBackgroundTheme()
		} else {
			t = NewNoColorDarkBackgroundTheme()
		}
		e.syntaxHighlight = false
	}
	e.Theme = t
	syntax.DefaultTextConfig = *(t.TextConfig())
}

// setDefaultTheme sets the default colors
func (e *Editor) setDefaultTheme() {
	if initialLightBackground == nil {
		b := false
		initialLightBackground = &b
	}
	e.SetTheme(NewDefaultTheme())
}

// setLightTheme sets the light theme suitable for xterm
func (e *Editor) setLightTheme() {
	if initialLightBackground == nil {
		b := true
		initialLightBackground = &b
	}
	e.SetTheme(NewLightTheme())
}

// setRedBlackTheme sets a red/black/gray theme
func (e *Editor) setRedBlackTheme() {
	if initialLightBackground == nil {
		b := false
		initialLightBackground = &b
	}
	e.SetTheme(NewRedBlackTheme())
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
