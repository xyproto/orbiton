package main

import (
	"strings"

	"github.com/xyproto/syntax"
)

var (
	arrowColor = syntax.DefaultTextConfig.Dollar
	fieldColor = syntax.DefaultTextConfig.Protected
)

// Syntax highlight pointer arrows in C and C++
func arrowReplace(s string) string {
	s = strings.Replace(s, ">-<", "><off><"+arrowColor+">-<", -1)
	s = strings.Replace(s, ">"+escapedGreaterThan, "><off><"+arrowColor+">"+escapedGreaterThan+"<off><"+fieldColor+">", -1)
	return s
}
