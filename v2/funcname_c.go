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

// startsWithCType checks if the line starts with a C type, composite type, or modifier+type
func startsWithCType(line string) bool {
	return hasPrefixWithSpace(line, cTypes) || hasPrefixWithSpace(line, cCompositeTypes)
}

// startsWithCModifierAndType checks if line starts with a modifier followed by a type
func startsWithCModifierAndType(line string) bool {
	for _, modifier := range cModifiers {
		// Check for modifier followed by space to avoid false matches
		modifierWithSpace := modifier + " "
		if strings.HasPrefix(line, modifierWithSpace) {
			remaining := strings.TrimPrefix(line, modifierWithSpace)
			if startsWithCType(remaining) {
				return true
			}
		}
	}
	return false
}

// cLooksLikeFunctionDef checks if a line looks like a C/C++ function definition
// This includes enhanced detection for incomplete function definitions split across lines
func (e *Editor) cLooksLikeFunctionDef(line string) bool {
	trimmedLine := e.StripSingleLineComment(strings.TrimSpace(line))

	// Filter out comments
	if strings.HasPrefix(trimmedLine, "/*") || strings.HasPrefix(trimmedLine, "//") || strings.HasPrefix(trimmedLine, "*") {
		return false
	}

	if strings.HasSuffix(trimmedLine, "()") {
		return true
	}
	if !strings.Contains(trimmedLine, "(") {
		return false
	}

	// Check for complete function definitions (lines ending with {, ), }, };, or } ;)
	endsLikeFuncDef := strings.HasSuffix(trimmedLine, "{") ||
		strings.HasSuffix(trimmedLine, ")") ||
		strings.HasSuffix(trimmedLine, "}") ||
		strings.HasSuffix(trimmedLine, "};") ||
		strings.HasSuffix(trimmedLine, "} ;")

	if endsLikeFuncDef {
		// Reject forward declarations (have ; but don't end with }; or } ; and aren't one-liners)
		hasSemicolon := strings.Contains(trimmedLine, ";")
		endsWithBraceSemi := strings.HasSuffix(trimmedLine, "};") || strings.HasSuffix(trimmedLine, "} ;")
		isOneLiner := strings.Contains(trimmedLine, "{") && strings.Contains(trimmedLine, "}")
		if hasSemicolon && !endsWithBraceSemi && !isOneLiner {
			return false
		}
		if startsWithCType(trimmedLine) {
			return true
		}
	} else if cLikeness(e.mode) >= 0.8 {
		// Handle incomplete function definitions (split across lines) - only for core C/C++ modes
		if strings.Contains(trimmedLine, ";") {
			return false
		}
		// Reject assignments (= before function call or lambda)
		if strings.Contains(trimmedLine, "=") {
			equalPos := strings.LastIndex(trimmedLine, "=")
			afterEqual := trimmedLine[equalPos+1:]
			if strings.Contains(afterEqual, "(") || strings.Contains(afterEqual, "[") {
				return false
			}
		}
		if startsWithCType(trimmedLine) || startsWithCModifierAndType(trimmedLine) {
			return true
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
					if firstWord != "typedef" && firstWord != "#define" &&
						!hasS(cControlFlow, firstWord) &&
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

// stripPointerRef removes leading * and & from a function name
func stripPointerRef(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "*")
	name = strings.TrimPrefix(name, "&")
	return strings.TrimSpace(name)
}

// wordLooksLikeCType checks if a word looks like a C type or composite type keyword
func wordLooksLikeCType(word string) bool {
	return hasPrefix(word, cTypes) || hasPrefix(word, cCompositeTypes)
}

// cExtractFunctionName extracts function name from a C/C++ function definition line
func (e *Editor) cExtractFunctionName(line string) string {
	s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), "{"))
	words := strings.Split(s, " ")

	for i, word := range words {
		if strings.Contains(word, "(") {
			// Found the word with opening parenthesis
			fields := strings.SplitN(word, "(", 2)
			if name := stripPointerRef(fields[0]); name != "" {
				return name
			}
		} else if i > 0 && i == len(words)-1 {
			// Last word without parenthesis - might be incomplete function def
			// Check if any previous word is a type/modifier
			for j := 0; j < i; j++ {
				if wordLooksLikeCType(words[j]) {
					if name := stripPointerRef(word); name != "" {
						return name
					}
					break
				}
			}
		}
	}
	return ""
}
