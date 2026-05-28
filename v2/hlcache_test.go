package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/xyproto/mode"
	"github.com/xyproto/themes"
	"github.com/xyproto/vt"
)

// BenchmarkWriteLinesLargeFileTop benchmarks WriteLines near the top of a large Go file
func BenchmarkWriteLinesLargeFileTop(b *testing.B) {
	e := newLargeBenchEditor(50000, mode.Go)
	c := vt.NewCanvasWithSize(80, 40)
	// Scroll near the top
	e.pos.SetOffsetY(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.WriteLines(c, 100, 140, 0, 0, false, true)
	}
}

// BenchmarkWriteLinesLargeFileBottom benchmarks WriteLines near the bottom of a large Go file
func BenchmarkWriteLinesLargeFileBottom(b *testing.B) {
	e := newLargeBenchEditor(50000, mode.Go)
	c := vt.NewCanvasWithSize(80, 40)
	// Scroll near the bottom
	e.pos.SetOffsetY(49900)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.WriteLines(c, 49900, 49940, 0, 0, false, true)
	}
}

// BenchmarkWriteLinesLargeMarkdownBottom benchmarks WriteLines near the bottom of a large Markdown file
func BenchmarkWriteLinesLargeMarkdownBottom(b *testing.B) {
	e := newLargeBenchEditor(50000, mode.Markdown)
	c := vt.NewCanvasWithSize(80, 40)
	e.pos.SetOffsetY(49900)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.WriteLines(c, 49900, 49940, 0, 0, false, true)
	}
}

// newLargeBenchEditor creates an editor with numLines lines for benchmarking
func newLargeBenchEditor(numLines int, m mode.Mode) *Editor {
	t := themes.NewDefaultTheme()
	e := NewCustomEditor(mode.DefaultTabsSpaces, 1, m, t, true, false, false, false, false, false, false)
	e.syntaxHighlight = true

	// Generate representative content based on mode
	for i := range numLines {
		var line string
		switch m {
		case mode.Go:
			switch {
			case i%50 == 0:
				line = fmt.Sprintf("func example%d() {", i/50)
			case i%50 == 49:
				line = "}"
			case i%10 == 5:
				line = fmt.Sprintf("\t// This is a comment on line %d", i)
			case i%20 == 15:
				line = fmt.Sprintf("\ts := \"string with { braces } on line %d\"", i)
			default:
				line = fmt.Sprintf("\tx := %d", i)
			}
		case mode.Markdown:
			switch {
			case i%100 == 0:
				line = fmt.Sprintf("# Heading %d", i/100)
			case i%100 == 10:
				line = "```go"
			case i%100 == 20:
				line = "```"
			default:
				line = fmt.Sprintf("Some text on line %d with %s content", i, strings.Repeat("word ", 5))
			}
		default:
			line = fmt.Sprintf("line %d content", i)
		}
		e.SetLine(LineIndex(i), line)
	}
	// Reset changed flag since we just populated the editor
	e.changed.Store(false)
	// Clear cache so benchmarks start fresh
	if e.hlCache != nil {
		e.hlCache.Invalidate()
	}
	return e
}

// TestHighlightCacheCorrectness verifies that rendering with a warmed cache
// produces identical output to rendering from scratch (no cache)
func TestHighlightCacheCorrectness(t *testing.T) {
	for _, m := range []mode.Mode{mode.Go, mode.Markdown, mode.Python} {
		t.Run(m.String(), func(t *testing.T) {
			e := newLargeBenchEditor(5000, m)
			c1 := vt.NewCanvasWithSize(80, 40)
			c2 := vt.NewCanvasWithSize(80, 40)

			offset := LineIndex(4900)

			// Render without cache (fresh state)
			e.hlCache.Invalidate()
			e.WriteLines(c1, offset, offset+40, 0, 0, false, true)
			var buf1 bytes.Buffer
			if err := c1.Snapshot(&buf1); err != nil {
				t.Fatal(err)
			}

			// Render again — this time the cache should be populated
			e.WriteLines(c2, offset, offset+40, 0, 0, false, true)
			var buf2 bytes.Buffer
			if err := c2.Snapshot(&buf2); err != nil {
				t.Fatal(err)
			}

			if buf1.String() != buf2.String() {
				t.Error("cached render differs from uncached render")
			}
		})
	}
}
