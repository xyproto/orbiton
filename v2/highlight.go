package main

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"text/scanner"
	"unicode"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/sourcegraph/annotate"
	"github.com/xyproto/env/v2"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

var (
	controlRuneReplacement = func() rune {
		if envVT100 {
			return '?'
		}
		return '¿' // for displaying control sequence characters. Could also use: �
	}()
	wrapMarkerRune = func() rune {
		if envVT100 {
			return '.'
		}
		return '·'
	}()
	ellipsisRune = func() rune {
		if envVT100 {
			return '~'
		}
		return '…'
	}()
)

// Kind represents a syntax highlighting kind (class) which will be assigned to tokens.
// A syntax highlighting scheme (style) maps text style properties to each token kind.
type Kind uint8

// Supported highlighting kinds.
const (
	Whitespace Kind = iota
	AndOr
	AngleBracket
	AssemblyEnd
	Class
	Comment
	Decimal
	Dollar
	Literal
	Keyword
	Mut
	Plaintext
	Private
	Protected
	Public
	Punctuation
	Self
	Star
	Static
	String
	Tag
	TextAttrName
	TextAttrValue
	TextTag
	Type
)

// TextConfig holds the Text class configuration to be used by annotators when highlighting code.
type TextConfig struct {
	AndOr         string
	AngleBracket  string
	AssemblyEnd   string
	Class         string
	Comment       string
	Decimal       string
	Dollar        string
	Keyword       string
	Literal       string
	Mut           string
	Plaintext     string
	Private       string
	Protected     string
	Public        string
	Punctuation   string
	Self          string
	Star          string
	Static        string
	String        string
	Tag           string
	TextAttrName  string
	TextAttrValue string
	TextTag       string
	Type          string
	Whitespace    string
}

// Option is a function that can modify TextConfig.
type Option func(*TextConfig)

var (
	colorTagRegex = regexp.MustCompile(`<([a-nA-Np-zP-Z]\w+)>`) // not starting with "o"
	tout          = vt.New()
	resizeMut     sync.RWMutex                                         // locked when the terminal emulator is being resized
	noGUI         = !env.Has("DISPLAY") && !env.Has("WAYLAND_DISPLAY") // no X, no Wayland
)

// DefaultTextConfig provides class names matching the color names of textoutput tags.
var DefaultTextConfig = TextConfig{
	AndOr:         "red",
	AngleBracket:  "red",
	AssemblyEnd:   "lightyellow",
	Class:         "white",
	Comment:       "darkgray",
	Decimal:       "red",
	Dollar:        "white",
	Keyword:       "red",
	Literal:       "white",
	Mut:           "magenta",
	Plaintext:     "white",
	Private:       "red",
	Protected:     "red",
	Public:        "red",
	Punctuation:   "red",
	Self:          "magenta",
	Star:          "white",
	Static:        "lightyellow",
	String:        "lightwhite",
	Tag:           "white",
	TextAttrName:  "white",
	TextAttrValue: "white",
	TextTag:       "white",
	Type:          "white",
	Whitespace:    "",
}

// GetClass returns the CSS class for a given token kind.
func (c TextConfig) GetClass(kind Kind) string {
	switch kind {
	case String:
		return c.String
	case Keyword:
		return c.Keyword
	case Comment:
		return c.Comment
	case Type:
		return c.Type
	case Literal:
		return c.Literal
	case Punctuation:
		return c.Punctuation
	case Plaintext:
		return c.Plaintext
	case Tag:
		return c.Tag
	case TextTag:
		return c.TextTag
	case TextAttrName:
		return c.TextAttrName
	case TextAttrValue:
		return c.TextAttrValue
	case Decimal:
		return c.Decimal
	case AndOr:
		return c.AndOr
	case AngleBracket:
		return c.AngleBracket
	case Dollar:
		return c.Dollar
	case Star:
		return c.Star
	case Static:
		return c.Static
	case Self:
		return c.Self
	case Class:
		return c.Class
	case Public:
		return c.Public
	case Private:
		return c.Private
	case Protected:
		return c.Protected
	case AssemblyEnd:
		return c.AssemblyEnd
	case Mut:
		return c.Mut
	}
	return ""
}

