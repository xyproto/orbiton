package kotlinsig

import (
	"strings"
	"unicode"
)

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	if firstChar := rune(s[0]); !unicode.IsLetter(firstChar) && firstChar != '_' {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

// Is tries to determine if the given string is most likely a Kotlin function signature or not
func Is(line string) bool {
	line = strings.TrimSpace(line)
	lower := strings.ToLower(line)

	// Skip lines that are clearly not function declarations
	if strings.Contains(line, "=") && !strings.Contains(line, "fun ") {
		return false
	}

	// Kotlin functions must contain "fun " keyword, "init", or "constructor"
	hasFun := strings.Contains(line, "fun ")
	hasInit := strings.Contains(lower, "init")
	hasConstructor := strings.Contains(lower, "constructor")

	if !hasFun && !hasInit && !hasConstructor {
		return false
	}

	// Handle init blocks first (they don't need parentheses)
	if hasInit {
		// init blocks just need to have "init" and typically braces
		if strings.Contains(line, "init") {
			return true
		}
	}

	// Check for parentheses (function parameters) - required for fun and constructor
	open := strings.Index(line, "(")
	close := strings.LastIndex(line, ")")
	if open == -1 || close == -1 || open > close {
		return false
	}

	// Skip common non-function constructs
	if strings.HasPrefix(lower, "return ") ||
		strings.HasPrefix(lower, "throw ") ||
		strings.HasPrefix(lower, "} catch ") ||
		strings.HasPrefix(lower, "super(") ||
		strings.HasPrefix(lower, "class ") ||
		strings.HasPrefix(lower, "interface ") ||
		strings.HasPrefix(lower, "object ") ||
		strings.HasPrefix(lower, "enum ") ||
		strings.HasPrefix(lower, "data class ") ||
		strings.HasPrefix(lower, "sealed class ") {
		return false
	}

	// Handle constructors
	if hasConstructor {
		// Primary or secondary constructors
		constructorIndex := strings.Index(lower, "constructor")
		if constructorIndex != -1 {
			// Constructors should have parentheses (already checked above)
			return true
		}
	}

	// If it's not a regular fun, init, or constructor, return false
	if !hasFun {
		return false
	}

	// Find the "fun" keyword and extract function name
	funIndex := strings.Index(lower, "fun ")
	if funIndex == -1 {
		return false
	}

	// Extract the part between "fun " and the opening parenthesis
	funStart := funIndex + 4 // length of "fun "
	funPart := strings.TrimSpace(line[funStart:open])

	// Handle generic functions (e.g., "fun <T> myFunction")
	if strings.HasPrefix(funPart, "<") {
		closeGeneric := strings.Index(funPart, ">")
		if closeGeneric != -1 && closeGeneric < len(funPart)-1 {
			funPart = strings.TrimSpace(funPart[closeGeneric+1:])
		}
	}

	// The function name should be a valid identifier
	tokens := strings.Fields(funPart)
	if len(tokens) == 0 {
		return false
	}

	functionName := tokens[len(tokens)-1]

	// Handle receiver functions (e.g., "String.myExtension")
	if strings.Contains(functionName, ".") {
		parts := strings.Split(functionName, ".")
		if len(parts) >= 2 {
			functionName = parts[len(parts)-1]
		}
	}

	return isIdentifier(functionName)
}
