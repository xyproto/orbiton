package main

import (
	"bytes"
	"fmt"
	"regexp"
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
	tout          = textoutput.NewTextOutput(true, true)
	resizeMut     sync.RWMutex                                  // locked when the terminal emulator is being resized
	colorTagRegex = regexp.MustCompile(`<([a-nA-Np-zP-Z]\w+)>`) // not starting with "o"
)

// WriteLines will draw editor lines from "fromline" to and up to "toline" to the canvas, at cx, cy
func (e *Editor) WriteLines(c *vt100.Canvas, fromline, toline LineIndex, cx, cy uint) {
	bg := e.Background.Background()
	tabString := strings.Repeat(" ", e.indentation.PerTab)
	inCodeBlock := false // used when highlighting Doc, Markdown, Python, Nim or Mojo

	// If the terminal emulator is being resized, then wait a bit
	resizeMut.Lock()
	defer resizeMut.Unlock()

	cw := c.Width()
	if fromline >= toline {
		return // errors.New("fromline >= toline in WriteLines")
	}
	numLinesToDraw := toline - fromline // Number of lines available on the canvas for drawing
	offsetY := fromline

	// logf("numlines: %d offsetY %d\n", numlines, offsetY)

	switch e.mode {
	// If in Markdown mode, figure out the current state of block quotes
	case mode.ASCIIDoc, mode.Markdown, mode.ReStructured, mode.SCDoc:
		// Figure out if "fromline" is within a markdown code block or not
		for i := LineIndex(0); i < fromline; i++ {
			trimmedLine := strings.TrimSpace(e.Line(i))
			// Check if the trimmed line starts with ~~~ or ```
			if strings.HasPrefix(trimmedLine, "~~~") || strings.HasPrefix(trimmedLine, "```") {
				if len(trimmedLine) > 3 && unicode.IsLetter([]rune(trimmedLine)[3]) {
					// If ~~~ or ``` is immediately followed by a letter, it is the start of a code block
					inCodeBlock = true
				} else if len(trimmedLine) > 4 && trimmedLine[3] == ' ' && unicode.IsLetter([]rune(trimmedLine)[4]) {
					// If ~~~ or ``` is immediately followed by a space and then a letter, it is the start of a code block
					inCodeBlock = true
				} else {
					// Toggle the flag for if we're in a code block or not
					inCodeBlock = !inCodeBlock
				}
			} else if strings.HasSuffix(trimmedLine, "~~~") || strings.HasSuffix(trimmedLine, "```") {
				// Check if the trimmed line ends with ~~~ or ```
				// Toggle the flag for if we're in a code block or not
				inCodeBlock = !inCodeBlock
			}
		}
	case mode.Nim, mode.Mojo, mode.Python:
		// Figure out if "fromline" is within a markdown code block or not
		for i := LineIndex(0); i < fromline; i++ {
			line := e.Line(i)
			trimmedLine := strings.TrimSpace(line)

			threeQuoteStart := strings.HasPrefix(trimmedLine, "\"\"\"") || strings.HasPrefix(trimmedLine, "'''")
			threeQuoteEnd := strings.HasSuffix(trimmedLine, "\"\"\"") || strings.HasSuffix(trimmedLine, "'''")

			if threeQuoteStart && threeQuoteEnd {
				inCodeBlock = false
			} else if threeQuoteStart || threeQuoteEnd {
				inCodeBlock = !inCodeBlock
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
			q.hasSingleLineComment = true
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
		lineRuneCount   uint
		lineStringCount uint
		line            string
		screenLine      string
		listItemRecord  []bool
		inListItem      bool
	)

	escapeFunction := Escape
	unEscapeFunction := UnEscape
	if e.mode == mode.Make || e.mode == mode.Just || e.mode == mode.Shell || e.mode == mode.Docker {
		escapeFunction = ShEscape
		unEscapeFunction = ShUnEscape
	}

	// Loop from 0 to numlines (used as y+offset in the loop) to draw the text
	for y := LineIndex(0); y < numLinesToDraw; y++ {
		lineRuneCount = 0   // per line rune counter, for drawing spaces afterwards (does not handle wide runes)
		lineStringCount = 0 // per line string counter, for drawing spaces afterwards (handles wide runes)

		line = trimRightSpace(e.Line(LineIndex(y + offsetY)))

		// already trimmed right, just trim left
		trimmedLine = strings.TrimLeftFunc(line, unicode.IsSpace)

		// expand tabs
		line = strings.ReplaceAll(line, "\t", tabString)

		if e.syntaxHighlight && !envNoColor {
			// Output a syntax highlighted line. Escape any tags in the input line.
			// textWithTags must be unescaped if there is not an error.
			if textWithTags, err := syntax.AsText([]byte(escapeFunction(line)), e.mode); err != nil {
				// Only output the line up to the width of the canvas
				screenLine = e.ChopLine(line, int(cw))
				// TODO: Check if just "fmt.Print" works here, for several terminal emulators
				fmt.Println(screenLine)
				lineRuneCount += uint(utf8.RuneCountInString(screenLine))
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
					coloredString = e.manPageHighlight(line, y == 0, y+1 == numLinesToDraw)
				case mode.ASCIIDoc, mode.Markdown, mode.ReStructured, mode.SCDoc:
					if highlighted, ok, codeBlockFound := e.markdownHighlight(line, inCodeBlock, listItemRecord, &inListItem); ok {
						coloredString = highlighted
						if codeBlockFound {
							inCodeBlock = !inCodeBlock
						}
					} else {
						// Syntax highlight the line if it's not picked up by the markdownHighlight function
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					}
					// If this is a list item, store true in "prevLineIsListItem"
					listItemRecord = append(listItemRecord, isListItem(line))
				case mode.Nim, mode.Mojo, mode.Python:
					trimmedLine = strings.TrimSpace(line)
					foundDocstringMarker := false

					if trimmedLine == "\"\"\"" || trimmedLine == "'''" { // only 3 letters
						inCodeBlock = !inCodeBlock
						foundDocstringMarker = true
					} else if strings.HasPrefix(trimmedLine, "\"\"\"") && strings.HasSuffix(trimmedLine, "\"\"\"") { // this could be 6 letters or more
						inCodeBlock = false
						foundDocstringMarker = true
					} else if strings.HasPrefix(trimmedLine, "'''") && strings.HasSuffix(trimmedLine, "'''") { // this could be 6 letters or more
						inCodeBlock = false
						foundDocstringMarker = true
					} else if strings.HasPrefix(trimmedLine, "\"\"\"") || strings.HasPrefix(trimmedLine, "'''") { // this is more than 3 letters
						// Toggle the flag for if we're in a code block or not
						inCodeBlock = true
						foundDocstringMarker = true
					} else if strings.HasSuffix(trimmedLine, "\"\"\"") || strings.HasSuffix(trimmedLine, "'''") { // this is more than 3 letters
						// Toggle the flag for if we're in a code block or not
						inCodeBlock = false
						foundDocstringMarker = true
					}

					if inCodeBlock || foundDocstringMarker {
						// Purple
						coloredString = unEscapeFunction(e.MultiLineString.Start(line))
					} else {
						// Regular highlight
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					}
				case mode.Config, mode.CMake, mode.JSON:
					if !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && (strings.Contains(trimmedLine, "/*") || strings.HasSuffix(trimmedLine, "*/")) {
						// No highlight
						coloredString = line
					} else if strings.HasPrefix(trimmedLine, "> ") {
						// If there is a } underneath and typing }, don't dedent, keep it at the same level!
						coloredString = unEscapeFunction(e.MultiLineString.Start(trimmedLine))
					} else if strings.Contains(trimmedLine, ":"+singleLineCommentMarker) {
						// If the line contains "://", then don't let the syntax package highlight it as a comment, by removing the gray color
						stringWithTags := strings.ReplaceAll(strings.ReplaceAll(string(textWithTags), "<"+e.Comment+">", "<"+e.Plaintext+">"), "</"+e.Comment+">", "</"+e.Plaintext+">")
						coloredString = unEscapeFunction(tout.DarkTags(strings.ReplaceAll(strings.ReplaceAll(stringWithTags, "<lightgreen>yes<", "<lightyellow>yes<"), "<lightred>no<", "<lightyellow>no<")))
					} else {
						// Regular highlight + highlight yes and no in blue when using the default color scheme
						// TODO: Modify (and rewrite) the syntax package instead.
						coloredString = unEscapeFunction(tout.DarkTags(strings.ReplaceAll(strings.ReplaceAll(string(textWithTags), "<lightgreen>yes<", "<lightyellow>yes<"), "<lightred>no<", "<lightyellow>no<")))
					}
				case mode.Zig:
					trimmedLine = strings.TrimSpace(line)
					// Handle doc comments (starting with ///)
					// and multi-line strings (starting with \\)
					if strings.HasPrefix(trimmedLine, "///") || strings.HasPrefix(trimmedLine, `\\`) {
						coloredString = unEscapeFunction(e.MultiLineString.Start(trimmedLine))
					} else {
						// Regular highlight
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					}
				case mode.Lilypond:
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "%") {
						coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
					} else if strings.HasPrefix(line, " ") && len(trimmedLine) > 0 && unicode.IsUpper([]rune(trimmedLine)[0]) {
						coloredString = unEscapeFunction(e.Foreground.Start(line))
					} else {
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					}
				case mode.Bat:
					trimmedLine = strings.TrimSpace(line)
					// In DOS batch files, ":" can be used both for labels and for single-line comments
					if strings.HasPrefix(trimmedLine, "@rem") || strings.HasPrefix(trimmedLine, "rem") || strings.HasPrefix(trimmedLine, ":") {
						// Handle single line comments
						coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
					} else {
						// Regular highlight
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					}
				case mode.Ada, mode.Agda, mode.Garnet, mode.Haskell, mode.Lua, mode.SQL, mode.Teal, mode.Terra: // not for OCaml and Standard ML
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "--") {
						// Handle single line comments
						coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
					} else if strings.HasPrefix(trimmedLine, "{-") && strings.HasSuffix(trimmedLine, "-}") {
						coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
					} else if strings.Contains(trimmedLine, "->") {
						coloredString = unEscapeFunction(tout.DarkTags(e.ArrowReplace(string(textWithTags))))
					} else {
						// Regular highlight
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					}
				case mode.Amber:
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "!!") {
						// Handle single line comments
						coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
					} else {
						// Regular highlight
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					}
				case mode.StandardML, mode.OCaml:
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "(*") && strings.HasSuffix(trimmedLine, "*)") {
						coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
					} else if strings.Contains(trimmedLine, "->") {
						coloredString = unEscapeFunction(tout.DarkTags(e.ArrowReplace(string(textWithTags))))
					} else {
						doneHighlighting = false
						break
					}
				case mode.Elm:
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "{-") && strings.HasSuffix(trimmedLine, "-}") {
						coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
					} else if strings.Contains(trimmedLine, "->") {
						coloredString = unEscapeFunction(tout.DarkTags(e.ArrowReplace(string(textWithTags))))
					} else {
						doneHighlighting = false
						break
					}
				case mode.Nroff:
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, `.\"`) {
						// Handle single line comments
						coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
					} else {
						// Regular highlight
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
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
							coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
						} else if doubleSemiCount == 1 {

							parts := strings.SplitN(line, ";;", 2)
							if newTextWithTags, err := syntax.AsText([]byte(escapeFunction(parts[0])), e.mode); err != nil {
								coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
							} else {
								coloredString = unEscapeFunction(tout.DarkTags(string(newTextWithTags)) + e.MultiLineComment.Get(";;"+parts[1]))
							}

						} else if strings.Count(trimmedLine, ";") == 1 {

							parts := strings.SplitN(line, ";", 2)
							if newTextWithTags, err := syntax.AsText([]byte(escapeFunction(parts[0])), e.mode); err != nil {
								coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
							} else {
								coloredString = unEscapeFunction(tout.DarkTags(string(newTextWithTags)) + e.MultiLineComment.Start(";"+parts[1]))
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
							coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
						} else {
							parts := strings.SplitN(line, "\"", 2)
							if newTextWithTags, err := syntax.AsText([]byte(escapeFunction(parts[0])), e.mode); err != nil {
								coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
							} else {
								coloredString = unEscapeFunction(tout.DarkTags(string(newTextWithTags)) + e.MultiLineComment.Start("\""+parts[1]))
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

					// logf("%s -[ %d ]-->\n\t%s\n", trimmedLine, addedPar, q.String())

					switch {
					case (e.mode == mode.Nim || e.mode == mode.Mojo || e.mode == mode.Python) && q.startedMultiLineString:
						// Python docstring
						coloredString = unEscapeFunction(e.MultiLineString.Get(line))
					case (e.mode == mode.Arduino || e.mode == mode.C || e.mode == mode.Cpp || e.mode == mode.ObjC || e.mode == mode.Shader || e.mode == mode.Make || e.mode == mode.Just) && !q.multiLineComment && (strings.HasPrefix(trimmedLine, "#if") || strings.HasPrefix(trimmedLine, "#else") || strings.HasPrefix(trimmedLine, "#elseif") || strings.HasPrefix(trimmedLine, "#endif") || strings.HasPrefix(trimmedLine, "#define") || strings.HasPrefix(trimmedLine, "#pragma")):
						coloredString = unEscapeFunction(e.MultiLineString.Get(line))
					case e.mode != mode.Shell && e.mode != mode.Docker && e.mode != mode.Make && e.mode != mode.Just && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.HasSuffix(trimmedLine, "*/") && !strings.Contains(trimmedLine, "/*"):
						coloredString = unEscapeFunction(e.MultiLineComment.Get(line))
					case (e.mode == mode.StandardML || e.mode == mode.OCaml) && (!strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.HasSuffix(trimmedLine, "*)") && !strings.Contains(trimmedLine, "(*")):
						coloredString = unEscapeFunction(e.MultiLineComment.Get(line))
					case (e.mode == mode.Elm || e.mode == mode.Haskell) && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.HasSuffix(trimmedLine, "-}") && !strings.Contains(trimmedLine, "{-") || q.multiLineComment:
						coloredString = unEscapeFunction(e.MultiLineComment.Get(line))
					case e.mode != mode.Shell && e.mode != mode.Docker && e.mode != mode.Make && e.mode != mode.Just && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.LastIndex(trimmedLine, "/*") > strings.LastIndex(trimmedLine, "*/"):
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					case (e.mode == mode.StandardML || e.mode == mode.OCaml) && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.LastIndex(trimmedLine, "(*") > strings.LastIndex(trimmedLine, "*)"):
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					case (e.mode == mode.Elm || e.mode == mode.Haskell) && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.LastIndex(trimmedLine, "{-") > strings.LastIndex(trimmedLine, "-}") || q.multiLineComment:
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					case q.containsMultiLineComments:
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					case e.mode != mode.Shell && e.mode != mode.Docker && e.mode != mode.Make && e.mode != mode.Just && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && (q.multiLineComment || q.stoppedMultiLineComment) && !strings.Contains(line, "\"/*") && !strings.Contains(line, "*/\"") && !strings.Contains(line, "\"(*") && !strings.Contains(line, "*)\"") && !strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "//"):
						// In the middle of a multi-line comment
						coloredString = unEscapeFunction(e.MultiLineComment.Get(line))
					case q.hasSingleLineComment || q.stoppedMultiLineComment:
						// Fix for interpreting URLs in shell scripts as single line comments
						if singleLineCommentMarker != "//" {
							commentColorName := e.Comment
							textWithTags = bytes.ReplaceAll(textWithTags, []byte(":<off><"+commentColorName+">//"), []byte("://"))
							textWithTags = bytes.ReplaceAll(textWithTags, []byte(" "+singleLineCommentMarker), []byte(" <"+commentColorName+">"+singleLineCommentMarker))
						}
						// A single line comment (the syntax module did the highlighting)
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					case !q.startedMultiLineString && q.backtick > 0:
						// A multi-line string
						coloredString = unEscapeFunction(e.MultiLineString.Get(line))
					case (e.mode != mode.HTML && e.mode != mode.XML && e.mode != mode.Markdown && e.mode != mode.Make && e.mode != mode.Just && e.mode != mode.Blank) && strings.Contains(line, "->"):
						// NOTE that if two color tags are placed after each other, they may cause blinking. Remember to turn <off> each color.
						coloredString = unEscapeFunction(tout.DarkTags(e.ArrowReplace(string(textWithTags))))
					default:
						// Regular code, may contain a comment at the end
						if strings.Contains(line, "://") {
							coloredString = unEscapeFunction(tout.DarkTags(e.replaceColorTagsInURL(string(textWithTags))))
						} else {
							coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
						}
					}

					// Do not color lines gray if they start with "#" and are not in a multiline comment and the current single line comment marker is "//"
					otherCommentMarker := "#"
					if singleLineCommentMarker == "#" {
						otherCommentMarker = "//"
					}
					if strings.HasPrefix(trimmedLine, otherCommentMarker) && !q.containsMultiLineComments && strings.HasPrefix(strings.TrimSpace(string(textWithTags)), "<"+e.Comment+">") {
						parts := strings.SplitN(line, otherCommentMarker, 2)
						commentMarkerString := tout.DarkTags(parts[0] + "<" + e.Dollar + ">" + otherCommentMarker + "<off>")
						theRestString := tout.DarkTags(parts[1])
						if theRestWithTags, err := syntax.AsText([]byte(escapeFunction(parts[1])), e.mode); err != nil {
							theRestString = tout.DarkTags(string(theRestWithTags))
						}
						coloredString = unEscapeFunction(commentMarkerString + theRestString)
					}

					// Take an extra pass on coloring the -> arrow, even if it's in a comment
					if !(e.mode == mode.HTML || e.mode == mode.XML || e.mode == mode.Markdown || e.mode == mode.Blank || e.mode == mode.Config || e.mode == mode.Shell || e.mode == mode.Docker || e.mode == mode.Just) && strings.Contains(line, "->") {
						arrowIndex := strings.Index(line, "->")
						commentMarkers := []string{"//", "/*", "(*", "{-"}
						arrowBeforeCommentMarker := true

						for _, marker := range commentMarkers {
							commentIndex := strings.Index(line, marker)
							if commentIndex != -1 && commentIndex < arrowIndex {
								if marker == "(*" && (e.mode == mode.OCaml || e.mode == mode.StandardML || e.mode == mode.Haskell) {
									arrowBeforeCommentMarker = false
									break
								}
								if marker == "{-" && (e.mode == mode.Elm || e.mode == mode.Haskell) {
									arrowBeforeCommentMarker = false
									break
								}
								if marker != "(*" && marker != "{-" {
									arrowBeforeCommentMarker = false
									break
								}
							}
						}

						if arrowBeforeCommentMarker {
							// arrow is before comment marker, color the arrow
							coloredString = unEscapeFunction(tout.DarkTags(e.ArrowReplace(string(textWithTags))))
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
				if e.mode != mode.Git && e.mode != mode.Email && e.rainbowParenthesis && q.None() && !q.hasSingleLineComment && !q.stoppedMultiLineComment {
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
				untilNextJumpLetter := 0
				hasSearchTerm := len(e.searchTerm) > 0
				jumpToLetterMode := e.jumpToLetterMode
				var fg vt100.AttributeColor
				var letter rune
				var tx, ty uint

				for runeIndex, ra := range runesAndAttributes {
					if skipX > 0 {
						skipX--
						continue
					}
					if ra.R == ' ' {
						fg = e.Foreground
					} else {
						fg = ra.A

						if matchForAnotherN > 0 {
							// Coloring an already found match
							fg = e.SearchHighlight
							matchForAnotherN--
						} else if hasSearchTerm && ra.R == searchTermRunes[0] {
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
						} else if jumpToLetterMode {
							letter = ra.R
							// Highlight some letters, and make it possible for the user to jump directly to these after pressing ctrl-l
							tx = cx + uint(lineRuneCount)           // the x position
							ty = cy + uint(y) + uint(e.pos.offsetY) // adding offset to get the position in the file and not only on the screen
							if untilNextJumpLetter <= 0 && !e.HasJumpLetter(letter) && e.RegisterJumpLetter(letter, ColIndex(tx), LineIndex(ty)) {
								untilNextJumpLetter = 60
								fg = e.JumpToLetterColor // foreground color for the highlighted "jump to letter"
							} else {
								untilNextJumpLetter--
								fg = e.CommentColor // foreground color for the non-highlighted letters
							}
						}
					}

					if ra.R == '\t' {
						c.Write(cx+lineRuneCount, cy+uint(y), fg, e.Background, tabString)
						lineRuneCount += uint(e.indentation.PerTab)
						lineStringCount += uint(e.indentation.PerTab)
					} else {
						letter = ra.R
						if unicode.IsControl(letter) {
							letter = controlRuneReplacement
						}
						tx = cx + lineRuneCount
						ty = cy + uint(y)
						if tx < cw {
							c.WriteRuneBNoLock(tx, ty, fg, bg, letter)
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
			c.Write(cx+lineRuneCount, cy+uint(y), e.Foreground, e.Background, screenLine)
			lineRuneCount += uint(utf8.RuneCountInString(screenLine)) // rune count
		}

		// Fill the rest of the line on the canvas with "blanks"
		// TODO: This may draw the wrong number of blanks, since lineRuneCount should really be the number of visible glyphs at this point
		yp := cy + uint(y)
		xp := cx + lineRuneCount
		c.WriteRunesB(xp, yp, e.Foreground, bg, ' ', cw-lineRuneCount)

		// Draw a red line to remind the user of where the N-column limit is
		if (e.showColumnLimit || e.mode == mode.Git) && lineRuneCount <= uint(e.wrapWidth) {
			c.WriteRune(uint(e.wrapWidth), yp, e.Theme.SearchHighlight, bg, '·')
		}

	}
}

// ArrowReplace can syntax highlight pointer arrows in C and C++ and function arrows in other languages
func (e *Editor) ArrowReplace(s string) string {
	arrowColor := e.Star
	if e.mode == mode.Arduino || e.mode == mode.C || e.mode == mode.Cpp || e.mode == mode.ObjC || e.mode == mode.Shader {
		arrowColor = syntax.DefaultTextConfig.Class
	}
	fieldColor := syntax.DefaultTextConfig.Protected
	s = strings.ReplaceAll(s, ">-<", "><off><"+arrowColor+">-<")
	return strings.ReplaceAll(s, ">"+Escape(">"), "><off><"+arrowColor+">"+Escape(">")+"<off><"+fieldColor+">")
}

// replaceColorTagsInURL handles a special case where the highlight package doesn't handle URL's with "//" too well.
// This function is a bit of a hack, until this editor uses a different syntax highlighting package.
func (e *Editor) replaceColorTagsInURL(input string) string {
	var (
		fields    = strings.Split(input, " ")
		newFields = make([]string, len(fields))
	)
	for i, field := range fields {
		if strings.Contains(field, ">:<off>") && strings.Contains(field, ">//") {
			newFields[i] = colorTagRegex.ReplaceAllString(field, "<"+e.Theme.String+">") + "<" + e.Theme.Plaintext + ">"
		} else {
			newFields[i] = field
		}
	}
	return strings.Join(newFields, " ")
}
