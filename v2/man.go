package main

import (
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/xyproto/vt"
)

// shellColorCodePattern matches shell color escape sequences (ANSI codes).
// Compiled lazily on first use to avoid startup cost.
var (
	shellColorCodePattern *regexp.Regexp
	shellColorCodeOnce    sync.Once
)

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
			i++ // Skip the _ character if the next rune is 0x08
		case prevRune == '_' && currRune == 0x08:
			i++ // Skip the 0x08 character if the previous rune was '_'
		case currRune == 0x08: // Encountered ^H
			i += 2 // Skip current 0x08 character and the following rune
		default:
			cleanedRunes = append(cleanedRunes, currRune)
			i++
		}
	}
	// Remove color codes (ANSI sequences), compiling the regex once lazily.
	cleanedString := string(cleanedRunes)
	shellColorCodeOnce.Do(func() {
		shellColorCodePattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	})
	return shellColorCodePattern.ReplaceAllString(cleanedString, "")
}

func (e *Editor) manPageHighlight(line string, firstLine, lastLine bool) string {
	line = handleManPageEscape(line)
	var (
		normal          = e.Foreground
		off             = vt.Stop()
		trimmedLine     = strings.TrimSpace(line)
		hasAnyWords     = hasWords(trimmedLine)
		innerSpaceCount = strings.Count(trimmedLine, " ")
	)
	if strings.Count(trimmedLine, "  ") > 10 && (firstLine || lastLine) { // first and last line
		return e.CommentColor.Get(line)
	}
	if strings.ToUpper(trimmedLine) == trimmedLine && !strings.HasPrefix(trimmedLine, "-") && hasAnyWords && !strings.HasPrefix(line, " ") { // a sub-section header
		return e.ManSectionColor.Get(line)
	}
	if strings.HasPrefix(trimmedLine, "-") { // a flag or parameter
		rs := []rune(e.MarkdownTextColor.String())
		inFlag := false
		spaceCount := 0
		foundLetter := false
		prevR := ' '
		for _, r := range line {
			if strings.HasPrefix(trimmedLine, "-") && strings.Count(line, "-") >= 1 && strings.Count(trimmedLine, " ") <= 1 {
				// One or two command line options, color them differently
				return e.MenuArrowColor.Get(line)
			}
			if !foundLetter && (unicode.IsLetter(r) || r == '_') {
				foundLetter = true
			}
			if r == ' ' {
				spaceCount++
				if innerSpaceCount > 8 {
					inFlag = false
				}
			} else {
				spaceCount = 0
			}
			if r != ' ' && (prevR == ' ' || prevR == '-') && (r == '-' || r == '[' || r == '_') && (prevR == '-' || !inFlag) {
				inFlag = true
				rs = append(rs, []rune(off+e.MenuArrowColor.String())...)
			} else if inFlag && !((prevR == ' ' || prevR == '-') && (r == '-' || r == '[' || r == ']' || r == '_')) { // Color the rest of the flag text in the textColor color (LightBlue)
				inFlag = false
				rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
			} else if foundLetter && spaceCount > 2 { // Color the rest of the line in the foreground color (LightGreen)
				rs = append(rs, []rune(off+normal.String())...)
			} else if r == ']' { // Color the rest of the line in the comment color (DarkGray)
				rs = append(rs, []rune(off+e.CommentColor.String())...)
			}
			rs = append(rs, r)
			prevR = r
		}
		rs = append(rs, []rune(off)...)
		return string(rs)
	}
	if allUpper(trimmedLine) {
		return e.MarkdownTextColor.Get(line)
	}
	// regular text, but highlight numbers (and hex numbers, if the number starts with a digit) + highlight "@"
	var (
		rs       []rune
		prevRune rune
		inDigits bool
		inWord   bool
		inAngles bool
		nextRune rune
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
		inWord = (unicode.IsLetter(r) || r == '_') || (inWord && unicode.IsLetter(r)) || (inWord && hexDigit(r))
		inAngles = (!inAngles && r == '<') || (inAngles && r != '>')
		if !inWord && unicode.IsDigit(r) && !inDigits {
			inDigits = true
			rs = append(rs, []rune(off+e.ItalicsColor.String())...)
		} else if inDigits && !inWord && !unicode.IsDigit(r) && !hexDigit(r) {
			inDigits = false
			rs = append(rs, []rune(off+normal.String())...)
		} else if inDigits && (unicode.IsDigit(r) || hexDigit(r)) {
			// don't apply a new color
		} else if !inWord && (r == '*' || r == '$' || r == '%' || r == '!' || r == '/' || r == '=' || r == '-') {
			rs = append(rs, []rune(off+e.MenuArrowColor.String())...)
		} else if r == '@' { // color @ gray and the rest of the string white
			rs = append(rs, []rune(off+e.CommentColor.String())...)
		} else if hasAlpha && r == '<' { // color < gray and the rest of the string white
			rs = append(rs, []rune(off+e.CommentColor.String())...)
		} else if hasAlpha && r == '>' { // color > gray and the rest of the string normal
			rs = append(rs, []rune(off+e.CommentColor.String())...)
		} else if inAngles || r == '>' {
			rs = append(rs, []rune(off+e.ItalicsColor.String())...)
		} else if inWord && unicode.IsUpper(prevRune) && ((unicode.IsUpper(r) && unicode.IsLetter(nextRune)) || (unicode.IsLower(r) && unicode.IsUpper(prevRune) && !unicode.IsLetter(nextRune))) {
			if unicode.IsUpper(r) {
				// This is for the leading and trailing letter of uppercase words
				rs = append(rs, []rune(off+e.ImageColor.String())...)
			} else {
				rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
			}
		} else if inWord && (unicode.IsUpper(r) || (unicode.IsUpper(prevRune) && unicode.IsLetter(r))) {
			if !unicode.IsLower(r) && (((unicode.IsUpper(nextRune) || nextRune == ' ') && unicode.IsLetter(prevRune)) || unicode.IsUpper(nextRune) || !unicode.IsLetter(nextRune)) {
				// This is for the center letters of uppercase words
				rs = append(rs, []rune(off+e.ImageColor.String())...)
			} else {
				rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
			}
		} else if inWord && unicode.IsUpper(r) {
			rs = append(rs, []rune(off+e.ImageColor.String())...)
		} else if !inWord || !unicode.IsUpper(r) {
			rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
		}
		rs = append(rs, r)
		if r == '@' || (hasAlpha && r == '<') { // color @ or < gray and the rest of the string white
			rs = append(rs, []rune(off+e.ItalicsColor.String())...)
		} else if hasAlpha && r == '>' { // color > gray and the rest of the string normal
			rs = append(rs, []rune(off+normal.String())...)
		}
		prevRune = r
	}
	return string(append(rs, []rune(off)...))
}
