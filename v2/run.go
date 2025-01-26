package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/xyproto/files"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

var runPID atomic.Int64

// stopBackgroundProcesses stops the "run" process that is running
// in the background, if runPID > 0.
func stopBackgroundProcesses() {
	if runPID.Load() > 0 {
		syscall.Kill(int(runPID.Load()), syscall.SIGKILL)
		runPID.Store(-1)
	}
}

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
	case mode.ABC:
		stopBackgroundProcesses()
		var audioOutputFlag string
		if isLinux {
			audioOutputFlag = "-Oj" // jack
		} else if isDarwin {
			audioOutputFlag = "-Od" // macOS
		}
		cmd = exec.Command("timidity", "--quiet", audioOutputFlag, filepath.Join(tempDir, "o.mid"))
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
	case mode.Markdown:
		cmd = exec.Command("algernon", "-m", sourceFilename)
	case mode.Lua:
		if e.LuaLove() {
			const macLovePath = "/Applications/love.app/Contents/MacOS/love"
			if files.Which("love") != "" {
				cmd = exec.Command("love", ".")
			} else if isDarwin && files.Exists(macLovePath) {
				cmd = exec.Command(macLovePath, sourceFilename)
			} else {
				return "", false, errors.New("please install LÖVE")
			}
		} else if e.LuaLovr() {
			const macLovrPath = "/Applications/lovr.app/Contents/MacOS/lovr"
			if files.Which("lovr") != "" {
				cmd = exec.Command("lovr", sourceFilename)
			} else if isDarwin && files.Exists(macLovrPath) {
				cmd = exec.Command(macLovrPath, sourceFilename)
			} else {
				return "", false, errors.New("please install LÖVR")
			}
		} else {
			cmd = exec.Command("lua", sourceFilename)
		}
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

	output, err := CombinedOutputSetPID(cmd)

	if e.mode != mode.ABC && err == nil { // success
		return trimRightSpace(string(output)), false, nil
	}
	if e.mode != mode.ABC && len(output) > 0 { // error, but text on stdout/stderr
		return trimRightSpace(string(output)), true, nil
	}
	// error and no text on stdout/stderr
	return "", false, err
}

// DrawOutput will draw a pane with the 5 last lines of the given output
func (e *Editor) DrawOutput(c *vt100.Canvas, maxLines int, title, collectedOutput string, backgroundColor vt100.AttributeColor, repositionCursorAfterDrawing, rightHandSide bool) {
	e.waitWithRedrawing.Store(true)

	w := c.Width()

	// Get the last maxLine lines, and create a string slice
	lines := strings.Split(collectedOutput, "\n")
	if l := len(lines); l > maxLines {
		lines = lines[l-maxLines:]
		// Add "[...]" as the first line
		lines = append([]string{"[...]", ""}, lines...)
	}

	boxMinWidth := w - 7

	_, maxLineLength := minMaxLength(lines)

	if maxLineLength < int(boxMinWidth) {
		boxMinWidth = uint(maxLineLength + 7)
	}

	// First create a box the size of the entire canvas
	canvasBox := NewCanvasBox(c)

	lowerBox := NewBox()

	if rightHandSide {
		lowerBox.LowerRightPlacement(canvasBox, int(boxMinWidth))
	} else {
		lowerBox.LowerLeftPlacement(canvasBox, int(boxMinWidth))
	}

	if title == "" {
		lowerBox.H = 5
	}

	lowerBox.Y -= 5
	lowerBox.H += 2

	if rightHandSide { // cosmetic adjustments
		lowerBox.W -= 2
	}

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(lowerBox, 1, 2)

	// Get the current theme for the stdout box
	bt := e.NewBoxTheme()
	bt.Background = &backgroundColor
	bt.UpperEdge = bt.LowerEdge

	e.DrawBox(bt, c, lowerBox)

	if title != "" {
		e.DrawTitle(bt, c, lowerBox, title, true)
	}

	e.DrawList(bt, c, listBox, lines, -1)

	// Blit
	c.HideCursorAndDraw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		e.EnableAndPlaceCursor(c)
	}
}

// CombinedOutputSetPID runs the command and returns its combined standard output and standard error.
// It also assignes the PID to the global runPID variable, right after the command has started.
func CombinedOutputSetPID(c *exec.Cmd) ([]byte, error) {
	if c.Stdout != nil || c.Stderr != nil {
		return []byte{}, errors.New("exec: stdout or stderr has already been set")
	}
	// Prepare a single buffer for both stdout and stderr
	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = &b
	// Start the process in the background
	err := c.Start()
	if err != nil {
		return b.Bytes(), err
	}
	// Get the PID of the running process
	if c.Process != nil {
		runPID.Store(int64(c.Process.Pid))
	} else {
		runPID.Store(-1)
	}
	// Wait for the process to complete
	err = c.Wait()
	// Ignore the error if the process was killed
	if err != nil && err.Error() == "signal: killed" { // ignore it if this process was killed
		err = nil
	}
	// Return the output bytes and the error, if any
	return b.Bytes(), err
}

// run tries to run the given command, without using a shell
func run(commandString string) error {
	parts := strings.Fields(commandString)
	if len(parts) == 0 {
		return errors.New("empty command")
	}
	if files.Which(parts[0]) == "" {
		return fmt.Errorf("could not find %s in path", parts[0])
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	return cmd.Run()
}
