package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	//"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cyrus-and/gdb"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

var (
	gdbOutput         bytes.Buffer
	originalDirectory string
	//gdbLogFile            = filepath.Join(userCacheDir, "o", "gdb.log")
	gdbConsole            strings.Builder
	watchMap              = make(map[string]string)
	lastSeenWatchVariable string
	prevLineNumber        = -1
)

// DebugStart will start a new debug session, using gdb.
// Will end the existing session first if e.gdb != nil.
func (e *Editor) DebugStart(sourceDir, sourceBaseFilename, executableBaseFilename string) (string, error) {

	//flogf(gdbLogFile, "[gdb] dir %s, src %s, exe %s\n", sourceDir, sourceBaseFilename, executableBaseFilename)

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
		gdbExecutable = which("rust-db")
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
						//flogf(gdbLogFile, "[gdb] frame: %v\n", frameMap)
						if lineNumberString, ok := frameMap["line"].(string); ok {
							if lineNumber, err := strconv.Atoi(lineNumberString); err == nil { // success
								// Got a line number, send the editor there, without any status messages
								if prevLineNumber != -1 {
									e.GoToLineNumber(LineNumber(prevLineNumber), nil, nil, true)
								}
								prevLineNumber = lineNumber
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
// e.gdb must not be nil. Returns whatever was outputted to gdb stdout.
func (e *Editor) DebugContinue() (string, error) {
	_, err := e.gdb.CheckedSend("exec-continue")
	if err != nil {
		return "", err
	}
	output := gdbOutput.String()
	gdbOutput.Reset()
	return output, nil
}

// DebugNext will continue the execution by stepping to the next instruction.
// e.gdb must not be nil. Returns whatever was outputted to gdb stdout.
func (e *Editor) DebugNext() (string, error) {
	_, err := e.gdb.CheckedSend("exec-next")
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

// DebugStep will continue the execution by stepping to the next line.
// e.gdb must not be nil. Returns whatever was outputted to gdb stdout.
func (e *Editor) DebugStep() (string, error) {
	_, err := e.gdb.CheckedSend("exec-step")
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
	// Also change to the original directory
	if originalDirectory != "" {
		os.Chdir(originalDirectory)
	}
	prevLineNumber = -1
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

	// Window is the background box that will be drawn in the upper right
	upperRightBox := NewBox()
	upperRightBox.UpperRightPlacement(canvasBox)

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(upperRightBox, 2)

	// Get the current theme for the watch box
	bt := NewBoxTheme()

	// Draw the background box and title
	e.DrawBox(bt, c, upperRightBox, true)

	title := "Running"
	if e.gdb == nil {
		title = "Not running"
	}
	if len(watchMap) == 0 {
		helpSlice := []string{
			"ctrl-space to step",
			"ctrl-w to add a watch",
			//"ctrl-r to toggle the register view",
			//"",
			//"gdb log: " + prettyPath(gdbLogFile),
		}
		// Draw the help text
		e.DrawList(c, listBox, helpSlice, -1)
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
	if e.showRegisters == 2 {
		// Don't draw anything
		return nil
	}
	filterWeirdRegisters := e.showRegisters != 1

	// First create a box the size of the entire canvas
	canvasBox := NewCanvasBox(c)

	// Window is the background box that will be drawn in the upper right
	lowerRightBox := NewBox()

	var title string
	if filterWeirdRegisters {
		title = "Changed registers"
		// narrow box
		lowerRightBox.LowerRightPlacement(canvasBox)
		e.redraw = true
	} else {
		title = "All changed registers"
		// wide box
		lowerRightBox.LowerPlacement(canvasBox)
		e.redraw = true
	}

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(lowerRightBox, 2)

	// Get the current theme for the register box
	bt := NewBoxTheme()

	// Draw the background box and title
	e.DrawBox(bt, c, lowerRightBox, true)

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
