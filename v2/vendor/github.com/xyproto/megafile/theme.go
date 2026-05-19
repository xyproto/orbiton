package megafile

import (
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/syntax"
	"github.com/xyproto/themes"
	"github.com/xyproto/vt"
)

// applyThemeFromEnv checks the O_THEME (or THEME) environment variable and
// applies the corresponding theme to the state. Also respects NO_COLOR.
func (s *State) applyThemeFromEnv() {
	if envNoColor {
		return
	}

	themeOverride := env.StrAlt("O_THEME", "OTHEME", "THEME")
	if themeOverride == "" {
		return
	}

	// If O_THEME points to a base16 scheme file, load it directly
	if strings.HasSuffix(themeOverride, ".yaml") || strings.HasSuffix(themeOverride, ".yml") {
		if themes.TermHas256Colors() {
			if theme, err := themes.NewThemeFromBase16File(themeOverride); err == nil {
				s.applyTheme(theme)
			}
		}
		return
	}

	ti, ok := themes.ThemeByEnvValue(themeOverride)
	if !ok {
		return
	}

	// For themes that need color registration, check terminal capabilities
	if ti.NeedsRegistration {
		if themes.TermHas256Colors() {
			ti.Register()
		} else if ti.Fallback16 != "" {
			// Fall back to the 16-color variant
			if fb, ok := themes.ThemeByEnvValue(ti.Fallback16); ok {
				ti = fb
			} else {
				return
			}
		} else {
			return
		}
	}

	theme := ti.New()
	s.applyTheme(theme)
}

// applyTheme sets the syntax highlighting config and adjusts UI colors from
// the given theme.
func (s *State) applyTheme(theme themes.Theme) {
	tc := theme.TextConfig()
	s.SyntaxTextConfig = tc
	s.Light = theme.Light
	syntax.DefaultTextConfig = *tc

	// Apply theme UI colors when they are set
	if theme.Foreground != 0 {
		s.WrittenTextColor = theme.Foreground
	}
	if theme.Background != 0 {
		s.Background = theme.Background
		s.EdgeBackground = theme.Background
	}
	if theme.SearchHighlight != 0 {
		s.HighlightForeground = theme.SearchHighlight
	}

	// Rebuild the vt text output so new custom color names take effect
	vt.RebuildTagReplacers()
}
