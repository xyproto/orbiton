package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cyrus-and/gdb"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

const (
	smallRegisterWindow = iota
	largeRegisterWindow
	noRegisterWindow
)

var (
	originalDirectory     string
	gdbLogFile            = filepath.Join(userCacheDir, "o", "gdb.log")
	gdbConsole            strings.Builder
	watchMap              = make(map[string]string)
	lastSeenWatchVariable string
	showInstructionPane   bool
	gdbOutput             bytes.Buffer
	lastGDBOutputLength   int
)

// DebugActivateBreakpoint sends break-insert to gdb together with the breakpoint in e.breakpoint, if available
func (e *Editor) DebugActivateBreakpoint(sourceBaseFilename string) (string, error) {
	if e.breakpoint != nil {
		if retvalMap, err := e.gdb.CheckedSend("break-insert", fmt.Sprintf("%s:%d", sourceBaseFilename, e.breakpoint.LineNumber())); err != nil {
			return fmt.Sprintf("%v", retvalMap), err
		}
		return "", nil
	}
	return "", errors.New("e.breakpoint is not set")
}

// DebugStart will start a new debug session, using gdb.
// Will end the existing session first if e.gdb != nil.
func (e *Editor) DebugStart(sourceDir, sourceBaseFilename, executableBaseFilename string) (string, error) {

	flogf(gdbLogFile, "[gdb] dir %s, src %s, exe %s\n", sourceDir, sourceBaseFilename, executableBaseFilename)

	// End any existing sessions
	e.DebugEnd()

	// Change directory to the sourcefile, temporarily
	var err error
	originalDirectory, err = os.Getwd()
	if err == nil { // cd success
		err = os.Chdir(sourceDir)
		if err != nil {
			return "", errors.New("could not change directory to " + sourceDir)
		}
	}

	// Use rust-gdb if we are debugging Rust
	var gdbExecutable string
	if e.mode == mode.Rust {
		gdbExecutable = which("rust-gdb")
	} else {
		gdbExecutable = which("gdb")
	}

	//flogf(gdbLogFile, "[gdb] starting %s: ", gdbExecutable)

	// Start a new gdb session
	e.gdb, err = gdb.NewCustom(gdbExecutable, func(notification map[string]interface{}) {
		// Handle messages from gdb, including frames that contains line numbers
		if payload, ok := notification["payload"]; ok && notification["type"] == "exec" {
			if payloadMap, ok := payload.(map[string]interface{}); ok {
				if frame, ok := payloadMap["frame"]; ok {
					if frameMap, ok := frame.(map[string]interface{}); ok {
						flogf(gdbLogFile, "[gdb] frame: %v\n", frameMap)
						if lineNumberString, ok := frameMap["line"].(string); ok {
							if lineNumber, err := strconv.Atoi(lineNumberString); err == nil { // success
								// TODO: Fetch a different line number?
								// Got a line number, send the editor there, without any status messages
								e.GoToLineNumber(LineNumber(lineNumber), nil, nil, true)
							}
						}
					}
				}
			}
		} else if payload, ok := notification["payload"]; ok && notification["type"] == "console" {
			if s, ok := payload.(string); ok {
				gdbConsole.WriteString(s)
			}
			//} else {
			//flogf(gdbLogFile, "[gdb] notification: %v\n", notification)
		}
	})
	if err != nil {
		e.gdb = nil
		//flogf(gdbLogFile, "%s\n", "fail")
		return "", err
	}
	if e.gdb == nil {
		//flogf(gdbLogFile, "%s\n", "fail")
		return "", errors.New("gdb.New returned no error, but e.gdb is nil")
	}
	//flogf(gdbLogFile, "%s\n", "ok")

	// Handle output to stdout (and stderr?) from programs that are being debugged
	go io.Copy(&gdbOutput, e.gdb)

	// Load the executable file
	if retvalMap, err := e.gdb.CheckedSend("file-exec-and-symbols", executableBaseFilename); err != nil {
		return fmt.Sprintf("%v", retvalMap), err
	}

	// Pass in arguments
	//e.gdb.Send("exec-arguments", "--version")

	// Pass the breakpoint, if it has been set with ctrl-b
	if e.breakpoint != nil {
		if retvalMap, err := e.gdb.CheckedSend("break-insert", fmt.Sprintf("%s:%d", sourceBaseFilename, e.breakpoint.LineNumber())); err != nil {
			return fmt.Sprintf("%v", retvalMap), err
		}
	}

	// Start from the top
	if _, err := e.gdb.CheckedSend("exec-run", "--start"); err != nil {
		output := gdbOutput.String()
		gdbOutput.Reset()
		return output, err
	}

	// Add any existing watches
	for varName := range watchMap {
		e.AddWatch(varName)
	}

	return "started gdb", nil
}

