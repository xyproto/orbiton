package main

import (
	"testing"

	"github.com/xyproto/mode"
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
			name: "filter by prefix without dot",
			items: []LSPCompletionItem{
				{Label: "fmt", Kind: 9},
				{Label: "Println", Kind: 3},
			},
			context:  "f",
			filtered: "",
			minItems: 1, // Only "fmt" should match prefix "f"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortAndFilterCompletions(tt.items, tt.context, "/tmp/test", []string{".go"}, mode.Go)

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

	result := sortAndFilterCompletions(items, "test.", "/tmp/test", []string{".go"}, mode.Go)

	if len(result) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(result))
	}

	if result[0].Label != "CommonFunc" {
		t.Errorf("Expected CommonFunc (preselected) to be first, got %s", result[0].Label)
	}
}

func TestCompletionPrefix(t *testing.T) {
	for _, tc := range []struct {
		line   string
		owner  string
		prefix string
		m      mode.Mode
		member bool
	}{
		// a dot earlier on the line must not swallow the prefix
		{"  environment.systemPackages = with pkgs; [ hel", "", "hel", mode.Nix, false},
		{"  environment.systemPackages = with pkgs; [ pkgs.hel", "pkgs", "hel", mode.Nix, true},
		{"  services.openssh.", "openssh", "", mode.Nix, true},
		// Nix attribute names may contain dashes and single quotes
		{"    pkgs.nix-pre", "pkgs", "nix-pre", mode.Nix, true},
		{"    buildPhase'", "", "buildPhase'", mode.Nix, false},
		// other languages keep dashes out of identifiers
		{"foo->ba", "foo", "ba", mode.C, true},
		{"x = a - b", "", "b", mode.C, false},
		{"std::vec", "std", "vec", mode.Cpp, true},
		{"obj.field.me", "field", "me", mode.Go, true},
		{"", "", "", mode.Go, false},
		{"   ", "", "", mode.Go, false},
	} {
		prefix, member, owner := completionPrefix(tc.line, tc.m)
		if prefix != tc.prefix || member != tc.member || owner != tc.owner {
			t.Errorf("completionPrefix(%q, %v) = (%q, %v, %q), want (%q, %v, %q)",
				tc.line, tc.m, prefix, member, owner, tc.prefix, tc.member, tc.owner)
		}
	}
}

func TestIsMemberAccess(t *testing.T) {
	for _, tc := range []struct {
		line string
		want bool
	}{
		{"foo.ba", true},
		{"foo->ba", true},
		{"foo.", true},
		{"printf", false},
		{"a = b + c", false},
	} {
		if got := isMemberAccess(tc.line); got != tc.want {
			t.Errorf("isMemberAccess(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

// an earlier dot on the line must not break filtering
func TestNixCompletionFiltering(t *testing.T) {
	items := []LSPCompletionItem{
		{Label: "hello", Kind: lspKindVariable},
		{Label: "helix", Kind: lspKindVariable},
		{Label: "zlib", Kind: lspKindVariable},
	}
	result := sortAndFilterCompletions(items, "  environment.systemPackages = with pkgs; [ hel", "/tmp/test", []string{".nix"}, mode.Nix)
	if len(result) != 2 {
		t.Fatalf("Expected 2 completions matching \"hel\", got %d: %v", len(result), result)
	}
	for _, item := range result {
		if item.Label == "zlib" {
			t.Errorf("zlib should not match the prefix \"hel\"")
		}
	}
}
