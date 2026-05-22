package main

import (
	"testing"
)

func TestRSTHighlightHelpers(t *testing.T) {
	// Test rstAdornmentLine
	tests := []struct {
		line string
		want bool
	}{
		{"=====", true},
		{"-----", true},
		{"~~~~~", true},
		{"^^", true},
		{"a", false},
		{"=-=", false},
		{"", false},
		{"=", false},
	}
	for _, tt := range tests {
		if got := rstAdornmentLine(tt.line); got != tt.want {
			t.Errorf("rstAdornmentLine(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}

	// Test rstDirective
	dirTests := []struct {
		line string
		want bool
	}{
		{".. code-block:: python", true},
		{".. image:: path/to/img.png", true},
		{".. note::", true},
		{".. This is a comment", false},
		{"..", false},
		{"not a directive", false},
	}
	for _, tt := range dirTests {
		if got := rstDirective(tt.line); got != tt.want {
			t.Errorf("rstDirective(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}

	// Test rstComment
	commentTests := []struct {
		line string
		want bool
	}{
		{".. This is a comment", true},
		{"..", true},
		{".. code-block:: python", false},
		{"not a comment", false},
	}
	for _, tt := range commentTests {
		if got := rstComment(tt.line); got != tt.want {
			t.Errorf("rstComment(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}

	// Test rstFieldList
	fieldTests := []struct {
		line string
		want bool
	}{
		{":param: description", true},
		{":type: str", true},
		{":", false},
		{"not a field", false},
		{"::", false},
	}
	for _, tt := range fieldTests {
		if got := rstFieldList(tt.line); got != tt.want {
			t.Errorf("rstFieldList(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}

	// Test rstListItem
	listTests := []struct {
		line string
		want bool
	}{
		{"* item", true},
		{"- item", true},
		{"+ item", true},
		{"1. item", true},
		{"#. item", true},
		{"1) item", true},
		{"(1) item", true},
		{"not a list", false},
	}
	for _, tt := range listTests {
		if got := rstListItem(tt.line); got != tt.want {
			t.Errorf("rstListItem(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}