// DebugContinue will continue the execution to the next breakpoint or to the end.
// e.gdb must not be nil.
func (e *Editor) DebugContinue() error {
	_, err := e.gdb.CheckedSend("exec-continue")
	return err
}

// DebugNext will continue the execution by stepping to the next line.
// e.gdb must not be nil.
func (e *Editor) DebugNext() error {
	_, err := e.gdb.CheckedSend("exec-next")
	if err != nil {
		return err
	}
	consoleString := strings.TrimSpace(gdbConsole.String())
	gdbConsole.Reset()
	// Interpret consoleString and extract the new variable names and values,
	// for variables there are watchpoints for.
	if consoleString != "" {
		var varName string
		var varValue string
		for _, line := range strings.Split(consoleString, "\n") {
			if strings.Contains(line, "watchpoint") && strings.Contains(line, ":") {
				fields := strings.SplitN(line, ":", 2)
				varName = strings.TrimSpace(fields[1])
			} else if varName != "" && strings.HasPrefix(line, "New value =") {
				fields := strings.SplitN(line, "=", 2)
				varValue = strings.TrimSpace(fields[1])
				watchMap[varName] = varValue
				lastSeenWatchVariable = varName
				varName = ""
				varValue = ""
			}
		}
	}
	return nil
}

// DebugNextInstruction will continue the execution by stepping to the next instruction.
// e.gdb must not be nil.
func (e *Editor) DebugNextInstruction() error {
	showInstructionPane = true
	_, err := e.gdb.CheckedSend("exec-next-instruction")
	if err != nil {
		return err
	}
	consoleString := strings.TrimSpace(gdbConsole.String())
	gdbConsole.Reset()
	// Interpret consoleString and extract the new variable names and values,
	// for variables there are watchpoints for.
	if consoleString != "" {
		var varName string
		var varValue string
		for _, line := range strings.Split(consoleString, "\n") {
			if strings.Contains(line, "watchpoint") && strings.Contains(line, ":") {
				fields := strings.SplitN(line, ":", 2)
				varName = strings.TrimSpace(fields[1])
			} else if varName != "" && strings.HasPrefix(line, "New value =") {
				fields := strings.SplitN(line, "=", 2)
				varValue = strings.TrimSpace(fields[1])
				watchMap[varName] = varValue
				lastSeenWatchVariable = varName
				varName = ""
				varValue = ""
			}
		}
	}
	return nil
}

// DebugStep will continue the execution by stepping.
// e.gdb must not be nil.
func (e *Editor) DebugStep() error {
	_, err := e.gdb.CheckedSend("exec-step")
	if err != nil {
		return err
	}
	consoleString := strings.TrimSpace(gdbConsole.String())
	gdbConsole.Reset()
	// Interpret consoleString and extract the new variable names and values,
	// for variables there are watchpoints for.
	if consoleString != "" {
		var varName string
		var varValue string
		for _, line := range strings.Split(consoleString, "\n") {
			if strings.Contains(line, "watchpoint") && strings.Contains(line, ":") {
				fields := strings.SplitN(line, ":", 2)
				varName = strings.TrimSpace(fields[1])
			} else if varName != "" && strings.HasPrefix(line, "New value =") {
				fields := strings.SplitN(line, "=", 2)
				varValue = strings.TrimSpace(fields[1])
				watchMap[varName] = varValue
				lastSeenWatchVariable = varName
				varName = ""
				varValue = ""
			}
		}
	}
	return nil
}

