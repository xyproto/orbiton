// Package syntax provides syntax highlighting for source code, using the same
// approach as Orbiton. It tokenizes input with text/scanner, classifies tokens
// into kinds, and wraps them in color tags that can be converted to ANSI escape
// codes by the vt package. Theme selection is supported via the O_THEME
// (or THEME) environment variable.
package syntax

import (
	"bytes"
	"io"
	"text/scanner"
	"unicode"

	"github.com/sourcegraph/annotate"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// Kind represents a syntax highlighting kind (class) which will be assigned to tokens.
// A syntax highlighting scheme (style) maps text style properties to each token kind.
type Kind uint8

// Supported highlighting kinds
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
	CurlyBracket
	IncludeSystem
)

// TextConfig holds the Text class configuration to be used by annotators when
// highlighting code.
type TextConfig struct {
	AndOr         string
	AngleBracket  string
	AssemblyEnd   string
	Class         string
	Comment       string
	CurlyBracket  string
	Decimal       string
	Dollar        string
	IncludeSystem string
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

// GetClass returns the set class for a given token Kind.
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
	case CurlyBracket:
		return c.CurlyBracket
	case IncludeSystem:
		return c.IncludeSystem
	}
	return ""
}

// Option is a type of the function that can modify
// one or more of the options in the TextConfig structure.
type Option func(*TextConfig)

// Printer implements an interface to render highlighted output
// (see TextPrinter for the implementation of this interface).
type Printer interface {
	Print(w io.Writer, kind Kind, tokText string) error
}

// TextPrinter implements Printer interface and is used to produce
// Text-based highligher.
type TextPrinter TextConfig

// Print is the function that emits highlighted source code using
// <color>...<off> wrapper tags.
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

// Annotate returns an annotation for the given token.
func (a TextAnnotator) Annotate(start int, kind Kind, tokText string) (*annotate.Annotation, error) {
	class := TextConfig(a).GetClass(kind)
	if class != "" {
		left := []byte("<" + class + ">")
		return &annotate.Annotation{
			Start: start, End: start + len(tokText),
			Left: left, Right: []byte("<off>"),
		}, nil
	}
	return nil, nil
}

