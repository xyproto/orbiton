package main

import (
	"errors"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/ianlancetaylor/demangle"
	"github.com/xyproto/files"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

const (
	smallRegisterWindow = iota
	largeRegisterWindow
	noRegisterWindow
)

var (
	originalDirectory        string
	showInstructionPane      bool
	errProgramStopped        = errors.New("program stopped") // must contain "program stopped"
	prevFlags                []string
	longInstructionPaneWidth int // should the instruction pane be extra wide, if so, how wide?
	lastDebugOutputLen       int
)

// DebugActivateBreakpoint sends break-insert to gdb together with the breakpoint in e.breakpoint, if available
func (e *Editor) DebugActivateBreakpoint(sourceBaseFilename string) (string, error) {
	if e.debugger == nil {
		return "", errors.New("debugger is not running")
	}
	if e.breakpoint != nil {
		if err := e.debugger.ActivateBreakpoint(sourceBaseFilename, int(e.breakpoint.LineNumber())); err != nil {
			return "", err
		}
		return "", nil
	}
	return "", errors.New("e.breakpoint is not set")
}

// DebugEnd will end the current debug session, but not set debugMode to false
func (e *Editor) DebugEnd() {
	if e.debugger != nil {
		e.debugger.End()
	}
	e.debugger = nil
	lastDebugOutputLen = 0
}

// DrawWatches will draw a box with the current watch expressions and values in the upper right
func (e *Editor) DrawWatches(c *vt.Canvas, repositionCursor bool) {
	// First create a box the size of the entire canvas
	canvasBox := NewCanvasBox(c)

	minWidth := 32

	// Window is the background box that will be drawn in the upper right
	upperRightBox := NewBox()
	upperRightBox.UpperRightPlacement(canvasBox, minWidth)

	w := int(c.Width())
	h := int(c.Height())

	// Then create a list box
	listBox := NewBox()

	marginY := 2
	if h < 35 {
		marginY = 1
	}
	listBox.FillWithMargins(upperRightBox, 2, marginY)

	// Get the current theme for the watch box
	bt := e.NewBoxTheme()

	title := "Running"
	bt.Background = &e.DebugRunningBackground

	programRunning := e.debugger != nil && e.debugger.ProgramRunning()

	if !programRunning {
		title = "Stopped"
		bt.Background = &e.DebugStoppedBackground
	}

	upperRightBox.Y--
	upperRightBox.H++
	if h > 35 {
		upperRightBox.H++
	}

	// Draw the background box
	e.DrawBox(bt, c, upperRightBox)

	watchMap := make(map[string]string)
	var lastSeenWatchVariable string
	if e.debugger != nil {
		watchMap = e.debugger.WatchMap()
		lastSeenWatchVariable = e.debugger.LastSeenWatch()
	}

	// Find the available space to draw help text in the upperRightBox
	availableHeight := max((listBox.H-marginY)-1,
		// Draw at least two rows of help text, no matter what
		2)
	if len(watchMap) == 0 {
		// Draw the help text, if the screen is wide enough
		if w > 120 {
			helpSlice := []string{
				"ctrl-space : step",
				"ctrl-n     : next instruction",
				"ctrl-f     : finish (step out)",
				"ctrl-r     : run to end",
				"ctrl-w     : add a watch",
				"ctrl-p     : reg. pane layout",
				"ctrl-i     : toggle step into",
			}
			if e.debugStepInto {
				helpSlice[0] = "ctrl-space : step into"
				helpSlice[1] = "ctrl-n     : next instruction (step into)"
			}
			if h < 32 {
				helpSlice = helpSlice[:availableHeight]
			}
			listBox.Y--
			e.DrawList(bt, c, listBox, helpSlice, -1)
		} else if w > 80 {
			narrowHelpSlice := []string{
				"ctrl-space: step",
				"ctrl-n: next inst.",
				"ctrl-f: step out",
				"ctrl-r: run to end",
				"ctrl-w: add watch",
				"ctrl-p: reg. pane",
				"ctrl-i: toggle into",
			}
			if e.debugStepInto {
				narrowHelpSlice[0] = "ctrl-space: step into"
				narrowHelpSlice[1] = "ctrl-n: into n. inst."
			}
			if h < 32 {
				narrowHelpSlice = narrowHelpSlice[:availableHeight]
			}
			e.DrawList(bt, c, listBox, narrowHelpSlice, -1)
		}
	} else {
		overview := []string{}
		foundLastSeen := false

		// First add the last seen variable, at the top of the list
		for k, v := range watchMap {
			if k == lastSeenWatchVariable {
				overview = append(overview, k+": "+v)
				foundLastSeen = true
				break
			}
		}

		// Then add the rest
		for k, v := range watchMap {
			if k == lastSeenWatchVariable {
				// Already added
				continue
			}
			overview = append(overview, k+": "+v)
		}

		// Highlight the top item if a debug session is active, and it was changed during this session
		if foundLastSeen && e.debugger != nil {
			// Draw the list of watches, where the last changed one is highlighted (and at the top)
			e.DrawList(bt, c, listBox, overview, 0)
		} else {
			// Draw the list of watches, with no highlights
			e.DrawList(bt, c, listBox, overview, -1)
		}
	}

	// Draw the title
	e.DrawTitle(bt, c, upperRightBox, title, true)

	// Blit
	c.HideCursorAndDraw()

	// Reposition the cursor
	if repositionCursor {
		e.EnableAndPlaceCursor(c)
	}
}

// DrawFlags will draw the currently set flags (like zero, carry etc) at the bottom right
func (e *Editor) DrawFlags(c *vt.Canvas, repositionCursor bool) {
	if e.debugger == nil {
		return
	}

	defer func() {
		// Reposition the cursor
		if repositionCursor {
			e.EnableAndPlaceCursor(c)
		}
	}()

	changedFlags := []string{}

	// Fetch the value of the machine flags (zero flag, carry etc)
	if flagNamesString, err := e.debugger.EvalExpression("$eflags"); err == nil {
		flagNamesString = strings.TrimPrefix(flagNamesString, "[ ")
		flagNamesString = strings.TrimSuffix(flagNamesString, " ]")
		flags := strings.Split(flagNamesString, " ")
		// Find which flags changed since last step
		for _, flag := range flags {
			if !slices.Contains(prevFlags, flag) {
				changedFlags = append(changedFlags, flag)
			}
		}
		prevFlags = flags
	}

	if len(changedFlags) == 0 {
		return
	}

	const title = "Changed flags:"

	lastWidthIndex := int(c.W() - 1)
	lastHeightIndex := int(c.H() - 1)

	// Length of all flags, joined with a "|" in between
	textLength := len(title + " " + strings.Join(changedFlags, "|"))

	// The left side margin, if the text is adjusted to the right
	x := uint(lastWidthIndex - textLength)

	// The bottom line
	y := uint(lastHeightIndex)

	// Title colors
	fg := e.StatusForeground
	bg := e.DebugOutputBackground

	// Draw the title
	c.Write(x, y, fg, bg, title+" ")
	x += ulen(title) + 1

	// Flag colors
	fg = e.StatusErrorForeground
	bg = e.StatusErrorBackground

	for i, flag := range changedFlags {
		if i > 0 {
			c.Write(x, y, e.DebugInstructionsForeground, bg, "|")
			x++
		}
		c.Write(x, y, fg, bg, flag)
		x += ulen(flag)
	}

	// Blit
	c.HideCursorAndDraw()
}

// DrawRegisters will draw a box with the current register values in the lower right
func (e *Editor) DrawRegisters(c *vt.Canvas, repositionCursor bool) error {
	defer func() {
		// Reposition the cursor
		if repositionCursor {
			e.EnableAndPlaceCursor(c)
		}
	}()

	if e.debugShowRegisters == noRegisterWindow || e.debugger == nil {
		// Don't draw anything
		return nil
	}
	filterWeirdRegisters := e.debugShowRegisters != largeRegisterWindow

	// First create a box the size of the entire canvas
	canvasBox := NewCanvasBox(c)

	// Window is the background box that will be drawn in the upper right
	lowerRightBox := NewBox()

	minWidth := 32

	var title string
	if filterWeirdRegisters {
		title = "Changed registers"
		// narrow box
		lowerRightBox.LowerRightPlacement(canvasBox, minWidth)
		if showInstructionPane {
			lowerRightBox.H = int(float64(lowerRightBox.H) * 0.9)
		}
		e.redraw.Store(true)
	} else {
		title = "All changed registers"
		// wide box
		lowerRightBox.LowerPlacement(canvasBox, 100)
		e.redraw.Store(true)
	}

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(lowerRightBox, 2, 2)

	// Get the current theme for the register box
	bt := e.NewBoxTheme()
	bt.Background = &e.DebugRegistersBackground

	// Draw the background box and title
	e.DrawBox(bt, c, lowerRightBox)

	e.DrawTitle(bt, c, lowerRightBox, title, true)

	if e.debugger != nil {
		// Tell the debugger whether to filter sub-registers
		if gdbD, ok := e.debugger.(*gdbDebugger); ok {
			gdbD.filterRegisters = filterWeirdRegisters
		}

		allChangedRegisters, err := e.debugger.ChangedRegisterMap()
		if err != nil {
			return err
		}

		var regSlice []string

		if filterWeirdRegisters {
			for reg, value := range allChangedRegisters {
				registryNameWithDigit := false
				for _, digit := range []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"} {
					if strings.Contains(reg, digit) {
						registryNameWithDigit = true
					}
				}
				spaceInValue := strings.Contains(value, " ")
				// Skip registers with numbers in their names, like "ymm12", for now
				if !registryNameWithDigit && !spaceInValue {
					regSlice = append(regSlice, reg+": "+value)
				}
			}
		} else {
			for reg, value := range allChangedRegisters {
				regSlice = append(regSlice, reg+": "+value)
			}
		}

		sort.Strings(regSlice)

		// Cutoff the slice by how high it is, if it's too long
		if len(regSlice) > listBox.H {
			if listBox.H > 0 {
				regSlice = regSlice[:listBox.H]
			}
		}

		// Draw the registers without numbers
		e.DrawList(bt, c, listBox, regSlice, -1)

	}

	// Blit
	c.HideCursorAndDraw()

	return nil
}

