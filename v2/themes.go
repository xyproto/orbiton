package main

// TODO: Restructure how themes are stored, so that it's easier to list all themes that works with a dark background or all that works with a light background, ref. initialLightBackground

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
	TextTag                     string
	TextAttrName                string
	RainbowParenColors          []AttributeColor
	HeaderBulletColor           AttributeColor
	MultiLineString             AttributeColor
	DebugInstructionsBackground AttributeColor
	Git                         AttributeColor
	MultiLineComment            AttributeColor
	SearchHighlight             AttributeColor
	StatusErrorBackground       AttributeColor
	StatusErrorForeground       AttributeColor
	StatusBackground            AttributeColor
	StatusForeground            AttributeColor
	Background                  AttributeColor
	Foreground                  AttributeColor
	MarkdownTextColor           AttributeColor
	BoxUpperEdge                AttributeColor
	HeaderTextColor             AttributeColor
	ListBulletColor             AttributeColor
	ListTextColor               AttributeColor
	ListCodeColor               AttributeColor
	CodeColor                   AttributeColor
	CodeBlockColor              AttributeColor
	ImageColor                  AttributeColor
	LinkColor                   AttributeColor
	QuoteColor                  AttributeColor
	QuoteTextColor              AttributeColor
	HTMLColor                   AttributeColor
	CommentColor                AttributeColor
	BoldColor                   AttributeColor
	ItalicsColor                AttributeColor
	StrikeColor                 AttributeColor
	TableColor                  AttributeColor
	CheckboxColor               AttributeColor
	XColor                      AttributeColor
	DebugInstructionsForeground AttributeColor
	UnmatchedParenColor         AttributeColor
	MenuTitleColor              AttributeColor
	MenuArrowColor              AttributeColor
	MenuTextColor               AttributeColor
	MenuHighlightColor          AttributeColor
	MenuSelectedColor           AttributeColor
	ManSectionColor             AttributeColor
	ManSynopsisColor            AttributeColor
	BoxTextColor                AttributeColor
	BoxBackground               AttributeColor
	BoxHighlight                AttributeColor
	DebugRunningBackground      AttributeColor
	DebugStoppedBackground      AttributeColor
	DebugRegistersBackground    AttributeColor
	DebugOutputBackground       AttributeColor
	TableBackground             AttributeColor
	JumpToLetterColor           AttributeColor
	NanoHelpForeground          AttributeColor
	NanoHelpBackground          AttributeColor
	HighlightForeground         AttributeColor
	HighlightBackground         AttributeColor
	StatusMode                  bool
	Light                       bool
}

// NewDefaultTheme creates a new default Theme struct
func NewDefaultTheme() Theme {
	return Theme{
		Name:                        "Default",
		Light:                       false,
		Foreground:                  LightBlue,
		Background:                  BackgroundDefault,
		StatusForeground:            White,
		StatusBackground:            BackgroundBlack,
		StatusErrorForeground:       LightRed,
		StatusErrorBackground:       BackgroundDefault,
		SearchHighlight:             LightMagenta,
		MultiLineComment:            Gray,
		MultiLineString:             Magenta,
		HighlightForeground:         White,
		HighlightBackground:         BackgroundDefault,
		Git:                         LightGreen,
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
		RainbowParenColors:          []AttributeColor{LightMagenta, LightRed, Yellow, LightYellow, LightGreen, LightBlue, Red},
		MarkdownTextColor:           LightBlue,
		HeaderBulletColor:           DarkGray,
		HeaderTextColor:             LightGreen,
		ListBulletColor:             Red,
		ListTextColor:               LightCyan,
		ListCodeColor:               Default,
		CodeColor:                   Default,
		CodeBlockColor:              Default,
		ImageColor:                  LightYellow,
		LinkColor:                   Magenta,
		QuoteColor:                  Yellow,
		QuoteTextColor:              LightCyan,
		HTMLColor:                   Default,
		CommentColor:                DarkGray,
		BoldColor:                   LightYellow,
		ItalicsColor:                White,
		StrikeColor:                 DarkGray,
		TableColor:                  Blue,
		CheckboxColor:               Default,
		XColor:                      LightYellow,
		TableBackground:             BackgroundDefault,
		UnmatchedParenColor:         White,
		MenuTitleColor:              LightYellow,
		MenuArrowColor:              Red,
		MenuTextColor:               Gray,
		MenuHighlightColor:          LightBlue,
		MenuSelectedColor:           LightCyan,
		ManSectionColor:             LightRed,
		ManSynopsisColor:            LightYellow,
		BoxTextColor:                Black,
		BoxBackground:               BackgroundBlue,
		BoxHighlight:                LightYellow,
		DebugRunningBackground:      BackgroundCyan,
		DebugStoppedBackground:      BackgroundMagenta,
		DebugRegistersBackground:    BackgroundBlue,
		DebugOutputBackground:       BackgroundYellow,
		DebugInstructionsForeground: LightYellow,
		DebugInstructionsBackground: BackgroundMagenta,
		BoxUpperEdge:                White,
		JumpToLetterColor:           LightRed,
		NanoHelpForeground:          Black,
		NanoHelpBackground:          BackgroundGray,
	}
}

