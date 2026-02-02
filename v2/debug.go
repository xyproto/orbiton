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
	gdbLogFile               = filepath.Join(userCacheDir, "o", "gdb.log")
	gdbConsole               strings.Builder
	watchMap                 = make(map[string]string)
	lastSeenWatchVariable    string
	showInstructionPane      bool
	gdbOutput                bytes.Buffer
	lastGDBOutputLength      int
	errProgramStopped        = errors.New("program stopped") // must contain "program stopped"
	programRunning           bool
	prevFlags                []string
	longInstructionPaneWidth int // should the instruction pane be extra wide, if so, how wide?
	gdbPathRust              *string
	gdbPathRegular           *string
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
func (e *Editor) DebugStart(sourceDir, sourceBaseFilename, executableBaseFilename string, doneFunc func()) (string, error) {
	if !noWriteToCache {
		flogf(gdbLogFile, "[gdb] dir %s, src %s, exe %s\n", sourceDir, sourceBaseFilename, executableBaseFilename)
	}

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

	// Find the path to either "rust-gdb" or "gdb", depending on the mode
	gdbPath := e.findGDB()

	// flogf(gdbLogFile, "[gdb] starting %s: ", gdbExecutable)

	// Start a new gdb session
	e.gdb, err = gdb.NewCustom([]string{gdbPath}, func(notification map[string]interface{}) {
		// Handle messages from gdb, including frames that contains line numbers
		if payload, ok := notification["payload"]; ok {
			switch notification["type"] {
			case "exec":
				if payloadMap, ok := payload.(map[string]interface{}); ok {
					if frame, ok := payloadMap["frame"]; ok {
						if frameMap, ok := frame.(map[string]interface{}); ok {
							if !noWriteToCache {
								flogf(gdbLogFile, "[gdb] frame: %v\n", frameMap)
							}
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
			case "console":
				// output on stdout
				if s, ok := payload.(string); ok {
					gdbConsole.WriteString(s)
				}
			case "notify":
				// notifications about events that are happening
				if notification["class"] == "thread-group-exited" {
					// gdb is done running
					programRunning = false
					doneFunc()
				}
			default:
				// logf("[gdb] unrecognized notification: %v\n", notification)
			}
			//} else {
			//    logf("[gdb] callback without payload: %v\n", notification)
		}
	})
	if err != nil {
		e.gdb = nil
		// flogf(gdbLogFile, "%s\n", "fail")
		return "", err
	}
	if e.gdb == nil {
		// flogf(gdbLogFile, "%s\n", "fail")
		return "", errors.New("gdb.New returned no error, but e.gdb is nil")
	}
	// flogf(gdbLogFile, "%s\n", "ok")

	// Handle output to stdout (and stderr?) from programs that are being debugged
	go io.Copy(&gdbOutput, e.gdb)

	// Load the executable file
	if retvalMap, err := e.gdb.CheckedSend("file-exec-and-symbols", executableBaseFilename); err != nil {
		return fmt.Sprintf("%v", retvalMap), err
	}

	// Pass in arguments
	// e.gdb.Send("exec-arguments", "--version")

	// Pass the breakpoint, if it has been set with ctrl-b
	if e.breakpoint != nil {
		if retvalMap, err := e.gdb.CheckedSend("break-insert", fmt.Sprintf("%s:%d", sourceBaseFilename, e.breakpoint.LineNumber())); err != nil {
			return fmt.Sprintf("%v", retvalMap), err
		}
	}

	// Assembly specific
	if e.mode == mode.Assembly {
		e.gdb.Send("break-insert", "-t", "1")
	}

	// Set the disassembly style
	e.gdb.Send("gdb-set", "disassembly-flavor", "intel")

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

	programRunning = true

	return "started gdb", nil
}

// DebugContinue will continue the execution to the next breakpoint or to the end.
// e.gdb must not be nil.
func (e *Editor) DebugContinue() error {
	if !programRunning {
		return errProgramStopped
	}
	_, err := e.gdb.CheckedSend("exec-continue")
	return err
}

// DebugRun will run the current program.
// e.gdb must not be nil.
func (e *Editor) DebugRun() error {
	_, err := e.gdb.CheckedSend("exec-run")
	return err
}

// DebugNext will continue the execution by stepping to the next line.
// e.gdb must not be nil.
func (e *Editor) DebugNext() error {
	if !programRunning {
		return errProgramStopped
	}
	gdbMI := "exec-next"
	if e.debugStepInto {
		gdbMI = "exec-step"
	}
	if _, err := e.gdb.CheckedSend(gdbMI); err != nil {
		return err
	}
	consoleString := strings.TrimSpace(gdbConsole.String())
	gdbConsole.Reset()
	// Interpret consoleString and extract the new variable names and values,
	// for variables there are watchpoints for.
	if consoleString != "" {
		var varName string
		for _, line := range strings.Split(consoleString, "\n") {
			if strings.Contains(line, "watchpoint") && strings.Contains(line, ":") {
				fields := strings.SplitN(line, ":", 2)
				varName = strings.TrimSpace(fields[1])
			} else if varName != "" && strings.HasPrefix(line, "New value =") {
				fields := strings.SplitN(line, "=", 2)
				watchMap[varName] = strings.TrimSpace(fields[1])
				lastSeenWatchVariable = varName
				varName = ""
			}
		}
	}
	if !programRunning {
		return errProgramStopped
	}
	return nil
}

// DebugNextInstruction will continue the execution by stepping to the next instruction.
// e.gdb must not be nil.
func (e *Editor) DebugNextInstruction() error {
	if !programRunning {
		return errProgramStopped
	}
	showInstructionPane = true
	gdbMI := "exec-next-instruction"
	if e.debugStepInto {
		gdbMI = "exec-step-instruction"
	}
	_, err := e.gdb.CheckedSend(gdbMI)
	if err != nil {
		return err
	}
	consoleString := strings.TrimSpace(gdbConsole.String())
	gdbConsole.Reset()
	// Interpret consoleString and extract the new variable names and values,
	// for variables there are watchpoints for.
	if consoleString != "" {
		var varName string
		for _, line := range strings.Split(consoleString, "\n") {
			if strings.Contains(line, "watchpoint") && strings.Contains(line, ":") {
				fields := strings.SplitN(line, ":", 2)
				varName = strings.TrimSpace(fields[1])
			} else if varName != "" && strings.HasPrefix(line, "New value =") {
				fields := strings.SplitN(line, "=", 2)
				watchMap[varName] = strings.TrimSpace(fields[1])
				lastSeenWatchVariable = varName
				varName = ""
			}
		}
	}
	if !programRunning {
		return errProgramStopped
	}
	return nil
}

// DebugStep will continue the execution by stepping.
// e.gdb must not be nil.
func (e *Editor) DebugStep() error {
	if !programRunning {
		return errProgramStopped
	}
	if _, err := e.gdb.CheckedSend("exec-step"); err != nil {
		return err
	}
	consoleString := strings.TrimSpace(gdbConsole.String())
	gdbConsole.Reset()
	// Interpret consoleString and extract the new variable names and values,
	// for variables there are watchpoints for.
	if consoleString != "" {
		var varName string
		for _, line := range strings.Split(consoleString, "\n") {
			if strings.Contains(line, "watchpoint") && strings.Contains(line, ":") {
				fields := strings.SplitN(line, ":", 2)
				varName = strings.TrimSpace(fields[1])
			} else if varName != "" && strings.HasPrefix(line, "New value =") {
				fields := strings.SplitN(line, "=", 2)
				watchMap[varName] = strings.TrimSpace(fields[1])
				lastSeenWatchVariable = varName
				varName = ""
			}
		}
	}
	if !programRunning {
		return errProgramStopped
	}
	return nil
}

// DebugFinish will "step out".
// e.gdb must not be nil. Returns whatever was outputted to gdb stdout.
func (e *Editor) DebugFinish() error {
	_, err := e.gdb.CheckedSend("exec-finish")
	if err != nil {
		return err
	}
	consoleString := strings.TrimSpace(gdbConsole.String())
	gdbConsole.Reset()
	// Interpret consoleString and extract the new variable names and values,
	// for variables there are watchpoints for.
	if consoleString != "" {
		var varName string
		for _, line := range strings.Split(consoleString, "\n") {
			if strings.Contains(line, "watchpoint") && strings.Contains(line, ":") {
				fields := strings.SplitN(line, ":", 2)
				varName = strings.TrimSpace(fields[1])
			} else if varName != "" && strings.HasPrefix(line, "New value =") {
				fields := strings.SplitN(line, "=", 2)
				watchMap[varName] = strings.TrimSpace(fields[1])
				lastSeenWatchVariable = varName
				varName = ""
			}
		}
	}
	if !programRunning {
		return errProgramStopped
	}
	return nil
}

// DebugRegisterNames will return all register names
func (e *Editor) DebugRegisterNames() ([]string, error) {
	if e.gdb == nil {
		return []string{}, errors.New("gdb must be running")
	}
	notification, err := e.gdb.CheckedSend("data-list-register-names")
	if err != nil {
		// flogf(gdbLogFile, "[gdb] data-list-register-names error: %s\n", err.Error())
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
					// flogf(gdbLogFile, "[gdb] data-list-register-names: %s\n", strings.Join(registerStringSlice, ","))
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
		// flogf(gdbLogFile, "[gdb] data-list-changed-registers error: %s\n", err.Error())
		return []int{}, err
	}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]interface{}); ok {
			// flogf(gdbLogFile, "[gdb] changed reg payload: %v\n", payloadMap)
			if registerInterfaces, ok := payloadMap["changed-registers"].([]interface{}); ok {
				changedRegisters := make([]int, len(registerInterfaces))
				for _, registerNumberString := range registerInterfaces {
					registerNumber, err := strconv.Atoi(registerNumberString.(string))
					if err != nil {
						return []int{}, err
					}
					changedRegisters = append(changedRegisters, registerNumber)
					// flogf(gdbLogFile, "[gdb] regnum %v %T\n", registerNumber, registerNumber)
				}
				return changedRegisters, nil
			}
		}
	}
	// flogf(gdbLogFile, "[gdb] data-list-register-values %v\n", registers)
	return []int{}, errors.New("could not find the register values in the payload returned from gdb")
}