// DrawInstructions will draw a box with the current instructions
func (e *Editor) DrawInstructions(c *vt.Canvas, repositionCursor bool) error {
	defer func() {
		// Reposition the cursor
		if repositionCursor {
			e.EnableAndPlaceCursor(c)
		}
	}()

	if showInstructionPane && e.debugger != nil {

		// First create a box the size of the entire canvas
		canvasBox := NewCanvasBox(c)

		// Window is the background box that will be drawn in the upper right
		centerBox := NewBox()

		minWidth := 32

		centerBox.EvenLowerRightPlacement(canvasBox, minWidth)
		e.redraw.Store(true)

		// Then create a list box
		listBox := NewBox()
		listBox.FillWithMargins(centerBox, 1, 1)

		// Get the current theme for the register box
		bt := e.NewBoxTheme()
		// bt.Text = &e.DebugInstructionsForeground
		bt.Background = &e.DebugInstructionsBackground

		title := "Next instructions"

		if e.debugger != nil {

			numberOfInstructionsToFetch := 5
			instructions, err := e.debugger.Disassemble(numberOfInstructionsToFetch)
			if err != nil { // We end up here if the program is done running, when stepping
				if err.Error() == "No registers." {
					// program is done running
				}
				return err
			}

			// Cutoff the slice by how high it is, if it's too long
			if len(instructions) > listBox.H {
				if listBox.H > 0 {
					instructions = instructions[:listBox.H]
				}
			}

			demangledLines := []string{}
			maxLen := 0
			for _, line := range instructions {
				demangledLine := line
				for word := range strings.FieldsSeq(line) {
					word := strings.TrimSpace(word)
					modifiedWord := word // maybe modified word
					if strings.HasPrefix(word, "<") && strings.HasSuffix(word, ">") {
						word = strings.TrimSpace(word[1 : len(word)-1])
						// This modification is needed for demangle to accept the symbol syntax
						modifiedWord = strings.Replace(word, "E+", "E.", 1)
					}
					modifiedWord = strings.TrimSuffix(modifiedWord, "@plt")
					if demangledWord, err := demangle.ToString(modifiedWord); err == nil { // success
						// logf("%s -> %s\n", modifiedWord, demangledWord)
						demangledLine = strings.ReplaceAll(demangledLine, word, demangledWord)
						//} else {
						//logf("could not demangle: %s\n", modifiedWord)
					}
				}
				if len(demangledLine) > maxLen {
					maxLen = len(demangledLine)
				}
				demangledLines = append(demangledLines, demangledLine)
			}

			// Adjust the box width, if needed
			if (centerBox.W - 4) < maxLen {
				centerBox.W = maxLen + 4
			}

			// Should the box cover the entire width?
			if longInstructionPaneWidth > 0 {
				centerBox.X = 0
				centerBox.W = longInstructionPaneWidth
			} else if w := int(c.W() - 1); (centerBox.X + centerBox.W) >= w {
				centerBox.X = 0
				centerBox.W = w
				longInstructionPaneWidth = w
			}

			// If the box reaches the bottom, move it up one step
			if (centerBox.Y + centerBox.H) >= int(c.H()-1) {
				centerBox.Y--
			}

			// Draw the background box
			e.DrawBox(bt, c, centerBox)

			// Position the list box
			listBox.FillWithMargins(centerBox, 1, 1)

			// Draw the registers without numbers, highlighting the first one
			e.DrawList(bt, c, listBox, demangledLines, 0)

		} else {
			// Just draw the background box
			e.DrawBox(bt, c, centerBox)
		}

		// Draw the title
		e.DrawTitle(bt, c, centerBox, title, true)

		// Blit
		c.HideCursorAndDraw()

	}

	return nil
}