// NewOrbTheme creates a new logical Theme struct with a refined color palette
func NewOrbTheme() Theme {
	return Theme{
		Name:                        "Orb",
		Light:                       false,
		Foreground:                  LightGray,
		Background:                  BackgroundBlack,
		StatusForeground:            LightGray,
		StatusBackground:            Gray,
		StatusErrorForeground:       LightRed,
		StatusErrorBackground:       BackgroundBlack,
		SearchHighlight:             LightMagenta,
		MultiLineComment:            Gray,
		MultiLineString:             LightCyan,
		HighlightForeground:         White,
		HighlightBackground:         BackgroundBlack,
		Git:                         LightCyan,
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
		RainbowParenColors:          []AttributeColor{LightRed, LightCyan, LightGreen, LightYellow, LightBlue, Gray, LightGray},
		MarkdownTextColor:           LightGray,
		HeaderBulletColor:           White,
		HeaderTextColor:             LightCyan,
		ListBulletColor:             LightRed,
		ListTextColor:               LightCyan,
		ListCodeColor:               White,
		CodeColor:                   White,
		CodeBlockColor:              White,
		ImageColor:                  LightGreen,
		LinkColor:                   LightCyan,
		QuoteColor:                  LightGreen,
		QuoteTextColor:              White,
		HTMLColor:                   White,
		CommentColor:                Gray,
		BoldColor:                   LightGreen,
		ItalicsColor:                LightGray,
		StrikeColor:                 White,
		TableColor:                  White,
		CheckboxColor:               White,
		XColor:                      LightGreen,
		TableBackground:             BackgroundBlack,
		UnmatchedParenColor:         LightRed,
		MenuTitleColor:              LightMagenta,
		MenuArrowColor:              White,
		MenuTextColor:               Blue,
		MenuHighlightColor:          LightCyan,
		MenuSelectedColor:           LightRed,
		ManSectionColor:             LightCyan,
		ManSynopsisColor:            LightGreen,
		BoxTextColor:                White,
		BoxBackground:               DarkGray,
		BoxHighlight:                LightYellow,
		DebugRunningBackground:      Cyan,
		DebugStoppedBackground:      LightRed,
		DebugRegistersBackground:    DarkGray,
		DebugOutputBackground:       LightGreen,
		DebugInstructionsForeground: LightGreen,
		DebugInstructionsBackground: DarkGray,
		BoxUpperEdge:                White,
		JumpToLetterColor:           LightRed,
		NanoHelpForeground:          White,
		NanoHelpBackground:          DarkGray,
	}
}

// NewPinetreeTheme creates a new Theme struct based on the base16-snazzy theme
func NewPinetreeTheme() Theme {
	return Theme{
		Name:                        "Pinetree",
		Light:                       false,
		Foreground:                  LightGray,
		Background:                  BackgroundBlack,
		StatusForeground:            LightGray,
		StatusBackground:            BackgroundBlack,
		StatusErrorForeground:       LightRed,
		StatusErrorBackground:       BackgroundBlack,
		SearchHighlight:             Yellow,
		MultiLineComment:            DarkGray,
		MultiLineString:             Magenta,
		HighlightForeground:         LightCyan,
		HighlightBackground:         BackgroundBlack,
		Git:                         LightGreen,
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
		RainbowParenColors:          []AttributeColor{LightMagenta, LightRed, Yellow, LightYellow, LightGreen, LightBlue, Red},
		MarkdownTextColor:           LightGray,
		HeaderBulletColor:           DarkGray,
		HeaderTextColor:             LightBlue,
		ListBulletColor:             LightRed,
		ListTextColor:               LightGray,
		ListCodeColor:               LightGreen,
		CodeColor:                   LightGreen,
		CodeBlockColor:              BackgroundBlack,
		ImageColor:                  Yellow,
		LinkColor:                   LightBlue,
		QuoteColor:                  Yellow,
		QuoteTextColor:              LightGray,
		HTMLColor:                   LightRed,
		CommentColor:                DarkGray,
		BoldColor:                   White,
		ItalicsColor:                LightBlue,
		StrikeColor:                 DarkGray,
		TableColor:                  LightBlue,
		CheckboxColor:               LightGray,
		XColor:                      LightRed,
		TableBackground:             BackgroundBlack,
		UnmatchedParenColor:         LightRed,
		MenuTitleColor:              LightGreen,
		MenuArrowColor:              LightRed,
		MenuTextColor:               LightGray,
		MenuHighlightColor:          LightCyan,
		MenuSelectedColor:           White,
		ManSectionColor:             LightRed,
		ManSynopsisColor:            Yellow,
		BoxTextColor:                Black,
		BoxBackground:               BackgroundBlue,
		BoxHighlight:                LightYellow,
		DebugRunningBackground:      BackgroundGreen,
		DebugStoppedBackground:      BackgroundMagenta,
		DebugRegistersBackground:    BackgroundBlue,
		DebugOutputBackground:       BackgroundYellow,
		DebugInstructionsForeground: LightYellow,
		DebugInstructionsBackground: BackgroundMagenta,
		BoxUpperEdge:                LightGray,
		JumpToLetterColor:           LightRed,
		NanoHelpForeground:          Black,
		NanoHelpBackground:          BackgroundGray,
	}
}

