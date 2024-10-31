package main

import (
	"strings"

	"github.com/xyproto/mode"
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

// LooksLikeFunctionDef tries to decide if the given line looks like a function definition or not
func (e *Editor) LooksLikeFunctionDef(line, funcPrefix string) bool {
	trimmedLine := strings.TrimSpace(line)
	if funcPrefix != "" && strings.HasPrefix(trimmedLine, funcPrefix) {
		return true
	}
	switch e.mode {
	// Very unscientific and approximate function definition detection for C and C++
	// TODO: Write a C parser and a C++ parser...
	case mode.C, mode.Cpp:
		if strings.HasSuffix(trimmedLine, "()") {
			return true
		}
		if !strings.Contains(trimmedLine, "(") { // must contain at least one "("
			return false
		}
		if !strings.HasSuffix(trimmedLine, "{") && !strings.HasSuffix(trimmedLine, ")") && !strings.HasSuffix(trimmedLine, "};") && !strings.HasSuffix(trimmedLine, "} ;") { // the line should end with either "{" or ")" or "};" or "} ;"
			return false
		}
		if strings.Contains(trimmedLine, ";") && !(strings.HasSuffix(trimmedLine, "};") || strings.HasSuffix(trimmedLine, "} ;")) {
			return false
		}
		xs := []string{"bool", "char", "const", "constexpr", "double", "float", "int", "int16_t", "int32_t", "int64_t", "int8_t", "long", "short", "signed", "size_t", "static", "uint", "uint16_t", "uint32_t", "uint64_t", "uint8_t", "unsigned", "void", "volatile "}
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
		fallthrough
	default:
		if strings.Contains(trimmedLine, "(") {
			fields := strings.SplitN(trimmedLine, "(", 2)
			if strings.Contains(fields[0], "=") {
				return false
			}
			if !strings.Contains(fields[0], " ") && strings.HasSuffix(trimmedLine, ") {") { // shell functions without a func prefix
				return true
			}
		}
		if strings.Index(trimmedLine, "=") < strings.Index(trimmedLine, "(") { // equal sign before the first (
			return false
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") { // lines that are not indented are more likely to be function definitions
			if strings.Contains(line, "(") && strings.Contains(line, ")") { // looking more and more like a function definition
				return true
			}
		}
	}
	return false
}

// FunctionName tries to extract the function name given a line with what looks like a function definition.
func (e *Editor) FunctionName(line string) string {
	var s string
	funcPrefix := e.FuncPrefix()
	if e.LooksLikeFunctionDef(line, funcPrefix) {
		trimmedLine := strings.TrimSpace(line)
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
	withoutFuncPrefix := strings.TrimSpace(strings.TrimPrefix(s, funcPrefix))
	if strings.Contains(withoutFuncPrefix, "(") {
		fields := strings.SplitN(withoutFuncPrefix, "(", 2)
		withoutFuncPrefix = strings.TrimSpace(fields[0])
	}
	if strings.Contains(withoutFuncPrefix, ".") {
		return ""
	}
	return withoutFuncPrefix
}

// FindCurrentFunctionName searches upwards until it finds a function definition.
// It returns either the found function name or an empty string.
// But! If the current line has no indentation AND is blank or closing (like "}"),
// then an empty string is returned.
func (e *Editor) FindCurrentFunctionName() string {
	startLineIndex := e.LineIndex()
	startLine := e.Line(startLineIndex)
	if !strings.HasPrefix(startLine, " ") && !strings.HasPrefix(startLine, "\t") { // no indentation on this line
		trimmedLine := strings.TrimSpace(startLine)
		if trimmedLine == "" { // and this line is empty
			lineAbove := e.LineAbove()
			if !strings.HasPrefix(lineAbove, " ") && !strings.HasPrefix(lineAbove, "\t") { // no indentation on the line above as well
				return "" // most likely an empty line between functions, so there is no function name to return here
			}
		}
		if strings.HasPrefix(trimmedLine, e.SingleLineCommentMarker()) || strings.HasPrefix(trimmedLine, "/*") || strings.HasSuffix(trimmedLine, "*/") || strings.HasPrefix(trimmedLine, "*") {
			return "" // probably on a comment before a function
		}
	}
	for i := startLineIndex; i >= 0; i-- {
		line := e.Line(i)
		if functionName := e.FunctionName(line); functionName != "" {
			// Found the current function name
			return functionName
		}
		if i < startLineIndex && (!strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t")) { // not indented, and not a function definition
			if trimmedLine := strings.TrimSpace(line); trimmedLine == "}" || trimmedLine == "end" {
				return ""
			}
		}
	}
	return ""
}

// WriteCurrentFunctionName writes (but does not redraw) the current function name we are within (if any),
// in the top right corner of the canvas.
func (e *Editor) WriteCurrentFunctionName(c *Canvas) {
	if !e.ProgrammingLanguage() {
		return
	}
	s := e.FindCurrentFunctionName()
	var (
		canvasWidth      = c.Width()
		x           uint = (canvasWidth - uint(len(s))) - 2 // 2 is the right side padding
		y           uint
	)
	c.Write(x, y, e.Foreground, e.Background, s)
}
