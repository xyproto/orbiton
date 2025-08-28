package main

import (
	"strconv"
	"strings"
	"unicode"
)

// ToggleCheckboxCurrentLine will attempt to toggle the Markdown checkbox on the current line of the editor.
// Returns true if toggled.
func (e *Editor) ToggleCheckboxCurrentLine() bool {
	var (
		line    = e.CurrentLine()
		trimmed = strings.TrimSpace(line)
		newLine string
		found   bool

		// Check each checkbox pattern and replace in one pass
		checkboxPatterns = [][2]string{
			{"- [ ]", "- [x]"},
			{"- [x]", "- [ ]"},
			{"- [X]", "- [ ]"},
			{"* [ ]", "* [x]"},
			{"* [x]", "* [ ]"},
			{"* [X]", "* [ ]"},
		}
	)

	for _, pattern := range checkboxPatterns {
		prefix := pattern[0]
		// Check if line starts with the pattern and is followed by space, end of line, or has content after
		if strings.HasPrefix(trimmed, prefix) && (len(trimmed) == len(prefix) || trimmed[len(prefix)] == ' ') {
			newLine = strings.Replace(line, prefix, pattern[1], 1)
			found = true
			break
		}
	}

	if found {
		e.SetLine(e.DataY(), newLine)
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
	}

	return found
}

// quotedWordReplace will replace quoted words with a highlighted version
// line is the uncolored string
// quote is the quote string (like "`" or "**")
// regular is the color of the regular text
// quoted is the color of the highlighted quoted text (including the quotes)
func quotedWordReplace(line string, quote rune, regular, quoted AttributeColor) string {
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
					s = append(s, []rune(Stop())...)
					s = append(s, []rune(quoted.String())...)
					s = append(s, r)
					continue
				}
				s = append(s, r)
				s = append(s, []rune(Stop())...)
				s = append(s, []rune(regular.String())...)
				continue
			}
			s = append(s, r)
			prevR = r                 // the previous r, for the next round
			nextR = r                 // default value, in case the next rune can not be fetched
			if (i + 2) < len(runes) { // + 2 since it must look 1 head for the next round
				nextR = []rune(line)[i+2]
			}
		}
		// End by turning the color off
		s = append(s, []rune(Stop())...)
		return string(s)
	}
	// Return the same line, but colored, if the quotes are not balanced
	return regular.Get(line)
}

func style(line, marker string, textColor, styleColor AttributeColor) string {
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
				result += part + Stop() + styleColor.String() + marker
			}
		default:
			// Odd case that is not the last case
			if len(part) == 0 {
				result += marker
			} else {
				result += part + marker + Stop() + textColor.String()
			}
		}
	}
	return result
}