// NewLitmusTheme creates a new default Theme struct
func NewLitmusTheme() Theme {
	return Theme{
		Name:                        "Litmus",
		Light:                       false,
		Foreground:                  Default,
		Background:                  BackgroundGray,
		StatusForeground:            Gray,
		StatusBackground:            BackgroundBlack,
		StatusErrorForeground:       LightRed,
		StatusErrorBackground:       BackgroundDefault,
		SearchHighlight:             LightMagenta,
		MultiLineComment:            Gray,
		MultiLineString:             Magenta,
		HighlightForeground:         LightRed,
		HighlightBackground:         BackgroundGray,
		Git:                         Black,
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
		RainbowParenColors:          []AttributeColor{LightMagenta, LightRed, Yellow, Green, Blue, LightBlue, Red},
		MarkdownTextColor:           Black,
		HeaderBulletColor:           DarkGray,
		HeaderTextColor:             Black,
		ListBulletColor:             Red,
		ListTextColor:               LightBlue,
		ListCodeColor:               Black,
		CodeColor:                   Black,
		CodeBlockColor:              Black,
		ImageColor:                  Red,
		LinkColor:                   Magenta,
		QuoteColor:                  Red,
		QuoteTextColor:              LightBlue,
		HTMLColor:                   Black,
		CommentColor:                DarkGray,
		BoldColor:                   Red,
		ItalicsColor:                DarkGray,
		StrikeColor:                 DarkGray,
		TableColor:                  Black,
		CheckboxColor:               Black,
		XColor:                      Red,
		TableBackground:             BackgroundGray,
		UnmatchedParenColor:         Yellow,
		MenuTitleColor:              Black,
		MenuArrowColor:              Red,
		MenuTextColor:               Gray,
		MenuHighlightColor:          Cyan,
		MenuSelectedColor:           LightBlue,
		ManSectionColor:             LightRed,
		ManSynopsisColor:            Red,
		BoxTextColor:                Black,
		BoxBackground:               BackgroundCyan,
		BoxHighlight:                Red,
		DebugRunningBackground:      BackgroundBlue,
		DebugStoppedBackground:      BackgroundMagenta,
		DebugRegistersBackground:    BackgroundCyan,
		DebugOutputBackground:       BackgroundGray,
		DebugInstructionsForeground: Red,
		DebugInstructionsBackground: BackgroundMagenta,
		BoxUpperEdge:                DarkGray,
		JumpToLetterColor:           LightRed,
		NanoHelpForeground:          Black,
		NanoHelpBackground:          BackgroundGray,
	}
}