// DrawDebugOutput will draw a pane with the 5 last lines of the collected stdout output from the debugger
func (e *Editor) DrawDebugOutput(c *vt.Canvas, repositionCursor bool) {
	// Check if the output pane should be shown or not
	if e.debugHideOutput || e.debugger == nil {
		return
	}

	const title = "stdout"

	// Gather the debugger stdout so far
	collectedOutput := strings.TrimSpace(e.debugger.Output())

	if l := len(collectedOutput); l > 0 && l != lastDebugOutputLen {
		lastDebugOutputLen = l

		// First create a box the size of the entire canvas
		canvasBox := NewCanvasBox(c)

		minWidth := 32

		lowerLeftBox := NewBox()
		lowerLeftBox.LowerLeftPlacement(canvasBox, minWidth)
		if showInstructionPane {
			lowerLeftBox.H = int(float64(lowerLeftBox.H) * 0.9)
		}

		// Then create a list box
		listBox := NewBox()
		listBox.FillWithMargins(lowerLeftBox, 2, 2)

		// Get the current theme for the stdout box
		bt := e.NewBoxTheme()
		bt.Background = &e.DebugOutputBackground
		bt.UpperEdge = bt.LowerEdge

		e.DrawBox(bt, c, lowerLeftBox)

		e.DrawTitle(bt, c, lowerLeftBox, title, true)

		// Get the last 5 lines, and create a string slice
		lines := strings.Split(collectedOutput, "\n")
		if l := len(lines); l > 5 {
			lines = lines[l-5:]
		}

		// Trim and shorten the lines
		var newLines []string
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if len(trimmedLine) > listBox.W {
				if listBox.W-7 > 0 {
					trimmedLine = trimmedLine[:listBox.W-7] + " [...]"
				} else {
					trimmedLine = trimmedLine[:listBox.W]
				}
			}
			newLines = append(newLines, trimmedLine)
		}
		lines = newLines

		e.DrawList(bt, c, listBox, lines, -1)

		// Blit
		c.HideCursorAndDraw()

		repositionCursor = true
	}

	// Reposition the cursor
	if repositionCursor {
		e.EnableAndPlaceCursor(c)
	}
}

