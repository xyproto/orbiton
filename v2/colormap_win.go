//go:build (!darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris) || !cgo
// +build !darwin,!dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris !cgo

package main

import (
	"github.com/mgutz/ansi"
)

type ANSIColor string

var LightColorMap = map[string]ANSIColor{
	"black":        ANSIColor(ansi.ColorCode("black")),
	"Black":        ANSIColor(ansi.ColorCode("black")),
	"red":          ANSIColor(ansi.ColorCode("red+h")),
	"Red":          ANSIColor(ansi.ColorCode("red+h")),
	"green":        ANSIColor(ansi.ColorCode("green+h")),
	"Green":        ANSIColor(ansi.ColorCode("green+h")),
	"yellow":       ANSIColor(ansi.ColorCode("yellow+h")),
	"Yellow":       ANSIColor(ansi.ColorCode("yellow+h")),
	"blue":         ANSIColor(ansi.ColorCode("blue+h")),
	"Blue":         ANSIColor(ansi.ColorCode("blue+h")),
	"magenta":      ANSIColor(ansi.ColorCode("magenta+h")),
	"Magenta":      ANSIColor(ansi.ColorCode("magenta+h")),
	"cyan":         ANSIColor(ansi.ColorCode("cyan+h")),
	"Cyan":         ANSIColor(ansi.ColorCode("cyan+h")),
	"gray":         ANSIColor(ansi.ColorCode("white")),
	"Gray":         ANSIColor(ansi.ColorCode("white")),
	"white":        ANSIColor(ansi.ColorCode("white+h")),
	"White":        ANSIColor(ansi.ColorCode("white+h")),
	"lightwhite":   ANSIColor(ansi.ColorCode("white+h")),
	"LightWhite":   ANSIColor(ansi.ColorCode("white+h")),
	"lightred":     ANSIColor(ansi.ColorCode("red+h")),
	"LightRed":     ANSIColor(ansi.ColorCode("red+h")),
	"lightgreen":   ANSIColor(ansi.ColorCode("green+h")),
	"LightGreen":   ANSIColor(ansi.ColorCode("green+h")),
	"lightyellow":  ANSIColor(ansi.ColorCode("yellow+h")),
	"LightYellow":  ANSIColor(ansi.ColorCode("yellow+h")),
	"lightblue":    ANSIColor(ansi.ColorCode("blue+h")),
	"LightBlue":    ANSIColor(ansi.ColorCode("blue+h")),
	"lightmagenta": ANSIColor(ansi.ColorCode("magenta+h")),
	"LightMagenta": ANSIColor(ansi.ColorCode("magenta+h")),
	"lightcyan":    ANSIColor(ansi.ColorCode("cyan+h")),
	"LightCyan":    ANSIColor(ansi.ColorCode("cyan+h")),
	"lightgray":    ANSIColor(ansi.ColorCode("white")),
	"LightGray":    ANSIColor(ansi.ColorCode("white")),
	"darkred":      ANSIColor(ansi.ColorCode("red")),
	"DarkRed":      ANSIColor(ansi.ColorCode("red")),
	"darkgreen":    ANSIColor(ansi.ColorCode("green")),
	"DarkGreen":    ANSIColor(ansi.ColorCode("green")),
	"darkyellow":   ANSIColor(ansi.ColorCode("yellow")),
	"DarkYellow":   ANSIColor(ansi.ColorCode("yellow")),
	"darkblue":     ANSIColor(ansi.ColorCode("blue")),
	"DarkBlue":     ANSIColor(ansi.ColorCode("blue")),
	"darkmagenta":  ANSIColor(ansi.ColorCode("magenta")),
	"DarkMagenta":  ANSIColor(ansi.ColorCode("magenta")),
	"darkcyan":     ANSIColor(ansi.ColorCode("cyan")),
	"DarkCyan":     ANSIColor(ansi.ColorCode("cyan")),
	"darkgray":     ANSIColor(ansi.ColorCode("default")),
	"DarkGray":     ANSIColor(ansi.ColorCode("default")),
}