// NewSynthwaveTheme creates a new Theme struct
func NewSynthwaveTheme() Theme {
	return Theme{
		Name:                        "Synthwave",
		Light:                       false,
		Foreground:                  LightBlue,
		Background:                  BackgroundDefault,
		StatusForeground:            White,
		StatusBackground:            BackgroundBlack,
		StatusErrorForeground:       Magenta,
		StatusErrorBackground:       BackgroundDefault,
		SearchHighlight:             LightMagenta,
		MultiLineComment:            Gray,
		MultiLineString:             Magenta,
		HighlightForeground:         White,
		HighlightBackground:         BackgroundDefault,
		Git:                         Cyan,
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
		RainbowParenColors:          []AttributeColor{LightRed, LightMagenta, Blue, LightCyan, LightBlue, Magenta, Cyan},
		MarkdownTextColor:           LightBlue,
		HeaderBulletColor:           DarkGray,
		HeaderTextColor:             Cyan,
		ListBulletColor:             Magenta,
		ListTextColor:               LightCyan,
		ListCodeColor:               Default,
		CodeColor:                   Default,
		CodeBlockColor:              Default,
		ImageColor:                  LightGray,
		LinkColor:                   LightMagenta,
		QuoteColor:                  Gray,
		QuoteTextColor:              LightCyan,
		HTMLColor:                   Default,
		CommentColor:                DarkGray,
		BoldColor:                   LightGray,
		ItalicsColor:                White,
		StrikeColor:                 DarkGray,
		TableColor:                  Blue,
		CheckboxColor:               Default,
		XColor:                      LightGray,
		TableBackground:             BackgroundDefault,
		UnmatchedParenColor:         LightRed, // to really stand out
		MenuTitleColor:              Cyan,
		MenuArrowColor:              Magenta,
		MenuTextColor:               Gray,
		MenuHighlightColor:          LightBlue,
		MenuSelectedColor:           LightCyan,
		ManSectionColor:             LightMagenta,
		ManSynopsisColor:            LightGray,
		BoxTextColor:                Black,
		BoxBackground:               BackgroundBlue,
		BoxHighlight:                LightGray,
		DebugRunningBackground:      BackgroundCyan,
		DebugStoppedBackground:      BackgroundRed,
		DebugRegistersBackground:    BackgroundBlue,
		DebugOutputBackground:       BackgroundGray,
		DebugInstructionsForeground: LightGray,
		DebugInstructionsBackground: BackgroundRed,
		BoxUpperEdge:                White,
		JumpToLetterColor:           LightMagenta,
		NanoHelpForeground:          Black,
		NanoHelpBackground:          BackgroundGray,
	}
}

// NewTealTheme creates a new Theme struct
func NewTealTheme() Theme {
	return Theme{
		Name:                        "Teal",
		Light:                       false,
		Foreground:                  Cyan,
		Background:                  BackgroundDefault,
		StatusForeground:            White,
		StatusBackground:            BackgroundBlack,
		StatusErrorForeground:       LightRed,
		StatusErrorBackground:       BackgroundGray,
		SearchHighlight:             Red,
		MultiLineComment:            Gray,
		MultiLineString:             Blue,
		HighlightForeground:         White,
		HighlightBackground:         BackgroundDefault,
		Git:                         Blue,
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
		RainbowParenColors:          []AttributeColor{LightRed, LightGray, LightBlue, Blue},
		MarkdownTextColor:           Cyan,
		HeaderBulletColor:           LightGray,
		HeaderTextColor:             LightGray,
		ListBulletColor:             LightGray,
		ListTextColor:               Cyan,
		ListCodeColor:               Cyan,
		CodeColor:                   LightGray,
		CodeBlockColor:              LightGray,
		ImageColor:                  LightGray,
		LinkColor:                   LightGray,
		QuoteColor:                  LightGray,
		QuoteTextColor:              LightGray,
		HTMLColor:                   Cyan,
		CommentColor:                DarkGray,
		BoldColor:                   LightGray,
		ItalicsColor:                LightGray,
		StrikeColor:                 DarkGray,
		TableColor:                  LightGray,
		CheckboxColor:               Cyan,
		XColor:                      LightGray,
		TableBackground:             BackgroundDefault,
		UnmatchedParenColor:         LightGray,
		MenuTitleColor:              LightBlue,
		MenuArrowColor:              LightCyan,
		MenuTextColor:               Gray,
		MenuHighlightColor:          Cyan,
		MenuSelectedColor:           White,
		ManSectionColor:             LightGray,
		ManSynopsisColor:            LightGray,
		BoxTextColor:                Black,
		BoxBackground:               BackgroundBlue,
		BoxHighlight:                LightGray,
		DebugRunningBackground:      BackgroundBlue,
		DebugStoppedBackground:      BackgroundGray,
		DebugRegistersBackground:    BackgroundGreen,
		DebugOutputBackground:       BackgroundCyan,
		DebugInstructionsForeground: LightCyan,
		DebugInstructionsBackground: BackgroundBlue,
		BoxUpperEdge:                LightGray,
		JumpToLetterColor:           LightGreen,
		NanoHelpForeground:          Black,
		NanoHelpBackground:          BackgroundGray,
	}
}

