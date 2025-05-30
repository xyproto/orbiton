package javasig

import (
	"strings"
	"unicode"
)

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	if firstChar := rune(s[0]); !unicode.IsLetter(firstChar) && firstChar != '_' && firstChar != '$' {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' {
			return false
		}
	}
	return true
}

// Is tries to determine if the given string is most likely a Java method/function signature or not
func Is(line string) bool {
	line = strings.TrimSpace(line)
	if strings.Contains(line, "=") {
		return false
	}
	if !strings.HasSuffix(line, "{") && !strings.HasSuffix(line, ";") {
		// some function signatures starts with ie. "protected " and has "throws" on the next line, before "{"
		if !strings.HasPrefix(line, "private ") && !strings.HasPrefix(line, "public ") && !strings.HasPrefix(line, "protected ") {
			return false
		}
	}
	open := strings.Index(line, "(")
	close := strings.Index(line, ")")
	if open == -1 || close == -1 || open > close {
		return false
	}
	lower := strings.ToLower(line)
	if strings.HasPrefix(lower, "new ") ||
		strings.HasPrefix(lower, "return ") ||
		strings.HasPrefix(lower, "throw ") ||
		strings.HasPrefix(lower, "} catch ") ||
		strings.HasPrefix(lower, "super(") ||
		strings.HasPrefix(lower, "class ") ||
		strings.HasPrefix(lower, "@interface ") {
		return false
	}
	before := strings.TrimSpace(line[:open])
	if strings.HasPrefix(before, "@") {
		parts := strings.Fields(before)
		if len(parts) > 1 {
			before = strings.Join(parts[1:], " ")
		}
	}
	tokens := strings.Fields(before)
	if len(tokens) < 2 {
		return false
	}
	methodName := tokens[len(tokens)-1]
	return isIdentifier(methodName)
}