// DebugFinish will "step out".
// e.gdb must not be nil. Returns whatever was outputted to gdb stdout.
func (e *Editor) DebugFinish() (string, error) {
	_, err := e.gdb.CheckedSend("exec-finish")
	if err != nil {
		return "", err
	}
	output := gdbOutput.String()
	gdbOutput.Reset()
	consoleString := strings.TrimSpace(gdbConsole.String())
	gdbConsole.Reset()
	// Interpret consoleString and extract the new variable names and values,
	// for variables there are watchpoints for.
	if consoleString != "" {
		var varName string
		var varValue string
		for _, line := range strings.Split(consoleString, "\n") {
			if strings.Contains(line, "watchpoint") && strings.Contains(line, ":") {
				fields := strings.SplitN(line, ":", 2)
				varName = strings.TrimSpace(fields[1])
			} else if varName != "" && strings.HasPrefix(line, "New value =") {
				fields := strings.SplitN(line, "=", 2)
				varValue = strings.TrimSpace(fields[1])
				watchMap[varName] = varValue
				lastSeenWatchVariable = varName
				varName = ""
				varValue = ""
			}
		}
	}
	return output, nil
}

// DebugRegisterNames will return all register names
func (e *Editor) DebugRegisterNames() ([]string, error) {
	if e.gdb == nil {
		return []string{}, errors.New("gdb must be running")
	}
	notification, err := e.gdb.CheckedSend("data-list-register-names")
	if err != nil {
		//flogf(gdbLogFile, "[gdb] data-list-register-names error: %s\n", err.Error())
		return []string{}, err
	}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]interface{}); ok {
			if registerNames, ok := payloadMap["register-names"]; ok {
				if registerSlice, ok := registerNames.([]interface{}); ok {
					registerStringSlice := make([]string, len(registerSlice))
					for i, interfaceValue := range registerSlice {
						if s, ok := interfaceValue.(string); ok {
							registerStringSlice[i] = s
						}
					}
					//flogf(gdbLogFile, "[gdb] data-list-register-names: %s\n", strings.Join(registerStringSlice, ","))
					return registerStringSlice, nil
				}
			}
		}
	}
	return []string{}, errors.New("could not find the register names in the payload returned from gdb")
}

// DebugChangedRegisters will return a list of all changed register numbers
func (e *Editor) DebugChangedRegisters() ([]int, error) {
	// Then get the register values
	notification, err := e.gdb.CheckedSend("data-list-changed-registers")
	if err != nil {
		//flogf(gdbLogFile, "[gdb] data-list-changed-registers error: %s\n", err.Error())
		return []int{}, err
	}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]interface{}); ok {
			//flogf(gdbLogFile, "[gdb] changed reg payload: %v\n", payloadMap)
			if registerInterfaces, ok := payloadMap["changed-registers"].([]interface{}); ok {
				changedRegisters := make([]int, len(registerInterfaces))
				for _, registerNumberString := range registerInterfaces {
					registerNumber, err := strconv.Atoi(registerNumberString.(string))
					if err != nil {
						return []int{}, err
					}
					changedRegisters = append(changedRegisters, registerNumber)
					//flogf(gdbLogFile, "[gdb] regnum %v %T\n", registerNumber, registerNumber)
				}
				return changedRegisters, nil
			}
		}
	}
	//flogf(gdbLogFile, "[gdb] data-list-register-values %v\n", registers)
	return []int{}, errors.New("could not find the register values in the payload returned from gdb")
}

// DebugDisassemble will return the next N assembly instructions
func (e *Editor) DebugDisassemble(n int) ([]string, error) {
	// Then get the register values
	notification, err := e.gdb.CheckedSend("data-disassemble -s $pc -e \"$pc + 20\" -- 0")
	if err != nil {
		//flogf(gdbLogFile, "[gdb] data-disassemble error: %s\n", err.Error())
		return []string{}, err
	}
	result := []string{}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]interface{}); ok {
			//flogf(gdbLogFile, "[gdb] disasm payload: %v\n", payloadMap)
			if asmDisSlice, ok := payloadMap["asm_insns"].([]interface{}); ok {
				for i, asmDis := range asmDisSlice {
					//flogf(gdbLogFile, "[gdb] disasm asm %d: %v %T\n", i, asmDis, asmDis)
					if asmMap, ok := asmDis.(map[string]interface{}); ok {
						instruction := asmMap["inst"]
						result = append(result, instruction.(string))
					}
					// Only collect n asm statements
					if i >= n {
						//flogf(gdbLogFile, "[gdb] result %v\n", result)
						break
					}
				}
				return result, nil
			}
		}
	}
	//flogf(gdbLogFile, "[gdb] disasm result: %v\n", result)
	return []string{}, errors.New("could not get disasm from gdb")
}

