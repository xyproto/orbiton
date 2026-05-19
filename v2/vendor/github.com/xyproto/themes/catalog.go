package themes

// ThemeInfo holds metadata about a built-in theme, for use in menus and listings.
type ThemeInfo struct {
	New               func() Theme // constructor
	Register          func()       // registers custom colors (call before New if NeedsRegistration)
	Name              string       // display name
	EnvValue          string       // O_THEME value (e.g., "synthwave", "gruvbox")
	Fallback16        string       // EnvValue of the 16-color fallback, if any
	NeedsRegistration bool         // requires Register() before use
	Needs256Colors    bool         // requires 256-color or true-color terminal
	Mono              bool         // monochrome theme (no syntax highlighting)
	Light             bool         // designed for light backgrounds
}

// BuiltinThemes returns all built-in themes in menu display order.
func BuiltinThemes() []ThemeInfo {
	noop := func() {}
	return []ThemeInfo{
		{
			Name:     "Default",
			EnvValue: "default",
			New:      NewDefaultTheme,
			Register: noop,
		},
		{
			Name:     "Synthwave",
			EnvValue: "synthwave",
			New:      NewSynthwaveTheme,
			Register: noop,
		},
		{
			Name:     "Red & Black",
			EnvValue: "redblack",
			New:      NewRedBlackTheme,
			Register: noop,
		},
		{
			Name:     "VS",
			EnvValue: "vs",
			Light:    true,
			New:      NewLightVSTheme,
			Register: noop,
		},
		{
			Name:     "Orb",
			EnvValue: "orb",
			New:      NewOrbTheme,
			Register: noop,
		},
		{
			Name:     "Litmus",
			EnvValue: "litmus",
			New:      NewLitmusTheme,
			Register: noop,
		},
		{
			Name:     "Teal",
			EnvValue: "teal",
			New:      NewTealTheme,
			Register: noop,
		},
		{
			Name:     "Blue Edit",
			EnvValue: "blueedit",
			New:      NewDarkBlueEditTheme,
			Register: noop,
		},
		{
			Name:     "Pinetree",
			EnvValue: "pinetree",
			New:      NewPinetreeTheme,
			Register: noop,
		},
		{
			Name:     "Zulu",
			EnvValue: "zulu",
			New:      NewZuluTheme,
			Register: noop,
		},
		{
			Name:              "Xoria",
			EnvValue:          "xoria",
			NeedsRegistration: true,
			Needs256Colors:    true,
			Fallback16:        "xoria16",
			New:               NewXoria256Theme,
			Register:          RegisterXoria256Colors,
		},
		{
			Name:     "Xoria 16",
			EnvValue: "xoria16",
			New:      NewXoria16Theme,
			Register: noop,
		},
		{
			Name:              "Gruvbox",
			EnvValue:          "gruvbox",
			NeedsRegistration: true,
			Needs256Colors:    true,
			Fallback16:        "gruvbox16",
			New:               NewGruvboxTheme,
			Register:          RegisterGruvboxColors,
		},
		{
			Name:     "Gruvbox 16",
			EnvValue: "gruvbox16",
			New:      NewGruvbox16Theme,
			Register: noop,
		},
		{
			Name:              "Monokai",
			EnvValue:          "monokai",
			NeedsRegistration: true,
			Needs256Colors:    true,
			Fallback16:        "monokai16",
			New:               NewMonokaiTheme,
			Register:          RegisterMonokaiColors,
		},
		{
			Name:     "Monokai 16",
			EnvValue: "monokai16",
			New:      NewMonokai16Theme,
			Register: noop,
		},
		{
			Name:     "Gray Mono",
			EnvValue: "graymono",
			Mono:     true,
			New:      NewGrayTheme,
			Register: noop,
		},
		{
			Name:     "Amber Mono",
			EnvValue: "ambermono",
			Mono:     true,
			New:      NewAmberTheme,
			Register: noop,
		},
		{
			Name:     "Green Mono",
			EnvValue: "greenmono",
			Mono:     true,
			New:      NewGreenTheme,
			Register: noop,
		},
		{
			Name:     "Blue Mono",
			EnvValue: "bluemono",
			Mono:     true,
			New:      NewBlueTheme,
			Register: noop,
		},
	}
}

// ThemeByEnvValue looks up a theme by its O_THEME value.
// Returns the ThemeInfo and true if found, or a zero value and false otherwise.
func ThemeByEnvValue(envValue string) (ThemeInfo, bool) {
	for _, t := range BuiltinThemes() {
		if t.EnvValue == envValue {
			return t, true
		}
	}
	return ThemeInfo{}, false
}
