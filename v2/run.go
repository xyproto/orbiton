package main

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

// CanRun checks if the current file mode supports running executables after building
func (e *Editor) CanRun() bool {
	switch e.mode {
	case mode.Kotlin, mode.Go, mode.Rust:
		return true
	}
	return false
}

// Run will attempt to run the corresponding output executable, given a source filename.
// This assumes that the BuildOrExport function has been successfully run first.
func (e *Editor) Run(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, filename string) (string, error) {
	sourceFilename, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	sourceDir := filepath.Dir(sourceFilename)
	var cmd *exec.Cmd

	switch e.mode {
	case mode.Kotlin:
		jarName := e.exeName(sourceFilename) + ".jar"
		cmd = exec.Command("java", "-jar", jarName)
		cmd.Dir = sourceDir
	case mode.Go:
		cmd = exec.Command("go", "run", filename)
		cmd.Dir = sourceDir
	case mode.Rust:
		exeName := filepath.Join(sourceDir, e.exeName(filename))
		cmd = exec.Command(exeName)
		cmd.Dir = sourceDir
	default:
		return "", errors.New("run: not implemented for " + e.mode.String())
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// DrawOutput will draw a pane with the 5 last lines of the given output
func (e *Editor) DrawOutput(c *vt100.Canvas, maxLines int, title, collectedOutput string, backgroundColor vt100.AttributeColor, repositionCursorAfterDrawing bool) {
	var minWidth = 32

	// Get the last maxLine lines, and create a string slice
	lines := strings.Split(collectedOutput, "\n")
	if l := len(lines); l > maxLines {
		lines = lines[l-maxLines:]
	}
	for _, line := range lines {
		if len(line) > minWidth {
			minWidth = len(line) + 5
		}
	}
	if minWidth > 79 {
		minWidth = 79
	}

	// First create a box the size of the entire canvas
	canvasBox := NewCanvasBox(c)

	lowerLeftBox := NewBox()
	lowerLeftBox.LowerLeftPlacement(canvasBox, minWidth)

	if title == "" {
		lowerLeftBox.H = 5
	}

	lowerLeftBox.Y -= 5
	lowerLeftBox.H += 2

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(lowerLeftBox, 2, 2)

	// Get the current theme for the stdout box
	bt := e.NewBoxTheme()
	bt.Background = &backgroundColor
	bt.UpperEdge = bt.LowerEdge

	e.DrawBox(bt, c, lowerLeftBox)

	if title != "" {
		e.DrawTitle(bt, c, lowerLeftBox, title)
	}

	e.DrawList(bt, c, listBox, lines, -1)

	// Blit
	c.Draw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		vt100.SetXY(uint(x), uint(y))
	}
}
