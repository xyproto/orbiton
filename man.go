package main

import (
	"strings"
	"unicode"

	"github.com/xyproto/vt100"
)

func handleManPageEscape(s string) string {
	var lineRunes []rune
	for _, r := range s {
		if r == 0x8 {
			// Encountered ^H
			// Pop the last appended rune and continue
			lineRunes = lineRunes[:len(lineRunes)-1]
			continue
		}
		lineRunes = append(lineRunes, r)
	}
	return string(lineRunes)
}

func (e *Editor) manPageHighlight(line, programName string) string {
	line = handleManPageEscape(line)
	var (
		coloredString   string
		normal          = e.Foreground
		off             = vt100.Stop()
		trimmedLine     = strings.TrimSpace(line)
		hasWords        = HasWords(trimmedLine)
		innerSpaceCount = strings.Count(trimmedLine, " ")
	)

	if strings.Count(trimmedLine, "  ") > 10 { // first and last line
		coloredString = e.CommentColor.Get(line)
	} else if strings.ToUpper(trimmedLine) == trimmedLine && !strings.HasPrefix(trimmedLine, "-") && hasWords && !strings.HasPrefix(line, " ") { // a sub-section header
		coloredString = e.ManSectionColor.Get(line)
	} else if strings.HasPrefix(trimmedLine, "-") { // a flag or parameter
		var rs []rune
		rs = append(rs, []rune(e.MarkdownTextColor.String())...)
		inFlag := false
		spaceCount := 0
		foundLetter := false
		prevR := ' '
		for _, r := range line {
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
			if r != ' ' && prevR == ' ' && (r == '-' || r == '[' || r == '_') && !inFlag {
				inFlag = true
				rs = append(rs, []rune(off+vt100.White.String())...)
				rs = append(rs, r)
			} else if prevR == ' ' && (r == '-' || r == '[' || r == '_') && inFlag {
				rs = append(rs, r)
			} else if inFlag { // Color the rest of the flag text in the textColor color (LightBlue)
				inFlag = false
				rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
				rs = append(rs, r)
			} else if foundLetter && spaceCount > 2 { // Color the rest of the line in the foreground color (LightGreen)
				rs = append(rs, []rune(off+normal.String())...)
				rs = append(rs, r)
			} else if r == ']' { // Color the rest of the line in the comment color (DarkGray)
				rs = append(rs, []rune(off+e.CommentColor.String())...)
				rs = append(rs, r)
			} else {
				rs = append(rs, r)
			}
			prevR = r
		}
		rs = append(rs, []rune(off)...)
		coloredString = string(rs)
	} else if allUpper(trimmedLine) { //|| (oneField(trimmedLine) && !strings.HasSuffix(trimmedLine, ".")) { // filename? command?
		coloredString = e.MarkdownTextColor.Get(line)
	} else { // regular text, but highlight numbers (and hex numbers, if the number starts with a digit) + highlight "@"
		var rs []rune
		rs = append(rs, []rune(normal.String())...)
		inDigits := false
		inWord := false
		hasAlpha := strings.Contains(trimmedLine, "@")
		inAngles := false
		lineRunes := []rune(line)
		for i, r := range line {
			if (unicode.IsLetter(r) || r == '_') && !inWord {
				inWord = true
			} else if inWord && !unicode.IsLetter(r) && !hexDigit(r) {
				inWord = false
			}
			if !inAngles && r == '<' {
				inAngles = true
			} else if inAngles && r == '>' {
				inAngles = false
			}
			if !inWord && unicode.IsDigit(r) && !inDigits {
				inDigits = true
				rs = append(rs, []rune(off+e.ItalicsColor.String())...)
				rs = append(rs, r)
			} else if inDigits && hexDigit(r) {
				rs = append(rs, r)
			} else if !inWord && inDigits {
				inDigits = false
				rs = append(rs, []rune(off+normal.String())...)
				rs = append(rs, r)
			} else if !inWord && (r == '*' || r == '$' || r == '%' || r == '!' || r == '/' || r == '=' || r == '-') {
				rs = append(rs, []rune(off+vt100.White.String())...)
				rs = append(rs, r)
			} else if r == '@' { // color @ gray and the rest of the string white
				rs = append(rs, []rune(off+e.CommentColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(off+e.ItalicsColor.String())...)
			} else if hasAlpha && r == '<' { // color < gray and the rest of the string white
				rs = append(rs, []rune(off+e.CommentColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(off+e.ItalicsColor.String())...)
			} else if hasAlpha && r == '>' { // color > gray and the rest of the string normal
				rs = append(rs, []rune(off+e.CommentColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(off+normal.String())...)
			} else if inAngles || r == '>' {
				rs = append(rs, []rune(off+e.ItalicsColor.String())...)
				rs = append(rs, r)
			} else if inWord && unicode.IsUpper(r) && (i+1 < len(lineRunes)) {
				nextRune := lineRunes[i+1]
				if unicode.IsUpper(nextRune) || !unicode.IsLetter(nextRune) {
					rs = append(rs, []rune(off+vt100.Yellow.String())...)
				} else {
					rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
				}
				rs = append(rs, r)
			} else if inWord && unicode.IsUpper(r) {
				rs = append(rs, []rune(off+vt100.Yellow.String())...)
				rs = append(rs, r)
			} else if !inWord || !unicode.IsUpper(r) {
				rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
				rs = append(rs, r)
			} else {
				rs = append(rs, r)
			}
			//prevRune = r
		}
		rs = append(rs, []rune(off)...)
		coloredString = string(rs)
	}
	return coloredString
}
