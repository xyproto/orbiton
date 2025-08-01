//go:build windows
// +build windows

package vt

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mgutz/ansi"
	"github.com/xyproto/env/v2"
)

// CharAttribute is a rune and a color attribute
type CharAttribute struct {
	A ANSIColor
	R rune
}

// TextOutput keeps state about verbosity and if colors are enabled
type TextOutput struct {
	color   bool
	enabled bool
	// Tag replacement structs, for performance
	lightReplacer *strings.Replacer
	darkReplacer  *strings.Replacer
}

// Respect the NO_COLOR environment variable
var EnvNoColor = env.Bool("NO_COLOR")

// New creates a new TextOutput struct, which is
// enabled by default and with colors turned on.
// If the NO_COLOR environment variable is set, colors are disabled.
func New() *TextOutput {
	o := &TextOutput{!EnvNoColor, true, nil, nil}
	o.initializeTagReplacers()
	return o
}

// NewTextOutput can initialize a new TextOutput struct,
// which can have colors turned on or off and where the
// output can be enabled (verbose) or disabled (silent).
// If NO_COLOR is set, colors are disabled, regardless.
func NewTextOutput(color, enabled bool) *TextOutput {
	if EnvNoColor {
		color = false
	}
	o := &TextOutput{color, enabled, nil, nil}
	o.initializeTagReplacers()
	return o
}

// OutputTags will output text that may have tags like "<blue>", "</blue>" or "<off>" for
// enabling or disabling color attributes. Respects the color/enabled settings
// of this TextOutput.
func (o *TextOutput) OutputTags(colors ...string) {
	if o.enabled {
		fmt.Println(o.Tags(colors...))
	}
}

// Write a message to stdout if output is enabled
func (o *TextOutput) Println(msg ...interface{}) {
	if o.enabled {
		fmt.Println(o.InterfaceTags(msg...))
	}
}

// Printf writes a formatted message to stdout if output is enabled
func (o *TextOutput) Printf(format string, args ...interface{}) {
	if o.enabled {
		fmt.Print(o.Tags(fmt.Sprintf(format, args...)))
	}
}

// Write a message to stdout if output is enabled
func (o *TextOutput) Print(msg ...interface{}) {
	if o.enabled {
		fmt.Print(o.InterfaceTags(msg...))
	}
}

// Write an error message in red to stderr if output is enabled
func (o *TextOutput) Err(msg string) {
	if o.enabled {
		if o.color {
			fmt.Fprintln(os.Stderr, ansi.Color(msg, "red"))
		} else {
			fmt.Fprintln(os.Stderr, msg)
		}
	}
}

// ErrExit writes an error message to stderr and quit with exit code 1
func (o *TextOutput) ErrExit(msg string) {
	o.Err(msg)
	os.Exit(1)
}

func (o *TextOutput) LightBlue(s string) string {
	if o.color {
		return ansi.Color(s, "blue+h")
	}
	return s
}

// Replace <blue> with starting a light blue color attribute and <off> with using the default attributes.
// </blue> can also be used for using the default attributes.
func (o *TextOutput) LightTags(colors ...string) string {
	return o.lightReplacer.Replace(strings.Join(colors, ""))
}

// Same as LightTags
func (o *TextOutput) Tags(colors ...string) string {
	return o.LightTags(colors...)
}

// InterfaceTags is the same as LightTags, but with interfaces
func (o *TextOutput) InterfaceTags(colors ...interface{}) string {
	var sb strings.Builder
	for _, color := range colors {
		if colorString, ok := color.(string); ok {
			sb.WriteString(colorString)
		} else {
			sb.WriteString(fmt.Sprintf("%s", color))
		}
	}
	return o.LightTags(sb.String())
}

// Replace <blue> with starting a light blue color attribute and <off> with using the default attributes.
// </blue> can also be used for using the default attributes.
func (o *TextOutput) DarkTags(colors ...string) string {
	return o.darkReplacer.Replace(strings.Join(colors, ""))
}

func (o *TextOutput) Disable() {
	o.enabled = false
}

func (o *TextOutput) Enable() {
	o.enabled = true
}