// DebugDisassemble will return the next N assembly instructions
func (e *Editor) DebugDisassemble(n int) ([]string, error) {
	// Then get the register values
	notification, err := e.gdb.CheckedSend("data-disassemble -s $pc -e \"$pc + 20\" -- 0")
	if err != nil {
		// flogf(gdbLogFile, "[gdb] data-disassemble error: %s\n", err.Error())
		return []string{}, err
	}
	result := []string{}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]interface{}); ok {
			// flogf(gdbLogFile, "[gdb] disasm payload: %v\n", payloadMap)
			if asmDisSlice, ok := payloadMap["asm_insns"].([]interface{}); ok {
				for i, asmDis := range asmDisSlice {
					// flogf(gdbLogFile, "[gdb] disasm asm %d: %v %T\n", i, asmDis, asmDis)
					if asmMap, ok := asmDis.(map[string]interface{}); ok {
						instruction := asmMap["inst"]
						result = append(result, instruction.(string))
					}
					// Only collect n asm statements
					if i >= n {
						// flogf(gdbLogFile, "[gdb] result %v\n", result)
						break
					}
				}
				return result, nil
			}
		}
	}
	// flogf(gdbLogFile, "[gdb] disasm result: %v\n", result)
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
		// flogf(gdbLogFile, "[gdb] data-list-register-values error: %s\n", err.Error())
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
							// flogf(gdbLogFile, "[gdb] data-list-register-values: %s %s\n", registerName, value)
						}
					}

					reg8byte := []string{"rax", "rcx", "rdx", "rbx", "rsi", "rdi", "rsp", "rbp", "r8", "r9", "r10", "r11", "r12", "r13", "r14", "r15"}
					reg4byte := []string{"eax", "ecx", "edx", "ebx", "esi", "edi", "esp", "ebp", "r8d", "r9d", "r10d", "r11d", "r12d", "r13d", "r14d", "r15d"}
					reg2byte := []string{"ax", "cx", "dx", "bx", "si", "di", "sp", "bp", "r8w", "r9w", "r10w", "r11w", "r12w", "r13w", "r14w", "r15w"}
					reg1byteL := []string{"al", "cl", "dl", "bl", "sil", "dil", "spl", "bpl", "r8b", "r9b", "r10b", "r11b", "r12b", "r13b", "r14b", "r15b"}
					reg1byteH := []string{"ah", "ch", "dh", "bh", "sil", "dil", "spl", "bpl", "r8b", "r9b", "r10b", "r11b", "r12b", "r13b", "r14b", "r15b"}

					// If only the right half of ie. rax has changed, delete rax from the list
					// If only the right half of ie. eax has changed, delete eax from the list
					// If only the right half of ie. ax has changed, delete ax from the list
					// But always keep al and ah

					// TODO: Think this through a bit better!

					filterRegisters := e.debugShowRegisters != largeRegisterWindow
					if filterRegisters {
						for _, regSlice := range [][]string{reg8byte, reg4byte, reg2byte} {
							for _, regName := range regSlice {
								if !hasKey(registers, regName) {
									continue
								}

								// Removing sub-registers goes here
								// If ie. "rax" is present, filter out "eax", "ax", "ah" and "al"
								for i, r8b := range reg8byte {
									if hasKey(registers, r8b) {
										delete(registers, reg4byte[i])
										delete(registers, reg2byte[i])
										delete(registers, reg1byteL[i])
										delete(registers, reg1byteH[i])
									}
								}
								// If ie. "eax" is present, filter out "ax", "ah" and "al"
								for i, r4b := range reg4byte {
									if hasKey(registers, r4b) {
										delete(registers, reg2byte[i])
										delete(registers, reg1byteL[i])
										delete(registers, reg1byteH[i])
									}
								}
								// If ie. "ax" is present, filter out "ah" and "al"
								for i, r2b := range reg2byte {
									if hasKey(registers, r2b) {
										delete(registers, reg1byteL[i])
										delete(registers, reg1byteH[i])
									}
								}

							}
						}
					}

					// Filter out "eflags" since it's covered by the status in the lower right corner
					delete(registers, "eflags")

					return registers, nil
				}
			}
		}
	}
	// flogf(gdbLogFile, "[gdb] data-list-register-values %v\n", registers)
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
		// flogf(gdbLogFile, "[gdb] data-list-register-values error: %s\n", err.Error())
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
							// flogf(gdbLogFile, "[gdb] data-list-register-values: %s %s\n", registerName, value)
						}
					}
					return registers, nil
				}
			}
		}
	}
	// flogf(gdbLogFile, "[gdb] data-list-register-values %v\n", registers)
	return nil, errors.New("could not find the register values in the payload returned from gdb")
}

