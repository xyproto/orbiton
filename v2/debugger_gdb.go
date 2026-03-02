package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cyrus-and/gdb"
	"github.com/xyproto/files"
	"github.com/xyproto/mode"
)

var (
	gdbPathRust    *string
	gdbPathRegular *string
	debugLogFile   = filepath.Join(userCacheDir, "o", "gdb.log")
)

// compile-time check that gdbDebugger implements Debugger
var _ Debugger = (*gdbDebugger)(nil)

type gdbDebugger struct {
	conn            *gdb.Gdb
	watchMap        map[string]string
	stopped         chan struct{} // signaled when a *stopped exec notification arrives
	lastWatch       string
	console         strings.Builder
	output          bytes.Buffer
	mode            mode.Mode
	running         bool
	recording       bool // true when "record full" is active
	stepInto        bool
	filterRegisters bool
}

func newGDBDebugger(m mode.Mode) *gdbDebugger {
	return &gdbDebugger{
		watchMap: make(map[string]string),
		mode:     m,
		stopped:  make(chan struct{}, 1),
	}
}

// findGDB will find "rust-gdb" for mode.Rust or "gdb" for other
// modes, but in a memoized way to avoid more than one lookup of each.
func findGDB(m mode.Mode) string {
	if m == mode.Rust {
		if gdbPathRust == nil {
			path := files.WhichCached("rust-gdb")
			gdbPathRust = &path
		}
		return *gdbPathRust
	}
	if gdbPathRegular == nil {
		path := files.WhichCached("gdb")
		if path == "" && isDarwin {
			path = files.WhichCached("ggdb")
		}
		gdbPathRegular = &path
	}
	return *gdbPathRegular
}

// Start begins a new debug session using GDB.
func (d *gdbDebugger) Start(sourceDir, sourceBaseFilename, executableBaseFilename string, lineFunc func(int), doneFunc func()) (string, error) {
	if !noWriteToCache {
		flogf(debugLogFile, "[gdb] dir %s, src %s, exe %s\n", sourceDir, sourceBaseFilename, executableBaseFilename)
	}

	// End any existing sessions
	d.End()

	// Change directory to the sourcefile, temporarily
	var err error
	originalDirectory, err = os.Getwd()
	if err == nil {
		err = os.Chdir(sourceDir)
		if err != nil {
			return "", errors.New("could not change directory to " + sourceDir)
		}
	}

	// Find the path to either "rust-gdb" or "gdb", depending on the mode
	gdbPath := findGDB(d.mode)

	// Start a new gdb session
	d.conn, err = gdb.NewCustom([]string{gdbPath}, func(notification map[string]any) {
		// If the program hit an exit syscall catchpoint, transparently stop
		// recording and continue so the program can exit without hanging.
		if notification["type"] == "exec" && notification["class"] == "stopped" {
			if isExitSyscallCatchpoint(notification) {
				go func() {
					conn := d.conn
					if conn == nil {
						return
					}
					if d.recording {
						conn.Send("interpreter-exec", "console", "record stop")
						d.recording = false
					}
					conn.Send("exec-continue")
				}()
				return
			}
			select {
			case d.stopped <- struct{}{}:
			default:
			}
		}
		if notification["type"] == "notify" && notification["class"] == "thread-group-exited" {
			d.running = false
			select {
			case d.stopped <- struct{}{}:
			default:
			}
			doneFunc()
		}

		// Handle messages from gdb, including frames that contains line numbers
		if payload, ok := notification["payload"]; ok {
			switch notification["type"] {
			case "exec":
				if payloadMap, ok := payload.(map[string]any); ok {
					if frame, ok := payloadMap["frame"]; ok {
						if frameMap, ok := frame.(map[string]any); ok {
							if !noWriteToCache {
								flogf(debugLogFile, "[gdb] frame: %v\n", frameMap)
							}
							if lineNumberString, ok := frameMap["line"].(string); ok {
								if lineNumber, err := strconv.Atoi(lineNumberString); err == nil {
									lineFunc(lineNumber)
								}
							}
						}
					}
				}
			case "console":
				if s, ok := payload.(string); ok {
					d.console.WriteString(s)
				}
			default:
			}
		}
	})
	if err != nil {
		d.conn = nil
		return "", err
	}
	if d.conn == nil {
		return "", errors.New("gdb.New returned no error, but conn is nil")
	}

	// Handle output to stdout from programs that are being debugged
	go io.Copy(&d.output, d.conn)

	// Load the executable file
	if retvalMap, err := d.conn.CheckedSend("file-exec-and-symbols", executableBaseFilename); err != nil {
		return fmt.Sprintf("%v", retvalMap), err
	}

	// Pass the breakpoint is handled by the caller via ActivateBreakpoint

	return "started gdb", nil
}

