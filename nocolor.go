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
		e.searchFg = vt100.Default
		e.gitColor = vt100.Default
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
	}
}

func (status *StatusBar) respectNoColorEnvironmentVariable() {
	if os.Getenv("NO_COLOR") != "" {
		status.fg = vt100.Default
		status.bg = vt100.BackgroundDefault
	}
}