// DebugStartSession builds and then connects to the debugger
func (e *Editor) DebugStartSession(c *vt.Canvas, tty *vt.TTY, status *StatusBar, optionalOutputExecutable string) error {
	absFilename, err := e.AbsFilename()
	if err != nil {
		return err
	}

	var outputExecutable string
	if optionalOutputExecutable == "" {
		outputExecutable, err = e.BuildOrExport(tty, c, status)
		if err != nil {
			e.debugMode = false
			e.redrawCursor.Store(true)
			return err
		}
	} else {
		outputExecutable = optionalOutputExecutable
	}

	if outputExecutable == "everything" {
		outputExecutable = e.exeName(absFilename, true)
	}

	outputExecutableClean := filepath.Clean(filepath.Join(filepath.Dir(absFilename), outputExecutable))
	if !files.Exists(outputExecutableClean) {
		e.debugMode = false
		e.redrawCursor.Store(true)
		return errors.New("could not find " + outputExecutableClean)
	}

	// Reuse existing debugger to preserve watches, or create a new one
	if e.debugger == nil {
		e.debugger = newGDBDebugger(e.mode)
	}

	lineFunc := func(lineNumber int) {
		e.GoToLineNumber(LineNumber(lineNumber), nil, nil, true)
	}

	doneFunc := func() {
		status.SetMessageAfterRedraw("Execution complete")
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
	}

	// Start GDB execution from the top
	msg, err := e.debugger.Start(filepath.Dir(absFilename), filepath.Base(absFilename), outputExecutable, lineFunc, doneFunc)
	if err != nil {
		e.debugger = nil
		e.redrawCursor.Store(true)
		if msg != "" {
			msg += ", "
		}
		msg += err.Error()
		return errors.New("could not start debugging: " + msg)
	}

	// Pass the breakpoint, if set
	if e.breakpoint != nil {
		if err := e.debugger.ActivateBreakpoint(filepath.Base(absFilename), int(e.breakpoint.LineNumber())); err != nil {
			e.debugger.End()
			e.debugger = nil
			return err
		}
	}

	// Setup assembly mode, disassembly style, and run
	if gdbD, ok := e.debugger.(*gdbDebugger); ok {
		if _, err := gdbD.SetupAndRun(e.mode == mode.Assembly); err != nil {
			e.debugger.End()
			e.debugger = nil
			return err
		}
	}

	e.GoToTop(c, nil)

	status.ClearAll(c, false)
	if e.breakpoint == nil {
		status.SetMessage("Running")
	} else {
		status.SetMessage("Running. Breakpoint at line " + e.breakpoint.LineNumber().String() + ".")
	}
	status.Show(c, e)
	return nil
}
