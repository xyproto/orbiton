package main

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/xyproto/vt100"
)

var (
	checkboxPrefixes = []string{"- [ ]", "- [x]", "- [X]", "* [ ]", "* [x]", "* [X]"}

	textColor         = vt100.LightBlue
	headerBulletColor = vt100.DarkGray
	headerTextColor   = vt100.LightGreen
	listBulletColor   = vt100.Red
	listTextColor     = vt100.LightCyan
	listCodeColor     = vt100.Default
	codeColor         = vt100.Default
	codeBlockColor    = vt100.Default
	imageColor        = vt100.LightYellow
	linkColor         = vt100.Magenta
	quoteColor        = vt100.Yellow
	quoteTextColor    = vt100.LightCyan
	htmlColor         = vt100.Default
	commentColor      = vt100.DarkGray
	boldColor         = vt100.LightYellow
	italicsColor      = vt100.White
	strikeColor       = vt100.DarkGray
	tableColor        = vt100.Blue
	checkboxColor     = vt100.Default     // a Markdown checkbox: [ ], [x] or [X]
	xColor            = vt100.LightYellow // the x in the checkbox: [x]
	tableBackground   = vt100.BackgroundDefault
)

// ToggleCheckboxCurrentLine will attempt to toggle the Markdown checkbox on the current line of the editor.
// Returns true if toggled.
func (e *Editor) ToggleCheckboxCurrentLine() bool {
	// Toggle Markdown checkboxes
	if line := e.CurrentLine(); hasAnyPrefixWord(strings.TrimSpace(line), checkboxPrefixes) {
		if strings.Contains(line, "[ ]") {
			e.SetLine(e.DataY(), strings.Replace(line, "[ ]", "[x]", 1))
			e.redraw = true
		} else if strings.Contains(line, "[x]") {
			e.SetLine(e.DataY(), strings.Replace(line, "[x]", "[ ]", 1))
			e.redraw = true
		} else if strings.Contains(line, "[X]") {
			e.SetLine(e.DataY(), strings.Replace(line, "[X]", "[ ]", 1))
			e.redraw = true
		}
		e.redrawCursor = e.redraw
		return true
	}
	return false
}

func runeCount(s string, r rune) int {
	counter := 0
	for _, e := range s {
		if e == r {
			counter++
		}
	}
	return counter
}

