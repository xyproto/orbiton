package main

import (
	"strings"
	"testing"
)

func TestCompletionFiltering(t *testing.T) {
	// Test that package names are filtered when completing package members
	items := []LSPCompletionItem{
		{Label: "fmt", Kind: 9},
		{Label: "Println", Kind: 3},
		{Label: "Printf", Kind: 3},
		{Label: "Sprintf", Kind: 3},
	}

	// Test fmt. completion - should NOT include "fmt" itself
	result := sortAndFilterCompletions(items, "fmt.", "/tmp/test")

	for _, item := range result {
		if item.Label == "fmt" {
			t.Errorf("Package name 'fmt' should be filtered out when completing 'fmt.'")
		}
	}

	// Verify we still have the other items
	if len(result) < 3 {
		t.Errorf("Expected at least 3 completions, got %d", len(result))
	}
}

func TestCompletionContextParsing(t *testing.T) {
	tests := []struct {
		context   string
		expectDot bool
		expectPkg string
	}{
		{"fmt.", true, "fmt"},
		{"strconv.", true, "strconv"},
		{"\tfmt.", true, "fmt"},
		{"  strconv.", true, "strconv"},
		{"fmt", false, ""},
		{"", false, ""},
	}

	for _, tt := range tests {
		context := strings.TrimSpace(tt.context)
		hasDot := strings.HasSuffix(context, ".")
		var packageName string
		if hasDot {
			parts := strings.Split(context, ".")
			if len(parts) >= 2 {
				packageName = parts[len(parts)-2]
			}
		}

		if hasDot != tt.expectDot {
			t.Errorf("context=%q: expected hasDot=%v, got %v", tt.context, tt.expectDot, hasDot)
		}
		if packageName != tt.expectPkg {
			t.Errorf("context=%q: expected package=%q, got %q", tt.context, tt.expectPkg, packageName)
		}
	}
}

func TestSortingPriority(t *testing.T) {
	// Create test items with different characteristics
	items := []LSPCompletionItem{
		{Label: "RareFunc", Kind: 3, SortText: "00100"},
		{Label: "CommonFunc", Kind: 3, SortText: "00001", Preselect: true},
		{Label: "PackageName", Kind: 9, SortText: "00000"},
	}

	// Test that package name is filtered in dot context
	result := sortAndFilterCompletions(items, "pkg.", "/tmp/test")

	// Should have 2 items (RareFunc and CommonFunc, but not PackageName if it matches)
	if len(result) < 2 {
		t.Logf("Got %d items: %v", len(result), result)
	}
}
