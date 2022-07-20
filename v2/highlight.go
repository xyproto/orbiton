package main

import (
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/xyproto/mode"
	"github.com/xyproto/stringpainter"
	"github.com/xyproto/syntax"
	"github.com/xyproto/textoutput"
	"github.com/xyproto/vt100"
)

const (
	blankRune              = ' '
	controlRuneReplacement = '¿' // for displaying control sequence characters. Could also use: �
)

var (
	tout      = textoutput.NewTextOutput(true, true)
	resizeMut sync.RWMutex // used when the terminal is resized
)

// WriteLines will draw editor lines from "fromline" to and up to "toline" to the canvas, at cx, cy
func (e *Editor) WriteLines(c *vt100.Canvas, fromline, toline LineIndex, cx, cy int) {

	resizeMut.Lock()
	defer resizeMut.Unlock()

	// Convert the background color to a background color code
	bg := e.Background.Background()

	tabString := strings.Repeat(" ", e.tabsSpaces.PerTab)
	cw := c.Width()
	if fromline >= toline {
		return //errors.New("fromline >= toline in WriteLines")
	}
	numLinesToDraw := toline - fromline // Number of lines available on the canvas for drawing
	offsetY := fromline

	inCodeBlock := false // used when highlighting Doc, Markdown or Python

	//logf("numlines: %d offsetY %d\n", numlines, offsetY)

	switch e.mode {
	// If in Markdown mode, figure out the current state of block quotes
	case mode.Doc, mode.Markdown:
		// Figure out if "fromline" is within a markdown code block or not
		for i := LineIndex(0); i < fromline; i++ {
			// Check if the untrimmed line starts with ~~~ or ```
			line := e.Line(i)
			if strings.HasPrefix(line, "~~~") || strings.HasPrefix(line, "```") {
				// Toggle the flag for if we're in a code block or not
				inCodeBlock = !inCodeBlock
			}
			// Note that code blocks in Markdown normally starts or ends with ~~~ or ``` as
			// the first thing happening on the line, so it's not important to check the trimmed line.
		}
	case mode.Python:
		// Figure out if "fromline" is within a markdown code block or not
		for i := LineIndex(0); i < fromline; i++ {
			// Check if the trimmed line starts with """ or '''
			line := e.Line(i)
			trimmedLine := strings.TrimSpace(line)

			if strings.HasPrefix(trimmedLine, "\"\"\"") && strings.HasSuffix(trimmedLine, "\"\"\"") {
				inCodeBlock = false
			} else if strings.HasPrefix(trimmedLine, "'''") && strings.HasSuffix(trimmedLine, "'''") {
				inCodeBlock = false
			} else if strings.HasPrefix(trimmedLine, "\"\"\"") || strings.HasPrefix(trimmedLine, "'''") {
				// Toggle the flag for if we're in a code block or not
				inCodeBlock = !inCodeBlock
			} else if strings.HasSuffix(trimmedLine, "\"\"\"") || strings.HasSuffix(trimmedLine, "'''") {
				// Toggle the flag for if we're in a code block or not
				inCodeBlock = false
			}

		}
	}

	var (
		trimmedLine             string
		singleLineCommentMarker = e.SingleLineCommentMarker()
		ignoreSingleQuotes      = (e.mode == mode.Lisp) || (e.mode == mode.Clojure)
	)

	q, err := NewQuoteState(singleLineCommentMarker, e.mode, ignoreSingleQuotes)
	if err != nil {
		return // err
	}

	// First loop from 0 up to to offset to figure out if we are already in a multiLine comment or a multiLine string at the current line
	for i := LineIndex(0); i < offsetY; i++ {
		trimmedLine = strings.TrimSpace(e.Line(LineIndex(i)))

		// Special case for ViM
		if e.mode == mode.Vim && strings.HasPrefix(trimmedLine, "\"") {
			q.singleLineComment = true
			q.startedMultiLineString = false
			q.stoppedMultiLineComment = false
			q.backtick = 0
			q.doubleQuote = 0
			q.singleQuote = 0
			continue
		}

		// Have a trimmed line. Want to know: the current state of which quotes, comments or strings we are in.
		// Solution, have a state struct!
		q.Process(trimmedLine)
	}
	// q should now contain the current quote state

	var (
		lineRuneCount   int
		lineStringCount uint
		line            string
		listItemRecord  []bool
		inListItem      bool
		screenLine      string
		programName     string
	)

	// Loop from 0 to numlines (used as y+offset in the loop) to draw the text
	for y := LineIndex(0); y < numLinesToDraw; y++ {
		lineRuneCount = 0   // per line rune counter, for drawing spaces afterwards (does not handle wide runes)
		lineStringCount = 0 // per line string counter, for drawing spaces afterwards (handles wide runes)

		line = e.Line(LineIndex(y + offsetY))

		line = strings.TrimRightFunc(line, unicode.IsSpace)

		// already trimmed right, just trim left
		trimmedLine = strings.TrimLeftFunc(line, unicode.IsSpace)

		// expand tabs
		line = strings.Replace(line, "\t", tabString, -1)

		if e.syntaxHighlight && !envNoColor {
			// Output a syntax highlighted line. Escape any tags in the input line.
			// textWithTags must be unescaped if there is not an error.
			if textWithTags, err := syntax.AsText([]byte(Escape(line)), e.mode); err != nil {
				// Only output the line up to the width of the canvas
				screenLine = e.ChopLine(line, int(cw))
				// TODO: Check if just "fmt.Print" works here, for several terminal emulators
				fmt.Println(screenLine)
				lineRuneCount += utf8.RuneCountInString(screenLine)
				//lineRuneCount += uint(len([]rune(screenLine)))
				lineStringCount += uint(len(screenLine))
			} else {
				var (
					// Color and unescape
					coloredString    string
					doneHighlighting = true
				)
				switch e.mode {
				case mode.Email, mode.Git:
					coloredString = e.gitHighlight(line)
				case mode.ManPage:
					if y == 0 {
						// Get the first word that consists only of letters, and use that as the man page program name
						fields := strings.FieldsFunc(trimmedLine, func(c rune) bool { return !unicode.IsLetter(c) })
						if len(fields) > 0 {
							programName = fields[0]
						}
					}
					cs := e.manPageHighlight(line, programName, y == 0, y+1 == numLinesToDraw)
					coloredString = cs
				case mode.Doc, mode.Markdown:
					if highlighted, ok, codeBlockFound := e.markdownHighlight(line, inCodeBlock, listItemRecord, &inListItem); ok {
						coloredString = highlighted
						if codeBlockFound {
							inCodeBlock = !inCodeBlock
						}
					} else {
						// Syntax highlight the line if it's not picked up by the markdownHighlight function
						coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					}
					// If this is a list item, store true in "prevLineIsListItem"
					listItemRecord = append(listItemRecord, isListItem(line))
				case mode.Python:
					trimmedLine = strings.TrimSpace(line)
					foundDocstringMarker := false

					if strings.HasPrefix(trimmedLine, "\"\"\"") && strings.HasSuffix(trimmedLine, "\"\"\"") {
						inCodeBlock = false
						foundDocstringMarker = true
					} else if strings.HasPrefix(trimmedLine, "'''") && strings.HasSuffix(trimmedLine, "'''") {
						inCodeBlock = false
						foundDocstringMarker = true
					} else if strings.HasPrefix(trimmedLine, "\"\"\"") || strings.HasPrefix(trimmedLine, "'''") {
						// Toggle the flag for if we're in a code block or not
						inCodeBlock = !inCodeBlock
						foundDocstringMarker = true
					} else if strings.HasSuffix(trimmedLine, "\"\"\"") || strings.HasSuffix(trimmedLine, "'''") {
						// Toggle the flag for if we're in a code block or not
						inCodeBlock = false
						foundDocstringMarker = true
					}

					if inCodeBlock || foundDocstringMarker {
						// Purple
						coloredString = UnEscape(e.MultiLineString.Start(line))
					} else {
						// Regular highlight
						coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					}
				case mode.Config, mode.CMake, mode.JSON:
					if !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && (strings.Contains(trimmedLine, "/*") || strings.HasSuffix(trimmedLine, "*/")) {
						// No highlight
						coloredString = line
					} else if strings.HasPrefix(trimmedLine, "> ") {
						// If there is a } underneath and typing }, don't dedent, keep it at the same level!
						coloredString = UnEscape(e.MultiLineString.Start(trimmedLine))
					} else {
						// Regular highlight + highlight yes and no in blue when using the default color scheme
						// TODO: Modify (and rewrite) the syntax package instead.
						coloredString = UnEscape(tout.DarkTags(strings.Replace(strings.Replace(string(textWithTags), "<lightgreen>yes<", "<lightyellow>yes<", -1), "<lightred>no<", "<lightyellow>no<", -1)))
					}
				case mode.Zig:
					trimmedLine = strings.TrimSpace(line)
					// Handle doc comments (starting with ///)
					// and multiline strings (starting with \\)
					if strings.HasPrefix(trimmedLine, "///") || strings.HasPrefix(trimmedLine, `\\`) {
						coloredString = UnEscape(e.MultiLineString.Start(trimmedLine))
					} else {
						// Regular highlight
						coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					}
				case mode.Bat:
					trimmedLine = strings.TrimSpace(line)
					// In DOS batch files, ":" can be used both for labels and for single-line comments
					if strings.HasPrefix(trimmedLine, "@rem") || strings.HasPrefix(trimmedLine, "rem") || strings.HasPrefix(trimmedLine, ":") {
						// Handle single line comments
						coloredString = UnEscape(e.MultiLineComment.Start(line))
					} else {
						// Regular highlight
						coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					}
				case mode.Ada, mode.Agda, mode.Haskell, mode.Lua, mode.SQL, mode.Teal, mode.Terra: // not for OCaml and Standard ML
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "--") {
						// Handle single line comments
						coloredString = UnEscape(e.MultiLineComment.Start(line))
					} else {
						// Regular highlight
						coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					}
				case mode.Amber:
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "!!") {
						// Handle single line comments
						coloredString = UnEscape(e.MultiLineComment.Start(line))
					} else {
						// Regular highlight
						coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					}
				case mode.StandardML, mode.OCaml:
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "(*") && strings.HasSuffix(trimmedLine, "*)") {
						coloredString = UnEscape(e.MultiLineComment.Start(line))
					} else {
						doneHighlighting = false
						break
						//coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					}
				case mode.Nroff:
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, `.\"`) {
						// Handle single line comments
						coloredString = UnEscape(e.MultiLineComment.Start(line))
					} else {
						// Regular highlight
						coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					}
				case mode.Log:
					coloredString = stringpainter.Colorize(line)
				case mode.Lisp, mode.Clojure:
					q.singleQuote = 0
					// Special case for Lisp single-line comments
					trimmedLine = strings.TrimSpace(line)
					if doubleSemiCount := strings.Count(trimmedLine, ";;"); doubleSemiCount > 0 {
						// Color the line with the same color as for multiLine comments
						if strings.HasPrefix(trimmedLine, ";") {
							coloredString = UnEscape(e.MultiLineComment.Start(line))
						} else if doubleSemiCount == 1 {

							parts := strings.SplitN(line, ";;", 2)
							if newTextWithTags, err := syntax.AsText([]byte(Escape(parts[0])), e.mode); err != nil {
								coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
							} else {
								coloredString = UnEscape(tout.DarkTags(string(newTextWithTags)) + e.MultiLineComment.Get(";;"+parts[1]))
							}

						} else if strings.Count(trimmedLine, ";") == 1 {

							parts := strings.SplitN(line, ";", 2)
							if newTextWithTags, err := syntax.AsText([]byte(Escape(parts[0])), e.mode); err != nil {
								coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
							} else {
								coloredString = UnEscape(tout.DarkTags(string(newTextWithTags)) + e.MultiLineComment.Start(";"+parts[1]))
							}

						}
						doneHighlighting = true
						break
					}
					doneHighlighting = false
				case mode.Vim:
					// Special case for ViM single-line comments
					trimmedLine = strings.TrimSpace(line)
					if strings.Count(trimmedLine, "\"") == 1 {
						// Color the line with the same color as for multiLine comments
						if strings.HasPrefix(trimmedLine, "\"") {
							coloredString = UnEscape(e.MultiLineComment.Start(line))
						} else {
							parts := strings.SplitN(line, "\"", 2)
							if newTextWithTags, err := syntax.AsText([]byte(Escape(parts[0])), e.mode); err != nil {
								coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
							} else {
								coloredString = UnEscape(tout.DarkTags(string(newTextWithTags)) + e.MultiLineComment.Start("\""+parts[1]))
							}
						}
						break
					}
					fallthrough
				default:
					doneHighlighting = false
				}

				if !doneHighlighting {

					// C, C++, Go, Rust etc

					trimmedLine = strings.TrimSpace(line)
					q.Process(trimmedLine)

					//logf("%s -[ %d ]-->\n\t%s\n", trimmedLine, addedPar, q.String())

					switch {
					case e.mode == mode.Python && q.startedMultiLineString:
						// Python docstring
						coloredString = UnEscape(e.MultiLineString.Get(line))
					case !q.multiLineComment && (strings.HasPrefix(trimmedLine, "#if") || strings.HasPrefix(trimmedLine, "#else") || strings.HasPrefix(trimmedLine, "#elseif") || strings.HasPrefix(trimmedLine, "#endif") || strings.HasPrefix(trimmedLine, "#define") || strings.HasPrefix(trimmedLine, "#pragma")):
						coloredString = UnEscape(e.MultiLineString.Get(line))
					case e.mode != mode.Shell && e.mode != mode.Makefile && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.HasSuffix(trimmedLine, "*/") && !strings.Contains(trimmedLine, "/*"):
						coloredString = UnEscape(e.MultiLineComment.Get(line))
					case (e.mode == mode.StandardML || e.mode == mode.OCaml) && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.HasSuffix(trimmedLine, "*)") && !strings.Contains(trimmedLine, "(*"):
						coloredString = UnEscape(e.MultiLineComment.Get(line))
					case e.mode != mode.Shell && e.mode != mode.Makefile && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.LastIndex(trimmedLine, "/*") > strings.LastIndex(trimmedLine, "*/"):
						coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					case (e.mode == mode.StandardML || e.mode == mode.OCaml) && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.LastIndex(trimmedLine, "(*") > strings.LastIndex(trimmedLine, "*)"):
						coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					case q.containsMultiLineComments:
						coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					case e.mode != mode.Shell && e.mode != mode.Makefile && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && (q.multiLineComment || q.stoppedMultiLineComment) && !strings.Contains(line, "\"/*") && !strings.Contains(line, "*/\"") && !strings.Contains(line, "\"(*") && !strings.Contains(line, "*)\"") && !strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "//"):
						// In the middle of a multi-line comment
						coloredString = UnEscape(e.MultiLineComment.Get(line))
					case q.singleLineComment || q.stoppedMultiLineComment:
						// A single line comment (the syntax module did the highlighting)
						coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					case !q.startedMultiLineString && q.backtick > 0:
						// A multi-line string
						coloredString = UnEscape(e.MultiLineString.Get(line))
					case (e.mode != mode.HTML && e.mode != mode.XML) && strings.Contains(line, "->"):
						// NOTE that if two color tags are placed after each other, they may cause blinking. Remember to turn <off> each color.
						coloredString = UnEscape(tout.DarkTags(arrowReplace(string(textWithTags))))
					default:
						// Regular code
						coloredString = UnEscape(tout.DarkTags(string(textWithTags)))
					}

					// Take an extra pass on coloring the -> arrow, even if it's in a comment
					if (e.mode != mode.HTML && e.mode != mode.XML) && strings.Contains(line, "->") {
						arrowIndex := strings.Index(line, "->")
						if i := strings.Index(line, "//"); i != -1 && i < arrowIndex {
							// arrow is after comment marker, do nothing
						} else if i := strings.Index(line, "/*"); i != -1 && i < arrowIndex {
							// arrow is after comment marker, do nothing
						} else if i := strings.Index(line, "(*"); i != -1 && i < arrowIndex {
							// arrow is after comment marker, do nothing
						} else {
							// arrow is before comment marker, color the arrow
							coloredString = UnEscape(tout.DarkTags(arrowReplace(string(textWithTags))))
						}
					}
				}

				// Extract a slice of runes and color attributes
				runesAndAttributes := tout.Extract(coloredString)

				// Also handle things like ZALGO HE COMES which contains unicode "mark" runes
				// TODO: Also handle all languages in v2/test/problematic.desktop
				for k, v := range runesAndAttributes {
					if unicode.IsMark(v.R) {
						// Replace the "unprintable" (for a terminal emulator) characters
						runesAndAttributes[k].R = controlRuneReplacement
					}
				}

				// If e.rainbowParenthesis is true and we're not in a comment or a string, enable rainbow parenthesis
				// TODO: Figure out why some parantheses in some comments are still highlighted
				if e.mode != mode.Git && e.rainbowParenthesis && q.None() && !q.singleLineComment && !q.stoppedMultiLineComment {
					thisLineParCount, thisLineBraCount := q.ParBraCount(trimmedLine)
					parCountBeforeThisLine := q.parCount - thisLineParCount
					braCountBeforeThisLine := q.braCount - thisLineBraCount
					if e.rainbowParen(&parCountBeforeThisLine, &braCountBeforeThisLine, &runesAndAttributes, singleLineCommentMarker, ignoreSingleQuotes) == errUnmatchedParenthesis {
						// Don't mark the rest of the parenthesis as wrong, even though this one is
						q.parCount = 0
						q.braCount = 0
					}
				}

				// Search term highlighting
				searchTermRunes := []rune(e.searchTerm)
				matchForAnotherN := 0

				// Output a line with the chars (Rune + AttributeColor)
				skipX := e.pos.offsetX
				for runeIndex, ra := range runesAndAttributes {
					if skipX > 0 {
						skipX--
						continue
					}
					letter := ra.R
					fg := ra.A
					if letter == ' ' {
						fg = e.Foreground
					}
					if matchForAnotherN > 0 {
						// Coloring an already found match
						fg = e.SearchHighlight
						matchForAnotherN--
					} else if len(e.searchTerm) > 0 && letter == searchTermRunes[0] {
						// Potential search highlight match
						length := utf8.RuneCountInString(e.searchTerm)
						counter := 0
						match := true
						for i := runeIndex; i < (runeIndex + length); i++ {
							if i >= len(runesAndAttributes) {
								match = false
								break
							}
							ra2 := runesAndAttributes[i]
							if ra2.R != []rune(e.searchTerm)[counter] {
								// mismatch, not a hit
								match = false
								break
							}
							counter++
						}
						if match {
							fg = e.SearchHighlight
							matchForAnotherN = length - 1
						}
					}
					if letter == '\t' {
						c.Write(uint(cx+lineRuneCount), uint(cy)+uint(y), fg, e.Background, tabString)
						lineRuneCount += e.tabsSpaces.PerTab
						lineStringCount += uint(e.tabsSpaces.PerTab)
					} else {
						if unicode.IsControl(letter) {
							letter = controlRuneReplacement
						}
						tx := uint(cx + lineRuneCount)
						ty := uint(cy) + uint(y)
						if tx < cw {
							c.WriteRuneB(tx, ty, fg, bg, letter)
							lineRuneCount++                              // 1 rune
							lineStringCount += uint(len(string(letter))) // 1 rune, expanded
						}
					}
				}
			}
		} else { // no syntax highlighting
			// Man pages are special
			if e.mode == mode.ManPage {
				line = handleManPageEscape(line)
			}
			// Output a regular line, scrolled to the current e.pos.offsetX
			screenLine = e.ChopLine(line, int(cw))
			c.Write(uint(cx+lineRuneCount), uint(cy)+uint(y), e.Foreground, e.Background, screenLine)
			lineRuneCount += utf8.RuneCountInString(screenLine) // rune count
			//lineRuneCount += uint(len([]rune(screenLine))) // rune count
			lineStringCount += uint(len(screenLine)) // string length, not rune length
		}

		// Fill the rest of the line on the canvas with "blanks"
		// TODO: This may draw the wrong number of blanks, since lineRuneCount should really be the number of visible glyphs at this point
		yp := uint(cy) + uint(y)
		xp := uint(cx + lineRuneCount)
		c.WriteRunesB(xp, yp, e.Foreground, bg, ' ', cw-uint(lineRuneCount))

	}
}

// Syntax highlight pointer arrows in C and C++
func arrowReplace(s string) string {
	arrowColor := syntax.DefaultTextConfig.Class
	fieldColor := syntax.DefaultTextConfig.Protected
	s = strings.Replace(s, ">-<", "><off><"+arrowColor+">-<", -1)
	s = strings.Replace(s, ">"+Escape(">"), "><off><"+arrowColor+">"+Escape(">")+"<off><"+fieldColor+">", -1)
	return s
}