// DebugChangedRegisterMap returns a map of all registers that were changed the last step, and their values
func (e *Editor) DebugChangedRegisterMap() (map[string]string, error) {
	// First get the names of the registers
	names, err := e.DebugRegisterNames()
	if err != nil {
		return nil, err
	}
	// Then get the changed register IDs
	changedRegisters, err := e.DebugChangedRegisters()
	if err != nil {
		return nil, err
	}
	// Then get the register IDs, then use them to get the register names and values
	notification, err := e.gdb.CheckedSend("data-list-register-values", "--skip-unavailable", "x")
	if err != nil {
		//flogf(gdbLogFile, "[gdb] data-list-register-values error: %s\n", err.Error())
		return nil, err
	}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]interface{}); ok {
			if registerValues, ok := payloadMap["register-values"]; ok {
				if registerSlice, ok := registerValues.([]interface{}); ok {
					registers := make(map[string]string, len(names))
					for _, singleRegisterMap := range registerSlice {
						if registerMap, ok := singleRegisterMap.(map[string]interface{}); ok {
							numberString, ok := registerMap["number"].(string)
							if !ok {
								return nil, errors.New("could not convert \"number\" interface to string")
							}
							registerNumber, err := strconv.Atoi(numberString)
							if err != nil {
								return nil, err
							}
							thisRegisterWasChanged := false
							for _, changedRegisterNumber := range changedRegisters {
								if changedRegisterNumber == registerNumber {
									thisRegisterWasChanged = true
									break
								}
							}
							if !thisRegisterWasChanged {
								// Continue to the next one in the list of all available registers
								continue
							}
							value, ok := registerMap["value"].(string)
							if !ok {
								return nil, errors.New("could not convert \"value\" interface to string")
							}
							registerName := names[registerNumber]
							registers[registerName] = value
							//flogf(gdbLogFile, "[gdb] data-list-register-values: %s %s\n", registerName, value)
						}
					}
					return registers, nil
				}
			}
		}
	}
	//flogf(gdbLogFile, "[gdb] data-list-register-values %v\n", registers)
	return nil, errors.New("could not find the register values in the payload returned from gdb")
}

// DebugRegisterMap will return a map of all register names and values
func (e *Editor) DebugRegisterMap() (map[string]string, error) {
	// First get the names of the registers
	names, err := e.DebugRegisterNames()
	if err != nil {
		return nil, err
	}
	// Then get the register IDs, then use them to get the register names and values
	notification, err := e.gdb.CheckedSend("data-list-register-values", "--skip-unavailable", "x")
	if err != nil {
		//flogf(gdbLogFile, "[gdb] data-list-register-values error: %s\n", err.Error())
		return nil, err
	}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]interface{}); ok {
			if registerValues, ok := payloadMap["register-values"]; ok {
				if registerSlice, ok := registerValues.([]interface{}); ok {
					registers := make(map[string]string, len(names))
					for _, singleRegisterMap := range registerSlice {
						if registerMap, ok := singleRegisterMap.(map[string]interface{}); ok {
							numberString, ok := registerMap["number"].(string)
							if !ok {
								return nil, errors.New("could not convert \"number\" interface to string")
							}
							registerNumber, err := strconv.Atoi(numberString)
							if err != nil {
								return nil, err
							}
							value, ok := registerMap["value"].(string)
							if !ok {
								return nil, errors.New("could not convert \"value\" interface to string")
							}
							registerName := names[registerNumber]
							registers[registerName] = value
							//flogf(gdbLogFile, "[gdb] data-list-register-values: %s %s\n", registerName, value)
						}
					}
					return registers, nil
				}
			}
		}
	}
	//flogf(gdbLogFile, "[gdb] data-list-register-values %v\n", registers)
	return nil, errors.New("could not find the register values in the payload returned from gdb")
}

