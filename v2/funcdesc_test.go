package main

import (
	"testing"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

func TestFunctionRangeForCurrentFunctionBraceBased(t *testing.T) {
	clearFunctionDescriptionState()
	t.Cleanup(clearFunctionDescriptionState)

	e := NewSimpleEditor(80)
	e.mode = mode.Go

	lines := []string{
		"func demo() {",
		"    a := 1",
		"    if a > 0 {",
		"        a++",
		"    }",
		"}",
		"",
		"func other() {",
		"}",
	}
	for i, line := range lines {
		e.SetLine(LineIndex(i), line)
	}

	e.pos.SetY(2)
	start, end, ok := e.functionRangeForCurrentFunction("demo")
	if !ok {
		t.Fatal("expected function range to be found")
	}
	if start != 0 {
		t.Fatalf("expected start line 0, got %d", start)
	}
	if end != 5 {
		t.Fatalf("expected end line 5, got %d", end)
	}
}

func TestPreferredDescriptionBoxYStaysStableWhenCursorMoves(t *testing.T) {
	clearFunctionDescriptionState()
	t.Cleanup(clearFunctionDescriptionState)

	e := NewSimpleEditor(80)
	e.mode = mode.Go

	lines := []string{
		"func demo() {",
		"    a := 1",
		"    if a > 0 {",
		"        a++",
		"    }",
		"}",
	}
	for i, line := range lines {
		e.SetLine(LineIndex(i), line)
	}

	setCurrentDescribedFunction("demo", 0, 5, true)
	c := vt.NewCanvas()

	e.pos.SetY(1)
	firstY := e.preferredDescriptionBoxY(c, "demo", 4, 2)

	e.pos.SetY(4)
	secondY := e.preferredDescriptionBoxY(c, "demo", 4, 2)

	if secondY != firstY {
		t.Fatalf("expected stable box Y, got %d then %d", firstY, secondY)
	}
}