func (o *TextOutput) initializeTagReplacers() {
	// Initialize tag replacement tables, with as few memory allocations as possible (no append)
	off := ansi.ColorCode("off")
	rs := make([]string, len(LightColorMap)*4+2)
	i := 0
	if o.color {
		for key, value := range LightColorMap {
			rs[i] = "<" + key + ">"
			i++
			rs[i] = string(value)
			i++
			rs[i] = "</" + key + ">"
			i++
			rs[i] = off
			i++
		}
		rs[i] = "<off>"
		i++
		rs[i] = off
	} else {
		for key := range LightColorMap {
			rs[i] = "<" + key + ">"
			i++
			rs[i] = ""
			i++
			rs[i] = "</" + key + ">"
			i++
			rs[i] = ""
			i++
		}
		rs[i] = "<off>"
		i++
		rs[i] = ""
	}
	o.lightReplacer = strings.NewReplacer(rs...)
	// Initialize the replacer for the dark color scheme, while reusing the rs slice
	i = 0
	if o.color {
		for key, value := range DarkColorMap {
			rs[i] = "<" + key + ">"
			i++
			rs[i] = string(value)
			i++
			rs[i] = "</" + key + ">"
			i++
			rs[i] = off
			i++
		}
		rs[i] = "<off>"
		i++
		rs[i] = off
	} else {
		for key := range DarkColorMap {
			rs[i] = "<" + key + ">"
			i++
			rs[i] = ""
			i++
			rs[i] = "</" + key + ">"
			i++
			rs[i] = ""
			i++
		}
		rs[i] = "<off>"
		i++
		rs[i] = ""
	}
	o.darkReplacer = strings.NewReplacer(rs...)
}

// ExtractToSlice iterates over an ANSI encoded string, parsing out color codes and places it in
// a slice of CharAttribute. Each CharAttribute in the slice represents a character in the
// input string and its corresponding color attributes. This function handles escaping sequences
// and converts ANSI color codes to AttributeColor structs.
// The returned uint is the number of stored elements.
func (o *TextOutput) ExtractToSlice(s string, pcc *[]CharAttribute) uint {
	var (
		escaped      bool
		colorcode    strings.Builder
		currentColor ANSIColor
	)
	counter := uint(0)
	for _, r := range s {
		switch {
		case escaped && r == 'm':
			colorAttributes := strings.Split(strings.TrimPrefix(colorcode.String(), "["), ";")
			if len(colorAttributes) != 1 || colorAttributes[0] != "0" {
				var primaryAttr, secondaryAttr ANSIColor
				for i, attribute := range colorAttributes {
					if attributeNumber, err := strconv.Atoi(attribute); err == nil {
						if i == 0 {
							primaryAttr = ANSIColor(ansi.ColorCode(strconv.Itoa(attributeNumber)))
						} else {
							secondaryAttr = ANSIColor(ansi.ColorCode(strconv.Itoa(attributeNumber)))
							break // Only handle two attributes for now
						}
					}
				}
				if secondaryAttr != "" {
					currentColor = primaryAttr + secondaryAttr
				} else {
					currentColor = primaryAttr
				}
			} else {
				currentColor = ANSIColor("")
			}
			colorcode.Reset()
			escaped = false
		case r == '\033':
			escaped = true
		case escaped && r != 'm':
			colorcode.WriteRune(r)
		default:
			if counter >= uint(len(*pcc)) {
				// Extend the slice
				newSlice := make([]CharAttribute, len(*pcc)*2+1)
				copy(newSlice, *pcc)
				*pcc = newSlice
			}
			(*pcc)[counter] = CharAttribute{currentColor, r}
			counter++
		}
	}
	return counter
}

// // Pair takes a string with ANSI codes and returns
// // a slice with two elements.
// func (o *TextOutput) Extract(s string) []CharAttribute {
// 	var (
// 		escaped      bool
// 		colorcode    strings.Builder
// 		word         strings.Builder
// 		cc           = make([]ANSIColor, 0, len(s))
// 		currentColor ANSIColor
// 	)
// 	for _, r := range s {
// 		if r == '\033' {
// 			escaped = true
// 			if len(word.String()) > 0 {
// 				//fmt.Println("cc", cc)
// 				word.Reset()
// 			}
// 			continue
// 		}
// 		if escaped {
// 			if r != 'm' {
// 				colorcode.WriteRune(r)
// 			} else if r == 'm' {
// 				s2 := strings.TrimPrefix(colorcode.String(), "[")
// 				attributeStrings := strings.Split(s2, ";")
// 				if len(attributeStrings) == 1 && attributeStrings[0] == "0" {
// 					currentColor = ""
// 				}
// 				for _, attributeString := range attributeStrings {
// 					attributeNumber, err := strconv.Atoi(attributeString)
// 					if err != nil {
// 						continue
// 					}
// 					currentColor = append(currentColor, byte(attributeNumber))
// 				}
// 				// Strip away leading 0 color attribute, if there are more than 1
// 				if len(currentColor) > 1 && currentColor[0] == 0 {
// 					currentColor = currentColor[1:]
// 				}
// 				// currentColor now contains the last found color attributes,
// 				// but as an AttributeColor.
// 				colorcode.Reset()
// 				escaped = false
// 			}
// 		} else {
// 			cc = append(cc, CharAttribute{r, currentColor})
// 		}
// 	}
// 	// if escaped is true here, there is something wrong
// 	return cc
//}
