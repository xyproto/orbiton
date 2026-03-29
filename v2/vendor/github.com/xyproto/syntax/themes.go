package syntax

import (
	"github.com/xyproto/env/v2"
)

// NewDefaultTextConfig returns the TextConfig for the default Orbiton theme.
func NewDefaultTextConfig() TextConfig {
	return TextConfig{
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
		AngleBracket:  "lightyellow",
		Dollar:        "lightred",
		Star:          "lightyellow",
		Static:        "lightyellow",
		Self:          "white",
		Class:         "lightred",
		Private:       "darkred",
		Protected:     "darkyellow",
		Public:        "darkgreen",
		Whitespace:    "",
		AssemblyEnd:   "cyan",
		Mut:           "darkyellow",
	}
}

// NewOrbTextConfig returns the TextConfig for the "orb" theme.
func NewOrbTextConfig() TextConfig {
	return TextConfig{
		String:        "cyan",
		Keyword:       "lightcyan",
		Comment:       "gray",
		Type:          "lightblue",
		Literal:       "lightcyan",
		Punctuation:   "lightgray",
		Plaintext:     "white",
		Tag:           "lightcyan",
		TextTag:       "lightcyan",
		TextAttrName:  "lightblue",
		TextAttrValue: "lightgreen",
		Decimal:       "white",
		AndOr:         "lightcyan",
		AngleBracket:  "lightcyan",
		Dollar:        "lightred",
		Star:          "lightgreen",
		Static:        "lightgreen",
		Self:          "white",
		Class:         "lightcyan",
		Private:       "lightred",
		Protected:     "lightyellow",
		Public:        "lightgreen",
		Whitespace:    "",
		AssemblyEnd:   "lightblue",
		Mut:           "lightgreen",
	}
}

// NewPinetreeTextConfig returns the TextConfig for the "pinetree" theme.
func NewPinetreeTextConfig() TextConfig {
	return TextConfig{
		String:        "lightgreen",
		Keyword:       "lightred",
		Comment:       "darkgray",
		Type:          "lightcyan",
		Literal:       "lightgreen",
		Punctuation:   "lightgray",
		Plaintext:     "lightgray",
		Tag:           "lightred",
		TextTag:       "lightred",
		TextAttrName:  "lightyellow",
		TextAttrValue: "lightgreen",
		Decimal:       "lightgreen",
		AndOr:         "lightred",
		AngleBracket:  "lightred",
		Dollar:        "lightgreen",
		Star:          "lightyellow",
		Static:        "lightblue",
		Self:          "lightgray",
		Class:         "lightblue",
		Private:       "darkred",
		Protected:     "darkyellow",
		Public:        "lightgreen",
		Whitespace:    "",
		AssemblyEnd:   "cyan",
		Mut:           "darkyellow",
	}
}

// NewZuluTextConfig returns the TextConfig for the "zulu" theme.
func NewZuluTextConfig() TextConfig {
	return TextConfig{
		String:        "lightyellow",
		Keyword:       "lightgreen",
		Comment:       "gray",
		Type:          "lightcyan",
		Literal:       "lightmagenta",
		Punctuation:   "lightgray",
		Plaintext:     "lightgray",
		Tag:           "lightcyan",
		TextTag:       "lightcyan",
		TextAttrName:  "lightcyan",
		TextAttrValue: "lightyellow",
		Decimal:       "lightmagenta",
		AndOr:         "lightgreen",
		AngleBracket:  "lightgreen",
		Dollar:        "lightmagenta",
		Star:          "lightyellow",
		Static:        "lightgreen",
		Self:          "white",
		Class:         "lightcyan",
		Private:       "darkyellow",
		Protected:     "yellow",
		Public:        "lightgreen",
		Whitespace:    "",
		AssemblyEnd:   "lightcyan",
		Mut:           "lightgreen",
	}
}

// NewLitmusTextConfig returns the TextConfig for the "litmus" theme.
func NewLitmusTextConfig() TextConfig {
	return TextConfig{
		String:        "blue",
		Keyword:       "lightred",
		Comment:       "darkgray",
		Type:          "cyan",
		Literal:       "black",
		Punctuation:   "black",
		Plaintext:     "black",
		Tag:           "black",
		TextTag:       "black",
		TextAttrName:  "black",
		TextAttrValue: "black",
		Decimal:       "black",
		AndOr:         "lightred",
		AngleBracket:  "lightred",
		Dollar:        "lightred",
		Star:          "magenta",
		Static:        "magenta",
		Self:          "black",
		Class:         "lightred",
		Private:       "red",
		Protected:     "yellow",
		Public:        "green",
		Whitespace:    "",
		AssemblyEnd:   "magenta",
		Mut:           "yellow",
	}
}