// NewRedBlackTheme creates a new red/black/gray/white Theme struct
func NewRedBlackTheme() Theme {
	// NOTE: Dark gray may not be visible with light terminal emulator themes
	return Theme{
		Name:                        "Red & black",
		Light:                       false,
		Foreground:                  LightGray,
		Background:                  BackgroundBlack, // Dark gray background, as opposed toBackgroundDefault
		StatusForeground:            White,
		StatusBackground:            BackgroundBlack,
		StatusErrorForeground:       LightRed,
		StatusErrorBackground:       BackgroundDefault,
		SearchHighlight:             Red,
		MultiLineComment:            DarkGray,
		MultiLineString:             LightGray,
		HighlightForeground:         LightGray,
		HighlightBackground:         BackgroundBlack,
		Git:                         LightGreen,
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
		RainbowParenColors:          []AttributeColor{LightGray, White, Red},
		MarkdownTextColor:           LightGray,
		HeaderBulletColor:           DarkGray,
		HeaderTextColor:             Red,
		ListBulletColor:             Red,
		ListTextColor:               LightGray,
		ListCodeColor:               Default,
		CodeColor:                   White,
		CodeBlockColor:              White,
		ImageColor:                  Red,
		LinkColor:                   DarkGray,
		QuoteColor:                  White,
		QuoteTextColor:              LightGray,
		HTMLColor:                   LightGray,
		CommentColor:                DarkGray,
		BoldColor:                   Red,
		ItalicsColor:                White,
		StrikeColor:                 DarkGray,
		TableColor:                  White,
		CheckboxColor:               Default,
		XColor:                      Red,
		TableBackground:             BackgroundBlack, // Dark gray background, as opposed toBackgroundDefault
		UnmatchedParenColor:         LightCyan,       // To really stand out
		MenuTitleColor:              LightRed,
		MenuArrowColor:              Red,
		MenuTextColor:               Gray,
		MenuHighlightColor:          LightGray,
		MenuSelectedColor:           DarkGray,
		ManSectionColor:             Red,
		ManSynopsisColor:            White,
		BoxTextColor:                Black,
		BoxBackground:               BackgroundGray,
		BoxHighlight:                Red,
		DebugRunningBackground:      BackgroundGray,
		DebugStoppedBackground:      BackgroundGray,
		DebugRegistersBackground:    BackgroundGray,
		DebugOutputBackground:       BackgroundGray,
		DebugInstructionsForeground: Red,
		DebugInstructionsBackground: BackgroundGray,
		BoxUpperEdge:                Black,
		JumpToLetterColor:           Red,
		NanoHelpForeground:          Black,
		NanoHelpBackground:          BackgroundGray,
	}
}

// NewLightBlueEditTheme creates a new blue/gray/yellow Theme struct, for light backgrounds
func NewLightBlueEditTheme() Theme {
	return Theme{
		Name:                        "Blue Edit Light",
		Light:                       true,
		StatusMode:                  false,
		Foreground:                  White,
		Background:                  BackgroundBlue,
		StatusForeground:            Black,
		StatusBackground:            BackgroundCyan,
		StatusErrorForeground:       Black,
		StatusErrorBackground:       BackgroundRed,
		SearchHighlight:             LightRed,
		MultiLineComment:            Gray,
		MultiLineString:             LightYellow,
		HighlightForeground:         LightYellow,
		HighlightBackground:         BackgroundBlue,
		Git:                         White,
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
		RainbowParenColors:          []AttributeColor{LightCyan, LightYellow, LightGreen, White},
		MarkdownTextColor:           White,
		HeaderBulletColor:           LightGray,
		HeaderTextColor:             White,
		ListBulletColor:             LightCyan,
		ListTextColor:               LightCyan,
		ListCodeColor:               White,
		CodeColor:                   White,
		CodeBlockColor:              White,
		ImageColor:                  LightYellow,
		LinkColor:                   LightYellow,
		QuoteColor:                  LightYellow,
		QuoteTextColor:              LightCyan,
		HTMLColor:                   White,
		CommentColor:                LightGray,
		BoldColor:                   LightYellow,
		ItalicsColor:                White,
		StrikeColor:                 LightGray,
		TableColor:                  White,
		CheckboxColor:               White,
		XColor:                      LightYellow,
		TableBackground:             BackgroundBlue,
		UnmatchedParenColor:         White,
		MenuTitleColor:              LightYellow,
		MenuArrowColor:              LightRed,
		MenuTextColor:               LightYellow,
		MenuHighlightColor:          LightRed,
		MenuSelectedColor:           White,
		ManSectionColor:             LightBlue,
		ManSynopsisColor:            LightBlue,
		BoxTextColor:                Black,
		BoxBackground:               BackgroundGray,
		BoxHighlight:                LightYellow,
		DebugRunningBackground:      BackgroundGray,
		DebugStoppedBackground:      BackgroundMagenta,
		DebugRegistersBackground:    BackgroundMagenta,
		DebugOutputBackground:       BackgroundYellow,
		DebugInstructionsForeground: LightYellow,
		DebugInstructionsBackground: BackgroundCyan,
		BoxUpperEdge:                White,
		JumpToLetterColor:           LightBlue,
		NanoHelpForeground:          Black,
		NanoHelpBackground:          BackgroundGray,
	}
}