// DebugEnd will end the current gdb session
func (e *Editor) DebugEnd() {
	if e.gdb != nil {
		e.gdb.Exit()
	}
	e.gdb = nil
	// Clear any existing output
	gdbOutput.Reset()
	gdbConsole.Reset()
	// Clear the last seen variable
	lastSeenWatchVariable = ""
	// Clear the previous GDB stdout buffer length
	lastGDBOutputLength = 0
	// Also change to the original directory
	if originalDirectory != "" {
		os.Chdir(originalDirectory)
	}
	//flogf(gdbLogFile, "[gdb] %s\n", "stopped")
}

// AddWatch will add a watchpoint / watch expression to gdb
func (e *Editor) AddWatch(expression string) (string, error) {
	var output string
	if e.gdb != nil {
		//flogf(gdbLogFile, "[gdb] adding watch: %s\n", expression)
		_, err := e.gdb.CheckedSend("break-watch", "-a", expression)
		if err != nil {
			return "", err
		}
		output = gdbOutput.String()
		gdbOutput.Reset()
		//flogf(gdbLogFile, "[gdb] output after adding watch: %s\n", output)
	}
	watchMap[expression] = "?"

	// Don't set this, the variable watch has not been seen yet
	// lastSeenWatchVariable = expression

	return output, nil
}

// DrawWatches will draw a box with the current watch expressions and values in the upper right
func (e *Editor) DrawWatches(c *vt100.Canvas, repositionCursor bool) {
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

	if h < 35 {
		listBox.FillWithMargins(upperRightBox, 2, 1)
	} else {
		listBox.FillWithMargins(upperRightBox, 2, 2)
	}

	// Get the current theme for the watch box
	bt := NewBoxTheme()

	// Draw the background box and title
	e.DrawBox(bt, c, upperRightBox, &e.BoxBackground)

	title := "Running"
	if e.gdb == nil {
		title = "Not running"
	}
	if len(watchMap) == 0 {
		// Draw the help text, if the screen is wide enough
		if w > 120 {
			helpSlice := []string{
				"ctrl-space : step",
				"ctrl-n     : next instruction",
				"ctrl-f     : finish (step out)",
				"ctrl-w     : add a watch",
				"ctrl-r     : reg. pane layout",
			}
			if h < 32 {
				helpSlice = helpSlice[:3]
			}
			e.DrawList(c, listBox, helpSlice, -1)
		} else if w > 80 {
			narrowHelpSlice := []string{
				"ctrl-space: step",
				"ctrl-n: next inst.",
				"ctrl-f: step out",
				"ctrl-w: add watch",
				"ctrl-r: reg. pane",
			}
			if h < 32 {
				narrowHelpSlice = narrowHelpSlice[:3]
			}
			e.DrawList(c, listBox, narrowHelpSlice, -1)
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
		if foundLastSeen && e.gdb != nil {
			// Draw the list of watches, where the last changed one is highlighted (and at the top)
			e.DrawList(c, listBox, overview, 0)
		} else {
			// Draw the list of watches, with no highlights
			e.DrawList(c, listBox, overview, -1)
		}
	}

	e.DrawTitle(c, upperRightBox, title)

	// Blit
	c.Draw()

	// Reposition the cursor
	if repositionCursor {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		vt100.SetXY(uint(x), uint(y))
	}
}

// DrawRegisters will draw a box with the current register values in the lower right
func (e *Editor) DrawRegisters(c *vt100.Canvas, repositionCursor bool) error {
	if e.debugShowRegisters == noRegisterWindow {
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

		e.redraw = true
	} else {
		title = "All changed registers"
		// wide box
		lowerRightBox.LowerPlacement(canvasBox, 100)
		e.redraw = true
	}

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(lowerRightBox, 2, 2)

	// Get the current theme for the register box
	bt := NewBoxTheme()

	// Draw the background box and title
	e.DrawBox(bt, c, lowerRightBox, &e.BoxBackground)

	e.DrawTitle(c, lowerRightBox, title)

	if e.gdb != nil {

		allChangedRegisters, err := e.DebugChangedRegisterMap()
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
			regSlice = regSlice[:listBox.H]
		}

		// Draw the registers without numbers
		e.DrawList(c, listBox, regSlice, -1)

	}

	// Blit
	c.Draw()

	// Reposition the cursor
	if repositionCursor {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		vt100.SetXY(uint(x), uint(y))
	}

	return nil
}