// Printer renders highlighted output.
type Printer interface {
	Print(w io.Writer, kind Kind, tokText string) error
}

// TextPrinter wraps TextConfig to implement Printer.
type TextPrinter TextConfig

// Print writes token text with start/end tags based on its kind.
func (p TextPrinter) Print(w io.Writer, kind Kind, tokText string) error {
	class := TextConfig(p).GetClass(kind)
	if class != "" {
		if _, err := io.WriteString(w, "<"+class+">"); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, tokText); err != nil {
		return err
	}
	if class != "" {
		if _, err := io.WriteString(w, "<off>"); err != nil {
			return err
		}
	}
	return nil
}

// Annotator produces syntax highlighting annotations.
type Annotator interface {
	Annotate(start int, kind Kind, tokText string) (*annotate.Annotation, error)
}

// TextAnnotator wraps TextConfig to implement Annotator.
type TextAnnotator TextConfig

// Print scans tokens from s, using Printer p for mode m.
func Print(s *scanner.Scanner, w io.Writer, p Printer, m mode.Mode) error {
	inComment := false
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		tokText := s.TokenText()
		if err := p.Print(w, tokenKind(tok, tokText, &inComment, m), tokText); err != nil {
			return err
		}
	}
	return nil
}

// AsText returns src highlighted for mode m, applying options to TextConfig.
func AsText(src []byte, m mode.Mode, options ...Option) ([]byte, error) {
	cfg := DefaultTextConfig
	for _, opt := range options {
		opt(&cfg)
	}
	var buf bytes.Buffer
	if err := Print(NewScanner(src), &buf, TextPrinter(cfg), m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// NewScanner returns a scanner.Scanner configured for syntax highlighting.
func NewScanner(src []byte) *scanner.Scanner {
	return NewScannerReader(bytes.NewReader(src))
}

// NewScannerReader returns a scanner.Scanner configured for syntax highlighting from r.
func NewScannerReader(r io.Reader) *scanner.Scanner {
	var s scanner.Scanner
	s.Init(r)
	s.Error = func(*scanner.Scanner, string) {}
	s.Whitespace = 0
	s.Mode ^= scanner.SkipComments
	return &s
}

// WriteLines will draw editor lines from "fromline" to and up to "toline" to the canvas, at cx, cy
func (e *Editor) WriteLines(c *vt.Canvas, fromline, toline LineIndex, cx, cy uint, shouldHighlightNow, hideCursorWhenDrawing bool) {
	// TODO: Use a channel for queuing up calls to the package to avoid race conditions

	// TODO: Refactor this function
	var (
		match                              bool
		arrowBeforeCommentMarker           bool
		inListItem                         bool
		inCodeBlock                        bool // used when highlighting Doc, Markdown, Python, Nim, Mojo or Starlark
		ok                                 bool
		codeBlockFound                     bool
		foundDocstringMarker               bool
		doneHighlighting                   = true
		hasSearchTerm                      = len(e.searchTerm) > 0
		ignoreSingleQuotes                 = e.mode == mode.Lisp || e.mode == mode.Clojure || e.mode == mode.Scheme || e.mode == mode.Ini
		numLinesToDraw                     int
		runeIndex                          int
		length                             int
		counter                            int
		i                                  int
		thisLineParCount, thisLineBraCount int
		parCountBeforeThisLine             int
		braCountBeforeThisLine             int
		doubleSemiCount                    int
		matchForAnotherN                   int
		untilNextJumpLetter                int
		arrowIndex                         int
		commentIndex                       int
		k                                  int
		lineRuneCount                      uint
		yp, xp                             uint
		tx, ty                             uint
		cw                                 uint
		marker                             string
		line                               string
		screenLine                         string
		trimmedLine                        string
		highlighted                        string
		stringWithTags                     string
		commentColorName                   string
		otherCommentMarker                 string
		commentMarkerString                string
		theRestString                      string
		coloredString                      string
		tabString                          = strings.Repeat(" ", e.indentation.PerTab)
		singleLineCommentMarker            = e.SingleLineCommentMarker()
		letter                             rune
		err                                error
		listItemRecord                     []bool
		textWithTags                       []byte
		newTextWithTags                    []byte
		theRestWithTags                    []byte
		commentMarkers                     = []string{"//", "/*", "(*", "{-"}
		parts                              []string
		li                                 LineIndex
		y                                  LineIndex
		offsetY                            LineIndex
		fg                                 vt.AttributeColor
		dottedLineColor                    vt.AttributeColor
		bg                                 vt.AttributeColor = e.Background.Background()
		ra, ra2                            vt.CharAttribute
		searchTermRunes                    = []rune(e.searchTerm) // Search term highlighting
		runesAndAttributes                 []vt.CharAttribute
		q                                  *QuoteState
		escapeFunction                     = Escape
		unEscapeFunction                   = UnEscape
		rw                                 int // rune width
		yesNoReplacer                      = strings.NewReplacer("<lightgreen>yes<", "<lightyellow>yes<", "<lightred>no<", "<lightyellow>no<")
		commentReplacer                    = strings.NewReplacer("<"+e.Comment+">", "<"+e.Plaintext+">", "</"+e.Comment+">", "</"+e.Plaintext+">")
	)

	// If the terminal emulator is being resized, then wait a bit
	resizeMut.Lock()
	defer resizeMut.Unlock()

	if hideCursorWhenDrawing {
		c.HideCursor()
		defer c.ShowCursor()
	}

	cw = c.Width()
	if fromline >= toline {
		return // errors.New("fromline >= toline in WriteLines")
	}
	numLinesToDraw = int(toline - fromline) // Number of lines available on the canvas for drawing
	offsetY = fromline

	// logf("numlines: %d offsetY %d\n", numlines, offsetY)

	dottedLineColor = e.Theme.SearchHighlight
	if e.wrapWhenTyping {
		dottedLineColor = e.Theme.StatusForeground
	}

	switch e.mode {
	// If in Markdown mode, figure out the current state of block quotes
	case mode.ASCIIDoc, mode.Markdown, mode.ReStructured, mode.SCDoc:
		// Figure out if "fromline" is within a markdown code block or not
		for li = LineIndex(0); li < fromline; li++ {
			trimmedLine = strings.TrimSpace(e.Line(li))
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
	case mode.Mojo, mode.Nim, mode.Python, mode.Starlark:
		// Figure out if "fromline" is within a markdown code block or not
		for li = LineIndex(0); li < fromline; li++ {
			line = e.Line(li)
			trimmedLine = strings.TrimSpace(line)
			inCodeBlock, _ = checkMultiLineString(trimmedLine, inCodeBlock)
		}
	}

	q, err = NewQuoteState(singleLineCommentMarker, e.mode, ignoreSingleQuotes)
	if err != nil {
		return // err
	}

	if e.mode != mode.Vim {
		// First loop from 0 up to to offset to figure out if we are already in a multiLine comment or a multiLine string at the current line
		for li = LineIndex(0); li < offsetY; li++ {
			trimmedLine = strings.TrimSpace(e.Line(li))
			// Have a trimmed line. Want to know: the current state of which quotes, comments or strings we are in.
			// Solution, have a state struct!
			q.Process(trimmedLine)
		}
	} else { // Special case for ViM
		// First loop from 0 up to to offset to figure out if we are already in a multiLine comment or a multiLine string at the current line
		for li = LineIndex(0); li < offsetY; li++ {
			trimmedLine = strings.TrimSpace(e.Line(li))
			if strings.HasPrefix(trimmedLine, "\"") {
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
	}
	// q should now contain the current quote state

	if e.mode == mode.Make || e.mode == mode.Just || e.mode == mode.Shell || e.mode == mode.Docker {
		escapeFunction = ShEscape
		unEscapeFunction = ShUnEscape
	}

	highlightCurrentLine := false

	// buffer for extracting char attributes from strings with terminal codes (will be expanded if it's too small)
	cc := make([]vt.CharAttribute, 256)

	// Loop from 0 to numlines (used as y+offset in the loop) to draw the text
	for y = LineIndex(0); y < LineIndex(numLinesToDraw); y++ {

		highlightCurrentLine = shouldHighlightNow && int(y) == e.pos.sy

		lineRuneCount = 0 // per line rune counter, for drawing spaces afterwards

		line = trimRightSpace(e.Line(LineIndex(y + offsetY)))

		// already trimmed right, just trim left
		trimmedLine = strings.TrimLeftFunc(line, unicode.IsSpace)

		// expand tabs
		line = strings.ReplaceAll(line, "\t", tabString)

		if e.syntaxHighlight && !envNoColor {
			// Output a syntax highlighted line. Escape any tags in the input line.
			// textWithTags must be unescaped if there is not an error.
			if textWithTags, err = AsText([]byte(escapeFunction(line)), e.mode); err != nil {
				// Only output the line up to the width of the canvas
				screenLine = e.ChopLine(line, int(cw))
				// TODO: Check if just "fmt.Print" works here, for several terminal emulators
				fmt.Println(screenLine)
				lineRuneCount += uint(runewidth.StringWidth(screenLine))
			} else {
				switch e.mode {
				case mode.Email, mode.Git:
					coloredString = e.gitHighlight(line)
				case mode.ManPage:
					coloredString = e.manPageHighlight(line, y == 0, int(y+1) == numLinesToDraw)
				case mode.ASCIIDoc, mode.Markdown, mode.ReStructured, mode.SCDoc:
					if highlighted, ok, codeBlockFound = e.markdownHighlight(line, inCodeBlock, listItemRecord, &inListItem); ok {
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
				case mode.Nim, mode.Mojo, mode.Python, mode.Starlark:
					trimmedLine = strings.TrimSpace(line)
					foundDocstringMarker = false

					inCodeBlock, foundDocstringMarker = checkMultiLineString(trimmedLine, inCodeBlock)

					if inCodeBlock || foundDocstringMarker {
						// Purple
						coloredString = unEscapeFunction(e.MultiLineString.Start(line))
					} else {
						// Regular highlight
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					}
				case mode.Config, mode.CMake, mode.JSON, mode.Ini, mode.FSTAB, mode.Nix:
					if !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && (strings.Contains(trimmedLine, "/*") || strings.HasSuffix(trimmedLine, "*/")) {
						// No highlight
						coloredString = line
					} else if (e.mode == mode.Ini || e.mode == mode.Config || e.mode == mode.FSTAB || e.mode == mode.Nix) && strings.HasPrefix(trimmedLine, ";") {
						// Commented out
						coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
					} else if strings.HasPrefix(trimmedLine, "> ") {
						// If there is a } underneath and typing }, don't dedent, keep it at the same level!
						coloredString = unEscapeFunction(e.MultiLineString.Start(line))
					} else if strings.Contains(trimmedLine, ":"+singleLineCommentMarker) {
						// If the line contains "://", then don't let the syntax package highlight it as a comment, by removing the gray color
						stringWithTags = commentReplacer.Replace(string(textWithTags))
						coloredString = unEscapeFunction(tout.DarkTags(yesNoReplacer.Replace(stringWithTags)))
					} else {
						// Regular highlight + highlight yes and no in blue when using the default color scheme
						// TODO: Modify (and rewrite) the syntax package instead.
						coloredString = unEscapeFunction(tout.DarkTags(yesNoReplacer.Replace(string(textWithTags))))
					}
				case mode.Zig:
					trimmedLine = strings.TrimSpace(line)
					// Handle doc comments (starting with ///)
					// and multi-line strings (starting with \\)
					if strings.HasPrefix(trimmedLine, "///") || strings.HasPrefix(trimmedLine, `\\`) {
						coloredString = unEscapeFunction(e.MultiLineString.Start(line))
					} else {
						// Regular highlight
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					}
				case mode.ABC, mode.Lilypond, mode.Perl, mode.Prolog:
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "%") && !(e.mode == mode.ABC && strings.HasPrefix(trimmedLine, "%%")) {
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
				case mode.Ada, mode.Agda, mode.Vibe67, mode.Elm, mode.Garnet, mode.Haskell, mode.Lua, mode.Nmap, mode.SQL, mode.Teal, mode.Terra: // not for OCaml and Standard ML
					trimmedLine = strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "--") {
						// Handle single line comments
						coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
					} else if e.mode == mode.Lua && strings.Contains(line, "--") {
						// Inline Lua comment, e.g. "local x = 42 -- set x to 42"
						parts := strings.SplitN(line, "--", 2)
						// Highlight the code portion before the comment
						if newTextWithTags, err = AsText([]byte(escapeFunction(parts[0])), e.mode); err != nil {
							coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
						} else {
							// Append the comment portion highlighted as a comment
							coloredString = unEscapeFunction(tout.DarkTags(string(newTextWithTags)) +
								e.MultiLineComment.Start("--"+parts[1]))
						}
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
					coloredString = vt.Colorize(line)
				case mode.Lisp, mode.Clojure:
					q.singleQuote = 0
					// Special case for Lisp single-line comments
					trimmedLine = strings.TrimSpace(line)
					if doubleSemiCount = strings.Count(trimmedLine, ";;"); doubleSemiCount > 0 {
						// Color the line with the same color as for multiLine comments
						if strings.HasPrefix(trimmedLine, ";") {
							coloredString = unEscapeFunction(e.MultiLineComment.Start(line))
						} else if doubleSemiCount == 1 {
							parts = strings.SplitN(line, ";;", 2)
							if newTextWithTags, err = AsText([]byte(escapeFunction(parts[0])), e.mode); err != nil {
								coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
							} else {
								coloredString = unEscapeFunction(tout.DarkTags(string(newTextWithTags)) + e.MultiLineComment.Get(";;"+parts[1]))
							}
						} else if strings.Count(trimmedLine, ";") == 1 {
							parts = strings.SplitN(line, ";", 2)
							if newTextWithTags, err = AsText([]byte(escapeFunction(parts[0])), e.mode); err != nil {
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
							parts = strings.SplitN(line, "\"", 2)
							if newTextWithTags, err = AsText([]byte(escapeFunction(parts[0])), e.mode); err != nil {
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
					case (e.mode == mode.Nim || e.mode == mode.Mojo || e.mode == mode.Python || e.mode == mode.Starlark) && q.startedMultiLineString:
						// Python docstring
						coloredString = unEscapeFunction(e.MultiLineString.Get(line))
					case (e.mode == mode.Arduino || e.mode == mode.C || e.mode == mode.Cpp || e.mode == mode.ObjC || e.mode == mode.Shader || e.mode == mode.Make || e.mode == mode.Just) && !q.multiLineComment && (strings.HasPrefix(trimmedLine, "#if") || strings.HasPrefix(trimmedLine, "#else") || strings.HasPrefix(trimmedLine, "#elseif") || strings.HasPrefix(trimmedLine, "#endif") || strings.HasPrefix(trimmedLine, "#elif") || strings.HasPrefix(trimmedLine, "#define") || strings.HasPrefix(trimmedLine, "#pragma")):
						coloredString = unEscapeFunction(e.MultiLineString.Get(line))
					case e.mode != mode.Shell && e.mode != mode.Docker && e.mode != mode.Make && e.mode != mode.Just && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.HasSuffix(trimmedLine, "*/") && !strings.Contains(trimmedLine, "/*"):
						coloredString = unEscapeFunction(e.MultiLineComment.Get(line))
					case (e.mode == mode.StandardML || e.mode == mode.OCaml) && (!strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.HasSuffix(trimmedLine, "*)") && !strings.Contains(trimmedLine, "(*")):
						coloredString = unEscapeFunction(e.MultiLineComment.Get(line))
					case (e.mode == mode.Elm || e.mode == mode.Haskell || e.mode == mode.Vibe67) && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.HasSuffix(trimmedLine, "-}") && !strings.Contains(trimmedLine, "{-") || q.multiLineComment:
						coloredString = unEscapeFunction(e.MultiLineComment.Get(line))
					case e.mode != mode.Shell && e.mode != mode.Docker && e.mode != mode.Make && e.mode != mode.Just && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.LastIndex(trimmedLine, "/*") > strings.LastIndex(trimmedLine, "*/"):
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					case (e.mode == mode.StandardML || e.mode == mode.OCaml) && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.LastIndex(trimmedLine, "(*") > strings.LastIndex(trimmedLine, "*)"):
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					case (e.mode == mode.Elm || e.mode == mode.Haskell || e.mode == mode.Vibe67) && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && strings.LastIndex(trimmedLine, "{-") > strings.LastIndex(trimmedLine, "-}") || q.multiLineComment:
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					case q.containsMultiLineComments:
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					case e.mode != mode.Shell && e.mode != mode.Docker && e.mode != mode.FSTAB && e.mode != mode.Nix && e.mode != mode.Make && e.mode != mode.Just && !strings.HasPrefix(trimmedLine, singleLineCommentMarker) && (q.multiLineComment || q.stoppedMultiLineComment) && !strings.Contains(line, "\"/*") && !strings.Contains(line, "*/\"") && !strings.Contains(line, "\"(*") && !strings.Contains(line, "*)\"") && !strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "//"):
						// In the middle of a multi-line comment
						coloredString = unEscapeFunction(e.MultiLineComment.Get(line))
					case q.hasSingleLineComment || q.stoppedMultiLineComment:
						// Fix for interpreting URLs in shell scripts as single line comments
						if singleLineCommentMarker != "//" {
							commentColorName = e.Comment
							textWithTags = bytes.ReplaceAll(textWithTags, []byte(":<off><"+commentColorName+">//"), []byte("://"))
							textWithTags = bytes.ReplaceAll(textWithTags, []byte(" "+singleLineCommentMarker), []byte(" <"+commentColorName+">"+singleLineCommentMarker))
						}
						// A single line comment (the syntax module did the highlighting)
						coloredString = unEscapeFunction(tout.DarkTags(string(textWithTags)))
					case !q.startedMultiLineString && q.backtick > 0:
						// A multi-line string
						coloredString = unEscapeFunction(e.MultiLineString.Get(line))
					case (e.mode != mode.HTML && e.mode != mode.XML && e.mode != mode.Markdown && e.mode != mode.Make && e.mode != mode.Just && e.mode != mode.Blank && e.mode != mode.Vibe67) && strings.Contains(line, "->"):
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
					otherCommentMarker = "#"
					if singleLineCommentMarker == "#" {
						otherCommentMarker = "//"
					}
					if strings.HasPrefix(trimmedLine, otherCommentMarker) && !q.containsMultiLineComments && strings.HasPrefix(strings.TrimSpace(string(textWithTags)), "<"+e.Comment+">") {
						parts = strings.SplitN(line, otherCommentMarker, 2)
						commentMarkerString = tout.DarkTags(parts[0] + "<" + e.Dollar + ">" + otherCommentMarker + "<off>")
						theRestString = tout.DarkTags(parts[1])
						if theRestWithTags, err = AsText([]byte(escapeFunction(parts[1])), e.mode); err != nil {
							theRestString = tout.DarkTags(string(theRestWithTags))
						}
						coloredString = unEscapeFunction(commentMarkerString + theRestString)
					}

					// Take an extra pass on coloring the -> arrow, even if it's in a comment
					if !(e.mode == mode.HTML || e.mode == mode.XML || e.mode == mode.Markdown || e.mode == mode.Blank || e.mode == mode.Config || e.mode == mode.Shell || e.mode == mode.Docker || e.mode == mode.Ini || e.mode == mode.Just) && strings.Contains(line, "->") {
						arrowIndex = strings.Index(line, "->")
						arrowBeforeCommentMarker = true

						for _, marker = range commentMarkers {
							commentIndex = strings.Index(line, marker)
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
				if len(coloredString) >= len(cc) {
					cc = make([]vt.CharAttribute, len(coloredString)*2)
				}
				n := tout.ExtractToSlice(coloredString, &cc)
				runesAndAttributes = cc[:n]

				// Also handle things like ZALGO HE COMES which contains unicode "mark" runes
				// TODO: Also handle all languages in v2/test/problematic.desktop
				for k, ra = range runesAndAttributes {
					if unicode.IsMark(ra.R) {
						// Replace the "unprintable" (for a terminal emulator) characters
						runesAndAttributes[k].R = controlRuneReplacement
					}
				}

				e.applyAccentHighlights(line, runesAndAttributes)

				// If e.rainbowParenthesis is true and we're not in a comment or a string, enable rainbow parenthesis
				if e.mode != mode.Git && e.mode != mode.Email && e.rainbowParenthesis && q.None() && !q.hasSingleLineComment && !q.stoppedMultiLineComment {
					thisLineParCount, thisLineBraCount = q.ParBraCount(trimmedLine)
					parCountBeforeThisLine = q.parCount - thisLineParCount
					braCountBeforeThisLine = q.braCount - thisLineBraCount
					if e.rainbowParen(&parCountBeforeThisLine, &braCountBeforeThisLine, &runesAndAttributes, singleLineCommentMarker, ignoreSingleQuotes) == errUnmatchedParenthesis {
						// Don't mark the rest of the parenthesis as wrong, even though this one is
						q.parCount = 0
						q.braCount = 0
					}
				}

				e.pos.mut.Lock()
				skipX := e.pos.offsetX
				e.pos.mut.Unlock()

				matchForAnotherN = 0
				untilNextJumpLetter = 0
				letter = rune(0)
				tx, ty = uint(0), uint(0)

				for runeIndex, ra = range runesAndAttributes {
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
							length = utf8.RuneCountInString(e.searchTerm)
							counter = 0
							match = true
							for i = runeIndex; i < (runeIndex + length); i++ {
								if i >= len(runesAndAttributes) {
									match = false
									break
								}
								ra2 = runesAndAttributes[i]
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
						} else if e.jumpToLetterMode {
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

					if ra.R != '\t' && !e.binaryFile {
						letter = ra.R
						rw = runewidth.RuneWidth(letter)
						if unicode.IsControl(letter) {
							letter = controlRuneReplacement
						} else if rw > 1 { // NOTE: This is a hack to prevent all the text from becoming skewed! Ideally, letter should be drawn, and other text should not be skewed.
							letter = controlRuneReplacement
						}
						tx = cx + lineRuneCount
						ty = cy + uint(y)
						if tx < cw {
							if highlightCurrentLine && (e.highlightCurrentText || e.highlightCurrentLine) {
								c.WriteRuneBNoLock(tx, ty, e.HighlightForeground, e.HighlightBackground, letter)
							} else {
								c.WriteRuneBNoLock(tx, ty, fg, bg, letter)
							}
							lineRuneCount += uint(rw)
						}
					} else {
						c.Write(cx+lineRuneCount, cy+uint(y), fg, e.Background, tabString)
						lineRuneCount += uint(e.indentation.PerTab)
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
		// TODO: This may draw the wrong number of blanks, since lineRuneCount should really be the number of visible glyphs at this point. This is problematic for emojis.
		yp = cy + uint(y)
		xp = cx + lineRuneCount
		if int(cw-lineRuneCount) > 0 {
			if highlightCurrentLine && e.highlightCurrentLine {
				c.WriteRunesB(xp, yp, e.HighlightForeground, e.HighlightBackground, ' ', cw-lineRuneCount)
			} else {
				c.WriteRunesB(xp, yp, e.Foreground, bg, ' ', cw-lineRuneCount)
			}
		}

		if noGUI {
			vt.SetXY(0, yp+1)
		}

		// Draw a dotted line to remind the user of where the N-column limit is
		if (e.showColumnLimit || e.mode == mode.Git) && lineRuneCount <= uint(e.wrapWidth) {
			c.WriteRune(uint(e.wrapWidth), yp, dottedLineColor, bg, wrapMarkerRune)
		}

	}
}

// ArrowReplace can syntax highlight pointer arrows in C and C++ and function arrows in other languages
func (e *Editor) ArrowReplace(s string) string {
	// TODO: Use the function that checks if e.mode is "C-like".
	// TODO: Don't hardcode colors here, introduce theme.CArrow, theme.Arrow and theme.ArrowField instead.
	arrowColor, fieldColor := e.arrowColorNames()
	s = strings.ReplaceAll(s, ">-<", "><off><"+arrowColor+">-<")
	return strings.ReplaceAll(s, ">"+Escape(">"), "><off><"+arrowColor+">"+Escape(">")+"<off><"+fieldColor+">")
}

func (e *Editor) arrowColorNames() (string, string) {
	if e.mode == mode.Arduino || e.mode == mode.C || e.mode == mode.Cpp || e.mode == mode.ObjC || e.mode == mode.Shader {
		return e.cArrowColorNames()
	}
	return e.Star, DefaultTextConfig.Protected
}

func (e *Editor) cArrowColorNames() (string, string) {
	arrowColor := DefaultTextConfig.Class
	fieldColor := DefaultTextConfig.Protected
	if e.Name == "Zulu" {
		arrowColor = "yellow"
		fieldColor = "cyan" // lightcyan
	}
	return arrowColor, fieldColor
}

func (e *Editor) ternaryAccentMode() bool {
	switch e.mode {
	case mode.Arduino, mode.C, mode.Cpp, mode.ObjC, mode.Shader,
		mode.C3, mode.CS, mode.Crystal, mode.D, mode.Dart, mode.Haxe,
		mode.Java, mode.JavaScript, mode.TypeScript, mode.PHP, mode.Perl,
		mode.Ruby, mode.Swift, mode.V:
		return true
	default:
		return false
	}
}

func colorNameToAttribute(colorName string) (vt.AttributeColor, bool) {
	if colorName == "" {
		return 0, false
	}
	if attr, ok := vt.DarkColorMap[colorName]; ok {
		return attr, true
	}
	if attr, ok := vt.DarkColorMap[strings.ToLower(colorName)]; ok {
		return attr, true
	}
	return 0, false
}

func isIdentifierRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (e *Editor) applyAccentHighlights(line string, runesAndAttributes []vt.CharAttribute) {
	arrowColor, _ := e.cArrowColorNames()
	arrowAttr, arrowOK := colorNameToAttribute(arrowColor)
	keywordAttr, keywordOK := colorNameToAttribute(e.Theme.Keyword)
	lineRunes := []rune(line)
	if len(lineRunes) == 0 || len(runesAndAttributes) == 0 {
		return
	}
	max := len(lineRunes)
	if len(runesAndAttributes) < max {
		max = len(runesAndAttributes)
	}
	if e.ternaryAccentMode() {
		if keywordOK {
			for i := 1; i+1 < max; i++ {
				if (lineRunes[i] == '?' || lineRunes[i] == ':') && lineRunes[i-1] == ' ' && lineRunes[i+1] == ' ' {
					runesAndAttributes[i].A = keywordAttr
				}
			}
		}
	}
	if arrowOK && (e.mode == mode.GDScript || e.mode == mode.Python) {
		for i := 0; i < max; i++ {
			if lineRunes[i] != '@' {
				continue
			}
			if i > 0 && isIdentifierRune(lineRunes[i-1]) {
				continue
			}
			if i+1 >= max || !isIdentifierRune(lineRunes[i+1]) {
				continue
			}
			end := i + 2
			for end < max && isIdentifierRune(lineRunes[end]) {
				end++
			}
			for j := i; j < end; j++ {
				runesAndAttributes[j].A = arrowAttr
			}
			i = end - 1
		}
	}
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

// ChopLine takes a string where the tabs have been expanded
// and scrolls it + chops it up for display in the current viewport.
// e.pos.offsetX and the given viewportWidth are respected.
func (e *Editor) ChopLine(line string, viewportWidth int) string {
	var (
		resultRunes []rune
		width       int
		offset      int
		w           int
	)
	// Skip runes until the horizontal offset has been reached
	for i, r := range line {
		w = runewidth.RuneWidth(r)
		if width+w > e.pos.offsetX {
			offset = i
			break
		}
		width += w
	}
	// Collect runes until the viewport width has been reached
	width = 0
	for _, r := range line[offset:] {
		w = runewidth.RuneWidth(r)
		if width+w > viewportWidth {
			break
		}
		resultRunes = append(resultRunes, r)
		width += w
	}
	return string(resultRunes)
}

// LastDataPosition returns the last X index for this line, for the data (does not expand tabs)
// Can be negative, if the line is empty.
func (e *Editor) LastDataPosition(n LineIndex) int {
	return utf8.RuneCountInString(e.Line(n)) - 1
}