var DarkColorMap = map[string]ANSIColor{
	"black":        ANSIColor(ansi.ColorCode("black")),
	"Black":        ANSIColor(ansi.ColorCode("black")),
	"red":          ANSIColor(ansi.ColorCode("red")),
	"Red":          ANSIColor(ansi.ColorCode("red")),
	"green":        ANSIColor(ansi.ColorCode("green")),
	"Green":        ANSIColor(ansi.ColorCode("green")),
	"yellow":       ANSIColor(ansi.ColorCode("yellow")),
	"Yellow":       ANSIColor(ansi.ColorCode("yellow")),
	"blue":         ANSIColor(ansi.ColorCode("blue")),
	"Blue":         ANSIColor(ansi.ColorCode("blue")),
	"magenta":      ANSIColor(ansi.ColorCode("magenta")),
	"Magenta":      ANSIColor(ansi.ColorCode("magenta")),
	"cyan":         ANSIColor(ansi.ColorCode("cyan")),
	"Cyan":         ANSIColor(ansi.ColorCode("cyan")),
	"gray":         ANSIColor(ansi.ColorCode("default")),
	"Gray":         ANSIColor(ansi.ColorCode("default")),
	"white":        ANSIColor(ansi.ColorCode("white+b")),
	"White":        ANSIColor(ansi.ColorCode("white+b")),
	"lightwhite":   ANSIColor(ansi.ColorCode("white+b")),
	"LightWhite":   ANSIColor(ansi.ColorCode("white+b")),
	"lightred":     ANSIColor(ansi.ColorCode("red+h")),
	"LightRed":     ANSIColor(ansi.ColorCode("red+h")),
	"lightgreen":   ANSIColor(ansi.ColorCode("green+h")),
	"LightGreen":   ANSIColor(ansi.ColorCode("green+h")),
	"lightyellow":  ANSIColor(ansi.ColorCode("yellow+h")),
	"LightYellow":  ANSIColor(ansi.ColorCode("yellow+h")),
	"lightblue":    ANSIColor(ansi.ColorCode("blue+h")),
	"LightBlue":    ANSIColor(ansi.ColorCode("blue+h")),
	"lightmagenta": ANSIColor(ansi.ColorCode("magenta+h")),
	"LightMagenta": ANSIColor(ansi.ColorCode("magenta+h")),
	"lightcyan":    ANSIColor(ansi.ColorCode("cyan+h")),
	"LightCyan":    ANSIColor(ansi.ColorCode("cyan+h")),
	"lightgray":    ANSIColor(ansi.ColorCode("default")),
	"LightGray":    ANSIColor(ansi.ColorCode("default")),
	"darkred":      ANSIColor(ansi.ColorCode("red")),
	"DarkRed":      ANSIColor(ansi.ColorCode("red")),
	"darkgreen":    ANSIColor(ansi.ColorCode("green")),
	"DarkGreen":    ANSIColor(ansi.ColorCode("green")),
	"darkyellow":   ANSIColor(ansi.ColorCode("yellow")),
	"DarkYellow":   ANSIColor(ansi.ColorCode("yellow")),
	"darkblue":     ANSIColor(ansi.ColorCode("blue")),
	"DarkBlue":     ANSIColor(ansi.ColorCode("blue")),
	"darkmagenta":  ANSIColor(ansi.ColorCode("magenta")),
	"DarkMagenta":  ANSIColor(ansi.ColorCode("magenta")),
	"darkcyan":     ANSIColor(ansi.ColorCode("cyan")),
	"DarkCyan":     ANSIColor(ansi.ColorCode("cyan")),
	"darkgray":     ANSIColor(ansi.ColorCode("default")),
	"DarkGray":     ANSIColor(ansi.ColorCode("default")),
}
