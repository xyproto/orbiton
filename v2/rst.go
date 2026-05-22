package main

import (
	"strings"
	"unicode"

	"github.com/xyproto/vt"
)

// rstAdornmentChars is the set of characters that RST uses for section title adornments
const rstAdornmentChars = "=-~^\"'`#*+._"

// rstAdornmentLine checks if a line consists entirely of one repeated adornment character
// and is at least 2 characters long (the minimum for an RST underline/overline)
func rstAdornmentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 2 {
		return false
	}
	ch := trimmed[0]
	if !strings.ContainsRune(rstAdornmentChars, rune(ch)) {
		return false
	}
	for i := 1; i < len(trimmed); i++ {
		if trimmed[i] != ch {
			return false
		}
	}
	return true
}

// rstDirective checks if a line is an RST directive (e.g., ".. code-block:: python")
func rstDirective(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, ".. ") && strings.Contains(trimmed, "::")
}

// rstComment checks if a line is an RST comment (starts with ".." but is not a directive)
func rstComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "..") {
		return false
	}
	// Bare ".." or ".. " followed by text without "::" is a comment
	if trimmed == ".." {
		return true
	}
	if strings.HasPrefix(trimmed, ".. ") && !strings.Contains(trimmed, "::") {
		return true
	}
	return false
}

// rstFieldList checks if a line is an RST field list entry (e.g., ":param name: description")
func rstFieldList(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, ":") || len(trimmed) < 3 {
		return false
	}
	// Look for the closing colon (must have at least one char between the colons)
	closeIdx := strings.Index(trimmed[1:], ":")
	return closeIdx > 0
}

// rstListItem checks if a line is an RST list item
func rstListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return false
	}
	firstWord := fields[0]

	// Bullet list items
	switch firstWord {
	case "*", "-", "+", "\u2022", "\u2023", "\u25e6":
		return true
	}

	// Numbered list items: "1.", "2.", "#."
	if strings.HasSuffix(firstWord, ".") {
		prefix := firstWord[:len(firstWord)-1]
		if prefix == "#" {
			return true
		}
		allDigits := true
		for _, r := range prefix {
			if !unicode.IsDigit(r) {
				allDigits = false
				break
			}
		}
		if allDigits && len(prefix) > 0 {
			return true
		}
	}

	// Numbered with parens: "1)", "(1)", "#)"
	if strings.HasSuffix(firstWord, ")") {
		inner := firstWord[:len(firstWord)-1]
		inner = strings.TrimPrefix(inner, "(")
		if inner == "#" {
			return true
		}
		allDigits := true
		for _, r := range inner {
			if !unicode.IsDigit(r) {
				allDigits = false
				break
			}
		}
		if allDigits && len(inner) > 0 {
			return true
		}
	}

	return false
}

// rstInlineMarkup applies RST inline markup highlighting to a line.
// RST uses *italic*, **bold**, “code“, and `interpreted text`.
func rstInlineMarkup(line string, textColor, italicsColor, boldColor, codeColor vt.AttributeColor) string {
	result := line
	// Bold: **text**
	if !withinBackticks(line, "**") {
		result = style(result, "**", textColor, boldColor)
	}
	// Italic: *text* (only if no ** present to avoid conflict)
	if !strings.Contains(line, "**") && !withinBackticks(line, "*") {
		result = style(result, "*", textColor, italicsColor)
	}
	// Inline code: ``text``
	if strings.Contains(result, "``") && strings.Count(result, "``")%2 == 0 {
		result = style(result, "``", textColor, codeColor)
	}
	return result
}

// rstHighlight returns a VT100 colored line, a bool that is true if highlighting
// was applied, and a bool that is true if it's the start or stop of a code block.
// The code block state is tracked by the caller across lines. It is toggled on when
// a line ends with "::" or a directive is found, and toggled off when a non-indented
// non-blank line appears while in code block mode.
func (e *Editor) rstHighlight(line string, inCodeBlock bool, prevLine, nextLine string) (string, bool, bool) {
	dataPos := 0
	for i, r := range line {
		if unicode.IsSpace(r) {
			dataPos = i + 1
		} else {
			break
		}
	}

	leadingSpace := line[:dataPos]
	rest := line[dataPos:]

	// Starting or ending a fenced code block (some RST parsers support ~~~ and ```)
	if strings.HasPrefix(rest, "~~~") || strings.HasPrefix(rest, "```") {
		return e.CodeBlockColor.Get(line), true, true
	}

	// In code block mode (after a :: line or directive), indented and blank lines are code
	if inCodeBlock {
		if rest == "" {
			return "", false, false
		}
		if len(leadingSpace) >= 3 {
			return e.CodeBlockColor.Get(line), true, false
		}
		// Non-indented non-blank line: exit code block mode.
		// Toggle (third return = true) and fall through to highlight normally.
		return e.rstHighlightNormal(line, leadingSpace, rest, dataPos, prevLine, nextLine, true)
	}

	return e.rstHighlightNormal(line, leadingSpace, rest, dataPos, prevLine, nextLine, false)
}