func emphasis(line string, textColor, italicsColor, boldColor, strikeColor AttributeColor) string {
	result := line
	if !withinBackticks(line, "~~") {
		result = style(result, "~~", textColor, strikeColor)
	}
	if !withinBackticks(line, "**") {
		result = style(result, "**", textColor, boldColor)
	}
	if !withinBackticks(line, "__") {
		result = style(result, "__", textColor, boldColor)
	}
	if !strings.Contains(line, "**") && !withinBackticks(line, "*") {
		result = style(result, "*", textColor, italicsColor)
	}
	if !strings.Contains(line, "__") && !withinBackticks(line, "_") {
		result = style(result, "_", textColor, italicsColor)
	}
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
func (e *Editor) markdownHighlight(line string, inCodeBlock bool, listItemRecord []bool, inListItem *bool) (string, bool, bool) {
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
		return e.CodeBlockColor.Get(line), true, true
	}

	if inCodeBlock {
		return e.CodeBlockColor.Get(line), true, false
	}

	// N is the number of lines to highlight with the same color for each numbered point or bullet point in a list
	N := 3
	prevNisListItem := false
	for i := len(listItemRecord) - 1; i > (len(listItemRecord) - N); i-- {
		if i >= 0 && listItemRecord[i] {
			prevNisListItem = true
		}
	}

	if leadingSpace == "    " && !strings.HasPrefix(rest, "*") && !strings.HasPrefix(rest, "-") && !prevNisListItem {
		// Four leading spaces means a quoted line
		// Also assume it's not a quote if it starts with "*" or "-"
		return e.CodeColor.Get(line), true, false
	}

	// An image (or a link to a single image) on a single line
	if (strings.HasPrefix(rest, "[!") || strings.HasPrefix(rest, "!")) && strings.HasSuffix(rest, ")") {
		return e.ImageColor.Get(line), true, false
	}

	// A link on a single line
	if strings.HasPrefix(rest, "[") && strings.HasSuffix(rest, ")") && strings.Count(rest, "[") == 1 {
		return e.LinkColor.Get(line), true, false
	}

	// A line with HTML tags that may link to an image, or just be an "a href" link
	if strings.HasPrefix(rest, "<") && strings.HasSuffix(rest, ">") {
		if strings.Contains(rest, "<img ") && strings.Contains(rest, "://") {
			// string includes "<img" and "://"
			return e.ImageColor.Get(line), true, false
		}
		if strings.Contains(rest, "<a ") && strings.Contains(rest, "://") {
			// string includes "<a" and "://"
			return e.LinkColor.Get(line), true, false
		}
		if strings.Count(rest, "<") == strings.Count(rest, ">") {
			// A list with HTML tags, matched evenly?
			return e.LinkColor.Get(line), true, false
		}
		// Maybe HTML tags. Maybe matched unevenly.
		return e.QuoteColor.Get(line), true, false
	}

	// A header line
	if strings.HasPrefix(rest, "---") || strings.HasPrefix(rest, "===") {
		return e.HeaderTextColor.Get(line), true, false
	}

	// HTML comments
	if strings.HasPrefix(rest, "<!--") || strings.HasPrefix(rest, "-->") {
		return e.CommentColor.Get(line), true, false
	}

	// A line with just a quote mark
	if strings.TrimSpace(rest) == ">" {
		return e.QuoteColor.Get(line), true, false
	}

	// A quote with something that follows
	if pos := strings.Index(rest, "> "); pos >= 0 && pos < 5 {
		words := strings.Fields(rest)
		if len(words) >= 2 {
			return e.QuoteColor.Get(words[0]) + " " + e.QuoteTextColor.Get(strings.Join(words[1:], " ")), true, false
		}
	}

	// HTML
	if strings.HasPrefix(rest, "<") || strings.HasPrefix(rest, ">") {
		return e.HTMLColor.Get(line), true, false
	}

	// Table
	if strings.HasPrefix(rest, "|") || strings.HasSuffix(rest, "|") {
		if strings.HasPrefix(line, "|-") {
			return e.TableColor.String() + line + e.TableBackground.String(), true, false
		}
		return strings.ReplaceAll(line, "|", e.TableColor.String()+"|"+e.TableBackground.String()), true, false
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

	// A list item that is a link on a single line, possibly with some text after the link
	if bracketPos := strings.Index(rest, "["); (firstWord == "-" || firstWord == "*") && bracketPos < 4 && strings.Count(rest, "](") >= 1 && strings.Count(rest, ")") >= 1 && strings.Index(rest, "]") != bracketPos+2 {
		// First comes the leading space and rest[:bracketPos], then comes "[" and then....
		twoParts := strings.SplitN(rest[bracketPos+1:], "](", 2)
		if len(twoParts) == 2 {
			lastParts := strings.SplitN(twoParts[1], ")", 2)
			if len(lastParts) == 2 {
				bulletColor := e.ListBulletColor
				labelColor := e.CodeColor
				linkColor := e.CommentColor
				bracketColor := e.Foreground
				// Then comes twoParts[0] and "](" and twoParts[1]
				return leadingSpace + bulletColor.Get(rest[:bracketPos]) + bracketColor.Get("[") + labelColor.Get(twoParts[0]) + bracketColor.Get("]") + e.CommentColor.Get("(") + linkColor.Get(lastParts[0]) + e.CommentColor.Get(")") + e.ListTextColor.Get(lastParts[1]), true, false
			}
		}
	}

	if consistsOf(firstWord, '#', []rune{'.', ' '}) {
		if strings.HasSuffix(lastWord, "#") && strings.Contains(rest, " ") {
			centerLen := len(rest) - (len(firstWord) + len(lastWord))
			if centerLen > 0 {
				centerText := rest[len(firstWord) : len(rest)-len(lastWord)]
				return leadingSpace + e.HeaderBulletColor.Get(firstWord) + e.HeaderTextColor.Get(centerText) + e.HeaderBulletColor.Get(lastWord), true, false
			}
			return leadingSpace + e.HeaderBulletColor.Get(rest), true, false
		} else if len(words) > 1 {
			return leadingSpace + e.HeaderBulletColor.Get(firstWord) + " " + e.HeaderTextColor.Get(emphasis(quotedWordReplace(line[dataPos+len(firstWord)+1:], '`', e.HeaderTextColor, e.CodeColor), e.HeaderTextColor, e.ItalicsColor, e.BoldColor, e.StrikeColor)), true, false // TODO: `
		}
		return leadingSpace + e.HeaderTextColor.Get(rest), true, false
	}

	if isListItem(line) {
		if strings.HasPrefix(rest, "- [ ] ") || strings.HasPrefix(rest, "- [x] ") || strings.HasPrefix(rest, "- [X] ") {
			return leadingSpace + e.ListBulletColor.Get(rest[:1]) + " " + e.CheckboxColor.Get(rest[2:3]) + e.XColor.Get(rest[3:4]) + e.CheckboxColor.Get(rest[4:5]) + " " + emphasis(quotedWordReplace(line[dataPos+6:], '`', e.ListTextColor, e.ListCodeColor), e.ListTextColor, e.ItalicsColor, e.BoldColor, e.StrikeColor), true, false
		}
		if len(words) > 1 {
			return leadingSpace + e.ListBulletColor.Get(firstWord) + " " + emphasis(quotedWordReplace(line[dataPos+len(firstWord)+1:], '`', e.ListTextColor, e.ListCodeColor), e.ListTextColor, e.ItalicsColor, e.BoldColor, e.StrikeColor), true, false
		}
		return leadingSpace + e.ListTextColor.Get(rest), true, false
	}

	// Leading hash without a space afterwards?
	if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "# ") {
		return e.MenuArrowColor.Get(line), true, false
	}

	if prevNisListItem {
		*inListItem = true
	}

	// A completely regular line of text that is also the continuation of a list item
	if *inListItem {
		return emphasis(quotedWordReplace(line, '`', e.ListTextColor, e.ListCodeColor), e.ListTextColor, e.ItalicsColor, e.BoldColor, e.StrikeColor), true, false
	}

	// A completely regular line of text
	return emphasis(quotedWordReplace(line, '`', e.MarkdownTextColor, e.CodeColor), e.MarkdownTextColor, e.ItalicsColor, e.BoldColor, e.StrikeColor), true, false
}
