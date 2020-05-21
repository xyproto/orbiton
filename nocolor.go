package main

import (
	"os"

	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

func (e *Editor) respectNoColorEnvironmentVariable() {
	if os.Getenv("NO_COLOR") != "" {
		e.fg = vt100.Default
		e.bg = vt100.BackgroundDefault

		// The following is not strictly needed, since the text will be black and white just by setting e.fg and e.bg above
		e.searchFg = vt100.Default
		e.gitColor = vt100.Default
		e.multiLineComment = vt100.Default
		e.multiLineString = vt100.Default
		syntax.DefaultTextConfig.String = ""
		syntax.DefaultTextConfig.Keyword = ""
		syntax.DefaultTextConfig.Comment = ""
		syntax.DefaultTextConfig.Type = ""
		syntax.DefaultTextConfig.Literal = ""
		syntax.DefaultTextConfig.Punctuation = ""
		syntax.DefaultTextConfig.Plaintext = ""
		syntax.DefaultTextConfig.Tag = ""
		syntax.DefaultTextConfig.TextTag = ""
		syntax.DefaultTextConfig.TextAttrName = ""
		syntax.DefaultTextConfig.TextAttrValue = ""
		syntax.DefaultTextConfig.Decimal = ""
		syntax.DefaultTextConfig.AndOr = ""
		syntax.DefaultTextConfig.Star = ""
		syntax.DefaultTextConfig.Class = ""
		syntax.DefaultTextConfig.Private = ""
		syntax.DefaultTextConfig.Protected = ""
		syntax.DefaultTextConfig.Public = ""
		syntax.DefaultTextConfig.Whitespace = ""
	}
}

func (status *StatusBar) respectNoColorEnvironmentVariable() {
	if os.Getenv("NO_COLOR") != "" {
		status.fg = vt100.Default
		status.bg = vt100.BackgroundDefault
	}
}