// ActivateBreakpoint sets a breakpoint at the given file and line.
func (d *gdbDebugger) ActivateBreakpoint(file string, line int) error {
	if d.conn == nil {
		return errors.New("gdb is not running")
	}
	if retvalMap, err := d.conn.CheckedSend("break-insert", fmt.Sprintf("%s:%d", file, line)); err != nil {
		return fmt.Errorf("%v: %w", retvalMap, err)
	}
	return nil
}

// SetupAndRun configures assembly mode, disassembly style, and starts execution.
// It also re-adds existing watches.
func (d *gdbDebugger) SetupAndRun(assemblyMode bool) (string, error) {
	if assemblyMode {
		d.conn.Send("break-insert", "-t", "1")
	}

	if runtime.GOARCH == "amd64" || runtime.GOARCH == "386" {
		d.conn.Send("gdb-set", "disassembly-flavor", "intel")
	}

	if isDarwin {
		d.conn.Send("gdb-set", "startup-with-shell", "off")
		// Some versions of GDB on macOS struggle with "exec-run --start"
		// using the MI interface. A more robust approach is setting a
		// temporary breakpoint at main and then just running.
		d.conn.Send("break-insert", "-t", "main")
		if _, err := d.conn.CheckedSend("exec-run"); err != nil {
			output := d.output.String()
			d.output.Reset()
			return output, err
		}
	} else {
		if _, err := d.conn.CheckedSend("exec-run", "--start"); err != nil {
			output := d.output.String()
			d.output.Reset()
			return output, err
		}
	}

	if !isDarwin {
		// Enable execution recording for reverse stepping
		d.conn.Send("interpreter-exec", "console", "record full")
		d.recording = true

		// Catch exit syscalls so we can stop recording before GDB tries to
		// record them (which causes GDB to hang with record full).
		d.conn.Send("interpreter-exec", "console", "catch syscall exit_group")
		d.conn.Send("interpreter-exec", "console", "catch syscall exit")
	}

	// Add any existing watches
	for varName := range d.watchMap {
		d.AddWatch(varName)
	}

	d.running = true

	// Drain any stale stop signals
	select {
	case <-d.stopped:
	default:
	}

	return "started gdb", nil
}

// isExitSyscallCatchpoint checks if a GDB notification is a *stopped event
// caused by hitting an exit syscall catchpoint (set to prevent record full hangs).
func isExitSyscallCatchpoint(notification map[string]any) bool {
	payload, ok := notification["payload"]
	if !ok {
		return false
	}
	payloadMap, ok := payload.(map[string]any)
	if !ok {
		return false
	}
	reason, _ := payloadMap["reason"].(string)
	return reason == "syscall-entry"
}

// nextInstructionIsSyscall disassembles the instruction at $pc and returns
// true if it is a "syscall" instruction. This is used to avoid stepping into
// an exit syscall while record full is active, which hangs GDB.
func (d *gdbDebugger) nextInstructionIsSyscall() bool {
	if d.conn == nil {
		return false
	}
	instructions, err := d.Disassemble(1)
	if err != nil || len(instructions) == 0 {
		return false
	}
	return strings.TrimSpace(instructions[0]) == "syscall"
}