// NewSynthwaveTextConfig returns the TextConfig for the "synthwave" theme.
func NewSynthwaveTextConfig() TextConfig {
	return TextConfig{
		String:        "lightgray",
		Keyword:       "magenta",
		Comment:       "gray",
		Type:          "lightblue",
		Literal:       "cyan",
		Punctuation:   "lightblue",
		Plaintext:     "cyan",
		Tag:           "cyan",
		TextTag:       "cyan",
		TextAttrName:  "cyan",
		TextAttrValue: "cyan",
		Decimal:       "white",
		AndOr:         "lightgray",
		AngleBracket:  "lightgray",
		Dollar:        "magenta",
		Star:          "lightgray",
		Static:        "lightgray",
		Self:          "white",
		Class:         "magenta",
		Private:       "magenta",
		Protected:     "blue",
		Public:        "green",
		Whitespace:    "",
		AssemblyEnd:   "cyan",
		Mut:           "darkgray",
	}
}

// NewTealTextConfig returns the TextConfig for the "teal" theme.
func NewTealTextConfig() TextConfig {
	return TextConfig{
		String:        "lightblue",
		Keyword:       "white",
		Comment:       "gray",
		Type:          "lightcyan",
		Literal:       "white",
		Punctuation:   "white",
		Plaintext:     "white",
		Tag:           "white",
		TextTag:       "white",
		TextAttrName:  "white",
		TextAttrValue: "lightblue",
		Decimal:       "white",
		AndOr:         "white",
		AngleBracket:  "white",
		Dollar:        "white",
		Star:          "white",
		Static:        "white",
		Self:          "white",
		Class:         "lightcyan",
		Private:       "lightgray",
		Protected:     "lightgray",
		Public:        "white",
		Whitespace:    "",
		AssemblyEnd:   "white",
		Mut:           "white",
	}
}

// NewRedBlackTextConfig returns the TextConfig for the "redblack" theme.
func NewRedBlackTextConfig() TextConfig {
	return TextConfig{
		String:        "white",
		Keyword:       "darkred",
		Comment:       "darkgray",
		Type:          "white",
		Literal:       "lightgray",
		Punctuation:   "darkred",
		Plaintext:     "lightgray",
		Tag:           "darkred",
		TextTag:       "darkred",
		TextAttrName:  "darkred",
		TextAttrValue: "darkred",
		Decimal:       "white",
		AndOr:         "darkred",
		AngleBracket:  "darkred",
		Dollar:        "white",
		Star:          "white",
		Static:        "white",
		Self:          "white",
		Class:         "darkred",
		Private:       "lightgray",
		Protected:     "lightgray",
		Public:        "white",
		Whitespace:    "",
		AssemblyEnd:   "darkred",
		Mut:           "lightgray",
	}
}

// NewLightBlueEditTextConfig returns the TextConfig for the light "blueedit" theme.
func NewLightBlueEditTextConfig() TextConfig {
	return TextConfig{
		String:        "lightyellow",
		Keyword:       "lightcyan",
		Comment:       "lightgray",
		Type:          "white",
		Literal:       "white",
		Punctuation:   "white",
		Plaintext:     "white",
		Tag:           "white",
		TextTag:       "white",
		TextAttrName:  "white",
		TextAttrValue: "white",
		Decimal:       "white",
		AndOr:         "lightyellow",
		AngleBracket:  "lightyellow",
		Dollar:        "lightred",
		Star:          "lightred",
		Static:        "lightred",
		Self:          "lightyellow",
		Class:         "lightcyan",
		Private:       "lightcyan",
		Protected:     "lightyellow",
		Public:        "white",
		Whitespace:    "",
		AssemblyEnd:   "lightcyan",
		Mut:           "lightyellow",
	}
}

// NewDarkBlueEditTextConfig returns the TextConfig for the dark "blueedit" theme.
func NewDarkBlueEditTextConfig() TextConfig {
	return TextConfig{
		String:        "lightyellow",
		Keyword:       "lightyellow",
		Comment:       "white",
		Type:          "white",
		Literal:       "white",
		Punctuation:   "white",
		Plaintext:     "white",
		Tag:           "white",
		TextTag:       "white",
		TextAttrName:  "white",
		TextAttrValue: "white",
		Decimal:       "lightgreen",
		AndOr:         "white",
		AngleBracket:  "white",
		Dollar:        "lightyellow",
		Star:          "lightyellow",
		Static:        "lightyellow",
		Self:          "lightgreen",
		Class:         "white",
		Private:       "white",
		Protected:     "white",
		Public:        "white",
		Whitespace:    "",
		AssemblyEnd:   "white",
		Mut:           "lightyellow",
	}
}

