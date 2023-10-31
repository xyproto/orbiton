package main

import (
	"testing"
)

func TestCapitalizeWords(t *testing.T) {
	if capitalizeWords("bob john") != "Bob John" {
		t.Fail()
	}
}

func TestWithinBackticks(t *testing.T) {
	tests := []struct {
		line     string
		what     string
		expected bool
	}{
		{
			line:     "This is a test `hello` string",
			what:     "hello",
			expected: true,
		},
		{
			line:     "This is a test `hello world` string",
			what:     "world",
			expected: true,
		},
		{
			line:     "This is a test `hello` string",
			what:     "world",
			expected: false,
		},
		{
			line:     "This is a test `hello world` string",
			what:     "hello world",
			expected: true,
		},
		{
			line:     "This is a test `hello world` string",
			what:     "test",
			expected: false,
		},
		{
			line:     "`hello` world",
			what:     "hello",
			expected: true,
		},
		{
			line:     "`hello` world",
			what:     "world",
			expected: false,
		},
		{
			line:     "This is `a` test",
			what:     "a",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := withinBackticks(tt.line, tt.what); got != tt.expected {
				t.Errorf("withinBackticks(%q, %q) = %v; expected %v", tt.line, tt.what, got, tt.expected)
			}
		})
	}
}