// waitForStop waits for a *stopped notification from GDB.
// If the step hangs (e.g., syscall with record full), it interrupts GDB,
// disables recording, and returns true to indicate the step was interrupted.
func (d *gdbDebugger) waitForStop() bool {
	// Wait for the primary stopped signal
	select {
	case <-d.stopped:
		// Step completed. Drain a possible follow-up (e.g., thread-group-exited).
		select {
		case <-d.stopped:
		case <-time.After(50 * time.Millisecond):
		}
		return false
	case <-time.After(2 * time.Second):
		// Step is taking too long — likely hung on a recorded syscall
	}

	// Interrupt GDB to regain control
	if d.conn != nil {
		d.conn.Interrupt()
	}

	// Wait for GDB to acknowledge the interrupt
	select {
	case <-d.stopped:
	case <-time.After(2 * time.Second):
	}

	// Disable recording to prevent future hangs on this instruction
	if d.conn != nil && d.recording {
		done := make(chan struct{})
		go func() {
			d.conn.Send("interpreter-exec", "console", "record stop")
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
		d.recording = false
	}

	return true
}

// End terminates the current gdb session.
func (d *gdbDebugger) End() {
	if d.conn != nil {
		conn := d.conn
		d.conn = nil
		// Run Exit in a goroutine with a timeout so a hung GDB cannot freeze the editor
		done := make(chan struct{})
		go func() {
			conn.Exit()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	d.output.Reset()
	d.console.Reset()
	d.lastWatch = ""
	if originalDirectory != "" {
		os.Chdir(originalDirectory)
	}
	d.running = false
	d.recording = false
	longInstructionPaneWidth = 0
}

// Continue resumes execution to the next breakpoint or end.
func (d *gdbDebugger) Continue() error {
	if !d.running {
		return errProgramStopped
	}
	_, err := d.conn.CheckedSend("exec-continue")
	return err
}

// Run starts (or restarts) the program from the beginning.
func (d *gdbDebugger) Run() error {
	_, err := d.conn.CheckedSend("exec-run")
	return err
}

// parseWatchOutput interprets gdb console output and extracts watch variable changes.
func (d *gdbDebugger) parseWatchOutput() {
	consoleString := strings.TrimSpace(d.console.String())
	d.console.Reset()
	if consoleString != "" {
		var varName string
		for line := range strings.SplitSeq(consoleString, "\n") {
			if strings.Contains(line, "watchpoint") && strings.Contains(line, ":") {
				fields := strings.SplitN(line, ":", 2)
				varName = strings.TrimSpace(fields[1])
			} else if varName != "" && strings.HasPrefix(line, "New value =") {
				fields := strings.SplitN(line, "=", 2)
				d.watchMap[varName] = strings.TrimSpace(fields[1])
				d.lastWatch = varName
				varName = ""
			}
		}
	}
}

// Next steps to the next source line (over calls).
func (d *gdbDebugger) Next() error {
	if !d.running {
		return errProgramStopped
	}
	if d.nextInstructionIsSyscall() {
		return errProgramStopped
	}
	gdbMI := "exec-next"
	if d.stepInto {
		gdbMI = "exec-step"
	}
	if _, err := d.conn.CheckedSend(gdbMI); err != nil {
		return err
	}
	if d.waitForStop() {
		return errRecordingStopped
	}
	d.parseWatchOutput()
	if !d.running || d.nextInstructionIsSyscall() {
		return errProgramStopped
	}
	return nil
}

// NextInstruction steps to the next machine instruction.
func (d *gdbDebugger) NextInstruction() error {
	if !d.running {
		return errProgramStopped
	}
	if d.nextInstructionIsSyscall() {
		return errProgramStopped
	}
	showInstructionPane = true
	gdbMI := "exec-next-instruction"
	if d.stepInto {
		gdbMI = "exec-step-instruction"
	}
	_, err := d.conn.CheckedSend(gdbMI)
	if err != nil {
		return err
	}
	if d.waitForStop() {
		return errRecordingStopped
	}
	d.parseWatchOutput()
	if !d.running || d.nextInstructionIsSyscall() {
		return errProgramStopped
	}
	return nil
}

// Step performs a single source-level step (into calls).
func (d *gdbDebugger) Step() error {
	if !d.running {
		return errProgramStopped
	}
	if d.nextInstructionIsSyscall() {
		return errProgramStopped
	}
	if _, err := d.conn.CheckedSend("exec-step"); err != nil {
		return err
	}
	if d.waitForStop() {
		return errRecordingStopped
	}
	d.parseWatchOutput()
	if !d.running || d.nextInstructionIsSyscall() {
		return errProgramStopped
	}
	return nil
}

// Finish runs until the current function returns (step out).
func (d *gdbDebugger) Finish() error {
	if d.nextInstructionIsSyscall() {
		return errProgramStopped
	}
	_, err := d.conn.CheckedSend("exec-finish")
	if err != nil {
		return err
	}
	if d.waitForStop() {
		return errRecordingStopped
	}
	d.parseWatchOutput()
	if !d.running || d.nextInstructionIsSyscall() {
		return errProgramStopped
	}
	return nil
}

// ReverseStep steps backward one source line.
func (d *gdbDebugger) ReverseStep() error {
	if !d.running {
		return errProgramStopped
	}
	_, err := d.conn.CheckedSend("interpreter-exec", "console", "reverse-step")
	if err != nil {
		return err
	}
	if d.waitForStop() {
		return errRecordingStopped
	}
	d.parseWatchOutput()
	if !d.running {
		return errProgramStopped
	}
	return nil
}

// ReverseNextInstruction steps backward one machine instruction.
func (d *gdbDebugger) ReverseNextInstruction() error {
	if !d.running {
		return errProgramStopped
	}
	_, err := d.conn.CheckedSend("interpreter-exec", "console", "reverse-stepi")
	if err != nil {
		return err
	}
	if d.waitForStop() {
		return errRecordingStopped
	}
	d.parseWatchOutput()
	if !d.running {
		return errProgramStopped
	}
	return nil
}

// RegisterNames returns all register names.
func (d *gdbDebugger) RegisterNames() ([]string, error) {
	if d.conn == nil {
		return []string{}, errors.New("gdb must be running")
	}
	notification, err := d.conn.CheckedSend("data-list-register-names")
	if err != nil {
		return []string{}, err
	}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]any); ok {
			if registerNames, ok := payloadMap["register-names"]; ok {
				if registerSlice, ok := registerNames.([]any); ok {
					registerStringSlice := make([]string, len(registerSlice))
					for i, interfaceValue := range registerSlice {
						if s, ok := interfaceValue.(string); ok {
							registerStringSlice[i] = s
						}
					}
					return registerStringSlice, nil
				}
			}
		}
	}
	return []string{}, errors.New("could not find the register names in the payload returned from gdb")
}

