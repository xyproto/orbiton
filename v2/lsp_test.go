package main

import (
	"testing"
)

func TestCompletionFiltering(t *testing.T) {
	tests := []struct {
		name     string
		items    []LSPCompletionItem
		context  string
		filtered string
		minItems int
	}{
		{
			name: "filter package name in dot context",
			items: []LSPCompletionItem{
				{Label: "fmt", Kind: 9},
				{Label: "Println", Kind: 3},
				{Label: "Printf", Kind: 3},
				{Label: "Sprintf", Kind: 3},
			},
			context:  "fmt.",
			filtered: "fmt",
			minItems: 3,
		},
		{
			name: "no filter without dot",
			items: []LSPCompletionItem{
				{Label: "fmt", Kind: 9},
				{Label: "Println", Kind: 3},
			},
			context:  "f",
			filtered: "",
			minItems: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortAndFilterCompletions(tt.items, tt.context, "/tmp/test")

			if tt.filtered != "" {
				for _, item := range result {
					if item.Label == tt.filtered {
						t.Errorf("Label %q should be filtered out", tt.filtered)
					}
				}
			}

			if len(result) < tt.minItems {
				t.Errorf("Expected at least %d completions, got %d", tt.minItems, len(result))
			}
		})
	}
}

func TestSortingPriority(t *testing.T) {
	items := []LSPCompletionItem{
		{Label: "RareFunc", Kind: 3, SortText: "00100"},
		{Label: "CommonFunc", Kind: 3, SortText: "00001", Preselect: true},
		{Label: "StructType", Kind: 22, SortText: "00050"},
	}

	result := sortAndFilterCompletions(items, "test.", "/tmp/test")

	if len(result) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(result))
	}

	if result[0].Label != "CommonFunc" {
		t.Errorf("Expected CommonFunc (preselected) to be first, got %s", result[0].Label)
	}
}