// rstHighlightNormal handles normal (non-code-block) RST highlighting.
// The toggleCodeBlock parameter signals the caller to toggle code block state.
func (e *Editor) rstHighlightNormal(line, leadingSpace, rest string, dataPos int, prevLine, nextLine string, toggleCodeBlock bool) (string, bool, bool) {
	// Directive line (e.g., ".. code-block:: python", ".. image:: path")
	if rstDirective(line) {
		trimmed := strings.TrimSpace(line)
		// Split at "::" to color directive name and arguments differently
		parts := strings.SplitN(trimmed, "::", 2)
		directivePart := parts[0] + "::"
		argPart := ""
		if len(parts) == 2 {
			argPart = parts[1]
		}
		// Directives introduce indented content blocks.
		// If we are already exiting a code block (toggleCodeBlock=true), the exit
		// and re-enter cancel each other out, so return false (no net toggle).
		return leadingSpace + e.QuoteColor.Get(directivePart) + e.ListTextColor.Get(argPart), true, !toggleCodeBlock
	}

	// Comment line (starts with ".." but not a directive)
	if rstComment(line) {
		// Comments also introduce indented continuation blocks.
		// Same cancel-out logic as directives.
		return e.CommentColor.Get(line), true, !toggleCodeBlock
	}

	// Section title adornment (underline/overline)
	if rstAdornmentLine(line) {
		return e.HeaderBulletColor.Get(line), true, toggleCodeBlock
	}

	// Section title: the line before or after an adornment line
	if rstAdornmentLine(nextLine) || rstAdornmentLine(prevLine) {
		if rest != "" && !rstAdornmentLine(line) {
			return leadingSpace + e.HeaderTextColor.Get(rest), true, toggleCodeBlock
		}
	}

	// Field list entry (e.g., ":param: value")
	if rstFieldList(line) {
		trimmed := strings.TrimSpace(line)
		// Find the field name between the first pair of colons
		closeIdx := strings.Index(trimmed[1:], ":")
		if closeIdx > 0 {
			fieldName := trimmed[:closeIdx+2]
			fieldValue := trimmed[closeIdx+2:]
			return leadingSpace + e.QuoteColor.Get(fieldName) + e.MarkdownTextColor.Get(fieldValue), true, toggleCodeBlock
		}
	}

	// Table lines (grid tables use +---+---+ and |   |   |, simple tables use === and ---)
	// Check tables before block quotes to avoid "|cell|" being misidentified
	if strings.HasPrefix(rest, "+") && strings.HasSuffix(rest, "+") && (strings.Contains(rest, "-") || strings.Contains(rest, "=")) {
		return e.TableColor.String() + line + e.TableBackground.String(), true, toggleCodeBlock
	}
	if strings.HasPrefix(rest, "|") && strings.HasSuffix(rest, "|") && strings.Count(rest, "|") >= 2 {
		return strings.ReplaceAll(line, "|", e.TableColor.String()+"|"+e.TableBackground.String()), true, toggleCodeBlock
	}

	// Block quote marker: line starting with "| " (line block) but not a table row
	if strings.HasPrefix(rest, "| ") || rest == "|" {
		return e.QuoteColor.Get(line), true, toggleCodeBlock
	}

	// RST list items
	if rstListItem(line) {
		words := strings.Fields(rest)
		if len(words) > 1 {
			firstWord := words[0]
			return leadingSpace + e.ListBulletColor.Get(firstWord) + " " + rstInlineMarkup(quotedWordReplace(line[dataPos+len(firstWord)+1:], '`', e.ListTextColor, e.ListCodeColor), e.ListTextColor, e.ItalicsColor, e.BoldColor, e.ListCodeColor), true, toggleCodeBlock
		}
		return leadingSpace + e.ListTextColor.Get(rest), true, toggleCodeBlock
	}

	// Links and references: .. _name: url
	if strings.HasPrefix(rest, ".. _") && strings.Contains(rest, ":") {
		return e.LinkColor.Get(line), true, toggleCodeBlock
	}

	// Substitution definition: .. |name| directive:: arg
	if strings.HasPrefix(rest, ".. |") && strings.Contains(rest, "|") {
		return e.QuoteColor.Get(line), true, toggleCodeBlock
	}

	// Doctest blocks
	if strings.HasPrefix(rest, ">>> ") {
		return e.CodeColor.Get(line), true, toggleCodeBlock
	}

	// A line ending with "::" introduces a literal block (code block).
	// Same cancel-out logic: if already exiting, exit+enter = no net toggle.
	if strings.HasSuffix(rest, "::") {
		colored := rstInlineMarkup(quotedWordReplace(line, '`', e.MarkdownTextColor, e.CodeColor), e.MarkdownTextColor, e.ItalicsColor, e.BoldColor, e.CodeColor)
		return colored, true, !toggleCodeBlock
	}

	// Indented content (at least 3 spaces): block quote when not in code block mode
	if len(leadingSpace) >= 3 {
		return e.QuoteTextColor.Get(line), true, toggleCodeBlock
	}

	// Empty line
	if strings.TrimSpace(rest) == "" {
		return "", false, toggleCodeBlock
	}

	// Regular text with inline markup
	return rstInlineMarkup(quotedWordReplace(line, '`', e.MarkdownTextColor, e.CodeColor), e.MarkdownTextColor, e.ItalicsColor, e.BoldColor, e.CodeColor), true, toggleCodeBlock
}