// NewDarkBlueEditTheme creates a new blue/gray/yellow Theme struct, for dark backgrounds
func NewDarkBlueEditTheme() Theme {
	return Theme{
		Name:                        "Blue Edit Dark",
		Light:                       false,
		StatusMode:                  false,
		Foreground:                  LightYellow,
		Background:                  BackgroundBlue,
		StatusForeground:            White,
		StatusBackground:            BackgroundCyan,
		StatusErrorForeground:       Red,
		StatusErrorBackground:       BackgroundCyan,
		SearchHighlight:             Red,
		MultiLineComment:            White,
		MultiLineString:             White,
		HighlightForeground:         LightYellow,
		HighlightBackground:         BackgroundBlue,
		Git:                         White,
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
		RainbowParenColors:          []AttributeColor{White, LightYellow},
		MarkdownTextColor:           White,
		HeaderBulletColor:           LightRed,
		HeaderTextColor:             White,
		ListBulletColor:             LightRed,
		ListTextColor:               White,
		ListCodeColor:               White,
		CodeColor:                   LightYellow,
		CodeBlockColor:              LightYellow,
		ImageColor:                  White,
		LinkColor:                   White,
		QuoteColor:                  LightYellow,
		QuoteTextColor:              LightYellow,
		HTMLColor:                   White,
		CommentColor:                LightYellow,
		BoldColor:                   White,
		ItalicsColor:                LightYellow,
		StrikeColor:                 LightYellow,
		TableColor:                  LightYellow,
		CheckboxColor:               White,
		XColor:                      White,
		TableBackground:             BackgroundBlue,
		UnmatchedParenColor:         LightRed,
		MenuTitleColor:              LightYellow,
		MenuArrowColor:              White,
		MenuTextColor:               LightGray,
		MenuHighlightColor:          LightYellow,
		MenuSelectedColor:           LightGreen,
		ManSectionColor:             White,
		ManSynopsisColor:            LightYellow,
		BoxTextColor:                LightYellow,
		BoxBackground:               LightYellow,
		BoxHighlight:                LightYellow,
		DebugRunningBackground:      BackgroundGray,
		DebugStoppedBackground:      BackgroundGray,
		DebugRegistersBackground:    BackgroundGray,
		DebugOutputBackground:       BackgroundGray,
		DebugInstructionsForeground: White,
		DebugInstructionsBackground: BackgroundGray,
		BoxUpperEdge:                LightYellow,
		JumpToLetterColor:           White,
		NanoHelpForeground:          Black,
		NanoHelpBackground:          BackgroundGray,
	}
}

// NewLightVSTheme creates a theme that is suitable for light xterm terminal emulator sessions
func NewLightVSTheme() Theme {
	return Theme{
		Name:                        "VS Light",
		Light:                       true,
		Foreground:                  Black,
		Background:                  BackgroundDefault,
		StatusForeground:            White,
		StatusBackground:            BackgroundBlack,
		StatusErrorForeground:       LightRed,
		StatusErrorBackground:       BackgroundDefault,
		SearchHighlight:             Red,
		MultiLineComment:            Gray,
		MultiLineString:             Red,
		HighlightForeground:         Red,
		HighlightBackground:         BackgroundDefault,
		Git:                         Blue,
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
		RainbowParenColors:          []AttributeColor{Magenta, Black, Blue, Green},
		MarkdownTextColor:           Default,
		HeaderBulletColor:           DarkGray,
		HeaderTextColor:             Blue,
		ListBulletColor:             Red,
		ListTextColor:               Default,
		ListCodeColor:               Red,
		CodeColor:                   Red,
		CodeBlockColor:              Red,
		ImageColor:                  Green,
		LinkColor:                   Magenta,
		QuoteColor:                  Yellow,
		QuoteTextColor:              LightCyan,
		HTMLColor:                   Default,
		CommentColor:                DarkGray,
		BoldColor:                   Blue,
		ItalicsColor:                Blue,
		StrikeColor:                 DarkGray,
		TableColor:                  Blue,
		CheckboxColor:               Default,
		XColor:                      Blue,
		TableBackground:             BackgroundDefault,
		UnmatchedParenColor:         Red,
		MenuTitleColor:              Blue,
		MenuArrowColor:              Red,
		MenuTextColor:               Black,
		MenuHighlightColor:          Red,
		MenuSelectedColor:           LightRed,
		ManSectionColor:             Red,
		ManSynopsisColor:            Blue,
		BoxTextColor:                Black,
		BoxBackground:               BackgroundGray,
		BoxHighlight:                Red,
		DebugRunningBackground:      BackgroundCyan,
		DebugStoppedBackground:      BackgroundDefault,
		DebugRegistersBackground:    BackgroundGray,
		DebugOutputBackground:       BackgroundGray,
		DebugInstructionsForeground: Black,
		DebugInstructionsBackground: BackgroundGray,
		BoxUpperEdge:                Black,
		JumpToLetterColor:           Red,
		NanoHelpForeground:          Black,
		NanoHelpBackground:          BackgroundGray,
	}
}

