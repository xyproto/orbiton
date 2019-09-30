// Package syntax provides syntax highlighting for code. It currently
// uses a language-independent lexer and performs decently on JavaScript, Java,
// Ruby, Python, Go, and C.
package syntax

import (
	"bytes"
	"io"
	"text/scanner"
	"unicode"
	"unicode/utf8"

	"github.com/sourcegraph/annotate"
)

// Kind represents a syntax highlighting kind (class) which will be assigned to tokens.
// A syntax highlighting scheme (style) maps text style properties to each token kind.
type Kind uint8

// A set of supported highlighting kinds
const (
	Whitespace Kind = iota
	String
	Keyword
	Comment
	Type
	Literal
	Punctuation
	Plaintext
	Tag
	TextTag
	TextAttrName
	TextAttrValue
	Decimal
)

//go:generate gostringer -type=Kind

// Printer implements an interface to render highlighted output
// (see TextPrinter for the implementation of this interface)
type Printer interface {
	Print(w io.Writer, kind Kind, tokText string) error
}

// TextConfig holds the Text class configuration to be used by annotators when
// highlighting code.
type TextConfig struct {
	String        string
	Keyword       string
	Comment       string
	Type          string
	Literal       string
	Punctuation   string
	Plaintext     string
	Tag           string
	TextTag       string
	TextAttrName  string
	TextAttrValue string
	Decimal       string
	Whitespace    string
}

// TextPrinter implements Printer interface and is used to produce
// Text-based highligher
type TextPrinter TextConfig

// Class returns the set class for a given token Kind.
func (c TextConfig) Class(kind Kind) string {
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
	}
	return ""
}

// Print is the function that emits highlighted source code using
// <color>...<off> wrapper tags
func (p TextPrinter) Print(w io.Writer, kind Kind, tokText string) error {
	class := ((TextConfig)(p)).Class(kind)
	if class != "" {
		_, err := w.Write([]byte(`<`))
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, class)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(`>`))
		if err != nil {
			return err
		}
	}
	w.Write([]byte(tokText))
	if class != "" {
		_, err := w.Write([]byte(`<off>`))
		if err != nil {
			return err
		}
	}
	return nil
}

type Annotator interface {
	Annotate(start int, kind Kind, tokText string) (*annotate.Annotation, error)
}

type TextAnnotator TextConfig

func (a TextAnnotator) Annotate(start int, kind Kind, tokText string) (*annotate.Annotation, error) {
	class := ((TextConfig)(a)).Class(kind)
	if class != "" {
		left := []byte(`<`)
		left = append(left, []byte(class)...)
		left = append(left, []byte(`>`)...)
		return &annotate.Annotation{
			Start: start, End: start + len(tokText),
			Left: left, Right: []byte("<off>"),
		}, nil
	}
	return nil, nil
}

// Option is a type of the function that can modify
// one or more of the options in the TextConfig structure.
type Option func(options *TextConfig)

// DefaultTextConfig provides class names that match the color names of
// textoutput tags: https://github.com/xyproto/textoutput
var DefaultTextConfig = TextConfig{
	String:        "lightwhite",
	Keyword:       "red",
	Comment:       "darkgray",
	Type:          "white",
	Literal:       "white",
	Punctuation:   "red",
	Plaintext:     "white",
	Tag:           "white",
	TextTag:       "white",
	TextAttrName:  "white",
	TextAttrValue: "white",
	Decimal:       "red",
	Whitespace:    "",
}

func Print(s *scanner.Scanner, w io.Writer, p Printer) error {
	tok := s.Scan()
	for tok != scanner.EOF {
		tokText := s.TokenText()
		err := p.Print(w, tokenKind(tok, tokText), tokText)
		if err != nil {
			return err
		}

		tok = s.Scan()
	}

	return nil
}

func Annotate(src []byte, a Annotator) (annotate.Annotations, error) {
	s := NewScanner(src)

	var anns annotate.Annotations
	read := 0

	tok := s.Scan()
	for tok != scanner.EOF {
		tokText := s.TokenText()

		ann, err := a.Annotate(read, tokenKind(tok, tokText), tokText)
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

// AsText converts source code into an Text-highlighted version;
// It accepts optional configuration parameters to control rendering
// (see OrderedList as one example)
func AsText(src []byte, options ...Option) ([]byte, error) {
	opt := DefaultTextConfig
	for _, f := range options {
		f(&opt)
	}

	var buf bytes.Buffer
	err := Print(NewScanner(src), &buf, TextPrinter(opt))
	if err != nil {
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

func tokenKind(tok rune, tokText string) Kind {
	switch tok {
	case scanner.Ident:
		if _, isKW := keywords[tokText]; isKW {
			return Keyword
		}
		if r, _ := utf8.DecodeRuneInString(tokText); unicode.IsUpper(r) {
			return Type
		}
		return Plaintext
	case scanner.Float, scanner.Int:
		return Decimal
	case scanner.Char, scanner.String, scanner.RawString:
		return String
	case scanner.Comment:
		return Comment
	}
	if unicode.IsSpace(tok) {
		return Whitespace
	}
	return Punctuation
}
