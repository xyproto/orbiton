package main

import (
	"strings"

	"github.com/xyproto/syntax"
)

func (e *Editor) arrowReplace(s string, c string) string {
	arrowColor := syntax.DefaultTextConfig.Keyword
	s = strings.Replace(s, ">-<", "><off><"+arrowColor+">-<", -1)
	s = strings.Replace(s, ">"+escapedGreaterThan, "><off><"+arrowColor+">"+escapedGreaterThan+"<off>", -1)
	return s
}
