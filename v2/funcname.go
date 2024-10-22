package main

import (
	"fmt"
	"strings"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

// FuncPrefix tries to return the function keyword for the current editor mode, if possible.
// This is not an exhaustive list. It can be used in connection with jumping to definitions.
// If no function prefix is found for this editor mode, an empty string is returned.
// The returned string may be prefixed or suffixed with a blank, on purpose.
func (e *Editor) FuncPrefix() string {
	switch e.mode {
	case mode.Clojure:
		return "defn "
	case mode.Crystal, mode.Nim, mode.Mojo, mode.Python, mode.Scala:
		return "def "
	case mode.GDScript, mode.Go:
		return "func "
	case mode.Kotlin:
		return "fun "
	case mode.Jakt, mode.JavaScript, mode.Koka, mode.Lua, mode.Shell, mode.TypeScript:
		return "function "
	case mode.Terra:
		return "terra "
	case mode.Odin:
		return "proc() "
	case mode.Hare, mode.Rust, mode.V, mode.Zig:
		return "fn "
	case mode.Erlang:
		// This is not "the definition of a function" in Erlang, but should work for many cases
		return " ->"
	case mode.Prolog:
		// This is not "the definition of a function" in Prolog, but should work for many cases
		return " :-"
	}
	return ""
}

func (e *Editor) LooksLikeFunctionDef(trimmedLine string) bool {
	if strings.HasPrefix(trimmedLine, e.FuncPrefix()) {
		return true
	}
	switch e.mode {
	// Very unscientific and approximate function definition detection for C and C++
	// TODO: Write a C parser and a C++ parser...
	case mode.C, mode.Cpp:
		if !(strings.HasSuffix(trimmedLine, "{") || !strings.HasSuffix(trimmedLine, ")")) { // the line should end with either "{" or ")"
			return false
		}
		if strings.Contains(trimmedLine, ";") && !(strings.HasSuffix(trimmedLine, "};") || strings.HasSuffix(trimmedLine, "} ;")) {
			return false
		}
		xs := []string{"void", "int", "unsigned", "static", "const", "signed", "volatile ", "char", "short", "long", "float", "double", "uint"}
		for _, x := range xs {
			if strings.HasPrefix(trimmedLine, x) {
				return true // it looks-ish like a function definition
			}
		}
		if strings.Contains(trimmedLine, " ") {
			fields := strings.SplitN(trimmedLine, " ", 2)
			if strings.Contains(fields[0], "*") {
				return true // it looks like it could return a pointer to a struct
			}
		}
		return false
	}
	return false
}

// WriteCurrentFunctionName writes (but does not redraw) the current function name we are within (if any) in the top right corner
func (e *Editor) WriteCurrentFuctionName(c *vt100.Canvas) {
	if !e.ProgrammingLanguage() {
		return
	}
	var s string
	trimmedLine := e.TrimmedLine()
	if e.LooksLikeFunctionDef(trimmedLine) {
		s = strings.TrimSpace(strings.TrimSuffix(trimmedLine, "{"))
		words := strings.Split(s, " ")
		for _, word := range words {
			if strings.HasPrefix(word, "(") {
				continue
			}
			if strings.Contains(word, "(") {
				fields := strings.SplitN(word, "(", 2)
				s = fields[0]
				break
			}
		}
	}
	if s == "" {
		s = fmt.Sprintf("%d%%", e.Percentage())
	}
	var (
		canvasWidth      = c.Width()
		x           uint = canvasWidth - uint(len(s))
		y           uint
	)
	c.Write(x, y, e.Foreground, e.Background, s)
}
