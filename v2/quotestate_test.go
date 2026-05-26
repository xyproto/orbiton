package main

import (
	"testing"

	"github.com/xyproto/mode"
)

func newGoQuoteState(t *testing.T) *QuoteState {
	t.Helper()
	q, err := NewQuoteState("//", mode.Go, false)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func newCQuoteState(t *testing.T) *QuoteState {
	t.Helper()
	q, err := NewQuoteState("//", mode.C, false)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func newPythonQuoteState(t *testing.T) *QuoteState {
	t.Helper()
	q, err := NewQuoteState("#", mode.Python, false)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func TestQuoteStateNoneInitial(t *testing.T) {
	q := newGoQuoteState(t)
	if !q.None() {
		t.Error("expected None() to be true for a fresh QuoteState")
	}
}

func TestQuoteStateDoubleQuote(t *testing.T) {
	q := newGoQuoteState(t)
	q.Process(`x = "hello"`)
	if !q.None() {
		t.Error("expected None() after a complete double-quoted string")
	}
}

func TestQuoteStateDoubleQuoteUnclosed(t *testing.T) {
	q := newGoQuoteState(t)
	q.Process(`x = "hello`)
	if q.None() {
		t.Error("expected NOT None() inside an unclosed double-quoted string")
	}
}

func TestQuoteStateEscapedQuote(t *testing.T) {
	q := newGoQuoteState(t)
	q.Process(`x = "say \"hi\""`)
	if !q.None() {
		t.Error("expected None() after a string with escaped quotes")
	}
}

func TestQuoteStateSingleQuote(t *testing.T) {
	q := newCQuoteState(t)
	q.Process(`c = '{'`)
	if !q.None() {
		t.Error("expected None() after a complete single-quoted char literal")
	}
}

func TestQuoteStateBacktick(t *testing.T) {
	q := newGoQuoteState(t)
	q.Process("s := `hello`")
	if !q.None() {
		t.Error("expected None() after a complete backtick string")
	}
}

func TestQuoteStateBacktickMultiLine(t *testing.T) {
	q := newGoQuoteState(t)
	q.Process("s := `hello")
	if q.None() {
		t.Error("expected NOT None() inside an unclosed backtick string")
	}
	q.Process("world`")
	if !q.None() {
		t.Error("expected None() after closing a multi-line backtick string")
	}
}

func TestQuoteStateSingleLineComment(t *testing.T) {
	q := newGoQuoteState(t)
	q.Process(`x := 1 // this is a comment with }`)
	// After processing, hasSingleLineComment is set but resets on next line
	if q.None() {
		t.Error("expected NOT None() on the line with a single-line comment")
	}
	// Process a new line — single-line comment state resets
	q.Process(`y := 2`)
	if !q.None() {
		t.Error("expected None() on the next line after a single-line comment")
	}
}

func TestQuoteStateMultiLineComment(t *testing.T) {
	q := newGoQuoteState(t)
	q.Process(`x := 1 /* start comment`)
	if q.None() {
		t.Error("expected NOT None() inside a multi-line comment")
	}
	q.Process(`still in comment }`)
	if q.None() {
		t.Error("expected NOT None() still inside the multi-line comment")
	}
	q.Process(`end of comment */ y := 2`)
	if !q.None() {
		t.Error("expected None() after closing a multi-line comment")
	}
}

func TestQuoteStateMultiLineCommentSingleLine(t *testing.T) {
	q := newGoQuoteState(t)
	q.Process(`x := 1 /* a comment */ + 2`)
	if !q.None() {
		t.Error("expected None() after an inline multi-line comment that opens and closes")
	}
}

func TestQuoteStatePythonComment(t *testing.T) {
	q := newPythonQuoteState(t)
	q.Process(`x = 1 # this has a }`)
	if q.None() {
		t.Error("expected NOT None() on a line with a Python comment")
	}
	q.Process(`y = 2`)
	if !q.None() {
		t.Error("expected None() on the next line after a Python comment")
	}
}

func TestQuoteStateBraceInString(t *testing.T) {
	q := newGoQuoteState(t)
	// Process a line with braces inside a string — should not affect brace counting
	q.Process(`fmt.Println("{}")`)
	if !q.None() {
		t.Error("expected None() after a line with braces inside a string")
	}
}

func TestQuoteStateParBraCountSkipsStrings(t *testing.T) {
	q := newGoQuoteState(t)
	par, bra := q.ParBraCount(`x := m["key"]`)
	if bra != 0 {
		t.Errorf("expected bracket count 0 (inside string), got %d", bra)
	}
	if par != 0 {
		t.Errorf("expected paren count 0, got %d", par)
	}
}

func TestQuoteStateParBraCountSkipsComments(t *testing.T) {
	q := newGoQuoteState(t)
	par, bra := q.ParBraCount(`x := 1 // f(a[0])`)
	if par != 0 {
		t.Errorf("expected paren count 0 (inside comment), got %d", par)
	}
	if bra != 0 {
		t.Errorf("expected bracket count 0 (inside comment), got %d", bra)
	}
}

func TestQuoteStateNestedQuotesInComment(t *testing.T) {
	q := newGoQuoteState(t)
	q.Process(`// x = "unclosed`)
	// The double quote inside a comment should NOT affect state
	if q.doubleQuote != 0 {
		t.Errorf("expected doubleQuote to be 0, got %d", q.doubleQuote)
	}
	q.Process(`y := 3`)
	if !q.None() {
		t.Error("expected None() on the line after a comment with quotes")
	}
}

func TestQuoteStateCommentInsideString(t *testing.T) {
	q := newGoQuoteState(t)
	q.Process(`s := "// not a comment"`)
	if !q.None() {
		t.Error("expected None() — // inside a string is not a comment")
	}
	if q.hasSingleLineComment {
		t.Error("expected hasSingleLineComment to be false for // inside a string")
	}
}

func TestQuoteStateMultiLineCommentInsideString(t *testing.T) {
	q := newGoQuoteState(t)
	q.Process(`s := "/* not a comment */"`)
	if !q.None() {
		t.Error("expected None() — /* inside a string is not a comment")
	}
	if q.multiLineComment {
		t.Error("expected multiLineComment to be false for /* inside a string")
	}
}

// TestFindMatchingCloseBraceSkipsStrings tests that brace matching skips braces in strings
func TestFindMatchingCloseBraceSkipsStrings(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.Go

	lines := []string{
		`func foo() {`,
		`    s := "}"`,
		`    return s`,
		`}`,
	}
	for i, line := range lines {
		e.SetLine(LineIndex(i), line)
	}

	result := e.findMatchingCloseBrace(0)
	if result != 3 {
		t.Errorf("expected closing brace on line 3, got %d", result)
	}
}

// TestFindMatchingCloseBraceSkipsSingleLineComment tests that braces in comments are skipped
func TestFindMatchingCloseBraceSkipsSingleLineComment(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.Go

	lines := []string{
		`func foo() {`,
		`    // }`,
		`    x := 1`,
		`}`,
	}
	for i, line := range lines {
		e.SetLine(LineIndex(i), line)
	}

	result := e.findMatchingCloseBrace(0)
	if result != 3 {
		t.Errorf("expected closing brace on line 3, got %d", result)
	}
}

// TestFindMatchingCloseBraceSkipsMultiLineComment tests that braces in multi-line comments are skipped
func TestFindMatchingCloseBraceSkipsMultiLineComment(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.Go

	lines := []string{
		`func foo() {`,
		`    /*`,
		`    }`,
		`    */`,
		`    x := 1`,
		`}`,
	}
	for i, line := range lines {
		e.SetLine(LineIndex(i), line)
	}

	result := e.findMatchingCloseBrace(0)
	if result != 5 {
		t.Errorf("expected closing brace on line 5, got %d", result)
	}
}

// TestFindMatchingCloseBraceSkipsBacktick tests that braces in backtick strings are skipped
func TestFindMatchingCloseBraceSkipsBacktick(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.Go

	lines := []string{
		`func foo() {`,
		"    s := `}",
		"}`",
		`    return s`,
		`}`,
	}
	for i, line := range lines {
		e.SetLine(LineIndex(i), line)
	}

	result := e.findMatchingCloseBrace(0)
	if result != 4 {
		t.Errorf("expected closing brace on line 4, got %d", result)
	}
}

// TestFindMatchingCloseBraceSkipsCharLiteral tests that braces in char literals are skipped
func TestFindMatchingCloseBraceSkipsCharLiteral(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.Go

	lines := []string{
		`func foo() {`,
		`    c := '}'`,
		`    return c`,
		`}`,
	}
	for i, line := range lines {
		e.SetLine(LineIndex(i), line)
	}

	result := e.findMatchingCloseBrace(0)
	if result != 3 {
		t.Errorf("expected closing brace on line 3, got %d", result)
	}
}

// TestFindMatchingCloseBraceNestedBraces tests correct matching with nested braces
func TestFindMatchingCloseBraceNestedBraces(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.Go

	lines := []string{
		`func foo() {`,
		`    if x > 0 {`,
		`        bar()`,
		`    }`,
		`    for i := range items {`,
		`        baz()`,
		`    }`,
		`}`,
	}
	for i, line := range lines {
		e.SetLine(LineIndex(i), line)
	}

	result := e.findMatchingCloseBrace(0)
	if result != 7 {
		t.Errorf("expected closing brace on line 7, got %d", result)
	}
}

// TestFindMatchingCloseBraceJavaScript tests JS with braces in template literals and comments
func TestFindMatchingCloseBraceJavaScript(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.JavaScript

	lines := []string{
		`function render() {`,
		"    const s = `${obj}`",
		`    // }`,
		`    return s`,
		`}`,
	}
	for i, line := range lines {
		e.SetLine(LineIndex(i), line)
	}

	result := e.findMatchingCloseBrace(0)
	if result != 4 {
		t.Errorf("expected closing brace on line 4, got %d", result)
	}
}

// TestFindMatchingCloseBraceCombined tests multiple distractors on the same line
func TestFindMatchingCloseBraceCombined(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.Go

	lines := []string{
		`func foo() {`,
		`    s := "}" + "{"  // } {`,
		`    return s`,
		`}`,
	}
	for i, line := range lines {
		e.SetLine(LineIndex(i), line)
	}

	result := e.findMatchingCloseBrace(0)
	if result != 3 {
		t.Errorf("expected closing brace on line 3, got %d", result)
	}
}
