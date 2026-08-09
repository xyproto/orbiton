package main

import (
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
	"github.com/xyproto/vt"
)

var (
	cursorPositionRegex = regexp.MustCompile(`\x1b\[(\d+);\d+H`)
	escapeSequenceRegex = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)
)

// captureStdout returns everything that f writes to stdout
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdout
	os.Stdout = w
	collected := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		collected <- string(b)
	}()
	f()
	w.Close()
	os.Stdout = saved
	return <-collected
}

// emittedColumnsPerRow sums the display width a drawn frame emits per terminal row
func emittedColumnsPerRow(frame string) map[string]int {
	columns := make(map[string]int)
	positions := cursorPositionRegex.FindAllStringSubmatchIndex(frame, -1)
	for i, pos := range positions {
		end := len(frame)
		if i+1 < len(positions) {
			end = positions[i+1][0]
		}
		text := escapeSequenceRegex.ReplaceAllString(frame[pos[1]:end], "")
		columns[frame[pos[2]:pos[3]]] += runewidth.StringWidth(text)
	}
	return columns
}

// TestDrawnFrameFitsTheCanvas checks that no drawn row is wider than the canvas.
// A too-wide row wraps, and a wrap on the bottom row scrolls, which puts the
// diffed frame out of sync with the screen and shows up as duplicated lines.
func TestDrawnFrameFitsTheCanvas(t *testing.T) {
	const w, h = 20, 6
	for _, tc := range []struct {
		name string
		line string
	}{
		{"narrow", strings.Repeat("x", 40)},
		{"cjk", strings.Repeat("中", 40)},
		{"emoji", strings.Repeat("😀", 40)},
		{"mixed", strings.Repeat("a中b😀", 10)},
		{"tabs", strings.Repeat("\t", 10) + strings.Repeat("y", 20)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewSimpleEditor(w)
			for range h {
				e.InsertStringAndMove(nil, tc.line)
				e.InsertLineBelow()
				e.pos.sy++
			}
			c := vt.NewCanvasWithSize(w, h)
			e.WriteLines(c, 0, h, 0, 0, true, false)
			frame := captureStdout(t, func() { c.Draw() })
			for row, columns := range emittedColumnsPerRow(frame) {
				if columns > w {
					t.Errorf("row %s emits %d columns, but the canvas is only %d wide", row, columns, w)
				}
			}
		})
	}
}

// TestDrawnFrameLeavesLineWrapAlone checks that drawing leaves autowrap off,
// the way vt.Init set it up, so that a too-wide row is cut off and not wrapped.
func TestDrawnFrameLeavesLineWrapAlone(t *testing.T) {
	c := vt.NewCanvasWithSize(20, 4)
	c.Write(0, 0, vt.White, vt.Black, "hello")
	frame := captureStdout(t, func() { c.Draw() })
	if strings.HasSuffix(strings.TrimSuffix(frame, "\033[?2026l"), "\033[?7h") {
		t.Error("the drawn frame enables autowrap")
	}
}