// NewDarkVSTheme creates a theme that is suitable for dark terminal emulator sessions
func NewDarkVSTheme() Theme {
	return Theme{
		Name:                        "VS Dark",
		Light:                       false,
		Foreground:                  Black,
		Background:                  BackgroundWhite,
		StatusForeground:            White,
		StatusBackground:            BackgroundBlue,
		StatusErrorForeground:       Red,
		StatusErrorBackground:       BackgroundCyan,
		SearchHighlight:             Red,
		MultiLineComment:            Gray,
		MultiLineString:             Red,
		HighlightForeground:         Black,
		HighlightBackground:         BackgroundWhite,
		Git:                         Blue,
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
		RainbowParenColors:          []AttributeColor{Magenta, Black, Blue, Green},
		MarkdownTextColor:           Black,
		HeaderBulletColor:           DarkGray,
		HeaderTextColor:             Blue,
		ListBulletColor:             Red,
		ListTextColor:               Black,
		ListCodeColor:               Red,
		CodeColor:                   Red,
		CodeBlockColor:              Red,
		ImageColor:                  DarkGray,
		LinkColor:                   Magenta,
		QuoteColor:                  Yellow,
		QuoteTextColor:              LightCyan,
		HTMLColor:                   Black,
		CommentColor:                DarkGray,
		BoldColor:                   Blue,
		ItalicsColor:                Blue,
		StrikeColor:                 DarkGray,
		TableColor:                  Blue,
		CheckboxColor:               Black,
		XColor:                      Blue,
		TableBackground:             DarkGray,
		UnmatchedParenColor:         Red,
		MenuTitleColor:              Blue,
		MenuArrowColor:              Red,
		MenuTextColor:               Black,
		MenuHighlightColor:          Red,
		MenuSelectedColor:           LightRed,
		ManSectionColor:             Red,
		ManSynopsisColor:            Blue,
		BoxTextColor:                Black,
		BoxBackground:               BackgroundGray,
		BoxHighlight:                Red,
		DebugRunningBackground:      BackgroundCyan,
		DebugStoppedBackground:      Gray,
		DebugRegistersBackground:    BackgroundGray,
		DebugOutputBackground:       BackgroundGray,
		DebugInstructionsForeground: Black,
		DebugInstructionsBackground: BackgroundGray,
		BoxUpperEdge:                Black,
		JumpToLetterColor:           Red,
		NanoHelpForeground:          Black,
		NanoHelpBackground:          BackgroundGray,
	}
}

// NewGrayTheme returns a theme where all text is light gray
func NewGrayTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Gray Mono"
	t.Foreground = LightGray
	t.Background = BackgroundDefault // black background
	//t.StatusBackground =BackgroundDefault
	//t.StatusErrorBackground =BackgroundDefault
	t.JumpToLetterColor = White // for jumping to a letter with ctrl-l
	return t
}

// NewAmberTheme returns a theme where all text is amber / yellow
func NewAmberTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Amber Mono"
	t.Foreground = Yellow
	t.Background = BackgroundDefault   // black background
	t.JumpToLetterColor = t.Foreground // for jumping to a letter with ctrl-l
	return t
}

// NewGreenTheme returns a theme where all text is green
func NewGreenTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Green Mono"
	t.Foreground = LightGreen
	t.Background = BackgroundDefault   // black background
	t.JumpToLetterColor = t.Foreground // for jumping to a letter with ctrl-l
	return t
}

// NewBlueTheme returns a theme where all text is blue
func NewBlueTheme() Theme {
	t := NewDefaultTheme()
	t.Name = "Blue Mono"
	t.Foreground = LightBlue
	t.Background = BackgroundDefault   // black background
	t.JumpToLetterColor = t.Foreground // for jumping to a letter with ctrl-l
	return t
}