// ChangedRegisters returns a list of all changed register numbers.
func (d *gdbDebugger) ChangedRegisters() ([]int, error) {
	notification, err := d.conn.CheckedSend("data-list-changed-registers")
	if err != nil {
		return []int{}, err
	}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]any); ok {
			if registerInterfaces, ok := payloadMap["changed-registers"].([]any); ok {
				changedRegisters := make([]int, 0, len(registerInterfaces))
				for _, registerNumberString := range registerInterfaces {
					registerNumber, err := strconv.Atoi(registerNumberString.(string))
					if err != nil {
						return []int{}, err
					}
					changedRegisters = append(changedRegisters, registerNumber)
				}
				return changedRegisters, nil
			}
		}
	}
	return []int{}, errors.New("could not find the register values in the payload returned from gdb")
}

// Disassemble returns the next n assembly instructions.
func (d *gdbDebugger) Disassemble(n int) ([]string, error) {
	notification, err := d.conn.CheckedSend("data-disassemble -s $pc -e \"$pc + 20\" -- 0")
	if err != nil {
		return []string{}, err
	}
	result := []string{}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]any); ok {
			if asmDisSlice, ok := payloadMap["asm_insns"].([]any); ok {
				for i, asmDis := range asmDisSlice {
					if asmMap, ok := asmDis.(map[string]any); ok {
						instruction := asmMap["inst"]
						result = append(result, instruction.(string))
					}
					if i >= n {
						break
					}
				}
				return result, nil
			}
		}
	}
	return []string{}, errors.New("could not get disasm from gdb")
}

// ChangedRegisterMap returns a map of all registers that were changed the last step, and their values.
func (d *gdbDebugger) ChangedRegisterMap() (map[string]string, error) {
	names, err := d.RegisterNames()
	if err != nil {
		return nil, err
	}
	changedRegisters, err := d.ChangedRegisters()
	if err != nil {
		return nil, err
	}

	notification, err := d.conn.CheckedSend("data-list-register-values", "--skip-unavailable", "x")
	if err != nil {
		return nil, err
	}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]any); ok {
			if registerValues, ok := payloadMap["register-values"]; ok {
				if registerSlice, ok := registerValues.([]any); ok {
					registers := make(map[string]string, len(names))
					for _, singleRegisterMap := range registerSlice {
						if registerMap, ok := singleRegisterMap.(map[string]any); ok {
							numberString, ok := registerMap["number"].(string)
							if !ok {
								return nil, errors.New("could not convert \"number\" interface to string")
							}
							registerNumber, err := strconv.Atoi(numberString)
							if err != nil {
								return nil, err
							}
							thisRegisterWasChanged := slices.Contains(changedRegisters, registerNumber)
							if !thisRegisterWasChanged {
								continue
							}
							value, ok := registerMap["value"].(string)
							if !ok {
								return nil, errors.New("could not convert \"value\" interface to string")
							}
							registerName := names[registerNumber]
							registers[registerName] = value
						}
					}

					reg8byte := []string{"rax", "rcx", "rdx", "rbx", "rsi", "rdi", "rsp", "rbp", "r8", "r9", "r10", "r11", "r12", "r13", "r14", "r15"}
					reg4byte := []string{"eax", "ecx", "edx", "ebx", "esi", "edi", "esp", "ebp", "r8d", "r9d", "r10d", "r11d", "r12d", "r13d", "r14d", "r15d"}
					reg2byte := []string{"ax", "cx", "dx", "bx", "si", "di", "sp", "bp", "r8w", "r9w", "r10w", "r11w", "r12w", "r13w", "r14w", "r15w"}
					reg1byteL := []string{"al", "cl", "dl", "bl", "sil", "dil", "spl", "bpl", "r8b", "r9b", "r10b", "r11b", "r12b", "r13b", "r14b", "r15b"}
					reg1byteH := []string{"ah", "ch", "dh", "bh", "sil", "dil", "spl", "bpl", "r8b", "r9b", "r10b", "r11b", "r12b", "r13b", "r14b", "r15b"}

					if d.filterRegisters {
						if runtime.GOARCH == "amd64" || runtime.GOARCH == "386" {
							for i, r64 := range reg8byte {
								if hasKey(registers, r64) {
									delete(registers, reg4byte[i])
									delete(registers, reg2byte[i])
									delete(registers, reg1byteL[i])
									delete(registers, reg1byteH[i])
								}
							}
							for i, r32 := range reg4byte {
								if hasKey(registers, r32) {
									delete(registers, reg2byte[i])
									delete(registers, reg1byteL[i])
									delete(registers, reg1byteH[i])
								}
							}
							for i, r16 := range reg2byte {
								if hasKey(registers, r16) {
									delete(registers, reg1byteL[i])
									delete(registers, reg1byteH[i])
								}
							}
						} else if runtime.GOARCH == "arm64" || runtime.GOARCH == "arm" {
							for i := 0; i <= 30; i++ {
								xReg := fmt.Sprintf("x%d", i)
								wReg := fmt.Sprintf("w%d", i)
								if hasKey(registers, xReg) {
									delete(registers, wReg)
								}
							}
						}
					}

					delete(registers, "eflags")
					delete(registers, "cpsr")
					delete(registers, "pstate")

					return registers, nil
				}
			}
		}
	}
	return nil, errors.New("could not find the register values in the payload returned from gdb")
}