// DebugEnd will end the current gdb session, but not set debugMode to false
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
	programRunning = false
	longInstructionPaneWidth = 0
	// flogf(gdbLogFile, "[gdb] %s\n", "stopped")
}

// AddWatch will add a watchpoint / watch expression to gdb
func (e *Editor) AddWatch(expression string) (string, error) {
	var output string
	if e.gdb != nil {
		// flogf(gdbLogFile, "[gdb] adding watch: %s\n", expression)
		_, err := e.gdb.CheckedSend("break-watch", "-a", expression)
		if err != nil {
			return "", err
		}
		output = gdbOutput.String()
		gdbOutput.Reset()
		// flogf(gdbLogFile, "[gdb] output after adding watch: %s\n", output)
	}
	watchMap[expression] = "?"

	// Don't set this, the variable watch has not been seen yet
	// lastSeenWatchVariable = expression

	return output, nil
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

	// Find the available space to draw help text in the upperRightBox
	availableHeight := (listBox.H - marginY) - 1
	if availableHeight < 2 {
		// Draw at least two rows of help text, no matter what
		availableHeight = 2
	}
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
		if foundLastSeen && e.gdb != nil {
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
	if e.gdb == nil {
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
	// data-evalutate-expression is the same as print, output and call in gdb
	if notification, err := e.gdb.CheckedSend("data-evaluate-expression", "$eflags"); err == nil {
		if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
			if payloadMap, ok := payload.(map[string]interface{}); ok {
				if flagNames, ok := payloadMap["value"]; ok {
					if flagNamesString, ok := flagNames.(string); ok {
						flagNamesString = strings.TrimPrefix(flagNamesString, "[ ")
						flagNamesString = strings.TrimSuffix(flagNamesString, " ]")
						flags := strings.Split(flagNamesString, " ")
						// Find which flags changed since last step
						for _, flag := range flags {
							if !hasS(prevFlags, flag) {
								changedFlags = append(changedFlags, flag)
							}
						}
						prevFlags = flags
					}
				}
			}
		}
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

	if e.debugShowRegisters == noRegisterWindow || e.gdb == nil {
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

	if showInstructionPane && e.gdb != nil {

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

		if e.gdb != nil {

			numberOfInstructionsToFetch := 5
			instructions, err := e.DebugDisassemble(numberOfInstructionsToFetch)
			if err != nil { // We end up here if the program is done running, when stepping
				if err.Error() == "No registers." {
					programRunning = false
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
				for _, word := range strings.Fields(line) {
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

// DrawGDBOutput will draw a pane with the 5 last lines of the collected stdoutput from GDB
func (e *Editor) DrawGDBOutput(c *vt.Canvas, repositionCursor bool) {
	// Check if the output pane should be shown or not
	if e.debugHideOutput || e.gdb == nil {
		return
	}

	const title = "stdout"

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

		// Get the current theme for the stdout box
		bt := e.NewBoxTheme()
		bt.Background = &e.DebugOutputBackground
		bt.UpperEdge = bt.LowerEdge

		e.DrawBox(bt, c, lowerLeftBox)

		e.DrawTitle(bt, c, lowerLeftBox, title, true)

		// Get the last 5 lines, and create a string slice
		lines := strings.Split(collectedGDBOutput, "\n")
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

// DebugStartSession builds and then connects to gdb
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

	outputExecutableClean := filepath.Clean(filepath.Join(filepath.Dir(absFilename), outputExecutable))
	if !files.Exists(outputExecutableClean) {
		e.debugMode = false
		e.redrawCursor.Store(true)
		return errors.New("could not find " + outputExecutableClean)
	}

	// Start GDB execution from the top
	msg, err := e.DebugStart(filepath.Dir(absFilename), filepath.Base(absFilename), outputExecutable, func() {
		// This happens when the program running under GDB is done running.
		programRunning = false
		status.SetMessageAfterRedraw("Execution complete")
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
	})
	if err != nil || e.gdb == nil {
		e.redrawCursor.Store(true)
		if msg != "" {
			msg += ", "
		}
		msg += err.Error()
		return errors.New("could not start debugging: " + msg)
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

// findGDB will find "rust-gdb" for mode.Rust or "gdb" for other
// modes, but in a memoized way to avoid more than one lookup of each.
func (e *Editor) findGDB() string {
	// Use rust-gdb if we are debugging Rust
	if e.mode == mode.Rust {
		if gdbPathRust == nil {
			path := files.WhichCached("rust-gdb")
			gdbPathRust = &path
			return path
		}
		return *gdbPathRust
	}
	if gdbPathRegular == nil {
		path := files.WhichCached("gdb")
		gdbPathRegular = &path
		return path
	}
	return *gdbPathRegular
}