// NewNoColorDarkBackgroundTheme creates a new theme without colors or syntax highlighting
func NewNoColorDarkBackgroundTheme() Theme {
	return Theme{
		Name:                        "No color",
		Light:                       false,
		Foreground:                  Default,
		Background:                  BackgroundDefault,
		StatusForeground:            White,
		StatusBackground:            BackgroundBlack,
		StatusErrorForeground:       White,
		StatusErrorBackground:       BackgroundDefault,
		SearchHighlight:             Default,
		MultiLineComment:            Default,
		MultiLineString:             Default,
		HighlightForeground:         White,
		HighlightBackground:         BackgroundDefault,
		Git:                         White,
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
		RainbowParenColors:          []AttributeColor{Gray},
		MarkdownTextColor:           Default,
		HeaderBulletColor:           Default,
		HeaderTextColor:             Default,
		ListBulletColor:             Default,
		ListTextColor:               Default,
		ListCodeColor:               Default,
		CodeColor:                   Default,
		CodeBlockColor:              Default,
		ImageColor:                  Default,
		LinkColor:                   Default,
		QuoteColor:                  Default,
		QuoteTextColor:              Default,
		HTMLColor:                   Default,
		CommentColor:                Default,
		BoldColor:                   Default,
		ItalicsColor:                Default,
		StrikeColor:                 Default,
		TableColor:                  Default,
		CheckboxColor:               Default,
		XColor:                      White,
		TableBackground:             BackgroundDefault,
		UnmatchedParenColor:         White,
		MenuTitleColor:              White,
		MenuArrowColor:              White,
		MenuTextColor:               Gray,
		MenuHighlightColor:          White,
		MenuSelectedColor:           Black,
		ManSectionColor:             White,
		ManSynopsisColor:            White,
		BoxTextColor:                Black,
		BoxBackground:               BackgroundGray,
		BoxHighlight:                Black,
		DebugRunningBackground:      BackgroundGray,
		DebugStoppedBackground:      BackgroundGray,
		DebugRegistersBackground:    BackgroundGray,
		DebugOutputBackground:       BackgroundGray,
		DebugInstructionsForeground: Black,
		DebugInstructionsBackground: BackgroundGray,
		BoxUpperEdge:                Black,
		JumpToLetterColor:           White,
		NanoHelpForeground:          Black,
		NanoHelpBackground:          BackgroundGray,
	}
}

// NewNoColorLightBackgroundTheme creates a new theme without colors or syntax highlighting
func NewNoColorLightBackgroundTheme() Theme {
	return Theme{
		Name:                        "No color",
		Light:                       true,
		Foreground:                  Default,
		Background:                  BackgroundDefault,
		StatusForeground:            Black,
		StatusBackground:            BackgroundWhite,
		StatusErrorForeground:       Black,
		StatusErrorBackground:       BackgroundDefault,
		SearchHighlight:             Default,
		MultiLineComment:            Default,
		MultiLineString:             Default,
		HighlightForeground:         Default,
		HighlightBackground:         BackgroundDefault,
		Git:                         Black,
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
		RainbowParenColors:          []AttributeColor{Gray},
		MarkdownTextColor:           Default,
		HeaderBulletColor:           Default,
		HeaderTextColor:             Default,
		ListBulletColor:             Default,
		ListTextColor:               Default,
		ListCodeColor:               Default,
		CodeColor:                   Default,
		CodeBlockColor:              Default,
		ImageColor:                  Default,
		LinkColor:                   Default,
		QuoteColor:                  Default,
		QuoteTextColor:              Default,
		HTMLColor:                   Default,
		CommentColor:                Default,
		BoldColor:                   Default,
		ItalicsColor:                Default,
		StrikeColor:                 Default,
		TableColor:                  Default,
		CheckboxColor:               Default,
		XColor:                      Black,
		TableBackground:             BackgroundDefault,
		UnmatchedParenColor:         Black,
		MenuTitleColor:              Black,
		MenuArrowColor:              Black,
		MenuTextColor:               Gray,
		MenuHighlightColor:          Black,
		MenuSelectedColor:           White,
		ManSectionColor:             Black,
		ManSynopsisColor:            Black,
		BoxTextColor:                White,
		BoxBackground:               BackgroundGray,
		BoxHighlight:                White,
		DebugRunningBackground:      BackgroundGray,
		DebugStoppedBackground:      BackgroundGray,
		DebugRegistersBackground:    BackgroundGray,
		DebugOutputBackground:       BackgroundGray,
		DebugInstructionsForeground: White,
		DebugInstructionsBackground: BackgroundGray,
		BoxUpperEdge:                White,
		JumpToLetterColor:           Black,
		NanoHelpBackground:          BackgroundGray,
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
// and also configures syntax highlighting by setting DefaultTextConfig.
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
	DefaultTextConfig = *(theme.TextConfig())
}

// setNoColorTheme sets the NoColor theme, and considers the background color
func (e *Editor) setNoColorTheme() {
	if initialLightBackground != nil && *initialLightBackground {
		e.Theme = NewNoColorLightBackgroundTheme()
	} else {
		e.Theme = NewNoColorDarkBackgroundTheme()
	}
	e.statusMode = e.Theme.StatusMode
	DefaultTextConfig = *(e.Theme.TextConfig())
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