// DefaultTextConfig provides class names that match the color names of
// textoutput tags: https://github.com/xyproto/textoutput
var DefaultTextConfig = TextConfig{
	AndOr:         "red",
	AngleBracket:  "red",
	AssemblyEnd:   "lightyellow",
	Class:         "white",
	Comment:       "darkgray",
	CurlyBracket:  "red",
	Decimal:       "red",
	Dollar:        "white",
	IncludeSystem: "red",
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

var (
	DefaultDarkTextConfig  = DefaultTextConfig
	DefaultLightTextConfig = TextConfig{
		AndOr:         "black",
		AngleBracket:  "black",
		AssemblyEnd:   "red",
		Class:         "blue",
		Comment:       "gray",
		CurlyBracket:  "black",
		Decimal:       "darkcyan",
		Dollar:        "red",
		IncludeSystem: "blue",
		Keyword:       "blue",
		Literal:       "darkcyan",
		Mut:           "black",
		Plaintext:     "black",
		Private:       "black",
		Protected:     "black",
		Public:        "black",
		Punctuation:   "black",
		Self:          "darkcyan",
		Star:          "black",
		Static:        "black",
		String:        "red",
		Tag:           "black",
		TextAttrName:  "black",
		TextAttrValue: "black",
		TextTag:       "black",
		Type:          "blue",
		Whitespace:    "",
	}
)

func NewTextConfig(light bool) TextConfig {
	if light {
		return DefaultLightTextConfig
	}
	return DefaultDarkTextConfig
}

type Style struct {
	Foreground vt.AttributeColor
	Background vt.AttributeColor
}

type Palette [IncludeSystem + 1]Style

func (p Palette) Style(k Kind) Style {
	if int(k) < len(p) {
		return p[k]
	}
	return Style{}
}

type Theme struct {
	Name    string
	Light   bool
	Palette Palette
}

func (t Theme) Style(k Kind) Style {
	return t.Palette.Style(k)
}

func NewTheme(name string, light bool, mapping map[Kind]Style) Theme {
	var pal Palette
	for k, s := range mapping {
		if int(k) < len(pal) {
			pal[k] = s
		}
	}
	return Theme{
		Name:    name,
		Light:   light,
		Palette: pal,
	}
}

var (
	DefaultDarkTheme = NewTheme("DefaultDark", false, map[Kind]Style{
		Whitespace:    {},
		Plaintext:     {Foreground: vt.LightGray},
		Keyword:       {Foreground: vt.Red},
		Type:          {Foreground: vt.LightGray},
		String:        {Foreground: vt.White},
		Comment:       {Foreground: vt.DarkGray},
		Decimal:       {Foreground: vt.Red},
		Punctuation:   {Foreground: vt.Red},
		CurlyBracket:  {Foreground: vt.Red},
		IncludeSystem: {Foreground: vt.Red},
	})

	DefaultLightTheme = NewTheme("DefaultLight", true, map[Kind]Style{
		Whitespace:    {},
		Plaintext:     {Foreground: vt.Black},
		Keyword:       {Foreground: vt.Blue},
		Type:          {Foreground: vt.Blue},
		String:        {Foreground: vt.Red},
		Comment:       {Foreground: vt.Gray},
		Decimal:       {Foreground: vt.Cyan},
		Punctuation:   {Foreground: vt.Black},
		CurlyBracket:  {Foreground: vt.Black},
		IncludeSystem: {Foreground: vt.Blue},
	})

	VSLightTheme = NewTheme("VSLight", true, map[Kind]Style{
		Whitespace:    {},
		Plaintext:     {Foreground: vt.TrueColor(0x00, 0x00, 0x00)},
		Keyword:       {Foreground: vt.TrueColor(0x00, 0x00, 0xff)},
		Type:          {Foreground: vt.TrueColor(0x26, 0x7f, 0x99)},
		String:        {Foreground: vt.TrueColor(0xa3, 0x15, 0x15)},
		Comment:       {Foreground: vt.TrueColor(0x00, 0x80, 0x00)},
		Decimal:       {Foreground: vt.TrueColor(0x09, 0x86, 0x58)},
		Punctuation:   {Foreground: vt.TrueColor(0x00, 0x00, 0x00)},
		CurlyBracket:  {Foreground: vt.TrueColor(0x00, 0x00, 0x00)},
		IncludeSystem: {Foreground: vt.TrueColor(0xa3, 0x15, 0x15)},
	})
)

type Styler interface {
	Style(Kind) Style
}

type DirectPrinter struct {
	Scheme Styler
}

func (dp DirectPrinter) Print(w io.Writer, kind Kind, tokText string) error {
	st := dp.Scheme.Style(kind)
	if st.Foreground != 0 {
		if _, err := io.WriteString(w, st.Foreground.String()); err != nil {
			return err
		}
	}
	if st.Background != 0 {
		if _, err := io.WriteString(w, st.Background.String()); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, tokText); err != nil {
		return err
	}
	if st.Foreground != 0 || st.Background != 0 {
		if _, err := io.WriteString(w, vt.NoColor); err != nil {
			return err
		}
	}
	return nil
}

func Highlight(src []byte, m mode.Mode, s Styler, w io.Writer) error {
	scan := NewScanner(src)
	dp := DirectPrinter{Scheme: s}
	inComment := false
	inInclude := false
	for tok := scan.Scan(); tok != scanner.EOF; tok = scan.Scan() {
		tokText := scan.TokenText()
		if err := dp.Print(w, tokenKind(tok, tokText, &inComment, &inInclude, m), tokText); err != nil {
			return err
		}
	}
	return nil
}

func HighlightToAttributes(src []byte, m mode.Mode, s Styler) []vt.CharAttribute {
	scan := NewScanner(src)
	inComment := false
	inInclude := false
	var attrs []vt.CharAttribute
	for tok := scan.Scan(); tok != scanner.EOF; tok = scan.Scan() {
		tokText := scan.TokenText()
		kind := tokenKind(tok, tokText, &inComment, &inInclude, m)
		st := s.Style(kind)
		color := st.Foreground
		if st.Background != 0 {
			color = color.Combine(st.Background)
		}
		for _, r := range tokText {
			attrs = append(attrs, vt.CharAttribute{
				R: r,
				A: color,
			})
		}
	}
	return attrs
}

func AsTextWithBackground(src []byte, m mode.Mode, light bool, options ...Option) ([]byte, error) {
	opt := NewTextConfig(light)
	for _, f := range options {
		f(&opt)
	}
	var buf bytes.Buffer
	if err := Print(NewScanner(src), &buf, TextPrinter(opt), m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Print scans tokens from s, using Printer p for mode m.
func Print(s *scanner.Scanner, w io.Writer, p Printer, m mode.Mode) error {
	switch m {
	case mode.C3:
		s.IsIdentRune = func(ch rune, i int) bool {
			return ch == '$' || ch == '@' || ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch) && i > 0
		}
	case mode.Clojure, mode.Lisp:
		s.IsIdentRune = func(ch rune, i int) bool {
			return ch == '*' || ch == '-' || ch == '+' || ch == '/' || ch == '?' || ch == '!' || ch == '.' || ch == ':' || ch == '&' || ch == '<' || ch == '>' || ch == '=' || ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch) && i > 0
		}
	case mode.Shell, mode.Fish, mode.Oil, mode.Make, mode.Just:
		s.IsIdentRune = func(ch rune, i int) bool {
			return ch == '-' || ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch) && i > 0
		}
	case mode.Spec:
		s.IsIdentRune = func(ch rune, i int) bool {
			return ch == '%' || ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch) && i > 0
		}
	case mode.Swift:
		s.IsIdentRune = func(ch rune, i int) bool {
			return ch == '#' || ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch) && i > 0
		}
	case mode.Tim:
		s.IsIdentRune = func(ch rune, i int) bool {
			return ch == '&' || ch == '<' || ch == '>' || ch == '^' || ch == '|' || ch == '~' || ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch) && i > 0
		}
	}
	inComment := false
	inInclude := false
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		tokText := s.TokenText()
		if err := p.Print(w, tokenKind(tok, tokText, &inComment, &inInclude, m), tokText); err != nil {
			return err
		}
	}
	return nil
}

