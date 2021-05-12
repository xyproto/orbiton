package main

import (
	"strings"

	"github.com/xyproto/syntax"
)

func (e *Editor) arrowReplace(s string, c string) string {
	arrowColor := syntax.DefaultTextConfig.Dollar
	fieldColor := syntax.DefaultTextConfig.Protected
	s = strings.Replace(s, ">-<", "><off><"+arrowColor+">-<", -1)
	s = strings.Replace(s, ">"+escapedGreaterThan, "><off><"+arrowColor+">"+escapedGreaterThan+"<off><" + fieldColor + ">", -1)
	return s
}
