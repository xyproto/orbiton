package main

import (
	"strings"

	"github.com/xyproto/mode"
)

// cLikeness returns a score (0.0-1.0) indicating how C-like a programming language mode is.
// 0.0 means "not at all C-like", 1.0 means "it is C".
func cLikeness(m mode.Mode) float64 {
	switch m {
	case mode.C:
		return 1.0 // Pure C
	case mode.Cpp:
		return 0.9 // C++ is very close to C
	case mode.Arduino:
		return 0.9 // Arduino uses C/C++ syntax
	case mode.ObjC:
		return 0.8 // Objective-C extends C
	case mode.Hare:
		return 0.7 // Hare is C-inspired with similar syntax
	case mode.Zig:
		return 0.7 // Zig is C-influenced with similar function syntax
	case mode.D:
		return 0.6 // D is C-family with similar syntax
	case mode.Shader:
		return 0.6 // Shader languages (GLSL/HLSL) are C-like
	case mode.Dart:
		return 0.5 // Dart has C-like syntax but different semantics
	case mode.JavaScript:
		return 0.5 // JavaScript has C-like syntax
	case mode.TypeScript:
		return 0.5 // TypeScript extends JavaScript with C-like features
	case mode.Jakt:
		return 0.5 // Jakt has C-influenced syntax
	case mode.Scala:
		return 0.4 // Scala can have C-like syntax but is more functional
	default:
		return 0.0 // Non-C-like languages
	}
}

// cLooksLikeFunctionDef checks if a line looks like a C/C++ function definition
// This includes enhanced detection for incomplete function definitions split across lines
func (e *Editor) cLooksLikeFunctionDef(line string) bool {
	trimmedLine := strings.TrimSpace(line)

	// Filter out comments
	if strings.HasPrefix(trimmedLine, "/*") || strings.HasPrefix(trimmedLine, "//") || strings.HasPrefix(trimmedLine, "*") {
		return false
	}

	if strings.HasSuffix(trimmedLine, "()") {
		return true
	}
	if !strings.Contains(trimmedLine, "(") { // must contain at least one "("
		return false
	}

	// Check for complete function definitions
	if strings.HasSuffix(trimmedLine, "{") || strings.HasSuffix(trimmedLine, ")") || strings.HasSuffix(trimmedLine, "};") || strings.HasSuffix(trimmedLine, "} ;") {
		if strings.Contains(trimmedLine, ";") && !(strings.HasSuffix(trimmedLine, "};") || strings.HasSuffix(trimmedLine, "} ;")) {
			return false
		}
		for _, x := range cTypes {
			if strings.HasPrefix(trimmedLine, x) {
				return true // it looks-ish like a function definition
			}
		}
	} else if cLikeness(e.mode) >= 0.8 {
		// Handle incomplete function definitions (split across lines) - only for core C/C++ modes
		// Look for patterns like "static int functionName(" or "void functionName(const"
		if strings.Contains(trimmedLine, ";") { // declarations with semicolons are not definitions
			return false
		}
		// Check if this looks like an assignment (= before the main function call or lambda)
		if strings.Contains(trimmedLine, "=") {
			// Find the last = (in case of operators like ==, !=, etc.)
			equalPos := strings.LastIndex(trimmedLine, "=")
			// Check if there's a ( or [ after the = (function call or lambda)
			afterEqual := trimmedLine[equalPos+1:]
			if strings.Contains(afterEqual, "(") || strings.Contains(afterEqual, "[") {
				return false // this is an assignment, not a function definition
			}
		}

		// Check if line starts with C/C++ type keywords or modifiers
		for _, x := range cTypes {
			if strings.HasPrefix(trimmedLine, x) {
				return true // looks like start of a function definition
			}
		}

		// Check for common C/C++ function modifiers/specifiers
		modifiers := []string{"constexpr ", "explicit ", "extern ", "inline ", "noexcept ", "override ", "static ", "virtual "}
		for _, modifier := range modifiers {
			if strings.HasPrefix(trimmedLine, modifier) {
				// Check if there's a type after the modifier
				remaining := strings.TrimPrefix(trimmedLine, modifier)
				for _, x := range cTypes {
					if strings.HasPrefix(remaining, x) {
						return true
					}
				}
			}
		}
	}

	// Detection for functions with custom return types (namespaced types, templates, etc.)
	// Matches lines at column 0 ending with function signature indicators
	if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
		if strings.HasSuffix(trimmedLine, "(") || strings.HasSuffix(trimmedLine, ",") || strings.HasSuffix(trimmedLine, "{") {
			// Confirm function-like characteristics: parentheses or scope resolution operator
			if strings.Contains(trimmedLine, "(") || strings.Contains(trimmedLine, "::") {
				parts := strings.Fields(trimmedLine)
				if len(parts) >= 2 {
					// Exclude common non-function declarations
					firstWord := strings.ToLower(parts[0])
					if firstWord != "typedef" && firstWord != "struct" && firstWord != "class" &&
						firstWord != "enum" && firstWord != "union" && firstWord != "#define" &&
						!strings.HasPrefix(firstWord, "#") {
						return true
					}
				}
				// C++ member functions and scoped functions
				if strings.Contains(trimmedLine, "::") && strings.Contains(trimmedLine, "(") {
					return true
				}
			}
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

// cExtractFunctionName extracts function name from a C/C++ function definition line
func (e *Editor) cExtractFunctionName(line string) string {
	trimmedLine := strings.TrimSpace(line)
	s := strings.TrimSpace(strings.TrimSuffix(trimmedLine, "{"))

	words := strings.Split(s, " ")
	for i, word := range words {
		if strings.Contains(word, "(") {
			// Found the word with opening parenthesis
			fields := strings.SplitN(word, "(", 2)
			functionName := strings.TrimSpace(fields[0])
			// Remove any pointer/reference symbols for C/C++
			functionName = strings.TrimPrefix(functionName, "*")
			functionName = strings.TrimPrefix(functionName, "&")
			if functionName != "" {
				return functionName
			}
		} else if i > 0 && i == len(words)-1 {
			// Enhanced incomplete function detection - only for C/C++
			// Last word without parenthesis - might be incomplete function def
			// Check if previous words contain types/modifiers
			hasTypeOrModifier := false
			for j := 0; j < i; j++ {
				prevWord := words[j]
				for _, cType := range cTypes {
					if strings.HasPrefix(prevWord, strings.TrimSpace(cType)) {
						hasTypeOrModifier = true
						break
					}
				}
				if hasTypeOrModifier {
					break
				}
			}
			if hasTypeOrModifier {
				functionName := strings.TrimSpace(word)
				functionName = strings.TrimPrefix(functionName, "*")
				functionName = strings.TrimPrefix(functionName, "&")
				if functionName != "" {
					return functionName
				}
			}
		}
	}

	return ""
}
