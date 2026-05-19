package main

import (
	"github.com/xyproto/syntax"
	"github.com/xyproto/themes"
	"github.com/xyproto/vt"
)

// TODO: Restructure how themes are stored, so that it's easier to list all themes that
// works with a dark background or all that works with a light background, ref. initialLightBackground

var initialLightBackground *bool

func syncInitialLightBackground() {
	themes.InitialLightBackground = initialLightBackground
}

func setInitialLightBackground(b bool) {
	initialLightBackground = &b
	syncInitialLightBackground()
}

// termHas256Colors reports whether the terminal actually supports >=256 colors.
func termHas256Colors() bool {
	return themes.TermHas256Colors()
}

// registerVSColors adds the VS Code light palette entries to the vt color maps
// and rebuilds the tag replacers so the custom color names are recognized.
func registerVSColors() {
	themes.RegisterVSColors()
	tout = vt.New()
}

// registerXoria256Colors adds the Xoria256 palette entries to the vt color maps
// and rebuilds the tag replacers so the custom color names are recognized.
func registerXoria256Colors() {
	themes.RegisterXoria256Colors()
	tout = vt.New()
}

// registerGruvboxColors adds the Gruvbox palette entries to the vt color maps
// and rebuilds the tag replacers so the custom color names are recognized.
func registerGruvboxColors() {
	themes.RegisterGruvboxColors()
	tout = vt.New()
}

// registerMonokaiColors adds the Monokai palette entries to the vt color maps
// and rebuilds the tag replacers so the custom color names are recognized.
func registerMonokaiColors() {
	themes.RegisterMonokaiColors()
	tout = vt.New()
}

// SetTheme assigns the given theme to the Editor,
// and also configures syntax highlighting by setting vt.DefaultTextConfig.
// Light/dark, syntax highlighting and no color information is also set.
// Respect the NO_COLOR environment variable. May set e.NoSyntaxHighlight to true.
func (e *Editor) SetTheme(theme themes.Theme, bs ...bool) {
	if len(bs) == 1 {
		setInitialLightBackground(bs[0])
	}
	if envNoColor {
		if initialLightBackground != nil && *initialLightBackground { // light
			theme = themes.NewNoColorLightBackgroundTheme()
		} else { // dark
			theme = themes.NewNoColorDarkBackgroundTheme()
		}
		e.syntaxHighlight = false
	}
	e.Theme = theme
	e.stickyStatusBars = theme.StickyStatusBars
	syntax.DefaultTextConfig = *(theme.TextConfig())
	if initialLightBackground != nil && *initialLightBackground { // light
		e.makeLightAdjustments()
	}
}

// setDefaultTheme sets the default colors
func (e *Editor) setDefaultTheme() {
	e.SetTheme(themes.NewDefaultTheme())
}

// setVSTheme sets the VS theme, preferring true-color when supported
func (e *Editor) setVSTheme(bs ...bool) {
	if len(bs) == 1 {
		setInitialLightBackground(bs[0])
	}
	if initialLightBackground != nil && *initialLightBackground { // light
		if termHas256Colors() {
			registerVSColors()
			e.SetTheme(themes.NewVSTrueColorTheme())
		} else {
			e.SetTheme(themes.NewLightVSTheme())
		}
	} else { // dark
		e.SetTheme(themes.NewDarkVSTheme())
	}
}

// setNoColorTheme sets the NoColor theme, and considers the background color
func (e *Editor) setNoColorTheme() {
	if initialLightBackground != nil && *initialLightBackground { // light
		e.Theme = themes.NewNoColorLightBackgroundTheme()
	} else { // dark
		e.Theme = themes.NewNoColorDarkBackgroundTheme()
	}
	e.stickyStatusBars = e.StickyStatusBars
	syntax.DefaultTextConfig = *(e.TextConfig())
	if initialLightBackground != nil && *initialLightBackground { // light
		e.makeLightAdjustments()
	}
}

// setLightVSTheme sets the light VS theme, preferring true-color when supported
func (e *Editor) setLightVSTheme() {
	if termHas256Colors() {
		registerVSColors()
		e.SetTheme(themes.NewVSTrueColorTheme())
	} else {
		e.SetTheme(themes.NewLightVSTheme())
	}
}

// setBlueEditTheme sets a blue/yellow/gray theme, for light or dark backgrounds
// if given "true" as an argument, then a light background is assumed
func (e *Editor) setBlueEditTheme(bs ...bool) {
	if len(bs) == 1 {
		setInitialLightBackground(bs[0])
	}
	if initialLightBackground != nil && *initialLightBackground { // light
		e.SetTheme(themes.NewLightBlueEditTheme())
	} else { // dark
		e.SetTheme(themes.NewDarkBlueEditTheme())
	}
}

// setGrayTheme sets a gray theme
func (e *Editor) setGrayTheme() {
	e.SetTheme(themes.NewGrayTheme())
}

// setAmberTheme sets an amber theme
func (e *Editor) setAmberTheme() {
	e.SetTheme(themes.NewAmberTheme())
}

// setGreenTheme sets a green theme
func (e *Editor) setGreenTheme() {
	e.SetTheme(themes.NewGreenTheme())
}

// setBlueTheme sets a blue theme
func (e *Editor) setBlueTheme() {
	e.SetTheme(themes.NewBlueTheme())
}

func (e *Editor) makeLightAdjustments() {
	if e.HighlightForeground == vt.White && e.Background != vt.BackgroundBlack && e.Light {
		e.HighlightForeground = vt.Black
	}
}
