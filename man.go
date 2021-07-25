package main

import (
	"strings"
	"unicode"

	"github.com/xyproto/vt100"
)

var (
	manSectionColor  = vt100.LightRed
	manSynopsisColor = vt100.LightYellow
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
	normal := e.fg

	foundSynopsis := false
	foundSectionAfterSynopsis := false

	line = handleManPageEscape(line)
	trimmedLine := strings.TrimSpace(line)
	hasWords := HasWords(trimmedLine)

	if !(prevLineIsBlank || prevLineIsSectionHeader) && strings.HasSuffix(trimmedLine, ")") && !strings.Contains(trimmedLine, ",") && (strings.HasPrefix(trimmedLine, programName) || firstLetterIsUpper(line)) { // top header or footer
		coloredString = commentColor.Get(line)
	} else if strings.ToUpper(trimmedLine) == trimmedLine && !strings.HasPrefix(trimmedLine, "-") && hasWords && !strings.HasPrefix(line, " ") { // a sub-section header
		if trimmedLine == "SYNOPSIS" {
			foundSynopsis = true
		} else if foundSynopsis {
			foundSectionAfterSynopsis = true
		}
		coloredString = manSectionColor.Get(line)
		lineIsSectionHeader = true
	} else if strings.HasPrefix(trimmedLine, "-") || strings.HasPrefix(trimmedLine, "[-") || strings.HasPrefix(trimmedLine, "[[-") && !foundSectionAfterSynopsis { // a flag or parameter
		var rs []rune
		rs = append(rs, []rune(textColor.String())...)
		inFlag := false
		spaceCount := 0
		foundLetter := false
		prevR := ' '
		for _, r := range line {
			if !foundLetter && unicode.IsLetter(r) {
				foundLetter = true
			}
			if r == ' ' {
				spaceCount++
			} else {
				spaceCount = 0
			}
			if prevR == ' ' && (r == '-' || r == '[') && !inFlag {
				inFlag = true
				rs = append(rs, []rune(vt100.Stop()+headerBulletColor.String())...)
				rs = append(rs, r)
			} else if prevR == ' ' && (r == '-' || r == '[') && inFlag {
				rs = append(rs, r)
			} else if inFlag { // Color the rest of the flag text in the textColor color (LightBlue)
				inFlag = false
				rs = append(rs, []rune(vt100.Stop()+textColor.String())...)
				rs = append(rs, r)
			} else if foundLetter && spaceCount > 2 { // Color the rest of the line in the foreground color (LightGreen)
				rs = append(rs, []rune(vt100.Stop()+normal.String())...)
				rs = append(rs, r)
			} else if r == ']' || r == '_' { // Color the rest of the line in the comment color (DarkGray)
				rs = append(rs, []rune(vt100.Stop()+commentColor.String())...)
				rs = append(rs, r)
			} else {
				rs = append(rs, r)
			}
			prevR = r
		}
		rs = append(rs, []rune(vt100.Stop())...)
		coloredString = string(rs)
	} else if (prevLineIsBlank || prevLineIsSectionHeader) && oneWordNoSpaces(trimmedLine) && !strings.Contains(trimmedLine, "=") {
		coloredString = manSynopsisColor.Get(line)
	} else if strings.Contains(trimmedLine, "://") && oneField(trimmedLine) { // URL
		coloredString = italicsColor.Get(line)
	} else if !hasWords { // the line has no words
		coloredString = italicsColor.Get(line)
	} else if strings.Contains(trimmedLine, "[") && !foundSectionAfterSynopsis { // synopsis
		parts := strings.SplitN(line, "[", 2)
		if strings.HasSuffix(trimmedLine, "]") {
			inBrackets := parts[1][:len(parts[1])-1]
			coloredString = manSynopsisColor.Get(parts[0]) + commentColor.Get("[") + italicsColor.Get(inBrackets)
			coloredString += commentColor.Get("]")
		} else {
			coloredString = manSynopsisColor.Get(parts[0]) + commentColor.Get("[") + italicsColor.Get(parts[1])
		}
	} else if specialRatio(trimmedLine) > 0.3 { // a lot of special characters
		coloredString = italicsColor.Get(line)
	} else if strings.Contains(trimmedLine, "(") && strings.Contains(trimmedLine, ")") { // regular text with paranthesis
		var rs []rune
		rs = append(rs, []rune(normal.String())...)
		inNum := false
		lineRunes := []rune(line)
		for i, r := range lineRunes {
			nextIsDigit := ((i + 1) < len(lineRunes)) && unicode.IsDigit(lineRunes[i+1])
			if r == '(' && nextIsDigit {
				inNum = true
				rs = append(rs, []rune(vt100.Stop()+commentColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(vt100.Stop()+manSectionColor.String())...)
			} else if r == ')' && inNum {
				inNum = false
				rs = append(rs, []rune(vt100.Stop()+commentColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(vt100.Stop()+normal.String())...)
			} else {
				rs = append(rs, r)
			}
		}
		rs = append(rs, []rune(vt100.Stop())...)
		coloredString = string(rs)
	} else { // regular text, but highlight numbers (and hex numbers, if the number starts with a digit) + highlight "@"
		var rs []rune
		rs = append(rs, []rune(normal.String())...)
		inDigits := false
		inWord := false
		hasAlpha := strings.Contains(trimmedLine, "@")
		for _, r := range line {
			if unicode.IsLetter(r) && !inWord {
				inWord = true
			} else if inWord && !unicode.IsLetter(r) && !hexDigit(r) {
				inWord = false
			}
			if !inWord && unicode.IsDigit(r) && !inDigits {
				inDigits = true
				rs = append(rs, []rune(vt100.Stop()+italicsColor.String())...)
				rs = append(rs, r)
			} else if hexDigit(r) && inDigits {
				rs = append(rs, r)
			} else if !inWord && inDigits {
				inDigits = false
				rs = append(rs, []rune(vt100.Stop()+normal.String())...)
				rs = append(rs, r)
			} else if r == '@' { // color @ gray and the rest of the string white
				rs = append(rs, []rune(vt100.Stop()+commentColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(vt100.Stop()+italicsColor.String())...)
			} else if hasAlpha && r == '<' { // color < gray and the rest of the string white
				rs = append(rs, []rune(vt100.Stop()+commentColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(vt100.Stop()+italicsColor.String())...)
			} else if hasAlpha && r == '>' { // color > gray and the rest of the string normal
				rs = append(rs, []rune(vt100.Stop()+commentColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(vt100.Stop()+normal.String())...)
			} else {
				rs = append(rs, r)
			}
		}
		rs = append(rs, []rune(vt100.Stop())...)
		coloredString = string(rs)
	}
	return coloredString, lineIsSectionHeader
}
