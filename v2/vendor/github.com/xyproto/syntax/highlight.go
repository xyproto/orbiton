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
	"github.com/xyproto/mode"
)

// Kind represents a syntax highlighting kind (class) which will be assigned to tokens.
// A syntax highlighting scheme (style) maps text style properties to each token kind.
type Kind uint8

// A set of supported highlighting kinds
const (
	Whitespace Kind = iota
	AndOr
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

//go:generate gostringer -type=Kind

// Printer implements an interface to render highlighted output
// (see TextPrinter for the implementation of this interface)
type Printer interface {
	Print(w io.Writer, kind Kind, tokText string) error
}

// TextConfig holds the Text class configuration to be used by annotators when
// highlighting code.
type TextConfig struct {
	AndOr         string
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

// TextPrinter implements Printer interface and is used to produce
// Text-based highligher
type TextPrinter TextConfig

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

// Print is the function that emits highlighted source code using
// <color>...<off> wrapper tags
func (p TextPrinter) Print(w io.Writer, kind Kind, tokText string) (err error) {
	class := ((TextConfig)(p)).GetClass(kind)
	if class != "" {
		_, err = io.WriteString(w, "<")
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, class)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, ">")
		if err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, tokText)
	if err != nil {
		return err
	}
	if class != "" {
		_, err = io.WriteString(w, "<off>")
	}
	return err // can be nil
}

type Annotator interface {
	Annotate(start int, kind Kind, tokText string) (*annotate.Annotation, error)
}

type TextAnnotator TextConfig

func (a TextAnnotator) Annotate(start int, kind Kind, tokText string) (*annotate.Annotation, error) {
	class := ((TextConfig)(a)).GetClass(kind)
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
	AndOr:         "red",
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

func Print(s *scanner.Scanner, w io.Writer, p Printer, m mode.Mode) error {
	tok := s.Scan()
	inSingleLineComment := false
	for tok != scanner.EOF {
		tokText := s.TokenText()
		err := p.Print(w, tokenKind(tok, tokText, &inSingleLineComment, m), tokText)
		if err != nil {
			return err
		}

		tok = s.Scan()
	}

	return nil
}

func Annotate(src []byte, a Annotator, m mode.Mode) (annotate.Annotations, error) {
	var (
		anns                annotate.Annotations
		s                   = NewScanner(src)
		read                = 0
		inSingleLineComment = false
		tok                 = s.Scan()
	)
	for tok != scanner.EOF {
		tokText := s.TokenText()
		ann, err := a.Annotate(read, tokenKind(tok, tokText, &inSingleLineComment, m), tokText)
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
func AsText(src []byte, m mode.Mode, options ...Option) ([]byte, error) {
	opt := DefaultTextConfig
	for _, f := range options {
		f(&opt)
	}

	var buf bytes.Buffer
	err := Print(NewScanner(src), &buf, TextPrinter(opt), m)
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

func tokenKind(tok rune, tokText string, inSingleLineComment *bool, m mode.Mode) Kind {

	// Check if we are in a bash-style single line comment, this probably needs to check for even more languages
	if (m == mode.Assembly && tok == ';') || (m != mode.Assembly && m != mode.GoAssembly && m != mode.Clojure && m != mode.Lisp && m != mode.C && m != mode.Cpp && m != mode.Lua && tok == '#') {
		*inSingleLineComment = true
	} else if tok == '\n' {
		*inSingleLineComment = false
	}

	// Check if this is #include or #define
	if (m == mode.C || m == mode.Cpp) && (tokText == "include" || tokText == "define" || tokText == "ifdef" || tokText == "ifndef" || tokText == "endif" || tokText == "else" || tokText == "elif") {
		*inSingleLineComment = false
		return Keyword
	}

	// If we are, return the Comment kind
	if *inSingleLineComment {
		return Comment
	}

	// Check if this is the "as" or "mut" keyword, for Rust
	if m == mode.Rust {
		switch tokText {
		case "as":
			return Type // re-use color
		case "mut":
			return Mut
		}
	}

	// Check if this is the "self" keyword, for Python
	if m == mode.Python && tokText == "self" {
		return Self
	}

	// If not, do the regular switch
	switch tok {
	case scanner.Ident:
		if _, isKW := Keywords[tokText]; isKW {
			return Keyword
		}
		switch tokText {
		case "private":
			return Private
		case "public":
			return Public
		case "protected":
			return Protected
		case "class":
			return Class
		case "static":
			return Static
		case "JMP", "jmp", "LEAVE", "leave", "RET", "ret", "CALL", "call":
			if m == mode.Assembly || m == mode.GoAssembly {
				return AssemblyEnd
			}
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
	if tok == '&' || tok == '|' {
		return AndOr
	} else if tok == '*' {
		return Star
	} else if tok == '$' {
		return Dollar
	}
	if unicode.IsSpace(tok) {
		return Whitespace
	}
	return Punctuation
}