// DrawInstructions will draw a box with the current instructions
func (e *Editor) DrawInstructions(c *vt100.Canvas, repositionCursor bool) error {

	if showInstructionPane {

		// First create a box the size of the entire canvas
		canvasBox := NewCanvasBox(c)

		// Window is the background box that will be drawn in the upper right
		centerBox := NewBox()

		minWidth := 32

		centerBox.EvenLowerRightPlacement(canvasBox, minWidth)
		e.redraw = true

		// Then create a list box
		listBox := NewBox()
		listBox.FillWithMargins(centerBox, 1, 1)

		// Get the current theme for the register box
		bt := NewBoxTheme()

		title := "Instructions"

		// Draw the background box and title
		e.DrawBox(bt, c, centerBox, &e.BoxBackground)
		e.DrawTitle(c, centerBox, title)

		if e.gdb != nil {

			numberOfInstructionsToFetch := 3
			instructions, err := e.DebugDisassemble(numberOfInstructionsToFetch)
			if err != nil {
				return err
			}

			// Cutoff the slice by how high it is, if it's too long
			if len(instructions) > listBox.H {
				instructions = instructions[:listBox.H]
			}

			// Draw the registers without numbers
			e.DrawList(c, listBox, instructions, -1)

		}

		// Blit
		c.Draw()

	}

	// Reposition the cursor
	if repositionCursor {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		vt100.SetXY(uint(x), uint(y))
	}

	return nil
}

func (e *Editor) usingGDBMightWork() bool {
	switch e.mode {
	case mode.Blank, mode.Git, mode.Markdown, mode.Makefile, mode.Shell, mode.Config, mode.Python, mode.Text, mode.CMake, mode.Vim, mode.Clojure, mode.Lisp, mode.Kotlin, mode.Java, mode.Gradle, mode.HIDL, mode.AIDL, mode.SQL, mode.Oak, mode.Lua, mode.Bat, mode.HTML, mode.XML, mode.PolicyLanguage, mode.Nroff, mode.Scala, mode.JSON, mode.CS, mode.JavaScript, mode.TypeScript, mode.ManPage, mode.Amber, mode.Bazel, mode.Perl, mode.M4, mode.Basic, mode.Log:
		// Nope
		return false
	case mode.Zig:
		// Could maybe have worked, but it didn't
		return false
	case mode.GoAssembly, mode.Go, mode.Haskell, mode.OCaml, mode.StandardML, mode.Assembly, mode.V, mode.Crystal, mode.Nim, mode.ObjectPascal, mode.Cpp, mode.Ada, mode.Odin, mode.Battlestar, mode.D, mode.Agda:
		// Maybe, but needs testing
		return true
	case mode.Rust, mode.C:
		// Yes, tested
		return true
	}
	// Unrecognized, assume that gdb might work with it?
	return true
}

// DrawOutput will draw a pane with the 5 last lines of the collected stdoutput from GDB
func (e *Editor) DrawOutput(c *vt100.Canvas, repositionCursor bool) {

	// Check if the output pane should be shown or not
	if e.debugHideOutput {
		return
	}

	const title = "stdout buffer"

	// Gather the GDB stdout so far
	collectedGDBOutput := strings.TrimSpace(gdbOutput.String())

	if l := len(collectedGDBOutput); l > 0 && l != lastGDBOutputLength {
		lastGDBOutputLength = l

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

		// Get the current theme for the watch box
		bt := NewBoxTheme()

		// Draw the background box and title
		e.DrawBox(bt, c, lowerLeftBox, &e.BoxBackground)

		e.DrawTitle(c, lowerLeftBox, title)

		// Get the last 5 lines, and create a string slice
		lines := strings.Split(collectedGDBOutput, "\n")
		if l := len(lines); l > 5 {
			lines = lines[l-5:]
		}

		e.DrawList(c, listBox, lines, -1)

		// Blit
		c.Draw()

	}

	// Reposition the cursor
	if repositionCursor {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		vt100.SetXY(uint(x), uint(y))
	}
}
