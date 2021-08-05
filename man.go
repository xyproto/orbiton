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

func (e *Editor) manPageHighlight(line, programName string, prevLineIsBlank, prevLineIsSectionHeader bool) (string, bool) {
	var coloredString string

	lineIsSectionHeader := false
	normal := e.Foreground
	off := vt100.Stop()

	foundSynopsis := false
	foundSectionAfterSynopsis := false

	line = handleManPageEscape(line)
	trimmedLine := strings.TrimSpace(line)
	hasWords := HasWords(trimmedLine)

	if !(prevLineIsBlank || prevLineIsSectionHeader) && strings.Count(trimmedLine, ")") == 2 && strings.Count(trimmedLine, "(") == 2 && strings.HasSuffix(trimmedLine, ")") && !strings.Contains(trimmedLine, ",") && (strings.HasPrefix(trimmedLine, programName) || firstLetterIsUpper(line)) { // top header or footer
		coloredString = e.CommentColor.Get(line)
	} else if strings.ToUpper(trimmedLine) == trimmedLine && !strings.HasPrefix(trimmedLine, "-") && hasWords && !strings.HasPrefix(line, " ") { // a sub-section header
		if trimmedLine == "SYNOPSIS" {
			foundSynopsis = true
		} else if foundSynopsis {
			foundSectionAfterSynopsis = true
		}
		coloredString = e.ManSectionColor.Get(line)
		lineIsSectionHeader = true
	} else if strings.HasPrefix(trimmedLine, "-") || strings.HasPrefix(trimmedLine, "[-") || strings.HasPrefix(trimmedLine, "[[-") && !foundSectionAfterSynopsis { // a flag or parameter
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
			} else {
				spaceCount = 0
			}
			if prevR == ' ' && (r == '-' || r == '[' || r == '_') && !inFlag {
				inFlag = true
				rs = append(rs, []rune(off+e.HeaderBulletColor.String())...)
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
	} else if (prevLineIsBlank || prevLineIsSectionHeader) && oneWordNoSpaces(trimmedLine) && !strings.Contains(trimmedLine, "=") && foundSectionAfterSynopsis {
		coloredString = e.ManSynopsisColor.Get(line)
	} else if strings.Contains(trimmedLine, "://") && oneField(trimmedLine) { // URL
		coloredString = e.ItalicsColor.Get(line)
	} else if strings.Contains(trimmedLine, "[") && !foundSectionAfterSynopsis && !strings.Contains(trimmedLine, ".") { // synopsis
		parts := strings.SplitN(line, "[", 2)
		trimmedParts := strings.SplitN(trimmedLine, "[", 2)
		if strings.Count(trimmedParts[0], " ") > 2 {
			coloredString = normal.Get(line)
		} else if strings.HasSuffix(trimmedLine, "]") {
			inBrackets := parts[1][:len(parts[1])-1]
			coloredString = e.ManSynopsisColor.Get(parts[0]) + e.CommentColor.Get("[") + e.ItalicsColor.Get(inBrackets)
			coloredString += e.CommentColor.Get("]")
		} else {
			coloredString = e.ManSynopsisColor.Get(parts[0]) + e.CommentColor.Get("[") + e.ItalicsColor.Get(parts[1])
		}
	} else if strings.Contains(trimmedLine, "(") && strings.Contains(trimmedLine, ")") { // regular text with paranthesis
		var rs []rune
		rs = append(rs, []rune(normal.String())...)
		inNum := false
		inUpper := false
		lineRunes := []rune(line)
		for i, r := range lineRunes {
			nextIsDigit := ((i + 1) < len(lineRunes)) && unicode.IsDigit(lineRunes[i+1])
			nextIsUpper := ((i + 1) < len(lineRunes)) && unicode.IsUpper(lineRunes[i+1])
			if r == '(' && nextIsDigit {
				inNum = true
				rs = append(rs, []rune(off+e.CommentColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(off+e.ManSectionColor.String())...)
			} else if inNum && !nextIsDigit {
				inNum = false
				rs = append(rs, []rune(off+e.CommentColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(off+normal.String())...)
			} else if unicode.IsUpper(r) && nextIsUpper {
				inUpper = true
				rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(off+normal.String())...)
			} else if inUpper && !nextIsUpper {
				inUpper = false
				rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(off+normal.String())...)
			} else if unicode.IsUpper(r) && nextIsUpper {
				rs = append(rs, []rune(off+e.MarkdownTextColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(off+normal.String())...)
			} else {
				rs = append(rs, r)
			}
		}
		rs = append(rs, []rune(off)...)
		coloredString = string(rs)
	} else if allUpper(trimmedLine) || ((prevLineIsBlank || prevLineIsSectionHeader) && oneField(trimmedLine) && !strings.HasSuffix(trimmedLine, ".")) { // filename? command?
		coloredString = e.MarkdownTextColor.Get(line)
	} else { // regular text, but highlight numbers (and hex numbers, if the number starts with a digit) + highlight "@"
		var rs []rune
		rs = append(rs, []rune(normal.String())...)
		inDigits := false
		inWord := false
		hasAlpha := strings.Contains(trimmedLine, "@")
		for _, r := range line {
			if (unicode.IsLetter(r) || r == '_') && !inWord {
				inWord = true
			} else if inWord && !unicode.IsLetter(r) && !hexDigit(r) {
				inWord = false
			}
			if !inWord && unicode.IsDigit(r) && !inDigits {
				inDigits = true
				rs = append(rs, []rune(off+e.ItalicsColor.String())...)
				rs = append(rs, r)
			} else if hexDigit(r) && inDigits {
				rs = append(rs, r)
			} else if !inWord && inDigits {
				inDigits = false
				rs = append(rs, []rune(off+normal.String())...)
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
			} else {
				rs = append(rs, r)
			}
		}
		rs = append(rs, []rune(off)...)
		coloredString = string(rs)
	}
	return coloredString, lineIsSectionHeader
}
