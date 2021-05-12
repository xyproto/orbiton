package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode"

	"github.com/xyproto/syntax"
	"github.com/xyproto/textoutput"
	"github.com/xyproto/vt100"
)

const controlRuneReplacement = '¿' // for displaying control sequence characters. Could also use: �

var writeLinesMutex sync.RWMutex

// WriteLines will draw editor lines from "fromline" to and up to "toline" to the canvas, at cx, cy
func (e *Editor) WriteLines(c *vt100.Canvas, fromline, toline LineIndex, cx, cy int) error {

	// Only one call to WriteLines at the time, thank you
	writeLinesMutex.Lock()
	defer writeLinesMutex.Unlock()

	// Convert the background color to a background color code
	bg := e.bg.Background()

	o := textoutput.NewTextOutput(true, true)
	tabString := strings.Repeat(" ", e.tabs.spacesPerTab)
	w := c.Width()
	//h := c.Height()
	if fromline >= toline {
		return errors.New("fromline >= toline in WriteLines")
	}
	numlines := toline - fromline // Number of lines available on the canvas for drawing
	offsetY := fromline
	inCodeBlock := false // used when highlighting Markdown or Python

	expandedRunes := false // used for detecting wide unicode symbols

	//logf("numlines: %d offsetY %d\n", numlines, offsetY)

	// If in Markdown mode, figure out the current state of block quotes
	if e.mode == modeMarkdown {
		// Figure out if "fromline" is within a markdown code block or not
		for i := LineIndex(0); i < fromline; i++ {
			// Check if the untrimmed line starts with ~~~ or ```
			contents := e.Line(i)
			if strings.HasPrefix(contents, "~~~") || strings.HasPrefix(contents, "```") {
				// Toggle the flag for if we're in a code block or not
				inCodeBlock = !inCodeBlock
			}
		}
	} else if e.mode == modePython {
		// Figure out if "fromline" is within a markdown code block or not
		for i := LineIndex(0); i < fromline; i++ {
			// Check if the untrimmed line starts with """ or '''
			contents := e.Line(i)
			if strings.HasPrefix(contents, "\"\"\"") || strings.HasPrefix(contents, "'''") {
				// Toggle the flag for if we're in a code block or not
				inCodeBlock = !inCodeBlock
			}
		}
	}
	var (
		trimmedLine             string
		singleLineCommentMarker = e.SingleLineCommentMarker()
		q                       = NewQuoteState(singleLineCommentMarker, e.mode)
		ignoreSingleQuotes      = e.mode == modeLisp
	)
	// First loop from 0 up to to offset to figure out if we are already in a multiLine comment or a multiLine string at the current line
	for i := LineIndex(0); i < offsetY; i++ {
		trimmedLine = strings.TrimSpace(e.Line(LineIndex(i)))

		// Special case for ViM
		if e.mode == modeVim && strings.HasPrefix(trimmedLine, "\"") {
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
		q.Process(trimmedLine, ignoreSingleQuotes)
	}
	// q should now contain the current quote state
	var (
		lineRuneCount         uint
		lineStringCount       uint
		line                  string
		assemblyStyleComments = ((e.mode == modeAssembly) || (e.mode == modeLisp)) && !e.firstLineHash
		prevLineIsListItem    bool
		inListItem            bool
		screenLine            string
	)
	// Then loop from 0 to numlines (used as y+offset in the loop) to draw the text
	for y := LineIndex(0); y < numlines; y++ {
		lineRuneCount = 0   // per line rune counter, for drawing spaces afterwards (does not handle wide runes)
		lineStringCount = 0 // per line string counter, for drawing spaces afterwards (handles wide runes)

		line = e.Line(LineIndex(y + offsetY))

		line = strings.TrimRightFunc(line, unicode.IsSpace)

		trimmedLine = strings.TrimSpace(line)

		if strings.Contains(line, "\t") {
			line = strings.Replace(line, "\t", tabString, -1)
		}

		//screenLine = line

		//redrawCursor = containsNonASCII(line)

		if e.syntaxHighlight && !e.noColor {
			// Output a syntax highlighted line. Escape any tags in the input line.
			// textWithTags must be unescaped if there is not an error.
			if textWithTags, err := syntax.AsText([]byte(Escape(line)), assemblyStyleComments); err != nil {
				// Only output the line up to the width of the canvas
				screenLine = e.ChopLine(line, int(w))
				// TODO: Check if just "fmt.Print" works here, for several terminal emulators
				fmt.Println(screenLine)
				lineRuneCount += uint(len([]rune(screenLine)))
				lineStringCount += uint(len(screenLine))
			} else {
				var (
					// Color and unescape
					coloredString    string
					doneHighlighting = true
				)
				switch e.mode {
				case modeGit:
					coloredString = e.gitHighlight(line)
				case modeMarkdown:
					if highlighted, ok, codeBlockFound := markdownHighlight(line, inCodeBlock, prevLineIsListItem, &inListItem); ok {
						coloredString = highlighted
						if codeBlockFound {
							inCodeBlock = !inCodeBlock
						}
					} else {
						// Syntax highlight the line if it's not picked up by the markdownHighlight function
						coloredString = UnEscape(o.DarkTags(string(textWithTags)))
					}
					// If this is a list item, store true in "prevLineIsListItem"
					prevLineIsListItem = isListItem(line)
				case modePython:
					trimmedLine = strings.TrimSpace(line)
					foundDockstringMarker := false
					if strings.HasPrefix(trimmedLine, "\"\"\"") {
						inCodeBlock = !inCodeBlock
						foundDockstringMarker = true
					} else if strings.HasSuffix(trimmedLine, "\"\"\"") {
						inCodeBlock = !inCodeBlock
						foundDockstringMarker = true
					}
					if inCodeBlock || foundDockstringMarker {
						// Purple
						coloredString = UnEscape(e.multiLineString.Start(trimmedLine))
					} else {
						// Regular highlight
						coloredString = UnEscape(o.DarkTags(string(textWithTags)))
					}
				case modeConfig, modeShell, modeCMake, modeJSON:
					if strings.Contains(trimmedLine, "/*") || strings.HasSuffix(trimmedLine, "*/") {
						// No highlight
						coloredString = line
					} else if strings.HasPrefix(trimmedLine, "> ") {
						// If there is a } underneath and typing }, don't dedent, keep it at the same level!
						coloredString = UnEscape(e.multiLineString.Start(trimmedLine))
					} else {
						// Regular highlight + highlight yes and no in blue when using the default color scheme
						// TODO: Modify (and rewrite) the syntax package instead.
						coloredString = UnEscape(o.DarkTags(strings.Replace(strings.Replace(string(textWithTags), "<lightgreen>yes<", "<lightyellow>yes<", -1), "<lightred>no<", "<lightyellow>no<", -1)))
					}
				case modeStandardML, modeOCaml:
					// Handle single line comments starting with (* and ending with *)
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "(*") && strings.HasSuffix(trimmedLine, "*)") {
						coloredString = UnEscape(e.multiLineComment.Start(trimmedLine))
					} else {
						// Regular highlight
						coloredString = UnEscape(o.DarkTags(string(textWithTags)))
					}
				case modeZig:
					trimmedLine = strings.TrimSpace(line)
					// Handle doc comments (starting with ///)
					// and multiline strings (starting with \\)
					if strings.HasPrefix(trimmedLine, "///") || strings.HasPrefix(trimmedLine, `\\`) {
						coloredString = UnEscape(e.multiLineString.Start(trimmedLine))
					} else {
						// Regular highlight
						coloredString = UnEscape(o.DarkTags(string(textWithTags)))
					}
				case modeBat:
					trimmedLine = strings.TrimSpace(line)
					// In DOS batch files, ":" can be used both for labels and for single-line comments
					if strings.HasPrefix(trimmedLine, "@rem") || strings.HasPrefix(trimmedLine, "rem") || strings.HasPrefix(trimmedLine, ":") {
						// Handle single line comments
						coloredString = UnEscape(e.multiLineComment.Start(line))
					} else {
						// Regular highlight
						coloredString = UnEscape(o.DarkTags(string(textWithTags)))
					}
				case modeSQL, modeLua, modeHaskell, modeAda:
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "--") {
						// Handle single line comments
						coloredString = UnEscape(e.multiLineComment.Start(line))
					} else {
						// Regular highlight
						coloredString = UnEscape(o.DarkTags(string(textWithTags)))
					}
				case modeNroff:
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, `.\"`) {
						// Handle single line comments
						coloredString = UnEscape(e.multiLineComment.Start(line))
					} else {
						// Regular highlight
						coloredString = UnEscape(o.DarkTags(string(textWithTags)))
					}
				case modeLisp:
					q.singleQuote = 0
					// Special case for Lisp single-line comments
					trimmedLine = strings.TrimSpace(line)
					if strings.Count(trimmedLine, ";;") == 1 {
						// Color the line with the same color as for multiLine comments
						if strings.HasPrefix(trimmedLine, ";") {
							coloredString = UnEscape(e.multiLineComment.Start(line))
						} else if strings.Count(trimmedLine, ";;") == 1 {

							parts := strings.SplitN(line, ";;", 2)
							if newTextWithTags, err := syntax.AsText([]byte(Escape(parts[0])), false); err != nil {
								coloredString = UnEscape(o.DarkTags(string(textWithTags)))
							} else {
								coloredString = UnEscape(o.DarkTags(string(newTextWithTags)) + e.multiLineComment.Get(";;"+parts[1]))
							}

						} else if strings.Count(trimmedLine, ";") == 1 {

							parts := strings.SplitN(line, ";", 2)
							if newTextWithTags, err := syntax.AsText([]byte(Escape(parts[0])), false); err != nil {
								coloredString = UnEscape(o.DarkTags(string(textWithTags)))
							} else {
								coloredString = UnEscape(o.DarkTags(string(newTextWithTags)) + e.multiLineComment.Start(";"+parts[1]))
							}

						}
						doneHighlighting = true
						break
					}
					doneHighlighting = false
				case modeVim:
					// Special case for ViM single-line comments
					trimmedLine = strings.TrimSpace(line)
					if strings.Count(trimmedLine, "\"") == 1 {
						// Color the line with the same color as for multiLine comments
						if strings.HasPrefix(trimmedLine, "\"") {
							coloredString = UnEscape(e.multiLineComment.Start(line))
						} else {
							parts := strings.SplitN(line, "\"", 2)
							if newTextWithTags, err := syntax.AsText([]byte(Escape(parts[0])), false); err != nil {
								coloredString = UnEscape(o.DarkTags(string(textWithTags)))
							} else {
								coloredString = UnEscape(o.DarkTags(string(newTextWithTags)) + e.multiLineComment.Start("\""+parts[1]))
							}
						}
						break
					}
					fallthrough
				default:
					doneHighlighting = false
				}

				// Looks good so far
				//panic(line + coloredString)

				if !doneHighlighting {

					// C, C++, Go, Rust etc

					trimmedLine = strings.TrimSpace(line)
					q.Process(trimmedLine, ignoreSingleQuotes)

					//logf("%s -[ %d ]-->\n\t%s\n", trimmedLine, addedPar, q.String())

					switch {
					case e.mode == modePython && q.startedMultiLineString:
						// Python docstring
						coloredString = UnEscape(e.multiLineString.Start(line))
					case !strings.HasPrefix(trimmedLine, "/*") && !strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "//") && strings.Count(trimmedLine, "/*") == 1 && strings.HasSuffix(trimmedLine, "*/"): // A special case for "line /* comment */"
						coloredString = UnEscape(o.DarkTags(strings.Replace(line, "/*", e.multiLineComment.String()+"/*", 1)))
					case (q.multiLineComment || q.stoppedMultiLineComment) && !strings.Contains(line, "\"/*") && !strings.Contains(line, "*/\"") && !strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "//"):
						// In the middle of a multi-line comment
						coloredString = UnEscape(e.multiLineComment.Start(line))
					case q.singleLineComment || q.stoppedMultiLineComment:
						// A single line comment (the syntax module did the highlighting)
						coloredString = UnEscape(o.DarkTags(string(textWithTags)))
					case !q.startedMultiLineString && q.backtick > 0:
						// A multi-line string
						coloredString = UnEscape(e.multiLineString.Start(line))
					case (e.mode != modeHTML && e.mode != modeXML) && strings.Contains(line, "->"):
						// Pointer arrow
						kwc := syntax.DefaultTextConfig.Keyword
						// color off after -> and before e.fg is key for the color not to blink when scrolling
						coloredString = UnEscape(o.DarkTags(strings.Replace(line, "->", "<"+kwc+">-><off>"+e.fg.String(), -1)))
					default:
						// Regular code
						coloredString = UnEscape(o.DarkTags(string(textWithTags)))
					}
				}

				// Slice of runes and color attributes, while at the same time highlighting search terms
				runesAndAttributes := o.Extract(coloredString)

				// If e.rainbowParenthesis is true and we're not in a comment or a string, enable rainbow parenthesis
				if e.rainbowParenthesis && q.None() && !q.singleLineComment {
					thisLineParCount, thisLineBraCount := q.ParBraCount(trimmedLine, ignoreSingleQuotes)
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
						fg = e.fg
					}
					if matchForAnotherN > 0 {
						// Coloring an already found match
						fg = e.searchFg
						matchForAnotherN--
					} else if len(e.searchTerm) > 0 && letter == searchTermRunes[0] {
						// Potential search highlight match
						length := len([]rune(e.searchTerm))
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
						// match?
						if match {
							fg = e.searchFg
							matchForAnotherN = length - 1
						}
					}
					if letter == '\t' {
						c.Write(uint(cx)+lineRuneCount, uint(cy)+uint(y), fg, e.bg, tabString)
						lineRuneCount += uint(e.tabs.spacesPerTab)
						lineStringCount += uint(e.tabs.spacesPerTab)
					} else {
						//logf(string(letter))
						if unicode.IsControl(letter) {
							letter = controlRuneReplacement
						}
						c.WriteRuneB(uint(cx)+lineRuneCount, uint(cy)+uint(y), fg, bg, letter)
						lineRuneCount++                              // 1 rune
						lineStringCount += uint(len(string(letter))) // 1 rune, expanded
					}
				}
			}
		} else {
			// Output a regular line, scrolled to the current e.pos.offsetX
			screenLine = e.ChopLine(line, int(w))
			c.Write(uint(cx)+lineRuneCount, uint(cy)+uint(y), e.fg, e.bg, screenLine)
			lineRuneCount += uint(len([]rune(screenLine))) // rune count
			lineStringCount += uint(len(screenLine))       // string length, not rune length

		}

		var (
			xp uint
			yp = uint(cy) + uint(y)
		)

		// NOTE: Work in progress

		// TODO: This number must be sent into the WriteRune function and stored per line in the canvas!
		//       This way, the canvas can go the start of the line for every line with a runeLengthDiff > 0.
		runeLengthDiff := int(lineStringCount) - int(lineRuneCount)
		if runeLengthDiff > 2 {
			expandedRunes = true
		}

		// Fill the rest of the line on the canvas with "blanks"
		for x := lineRuneCount; x < w; x++ {
			xp = uint(cx) + x
			r := ' '
			//lineNumber := strconv.Itoa(runeLengthDiff)
			//r = []rune(lineNumber)[len([]rune(lineNumber))-1]
			c.WriteRuneB(xp, yp, e.fg, bg, r)
		}
		//c.WriteRuneB(xp, yp, e.fg, e.bg, '\n')
	}

	if expandedRunes {
		return errors.New("unsupported unicode text")
		// TODO: Write something that is great at laying out unicode runes, then build on that.
	}

	return nil
}
