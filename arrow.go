package main

import (
	"strings"

	"github.com/xyproto/syntax"
)

// Syntax highlight pointer arrows in C and C++
func arrowReplace(s string) string {
	arrowColor := syntax.DefaultTextConfig.Class
	fieldColor := syntax.DefaultTextConfig.Protected
	s = strings.Replace(s, ">-<", "><off><"+arrowColor+">-<", -1)
	s = strings.Replace(s, ">"+Escape(">"), "><off><"+arrowColor+">"+Escape(">")+"<off><"+fieldColor+">", -1)
	return s
}