// quotedWordReplace will replace quoted words with a highlighted version
// line is the uncolored string
// quote is the quote string (like "`" or "**")
// regular is the color of the regular text
// quoted is the color of the highlighted quoted text (including the quotes)
func quotedWordReplace(line string, quote rune, regular, quoted vt100.AttributeColor) string {
	// Now do backtick replacements
	if strings.ContainsRune(line, quote) && runeCount(line, quote)%2 == 0 {
		inQuote := false
		s := make([]rune, 0, len(line)*2)
		// Start by setting the color to the regular one
		s = append(s, []rune(regular.String())...)
		var prevR, nextR rune
		runes := []rune(line)
		for i, r := range runes {
			// Look for quotes, but also handle **`asdf`** and __`asdf`__
			if r == quote && prevR != '*' && nextR != '*' && prevR != '_' && nextR != '_' {
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
			prevR = r                 // the previous r, for the next round
			nextR = r                 // default value, in case the next rune can not be fetched
			if (i + 2) < len(runes) { // + 2 since it must look 1 head for the next round
				nextR = []rune(line)[i+2]
			}
		}
		// End by turning the color off
		s = append(s, []rune(vt100.Stop())...)
		return string(s)
	}
	// Return the same line, but colored, if the quotes are not balanced
	return regular.Get(line)
}

func style(line, marker string, textColor, styleColor vt100.AttributeColor) string {
	n := strings.Count(line, marker)
	if n < 2 {
		// There must be at least two found markers
		return line
	}
	if n%2 != 0 {
		// The markers must be found in pairs
		return line
	}
	// Split the line up in parts, then combine the parts, with colors
	parts := strings.Split(line, marker)
	lastIndex := len(parts) - 1
	result := ""
	for i, part := range parts {
		switch {
		case i == lastIndex:
			// Last case
			result += part
		case i%2 == 0:
			// Even case that is not the last case
			if len(part) == 0 {
				result += marker
			} else {
				result += part + vt100.Stop() + styleColor.String() + marker
			}
		default:
			// Odd case that is not the last case
			if len(part) == 0 {
				result += marker
			} else {
				result += part + marker + vt100.Stop() + textColor.String()
			}
		}
	}
	return result
}

func emphasis(line string, textColor, italicsColor, boldColor, strikeColor vt100.AttributeColor) string {
	result := line
	result = style(result, "~~", textColor, strikeColor)
	result = style(result, "**", textColor, boldColor)
	result = style(result, "__", textColor, boldColor)
	// For now, nested emphasis and italics are not supported, only bold and strikethrough
	// TODO: Implement nested emphasis and italics
	//result = style(result, "*", textColor, italicsColor)
	//result = style(result, "_", textColor, italicsColor)
	return result
}

// isListItem checks if the given line is likely to be a Markdown list item
func isListItem(line string) bool {
	trimmedLine := strings.TrimSpace(line)
	fields := strings.Fields(trimmedLine)
	if len(fields) == 0 {
		return false
	}
	firstWord := fields[0]

	// Check if this is a regular list item
	switch firstWord {
	case "*", "-", "+":
		return true
	}

	// Check if this is a numbered list item
	if strings.HasSuffix(firstWord, ".") {
		if _, err := strconv.Atoi(firstWord[:len(firstWord)-1]); err == nil { // success
			return true
		}
	}

	return false
}

// markdownHighlight returns a VT100 colored line, a bool that is true if it worked out and a bool that is true if it's the start or stop of a block quote
func markdownHighlight(line string, inCodeBlock, prevLineIsListItem bool, inListItem *bool) (string, bool, bool) {

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
	if strings.HasPrefix(rest, "~~~") || strings.HasPrefix(rest, "```") { // TODO: fix syntax highlighting when this comment is removed `
		return codeBlockColor.Get(line), true, true
	}

	if inCodeBlock {
		return codeBlockColor.Get(line), true, false
	}

	if leadingSpace == "    " && !strings.HasPrefix(rest, "*") && !strings.HasPrefix(rest, "-") && !prevLineIsListItem {
		// Four leading spaces means a quoted line
		// Also assume it's not a quote if it starts with "*" or "-"
		return codeColor.Get(line), true, false
	}

	// An image (or a link to a single image) on a single line
	if (strings.HasPrefix(rest, "[!") || strings.HasPrefix(rest, "!")) && strings.HasSuffix(rest, ")") {
		return imageColor.Get(line), true, false
	}

	// A link on a single line
	if strings.HasPrefix(rest, "[") && strings.HasSuffix(rest, ")") && strings.Count(rest, "[") == 1 {
		return linkColor.Get(line), true, false
	}

	// A header line
	if strings.HasPrefix(rest, "---") || strings.HasPrefix(rest, "===") {
		return headerTextColor.Get(line), true, false
	}

	// HTML comments
	if strings.HasPrefix(rest, "<!--") || strings.HasPrefix(rest, "-->") {
		return commentColor.Get(line), true, false
	}

	// A line with just a quote mark
	if strings.TrimSpace(rest) == ">" {
		return quoteColor.Get(line), true, false
	}

	// A quote with something that follows
	if pos := strings.Index(rest, "> "); pos >= 0 && pos < 5 {
		words := strings.Fields(rest)
		if len(words) >= 2 {
			return quoteColor.Get(words[0]) + " " + quoteTextColor.Get(strings.Join(words[1:], " ")), true, false
		}
	}

	// HTML
	if strings.HasPrefix(rest, "<") || strings.HasPrefix(rest, ">") {
		return htmlColor.Get(line), true, false
	}

	// Table
	if strings.HasPrefix(rest, "|") || strings.HasSuffix(rest, "|") {
		if strings.HasPrefix(line, "|-") {
			return tableColor.String() + line + tableBackground.String(), true, false
		}
		return strings.Replace(line, "|", tableColor.String()+"|"+tableBackground.String(), -1), true, false
	}

	// Split the rest of the line into words
	words := strings.Fields(rest)
	if len(words) == 0 {
		*inListItem = false
		// Nothing to do here
		return "", false, false
	}

	// Color differently depending on the leading word
	firstWord := words[0]
	lastWord := words[len(words)-1]

	switch {
	case consistsOf(firstWord, '#', []rune{'.', ' '}):
		if strings.HasSuffix(lastWord, "#") && strings.Contains(rest, " ") {
			centerLen := len(rest) - (len(firstWord) + len(lastWord))
			if centerLen > 0 {
				centerText := rest[len(firstWord) : len(rest)-len(lastWord)]
				return leadingSpace + headerBulletColor.Get(firstWord) + headerTextColor.Get(centerText) + headerBulletColor.Get(lastWord), true, false
			}
			return leadingSpace + headerBulletColor.Get(rest), true, false
		} else if len(words) > 1 {
			return leadingSpace + headerBulletColor.Get(firstWord) + " " + headerTextColor.Get(emphasis(quotedWordReplace(line[dataPos+len(firstWord)+1:], '`', headerTextColor, codeColor), headerTextColor, italicsColor, boldColor, strikeColor)), true, false // TODO: `
		}
		return leadingSpace + headerTextColor.Get(rest), true, false
	case isListItem(line):
		if strings.HasPrefix(rest, "- [ ] ") || strings.HasPrefix(rest, "- [x] ") || strings.HasPrefix(rest, "- [X] ") {
			return leadingSpace + listBulletColor.Get(rest[:1]) + " " + checkboxColor.Get(rest[2:3]) + xColor.Get(rest[3:4]) + checkboxColor.Get(rest[4:5]) + " " + emphasis(quotedWordReplace(line[dataPos+6:], '`', listTextColor, listCodeColor), listTextColor, italicsColor, boldColor, strikeColor), true, false
		}
		if len(words) > 1 {
			return leadingSpace + listBulletColor.Get(firstWord) + " " + emphasis(quotedWordReplace(line[dataPos+len(firstWord)+1:], '`', listTextColor, listCodeColor), listTextColor, italicsColor, boldColor, strikeColor), true, false
		}
		return leadingSpace + listTextColor.Get(rest), true, false
	}

	// Leading hash without a space afterwards?
	if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "# ") {
		return vt100.Red.Get(line), true, false
	}

	// TODO: Refactor the "in list item" functionality
	if prevLineIsListItem {
		*inListItem = true
	}

	// A completely regular line of text that is also the continuation of a list item
	if *inListItem {
		return emphasis(quotedWordReplace(line, '`', listTextColor, listCodeColor), listTextColor, italicsColor, boldColor, strikeColor), true, false
	}

	// A completely regular line of text
	return emphasis(quotedWordReplace(line, '`', textColor, codeColor), textColor, italicsColor, boldColor, strikeColor), true, false
}
