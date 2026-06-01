package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xyproto/vt"
)

// Breadcrumb represents a navigation point that can be jumped back to
type Breadcrumb struct {
	BackFunc func()
	Label    string // display label, e.g. "main.c" or "main.go:42"
}

// breadcrumbs is the global stack of navigation breadcrumbs
var breadcrumbs []Breadcrumb

// breadcrumbBarShown tracks whether the breadcrumb bar has been activated
// (it stays visible even when the stack is empty, showing just the current filename)
var breadcrumbBarShown bool

// pushBreadcrumb adds a new breadcrumb to the navigation stack
func pushBreadcrumb(label string, back func()) {
	breadcrumbs = append(breadcrumbs, Breadcrumb{Label: label, BackFunc: back})
	breadcrumbBarShown = true
}

// popBreadcrumb removes and returns the last breadcrumb, or false if empty
func popBreadcrumb() (Breadcrumb, bool) {
	if len(breadcrumbs) == 0 {
		return Breadcrumb{}, false
	}
	lastIndex := len(breadcrumbs) - 1
	bc := breadcrumbs[lastIndex]
	breadcrumbs = breadcrumbs[:lastIndex]
	return bc, true
}

// clearBreadcrumbs removes all breadcrumbs and hides the bar
func clearBreadcrumbs() {
	breadcrumbs = breadcrumbs[:0]
	breadcrumbBarShown = false
}

// hasBreadcrumbs returns true if there are breadcrumbs to jump back to
func hasBreadcrumbs() bool {
	return len(breadcrumbs) > 0
}

// breadcrumbBarHeight returns 1 when the breadcrumb bar should be displayed
// (once activated, it stays visible until explicitly cleared or sticky bars are shown),
// 0 otherwise.
func (e *Editor) breadcrumbBarHeight() uint {
	if breadcrumbBarShown && !e.InBookMode() && !e.nanoMode.Load() && !e.stickyStatusBars {
		return 1
	}
	return 0
}

// breadcrumbTrailWithCurrent returns the display string for the breadcrumb bar,
// including the current filename at the end of the trail.
// Consecutive entries from the same file are collapsed into a single filename entry.
// When the stack is empty but the bar is visible, shows just the current filename.
func breadcrumbTrailWithCurrent(currentFilename string) string {
	currentBase := filepath.Base(currentFilename)
	if len(breadcrumbs) == 0 {
		return currentBase
	}

	// Build a deduplicated list of file labels: collapse consecutive
	// breadcrumbs that refer to the same base filename into one entry.
	var parts []string
	prevBase := ""
	for i := range breadcrumbs {
		base := breadcrumbFileBase(breadcrumbs[i].Label)
		if base != prevBase {
			parts = append(parts, base)
			prevBase = base
		}
	}

	// Append the current file if it differs from the last entry
	if len(parts) == 0 || parts[len(parts)-1] != currentBase {
		parts = append(parts, currentBase)
	}

	// Truncate from the left if too many
	const maxDisplay = 5
	if len(parts) > maxDisplay {
		parts = append([]string{"…"}, parts[len(parts)-maxDisplay:]...)
	}

	return strings.Join(parts, " > ")
}

// breadcrumbFileBase extracts the base filename from a breadcrumb label.
// Labels may be "file.c:42" or just "file.c".
func breadcrumbFileBase(label string) string {
	if idx := strings.LastIndex(label, ":"); idx > 0 {
		// Check if everything after ":" is digits (a line number)
		suffix := label[idx+1:]
		isLineNum := len(suffix) > 0
		for _, r := range suffix {
			if r < '0' || r > '9' {
				isLineNum = false
				break
			}
		}
		if isLineNum {
			return label[:idx]
		}
	}
	return label
}

// drawBreadcrumbBar paints the breadcrumb navigation bar at row 0.
func (e *Editor) drawBreadcrumbBar(c *vt.Canvas) {
	if e.breadcrumbBarHeight() == 0 {
		return
	}
	w := int(c.W())
	trail := breadcrumbTrailWithCurrent(e.filename)
	text := "<->" + trail + "<->"
	e.drawBar(c, 0, w, text, e.stickyBarStyle())
}

// breadcrumbLabel creates a display label from a filename and line index
// (e.g. "main.go:42"), used for function/definition jumps.
func breadcrumbLabel(filename string, lineIndex LineIndex) string {
	base := filepath.Base(filename)
	lineNum := int(lineIndex) + 1
	return fmt.Sprintf("%s:%d", base, lineNum)
}

// breadcrumbFileLabel creates a display label with just the filename
// (e.g. "main.c"), used for file-to-file jumps like #include.
func breadcrumbFileLabel(filename string) string {
	return filepath.Base(filename)
}