// RegisterMap returns a map of all register names to values.
func (d *gdbDebugger) RegisterMap() (map[string]string, error) {
	names, err := d.RegisterNames()
	if err != nil {
		return nil, err
	}
	notification, err := d.conn.CheckedSend("data-list-register-values", "--skip-unavailable", "x")
	if err != nil {
		return nil, err
	}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]any); ok {
			if registerValues, ok := payloadMap["register-values"]; ok {
				if registerSlice, ok := registerValues.([]any); ok {
					registers := make(map[string]string, len(names))
					for _, singleRegisterMap := range registerSlice {
						if registerMap, ok := singleRegisterMap.(map[string]any); ok {
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
						}
					}
					return registers, nil
				}
			}
		}
	}
	return nil, errors.New("could not find the register values in the payload returned from gdb")
}

// EvalExpression evaluates an expression and returns the result.
func (d *gdbDebugger) EvalExpression(expr string) (string, error) {
	if d.conn == nil {
		return "", errors.New("gdb is not running")
	}
	notification, err := d.conn.CheckedSend("data-evaluate-expression", expr)
	if err != nil {
		return "", err
	}
	if payload, ok := notification["payload"]; ok && notification["class"] == "done" {
		if payloadMap, ok := payload.(map[string]any); ok {
			if value, ok := payloadMap["value"]; ok {
				if s, ok := value.(string); ok {
					return s, nil
				}
			}
		}
	}
	return "", errors.New("could not evaluate expression")
}

// AddWatch adds a watchpoint for the given expression.
func (d *gdbDebugger) AddWatch(expression string) (string, error) {
	var output string
	if d.conn != nil {
		_, err := d.conn.CheckedSend("break-watch", "-a", expression)
		if err != nil {
			return "", err
		}
		output = d.output.String()
		d.output.Reset()
	}
	d.watchMap[expression] = "?"
	return output, nil
}

// Output returns the collected stdout from the debugged program.
func (d *gdbDebugger) Output() string { return d.output.String() }

// OutputLen returns the length of the collected output.
func (d *gdbDebugger) OutputLen() int { return d.output.Len() }

// ConsoleString returns and clears the debugger console output.
func (d *gdbDebugger) ConsoleString() string {
	s := d.console.String()
	d.console.Reset()
	return s
}

// WatchMap returns the current watch variable names and values.
func (d *gdbDebugger) WatchMap() map[string]string { return d.watchMap }

// LastSeenWatch returns the name of the last changed watch variable.
func (d *gdbDebugger) LastSeenWatch() string { return d.lastWatch }

// ProgramRunning returns whether the debugged program is currently running.
func (d *gdbDebugger) ProgramRunning() bool { return d.running }

// SetStepInto controls whether stepping goes into function calls.
func (d *gdbDebugger) SetStepInto(stepInto bool) { d.stepInto = stepInto }