// Annotate tokenizes src and returns annotations for mode m.
func Annotate(src []byte, a Annotator, m mode.Mode) (annotate.Annotations, error) {
	var (
		anns      annotate.Annotations
		s         = NewScanner(src)
		read      = 0
		inComment = false
		inInclude = false
		tok       = s.Scan()
	)
	for tok != scanner.EOF {
		tokText := s.TokenText()
		ann, err := a.Annotate(read, tokenKind(tok, tokText, &inComment, &inInclude, m), tokText)
		if err != nil {
			return nil, err
		}
		read += len(tokText)
		if ann != nil {
			anns = append(anns, ann)
		}
		tok = s.Scan()
	}
	return anns, nil
}

// AsText converts source code into a Text-highlighted version.
// It accepts optional configuration parameters to control rendering.
func AsText(src []byte, m mode.Mode, options ...Option) ([]byte, error) {
	opt := DefaultTextConfig
	for _, f := range options {
		f(&opt)
	}
	var buf bytes.Buffer
	if err := Print(NewScanner(src), &buf, TextPrinter(opt), m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// NewScanner is a helper that takes a []byte src, wraps it in a reader and creates a Scanner.
func NewScanner(src []byte) *scanner.Scanner {
	return NewScannerReader(bytes.NewReader(src))
}

// NewScannerReader takes a reader src and creates a Scanner.
func NewScannerReader(src io.Reader) *scanner.Scanner {
	var s scanner.Scanner
	s.Init(src)
	s.Error = func(_ *scanner.Scanner, _ string) {}
	s.Whitespace = 0
	s.Mode = s.Mode ^ scanner.SkipComments
	return &s
}
