package main

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

// CanRun checks if the current file mode supports running executables after building
func (e *Editor) CanRun() bool {
	switch e.mode {
	case mode.Blank, mode.AIDL, mode.Amber, mode.Bazel, mode.Config, mode.Doc, mode.Email, mode.Git, mode.HIDL, mode.HTML, mode.JSON, mode.Log, mode.M4, mode.ManPage, mode.Markdown, mode.Nroff, mode.PolicyLanguage, mode.ReStructured, mode.Shader, mode.SQL, mode.Text, mode.XML:
		return false
	case mode.Shell: // don't run, because it's not a good idea
		return false
	case mode.Zig: // TODO: Find out why running Zig programs is problematic, terminal emulator wise
		return false
	}
	return true
}

// Run will attempt to run the corresponding output executable, given a source filename.
// It's an advantage if the BuildOrExport function has been successfully run first.
func (e *Editor) Run() (string, error) {
	sourceFilename, err := filepath.Abs(e.filename)
	if err != nil {
		return "", err
	}

	sourceDir := filepath.Dir(sourceFilename)

	var cmd *exec.Cmd

	// Make sure not to do anything with cmd here until it has been initialized by the switch below!

	switch e.mode {
	case mode.CMake:
		cmd = exec.Command("cmake", "-B", "build", "-D", "CMAKE_BUILD_TYPE=Debug", "-S", sourceDir)
	case mode.Kotlin:
		jarName := e.exeName(sourceFilename, false) + ".jar"
		cmd = exec.Command("java", "-jar", jarName)
	case mode.Go:
		cmd = exec.Command("go", "run", sourceFilename)
	case mode.Lua:
		cmd = exec.Command("lua", sourceFilename)
	case mode.Make:
		cmd = exec.Command("make")
	case mode.Java:
		cmd = exec.Command("java", "-jar", "main.jar")
	case mode.Just:
		cmd = exec.Command("just")
	case mode.Python:
		cmd = exec.Command("python", sourceFilename)
	default:
		exeName := filepath.Join(sourceDir, e.exeName(e.filename, true))
		cmd = exec.Command(exeName)
	}

	cmd.Dir = sourceDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// DrawOutput will draw a pane with the 5 last lines of the given output
func (e *Editor) DrawOutput(c *vt100.Canvas, maxLines int, title, collectedOutput string, backgroundColor vt100.AttributeColor, repositionCursorAfterDrawing bool) {
	minWidth := 32

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
