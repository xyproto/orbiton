package main

import (
	"strings"

	"github.com/xyproto/javasig"
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
	case mode.Crystal, mode.Nim, mode.Mojo, mode.Python, mode.Scala, mode.Starlark:
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
	case mode.Java:
		return "void" // TODO
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
	case mode.Java:
		return javasig.Is(trimmedLine)
	// Very unscientific and approximate function definition detection for C and C++
	// TODO: Write a C parser and a C++ parser...
	case mode.Arduino, mode.C, mode.Cpp, mode.D, mode.Dart, mode.Hare, mode.Jakt, mode.JavaScript, mode.Kotlin, mode.ObjC, mode.Scala, mode.Shader, mode.TypeScript, mode.Zig:
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
		for _, x := range cTypes {
			if strings.HasPrefix(trimmedLine, x) {
				return true // it looks-ish like a function definition
			}
		}
		if e.mode == mode.Kotlin && strings.HasPrefix(trimmedLine, "suspend "+funcPrefix) {
			return true
		}
		if strings.Contains(trimmedLine, " ") {
			fields := strings.SplitN(trimmedLine, " ", 2)
			if strings.Contains(fields[0], "*") {
				return true // it looks like it could return a pointer to a struct
			}
		}
		fallthrough
	case mode.Odin:
		if strings.Contains(trimmedLine, " :: proc(") || strings.Contains(trimmedLine, " :: proc \"") {
			return true
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
	if e.mode == mode.Odin {
		if strings.Contains(line, " :: proc(") || strings.Contains(line, " :: proc \"") {
			fields := strings.SplitN(line, " :: proc", 2)
			name := strings.TrimSpace(fields[0])
			if name != "" {
				return name
			}
		}
	}
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

// isBraceBasedLanguage returns true if the language uses braces to delimit function bodies
func (e *Editor) isBraceBasedLanguage() bool {
	switch e.mode {
	case mode.Arduino, mode.C, mode.C3, mode.Cpp, mode.CS, mode.CSS, mode.D, mode.Dart,
		mode.Go, mode.Hare, mode.Haxe, mode.Jakt, mode.Java, mode.JavaScript, mode.JSON,
		mode.Kotlin, mode.Mojo, mode.ObjC, mode.Rust, mode.Scala, mode.Shader,
		mode.Swift, mode.TypeScript, mode.V, mode.Zig:
		return true
	}
	return false
}

// findMatchingCloseBrace finds the closing brace that matches an opening brace at the given line
// Returns the line index of the matching closing brace, or -1 if not found
func (e *Editor) findMatchingCloseBrace(openBraceLineIndex LineIndex) LineIndex {
	braceCount := 0
	totalLines := LineIndex(e.Len())

	for i := openBraceLineIndex; i < totalLines; i++ {
		line := e.Line(i)
		for _, char := range line {
			switch char {
			case '{':
				braceCount++
			case '}':
				braceCount--
				if braceCount == 0 {
					return i
				}
			}
		}
	}
	return -1 // No matching closing brace found
}

// isWithinFunctionBody checks if the current cursor position is within the body of the function
// that starts at the given function definition line
func (e *Editor) isWithinFunctionBody(funcDefLineIndex LineIndex) bool {
	if !e.isBraceBasedLanguage() {
		// For non-brace languages, use the original simpler logic
		return true
	}

	currentLineIndex := e.LineIndex()

	// Find the line with the opening brace for this function
	openBraceLineIndex := LineIndex(-1)
	totalLines := LineIndex(e.Len())

	// Look for opening brace starting from the function definition line
	// Search more lines to handle multi-line method signatures, annotations, etc.
	searchLimit := funcDefLineIndex + 20
	if searchLimit > totalLines {
		searchLimit = totalLines
	}

	for i := funcDefLineIndex; i < searchLimit; i++ {
		line := e.Line(i)
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines and comments while looking for the opening brace
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "//") ||
			strings.HasPrefix(trimmedLine, "/*") || strings.HasPrefix(trimmedLine, "*") {
			continue
		}

		if strings.Contains(line, "{") {
			openBraceLineIndex = i
			break
		}

		// If we hit another method signature or class/interface declaration, stop searching
		if i > funcDefLineIndex && (e.LooksLikeFunctionDef(line, e.FuncPrefix()) ||
			strings.Contains(trimmedLine, "class ") || strings.Contains(trimmedLine, "interface ") ||
			strings.Contains(trimmedLine, "enum ")) {
			break
		}
	}

	if openBraceLineIndex == -1 {
		// No opening brace found, might be an abstract method or interface method
		return false
	}

	// If cursor is before the opening brace, it's not within the function body
	if currentLineIndex < openBraceLineIndex {
		return false
	}

	// Find the matching closing brace
	closeBraceLineIndex := e.findMatchingCloseBrace(openBraceLineIndex)

	if closeBraceLineIndex == -1 {
		// No matching closing brace found, assume we're within the function
		return true
	}

	// Check if cursor is between opening and closing braces (inclusive of closing brace)
	return currentLineIndex <= closeBraceLineIndex
}

// FindCurrentFunctionName searches upwards until it finds a function definition.
// It returns either the found function name or an empty string.
// But! If the current line has no indentation AND is blank or closing (like "}"),
// then an empty string is returned.
// For brace-based languages like Java, it also verifies that the cursor is actually
// within the function body, not just above the function declaration.
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
			// Found a function definition, but verify we're actually within its body
			if e.isWithinFunctionBody(i) {
				return functionName
			}
			// We found a function but we're not within its body (e.g., we're above it)
			return ""
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
func (e *Editor) WriteCurrentFunctionName(c *vt100.Canvas) {
	if !ProgrammingLanguage(e.mode) {
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
