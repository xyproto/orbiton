package main

import (
	"strings"
	"testing"
)

// TestBookWrapBodyConsecutiveSpaces verifies that inserting a space next to
// an existing wrap-boundary space produces a visible change in the wrapped
// output. Previously both spaces would be absorbed into the invisible
// trailing region of the preceding row, so pressing space on the second
// sub-row of a soft-wrapped line looked like "nothing happened".
func TestBookWrapBodyConsecutiveSpaces(t *testing.T) {
	before := "one two three four five six seven eight nine ten"
	// Simulate inserting a single space right before "five" (between the
	// existing wrap-boundary space and the word).
	insertAt := strings.Index(before, "five")
	after := before[:insertAt] + " " + before[insertAt:]

	for availW := 20; availW <= 30; availW++ {
		segsBefore := bookWrapBody(before, availW)
		segsAfter := bookWrapBody(after, availW)
		// The wrapped output as concatenated row text must differ
		// (ignoring the trailing-space of any row, which is invisible
		// on a terminal).
		joinVisible := func(segs []wrapSegment) string {
			var b strings.Builder
			for i, s := range segs {
				if i > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(strings.TrimRight(s.text, " "))
			}
			return b.String()
		}
		vb := joinVisible(segsBefore)
		va := joinVisible(segsAfter)
		if vb == va {
			t.Errorf("availW=%d: inserting a space at a wrap boundary produced no visible change.\nbefore:\n%s\nafter:\n%s", availW, vb, va)
		}
	}
}