// NewLightVSTextConfig returns the TextConfig for the light "vs" theme.
func NewLightVSTextConfig() TextConfig {
	return TextConfig{
		String:        "red",
		Keyword:       "blue",
		Comment:       "gray",
		Type:          "blue",
		Literal:       "darkcyan",
		Punctuation:   "black",
		Plaintext:     "black",
		Tag:           "black",
		TextTag:       "black",
		TextAttrName:  "black",
		TextAttrValue: "black",
		Decimal:       "darkcyan",
		AndOr:         "black",
		AngleBracket:  "black",
		Dollar:        "red",
		Star:          "black",
		Static:        "black",
		Self:          "darkcyan",
		Class:         "blue",
		Private:       "black",
		Protected:     "black",
		Public:        "black",
		Whitespace:    "",
		AssemblyEnd:   "red",
		Mut:           "black",
	}
}

// NewDarkVSTextConfig returns the TextConfig for the dark "vs" theme.
func NewDarkVSTextConfig() TextConfig {
	return TextConfig{
		String:        "red",
		Keyword:       "blue",
		Comment:       "gray",
		Type:          "blue",
		Literal:       "darkcyan",
		Punctuation:   "black",
		Plaintext:     "black",
		Tag:           "black",
		TextTag:       "black",
		TextAttrName:  "black",
		TextAttrValue: "black",
		Decimal:       "darkcyan",
		AndOr:         "black",
		AngleBracket:  "black",
		Dollar:        "red",
		Star:          "red",
		Static:        "red",
		Self:          "darkcyan",
		Class:         "blue",
		Private:       "black",
		Protected:     "black",
		Public:        "black",
		Whitespace:    "",
		AssemblyEnd:   "red",
		Mut:           "black",
	}
}

// NewNoColorTextConfig returns an empty TextConfig with no colors.
func NewNoColorTextConfig() TextConfig {
	return TextConfig{}
}

// TextConfigByName returns the TextConfig for the given theme name.
// If the name is not recognized, the default TextConfig is returned.
func TextConfigByName(name string) TextConfig {
	switch name {
	case "default":
		return NewDefaultTextConfig()
	case "orb":
		return NewOrbTextConfig()
	case "pinetree":
		return NewPinetreeTextConfig()
	case "zulu":
		return NewZuluTextConfig()
	case "litmus":
		return NewLitmusTextConfig()
	case "synthwave":
		return NewSynthwaveTextConfig()
	case "teal":
		return NewTealTextConfig()
	case "redblack":
		return NewRedBlackTextConfig()
	case "blueedit":
		return NewDarkBlueEditTextConfig()
	case "vs":
		return NewDarkVSTextConfig()
	case "graymono", "ambermono", "greenmono", "bluemono":
		return NewNoColorTextConfig()
	default:
		return NewDefaultTextConfig()
	}
}

// LightTextConfigByName will return the TextConfig for the given theme name,
// preferring the light variant when one exists.
func LightTextConfigByName(name string) TextConfig {
	switch name {
	case "blueedit":
		return NewLightBlueEditTextConfig()
	case "vs":
		return NewLightVSTextConfig()
	default:
		return TextConfigByName(name)
	}
}

// TextConfigFromEnv will return the TextConfig selected by the O_THEME (or THEME)
// environment variable, falling back to the default.
// If NO_COLOR is set, an empty TextConfig (no colors) is returned.
// If O_LIGHT is set, light theme variants are preferred.
func TextConfigFromEnv() TextConfig {
	if env.Bool("NO_COLOR") {
		return NewNoColorTextConfig()
	}
	name := env.StrAlt("O_THEME", "THEME")
	if name == "" {
		return DefaultTextConfig
	}
	if env.Bool("O_LIGHT") {
		return LightTextConfigByName(name)
	}
	return TextConfigByName(name)
}

// SetDefaultTextConfigFromEnv will update DefaultTextConfig based on O_THEME.
func SetDefaultTextConfigFromEnv() {
	DefaultTextConfig = TextConfigFromEnv()
}
