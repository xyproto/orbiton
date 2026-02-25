package main

import (
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/xyproto/vt"
)

// Patterns for stripping ANSI color codes and OSC sequences (like hyperlinks),
// compiled lazily on first use
var (
	shellColorCodePattern *regexp.Regexp
	oscPattern            *regexp.Regexp
	escapePatternOnce     sync.Once
)

// handleManPageEscape strips nroff overstrike codes and ANSI/OSC escape sequences
func handleManPageEscape(input string) string {
	var (
		prevRune, currRune, nextRune rune
		cleanedRunes                 []rune
		inputRunes                   = []rune(input)
		lenInputRunes                = len(inputRunes)
		lastIndex                    = lenInputRunes - 1
	)
	for i := 0; i < lenInputRunes; { // NOTE: no i++
		prevRune, currRune = currRune, inputRunes[i]
		if i < lastIndex {
			nextRune = inputRunes[i+1]
		} else {
			nextRune = rune(0)
		}
		switch {
		case currRune == '_' && nextRune == 0x08:
			i++ // skip _ before backspace (nroff underline)
		case prevRune == '_' && currRune == 0x08:
			i++ // skip backspace after _ (nroff underline)
		case currRune == 0x08:
			i += 2 // skip backspace and the following rune (nroff bold)
		default:
			cleanedRunes = append(cleanedRunes, currRune)
			i++
		}
	}
	// Remove ANSI and OSC sequences, compiling the regexes once lazily
	cleanedString := string(cleanedRunes)
	escapePatternOnce.Do(func() {
		shellColorCodePattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
		oscPattern = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)
	})
	cleanedString = shellColorCodePattern.ReplaceAllString(cleanedString, "")
	return oscPattern.ReplaceAllString(cleanedString, "")
}

// findFlagTokenEnd returns the byte offset in s where the leading flag token(s)
// end, treating flags separated by ", " as one group.
// For "--zero end each..." it returns 6, for "-a, --all" it returns 9.
func findFlagTokenEnd(s string) int {
	i := 0
	n := len(s)
	for i < n {
		if s[i] != '-' {
			return i
		}
		for i < n && s[i] != ' ' && s[i] != ',' {
			i++
		}
		if i >= n {
			return n
		}
		// ", -" means another flag follows
		if s[i] == ',' && i+2 < n && s[i+1] == ' ' && s[i+2] == '-' {
			i += 2
			continue
		}
		return i
	}
	return i
}

// looksLikeFlags checks if s looks like a man page flag specification,
// such as "-a", "--all", "-f, --classify[=WHEN]" or "-I dir"
func looksLikeFlags(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || !strings.HasPrefix(s, "-") {
		return false
	}
	// Prose punctuation does not belong in a flag portion
	if strings.ContainsAny(s, ".:;()'\"") {
		return false
	}
	// A trailing comma is a dangling sentence fragment
	if strings.HasSuffix(s, ",") {
		return false
	}
	// Each comma-separated piece should start with a dash or bracket
	for i, part := range strings.Split(s, ", ") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		if i > 0 && !strings.HasPrefix(p, "-") && !strings.HasPrefix(p, "[") {
			return false
		}
	}
	return true
}

