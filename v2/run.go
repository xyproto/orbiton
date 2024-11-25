package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xyproto/files"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

// Run will attempt to run the corresponding output executable, given a source filename.
// It's an advantage if the BuildOrExport function has been successfully run first.
// The bool is true only if the command exited with an exit code != 0 and there is text on stderr,
// which implies that the error style / background color should be used when presenting the output.
func (e *Editor) Run() (string, bool, error) {
	sourceFilename, err := filepath.Abs(e.filename)
	if err != nil {
		return "", false, err
	}

	sourceDir := filepath.Dir(sourceFilename)

	pyCacheDir := filepath.Join(userCacheDir, "o", "python")
	if noWriteToCache {
		pyCacheDir = filepath.Join(sourceDir, "o", "python")
	}

	var cmd *exec.Cmd

	// Make sure not to do anything with cmd here until it has been initialized by the switch below!

	switch e.mode {
	case mode.CMake:
		cmd = exec.Command("cmake", "-B", "build", "-D", "CMAKE_BUILD_TYPE=Debug", "-S", sourceDir)
	case mode.Kotlin:
		jarName := e.exeName(sourceFilename, false) + ".jar"
		cmd = exec.Command("java", "-jar", jarName)
	case mode.Go:
		if strings.HasSuffix(sourceFilename, "_test.go") {
			// TODO: go test . -run NameOfTest and fetch NameOfTest from the test function that the cursor is within, if available
			cmd = exec.Command("go", "test", ".")
		} else {
			cmd = exec.Command("go", "run", sourceFilename)
		}
	case mode.Lilypond:
		ext := filepath.Ext(sourceFilename)
		firstName := strings.TrimSuffix(filepath.Base(sourceFilename), ext)
		pdfFilename := firstName + ".pdf"
		if isDarwin {
			cmd = exec.Command("open", pdfFilename)
		} else {
			cmd = exec.Command("xdg-open", pdfFilename)
		}
	case mode.Lua:
		cmd = exec.Command("lua", sourceFilename)
	case mode.Make:
		cmd = exec.Command("make")
	case mode.Java:
		cmd = exec.Command("java", "-jar", "main.jar")
	case mode.Just:
		cmd = exec.Command("just")
	case mode.Odin:
		if efn := e.exeName(e.filename, true); files.IsExecutable(efn) {
			cmd = exec.Command(filepath.Join(sourceDir, efn))
		}
	case mode.Python:
		if isDarwin {
			cmd = exec.Command("python3", sourceFilename)
		} else {
			cmd = exec.Command("python", sourceFilename)
		}
		cmd.Env = append(cmd.Env, "PYTHONUTF8=1")
		if !files.Exists(pyCacheDir) {
			os.MkdirAll(pyCacheDir, 0o700)
		}
		cmd.Env = append(cmd.Env, "PYTHONPYCACHEPREFIX="+pyCacheDir)
	default:
		cmd = exec.Command(filepath.Join(sourceDir, e.exeName(e.filename, true)))
	}

	cmd.Dir = sourceDir

	// If inputFileWhenRunning has been specified (or is input.txt),
	// check if that file can be used as stdin for the command to be run
	if inputFileWhenRunning != "" {
		inputFile, err := os.Open(inputFileWhenRunning)
		if err != nil {
			// Do not retry until the editor has been started again
			inputFileWhenRunning = ""
		} else {
			defer inputFile.Close()
			// Use the file as the input for stdin
			cmd.Stdin = inputFile
		}
	}

	output, err := cmd.CombinedOutput()
	if err == nil { // success
		return trimRightSpace(string(output)), false, nil
	}
	if len(output) > 0 { // error, but text on stdout/stderr
		return trimRightSpace(string(output)), true, nil
	}
	// error and no text on stdout/stderr
	return "", false, err
}

// DrawOutput will draw a pane with the 5 last lines of the given output
func (e *Editor) DrawOutput(c *vt100.Canvas, maxLines int, title, collectedOutput string, backgroundColor vt100.AttributeColor, repositionCursorAfterDrawing bool) {
	e.waitWithRedrawing.Store(true)

	minWidth := 32

	// Get the last maxLine lines, and create a string slice
	lines := strings.Split(collectedOutput, "\n")
	if l := len(lines); l > maxLines {
		lines = lines[l-maxLines:]
		// Add "[...]" as the first line
		lines = append([]string{"[...]", ""}, lines...)
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
	listBox.FillWithMargins(lowerLeftBox, 1, 2)

	// Get the current theme for the stdout box
	bt := e.NewBoxTheme()
	bt.Background = &backgroundColor
	bt.UpperEdge = bt.LowerEdge

	e.DrawBox(bt, c, lowerLeftBox)

	if title != "" {
		e.DrawTitle(bt, c, lowerLeftBox, title, true)
	}

	e.DrawList(bt, c, listBox, lines, -1)

	// Blit
	c.HideCursorAndDraw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		e.EnableAndPlaceCursor(c)
	}
}
