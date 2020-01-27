package main

import (
	"strings"
	"unicode"

	"github.com/xyproto/vt100"
)

func backTickReplace(line string, regular, quoted vt100.AttributeColor) string {
	// Now do backtick replacements
	if strings.Contains(line, "`") && strings.Count(line, "`")%2 == 0 {
		inQuote := false
		s := make([]rune, 0, len(line)*2)
		// Start by setting the color to the regular one
		s = append(s, []rune(regular.String())...)
		for _, r := range line {
			if r == '`' {
				inQuote = !inQuote
				if inQuote {
					s = append(s, []rune(vt100.Stop())...)
					s = append(s, []rune(quoted.String())...)
					s = append(s, r)
					continue
				} else {
					s = append(s, r)
					s = append(s, []rune(vt100.Stop())...)
					s = append(s, []rune(regular.String())...)
					continue
				}
			}
			s = append(s, r)
		}
		// End by turning the color off
		s = append(s, []rune(vt100.Stop())...)
		return string(s)
	}
	// Return the same line, but colored, if the quotes are not balanced
	return regular.Get(line)
}

// markdownHighlight returns a VT100 colored line, a bool that is true if it worked out and a bool that is true if it's the start or stop of a block quote
func markdownHighlight(line string, inCodeBlock bool) (string, bool, bool) {

	dataPos := 0
	for i, r := range line {
		if unicode.IsSpace(r) {
			dataPos = i + 1
		} else {
			break
		}
	}

	// First position of non-space on line is now dataPos
	leadingSpace := line[:dataPos]

	// Get the rest of the line that isn't whitespace
	rest := line[dataPos:]

	// Starting or ending a code block
	if strings.HasPrefix(rest, "~~~") || strings.HasPrefix(rest, "```") {
		return vt100.White.Get(line), true, true
	}

	if inCodeBlock {
		return vt100.White.Get(line), true, false
	}

	if leadingSpace == "    " && !strings.HasPrefix(rest, "*") {
		// Four leading spaces means a quoted line
		// Assume it's not a quote if it starts with "*"
		return vt100.White.Get(line), true, false
	}

	// An image (or a link to a single image) on a single line
	if (strings.HasPrefix(rest, "[!") || strings.HasPrefix(rest, "!")) && strings.HasSuffix(rest, ")") {
		return vt100.LightYellow.Get(line), true, false
	}

	// A link on a single line
	if strings.HasPrefix(rest, "[") && strings.HasSuffix(rest, ")") && strings.Count(rest, "[") == 1 {
		return vt100.LightYellow.Get(line), true, false
	}

	// A header line
	if strings.HasPrefix(rest, "---") {
		return vt100.LightGreen.Get(line), true, false
	}

	// HTML comments
	if strings.HasPrefix(rest, "<!--") || strings.HasPrefix(rest, "-->") {
		return vt100.DarkGray.Get(line), true, false
	}

	// A line with just a quote mark
	if strings.TrimSpace(rest) == ">" {
		return vt100.Red.Get(line), true, false
	}

	// A quote with something that follows
	if pos := strings.Index(rest, "> "); pos >= 0 && pos < 5 {
		words := strings.Fields(rest)
		if len(words) >= 2 {
			return vt100.Red.Get(words[0]) + " " + vt100.LightCyan.Get(strings.Join(words[1:], " ")), true, false
		}
	}

	// HTML
	if strings.HasPrefix(rest, "<") || strings.HasPrefix(rest, ">") {
		return vt100.LightRed.Get(line), true, false
	}

	// Split the rest of the line into words
	words := strings.Fields(rest)
	if len(words) == 0 {
		// Nothing to do here
		return "", false, false
	}

	// Color differently depending on the leading word
	firstWord := words[0]
	switch firstWord {
	case "#", "##", "###", "####", "#####", "######", "#######":
		if len(words) > 1 {
			return leadingSpace + vt100.LightGreen.Get(firstWord) + " " + vt100.LightGreen.Get(backTickReplace(line[dataPos+len(firstWord)+1:], vt100.LightGreen, vt100.White)), true, false
		}
		return leadingSpace + vt100.LightGreen.Get(rest), true, false
	case "*", "1.", "2.", "3.", "4.", "5.", "6.", "7.", "8.", "9.":
		if len(words) > 1 {
			return leadingSpace + vt100.LightRed.Get(firstWord) + " " + backTickReplace(line[dataPos+len(firstWord)+1:], vt100.LightMagenta, vt100.LightYellow), true, false
		}
		return leadingSpace + vt100.LightRed.Get(rest), true, false
	}

	// A completely regular line of text
	return backTickReplace(line, vt100.LightBlue, vt100.White), true, false
}