// manPageHighlight returns the given line with man page syntax highlighting
func (e *Editor) manPageHighlight(line string, firstLine, lastLine bool) string {
	line = handleManPageEscape(line)
	var (
		normal      = e.Foreground
		off         = vt.Stop()
		trimmedLine = strings.TrimSpace(line)
		hasAnyWords = hasWords(trimmedLine)
	)
	if strings.Count(trimmedLine, "  ") > 10 && (firstLine || lastLine) { // first and last line
		return e.CommentColor.Get(line)
	}
	if strings.ToUpper(trimmedLine) == trimmedLine && !strings.HasPrefix(trimmedLine, "-") && hasAnyWords && !strings.HasPrefix(line, " ") { // a sub-section header
		return e.ManSectionColor.Get(line)
	}
	// Detect flag lines: start with a dash followed by a letter or digit,
	// with only flag-like tokens before the description gap (2+ spaces).
	// Lines where a flag appears mid-sentence are treated as regular text.
	isFlagLine := false
	singleSpaceFlagEnd := 0
	if strings.HasPrefix(trimmedLine, "--") || (strings.HasPrefix(trimmedLine, "-") && len(trimmedLine) > 1 && (unicode.IsLetter(rune(trimmedLine[1])) || unicode.IsDigit(rune(trimmedLine[1])))) {
		flagPart := trimmedLine
		if idx := strings.Index(trimmedLine, "  "); idx >= 0 {
			flagPart = trimmedLine[:idx]
		}
		if looksLikeFlags(flagPart) {
			isFlagLine = true
		} else {
			// Extract just the leading flag token(s) and treat the rest
			// as a single-space-separated description
			end := findFlagTokenEnd(trimmedLine)
			if end > 0 && end < len(trimmedLine) {
				singleSpaceFlagEnd = end
			}
		}
	}
	if isFlagLine {
		// Short flag line with at most one space, like "-v" or "-h, --help"
		if strings.Count(trimmedLine, " ") <= 1 {
			return e.MenuArrowColor.Get(line)
		}
		// Flag line with description after a 2+ space gap
		var rs []rune
		inDescription := false
		spaceCount := 0
		pastIndent := false
		for _, r := range line {
			if inDescription {
				rs = append(rs, r)
				continue
			}
			if !pastIndent && r != ' ' {
				pastIndent = true
			}
			if pastIndent && r == ' ' {
				spaceCount++
			} else {
				spaceCount = 0
			}
			if pastIndent && spaceCount >= 2 {
				// Switch to description color
				inDescription = true
				rs = append(rs, []rune(off+normal.String())...)
			}
			rs = append(rs, r)
		}
		if !inDescription {
			// Entire line is flags, no description found
			return e.MenuArrowColor.Get(line)
		}
		result := e.MenuArrowColor.String() + string(rs) + off
		return result
	}
	// Line starts with a flag token followed by a single-space description,
	// like "--zero end each output line with NUL, not newline"
	if singleSpaceFlagEnd > 0 {
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		splitAt := indent + singleSpaceFlagEnd
		return e.MenuArrowColor.String() + line[:splitAt] + off + normal.String() + line[splitAt:] + off
	}
	if allUpper(trimmedLine) {
		return e.ImageColor.Get(line)
	}
	// Regular text: highlight numbers, inline flags, uppercase words and special chars
	var (
		rs           []rune
		prevRune     rune
		inDigits     bool
		inWord       bool
		inAngles     bool
		inInlineFlag bool // inside a --flag in prose text
		inUpperWord  bool // word starting with 2+ uppercase letters
		hasCamelCase bool // word has a lower-to-uppercase transition
		nextRune     rune
	)
	rs = append(rs, []rune(normal.String())...)
	hasAlpha := strings.Contains(trimmedLine, "@")
	lineRunes := []rune(line)
	for i, r := range line {
		if (i + 1) < len(lineRunes) {
			nextRune = lineRunes[i+1]
		} else {
			nextRune = ' '
		}
		// Detect inline long flags like --word in prose text
		if !inInlineFlag && r == '-' && nextRune == '-' && (prevRune == ' ' || prevRune == '\t' || prevRune == 0 || prevRune == '(') {
			inInlineFlag = true
		}
		if inInlineFlag {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '=' || r == '[' || r == ']' {
				rs = append(rs, []rune(off+e.MenuArrowColor.String())...)
				rs = append(rs, r)
				prevRune = r
				continue
			}
			inInlineFlag = false
		}
		inWord = (unicode.IsLetter(r) || r == '_') || (inWord && unicode.IsLetter(r)) || (inWord && hexDigit(r))
		// Track uppercase words so that FORMAT1, GPLv3+ etc get the same color,
		// but not camelCase words like OpenVZ
		if !inWord {
			hasCamelCase = false
		}
		if inWord && unicode.IsUpper(r) && unicode.IsLower(prevRune) {
			hasCamelCase = true
		}
		if inUpperWord {
			if hasCamelCase || !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '+' || r == '_') {
				inUpperWord = false
			}
		} else if inWord && unicode.IsUpper(r) && unicode.IsUpper(prevRune) && !hasCamelCase {
			inUpperWord = true
		}
		inAngles = (!inAngles && r == '<') || (inAngles && r != '>')
		if !inWord && unicode.IsDigit(r) && !inDigits {
			inDigits = true
			rs = append(rs, []rune(off+e.ItalicsColor.String())...)
		} else if inDigits && !inWord && !unicode.IsDigit(r) && !hexDigit(r) {
			inDigits = false
			rs = append(rs, []rune(off+normal.String())...)
		} else if !(inDigits && (unicode.IsDigit(r) || hexDigit(r))) {
			if inUpperWord {
				rs = append(rs, []rune(off+e.ImageColor.String())...)
			} else if !inWord && (r == '*' || r == '$' || r == '%' || r == '!') {
				rs = append(rs, []rune(off+e.MenuArrowColor.String())...)
			} else if r == '@' {
				rs = append(rs, []rune(off+e.CommentColor.String())...)
			} else if hasAlpha && r == '<' {
				rs = append(rs, []rune(off+e.CommentColor.String())...)
			} else if hasAlpha && r == '>' {
				rs = append(rs, []rune(off+e.CommentColor.String())...)
			} else if inAngles || r == '>' {
				rs = append(rs, []rune(off+e.ItalicsColor.String())...)
			} else if inWord && unicode.IsUpper(prevRune) && ((unicode.IsUpper(r) && unicode.IsLetter(nextRune)) || (unicode.IsLower(r) && unicode.IsUpper(prevRune) && !unicode.IsLetter(nextRune))) {
				if unicode.IsUpper(r) {
					// Leading and trailing letter of uppercase words
					rs = append(rs, []rune(off+e.ImageColor.String())...)
				} else {
					rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
				}
			} else if inWord && (unicode.IsUpper(r) || (unicode.IsUpper(prevRune) && unicode.IsLetter(r))) {
				if !unicode.IsLower(r) && (((unicode.IsUpper(nextRune) || nextRune == ' ') && unicode.IsLetter(prevRune)) || unicode.IsUpper(nextRune) || !unicode.IsLetter(nextRune)) {
					// Center letters of uppercase words
					rs = append(rs, []rune(off+e.ImageColor.String())...)
				} else {
					rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
				}
			} else if inWord && unicode.IsUpper(r) {
				rs = append(rs, []rune(off+e.ImageColor.String())...)
			} else if !inWord || !unicode.IsUpper(r) {
				rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
			}
		}
		rs = append(rs, r)
		if r == '@' || (hasAlpha && r == '<') {
			rs = append(rs, []rune(off+e.ItalicsColor.String())...)
		} else if hasAlpha && r == '>' {
			rs = append(rs, []rune(off+normal.String())...)
		}
		prevRune = r
	}
	return string(append(rs, []rune(off)...))
}
