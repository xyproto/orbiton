package main

import "strings"

// pascalRoutineKeywords are the keywords that can introduce a routine in Object Pascal / Lazarus.
// Matching is case-insensitive.
var pascalRoutineKeywords = []string{"function", "procedure", "constructor", "destructor", "operator"}

// trimPascalRoutineKeyword strips an optional "class " or "generic " qualifier and a routine keyword
// (followed by whitespace) from the start of a trimmed Object Pascal line. It returns the remainder
// and the matched keyword, or ("", "") if no routine keyword was found.
func trimPascalRoutineKeyword(trimmedLine string) (string, string) {
	for _, qualifier := range []string{"class ", "generic "} {
		if len(trimmedLine) > len(qualifier) && strings.EqualFold(trimmedLine[:len(qualifier)], qualifier) {
			trimmedLine = strings.TrimSpace(trimmedLine[len(qualifier):])
		}
	}
	for _, kw := range pascalRoutineKeywords {
		if len(trimmedLine) > len(kw) && strings.EqualFold(trimmedLine[:len(kw)], kw) {
			next := trimmedLine[len(kw)]
			if next == ' ' || next == '\t' {
				return strings.TrimSpace(trimmedLine[len(kw):]), kw
			}
		}
	}
	return "", ""
}

// isPascalIdentRune reports whether r is valid inside an Object Pascal identifier (including the '.'
// used to qualify a method with its class name).
func isPascalIdentRune(r rune) bool {
	return r == '_' || r == '.' || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// pascalExtractFunctionName extracts the routine name from a trimmed Object Pascal line that looks
// like a routine definition or declaration. Qualified names like "TFoo.Bar" are kept as-is.
// Returns an empty string if no name could be extracted.
func pascalExtractFunctionName(trimmedLine string) string {
	rest, keyword := trimPascalRoutineKeyword(trimmedLine)
	if rest == "" {
		return ""
	}
	end := 0
	for end < len(rest) && isPascalIdentRune(rune(rest[end])) {
		end++
	}
	if end == 0 && keyword == "operator" {
		// "operator + (a, b: TFoo) c: TFoo;", grab the symbol up to the next whitespace or '('
		for end < len(rest) {
			r := rune(rest[end])
			if r == ' ' || r == '\t' || r == '(' {
				break
			}
			end++
		}
	}
	return rest[:end]
}

// pascalLooksLikeFunctionDef reports whether a trimmed Object Pascal line looks like a routine
// definition or declaration.
func pascalLooksLikeFunctionDef(trimmedLine string) bool {
	return pascalExtractFunctionName(trimmedLine) != ""
}
