package main

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildErrorJumpedToSource(t *testing.T) {
	if buildErrorJumpedToSource(errors.New("plain")) {
		t.Fatal("expected plain errors to report no jump")
	}

	err := newBuildError("compile failure", true)
	if !buildErrorJumpedToSource(err) {
		t.Fatal("expected build error to report jump")
	}
}

func TestBuildErrorExplanationPrompt(t *testing.T) {
	prompt := buildErrorExplanationPrompt(
		"fn main() {\n    println!(greeting);\n}",
		3,
		`println!(greeting);`,
		"main.rs: format argument must be a string literal",
	)

	expected := []string{
		"For this function:",
		"fn main() {",
		"The user is currently looking at line 3:",
		"println!(greeting);",
		"main.rs: format argument must be a string literal",
		"Use at most 4 short lines.",
		"Use plain text only (no Markdown).",
	}
	for _, fragment := range expected {
		if !strings.Contains(prompt, fragment) {
			t.Fatalf("expected prompt to contain %q", fragment)
		}
	}
}

func TestTrimExplanationToMaxLines(t *testing.T) {
	input := "line1\n\nline2\nline3\nline4\nline5\nline6\n"
	got := trimExplanationToMaxLines(input, 4)
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	if lines[0] != "line1" || lines[1] != "" || lines[3] != "line3" {
		t.Fatalf("unexpected trimmed lines: %v", lines)
	}
}

func TestSanitizeOllamaTextRemovesFenceLines(t *testing.T) {
	input := "hello\n```rust\nlet x = 1;\n```\nworld\n  ```go"
	got := sanitizeOllamaText(input)
	lines := strings.Split(got, "\n")
	if len(lines) != 6 {
		t.Fatalf("unexpected number of lines: %d", len(lines))
	}
	if lines[1] != "" || lines[3] != "" || lines[5] != "" {
		t.Fatalf("expected code fence lines to be blank, got %q", got)
	}
}

func TestBuildErrorExplanationPendingState(t *testing.T) {
	clearBuildErrorExplanationState()
	if hasBuildErrorExplanation() {
		t.Fatal("expected no active build error explanation")
	}

	setBuildErrorExplanationPending()
	if !hasBuildErrorExplanation() {
		t.Fatal("expected active build error explanation while waiting for Ollama")
	}
	if !hasBuildErrorExplanationThinking() {
		t.Fatal("expected build error explanation to be thinking while waiting for Ollama")
	}

	clearBuildErrorExplanationState()
	if hasBuildErrorExplanation() {
		t.Fatal("expected cleared build error explanation state")
	}
	if hasBuildErrorExplanationThinking() {
		t.Fatal("expected build error explanation thinking state to be cleared")
	}
}

func TestSetBuildErrorExplanationIgnoredWhenInactive(t *testing.T) {
	clearBuildErrorExplanationState()
	setBuildErrorExplanation("demo", "text that should be ignored")

	if hasBuildErrorExplanation() {
		t.Fatal("expected inactive build error state to remain inactive")
	}
}
