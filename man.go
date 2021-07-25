package main

import (
	"strings"
	"unicode"

	"github.com/xyproto/vt100"
)

var manSectionColor = vt100.LightRed
var manSynopsisColor = vt100.LightYellow

func handleManPageEscape(s string) string {
	var lineRunes []rune
	skipNext := false
	for _, r := range s {
		if r == 0x8 {
			skipNext = true
			continue
		}
		if skipNext {
			skipNext = false
			continue
		}
		lineRunes = append(lineRunes, r)
	}
	return string(lineRunes)
}

func (e *Editor) manPageHighlight(line string) string {
	var coloredString string

	line = handleManPageEscape(line)

	trimmedLine := strings.TrimSpace(line)
	if strings.ToUpper(trimmedLine) == trimmedLine && !strings.HasPrefix(trimmedLine, "-") { // a sub-section header
		coloredString = manSectionColor.Get(line)
	} else if strings.HasPrefix(trimmedLine, "-") { // a flag or parameter
		var rs []rune
		rs = append(rs, []rune(textColor.String())...)
		inFlag := false
		for _, r := range line {
			if r == '-' && !inFlag {
				inFlag = true
				rs = append(rs, []rune(vt100.Stop()+headerBulletColor.String())...)
				rs = append(rs, r)
			} else if r == '-' && inFlag {
				rs = append(rs, r)
			} else if inFlag {
				inFlag = false
				rs = append(rs, []rune(vt100.Stop()+textColor.String())...)
				rs = append(rs, r)
			} else {
				rs = append(rs, r)
			}
		}
		rs = append(rs, []rune(vt100.Stop())...)
		coloredString = string(rs)
	} else if strings.HasSuffix(trimmedLine, ")") && !strings.Contains(trimmedLine, ",") { // top header or footer
		coloredString = commentColor.Get(line)
	} else if strings.HasSuffix(trimmedLine, "]") && strings.Contains(trimmedLine, "[") { // synopsis
		parts := strings.SplitN(line, "[", 2)
		inBrackets := parts[1][:len(parts[1])-1]
		coloredString = manSynopsisColor.Get(parts[0]) + commentColor.Get("[") + italicsColor.Get(inBrackets) + commentColor.Get("]")
	} else if strings.Contains(trimmedLine, "(") && strings.Contains(trimmedLine, ")") { // regular text with paranthesis
		var rs []rune
		rs = append(rs, []rune(e.fg.String())...)
		inNum := false
		lineRunes := []rune(line)
		for i, r := range lineRunes {
			nextIsNum := ((i + 1) < len(lineRunes)) && unicode.IsDigit(lineRunes[i+1])
			if r == '(' && nextIsNum {
				inNum = true
				rs = append(rs, []rune(vt100.Stop()+commentColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(vt100.Stop()+manSectionColor.String())...)
			} else if r == ')' && inNum {
				inNum = false
				rs = append(rs, []rune(vt100.Stop()+commentColor.String())...)
				rs = append(rs, r)
				rs = append(rs, []rune(vt100.Stop()+e.fg.String())...)
			} else {
				rs = append(rs, r)
			}
		}
		rs = append(rs, []rune(vt100.Stop())...)
		coloredString = string(rs)
	} else { // regular text, but highlight numbers (and hex numbers, if the number starts with a digit)
		var rs []rune
		rs = append(rs, []rune(e.fg.String())...)
		inDigits := false
		for _, r := range line {
			if unicode.IsDigit(r) && !inDigits {
				inDigits = true
				rs = append(rs, []rune(vt100.Stop()+italicsColor.String())...)
				rs = append(rs, r)
			} else if hexDigit(r) && inDigits {
				rs = append(rs, r)
			} else if inDigits {
				inDigits = false
				rs = append(rs, []rune(vt100.Stop()+e.fg.String())...)
				rs = append(rs, r)
			} else {
				rs = append(rs, r)
			}
		}
		rs = append(rs, []rune(vt100.Stop())...)
		coloredString = string(rs)
	}
	return coloredString
}
