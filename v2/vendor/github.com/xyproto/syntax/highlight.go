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
	case mode.Shell, mode.Make, mode.Just:
		s.IsIdentRune = func(ch rune, i int) bool {
			return ch == '-' || ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch) && i > 0
		}
	case mode.Swift:
		s.IsIdentRune = func(ch rune, i int) bool {
			return ch == '#' || ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch) && i > 0
		}
	case mode.Vibe67:
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
