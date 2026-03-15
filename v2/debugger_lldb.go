package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/xyproto/files"
)

// compile-time check that lldbDebugger implements Debugger
var _ Debugger = (*lldbDebugger)(nil)

// lldbDebugger implements the Debugger interface using LLDB's CLI.
// It is used on macOS where GDB does not support native arm64 debugging.
type lldbDebugger struct {
	cmd      *exec.Cmd
	ptyFile  *os.File   // PTY master (read/write to LLDB)
	mu       sync.Mutex // guards PTY writes and response reads
	watchMap map[string]string
	output   bytes.Buffer // program stdout
	console  strings.Builder
	prevRegs map[string]string // previous register values for change detection

	lastWatch string
	lineFunc  func(int)
	doneFunc  func()
	running   bool
	stepInto  bool
}

func newLLDBDebugger() *lldbDebugger {
	return &lldbDebugger{
		watchMap: make(map[string]string),
		prevRegs: make(map[string]string),
	}
}

// findLLDB returns the path to the lldb binary, or empty string if not found.
func findLLDB() string {
	return files.WhichCached("lldb")
}

// lineRegexp matches LLDB's frame output to extract file and line number.
// Example: "    frame #0: 0x... main`main at main.c:4:9"
var lineRegexp = regexp.MustCompile(`at\s+\S+:(\d+)`)

// send writes a command to LLDB and reads the response until the next prompt.
func (d *lldbDebugger) send(command string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.ptyFile == nil {
		return "", errors.New("lldb is not running")
	}

	if _, err := fmt.Fprintf(d.ptyFile, "%s\n", command); err != nil {
		return "", err
	}

	return d.readUntilPrompt()
}

// readUntilPrompt reads from the PTY until the "(lldb) " prompt is seen.
func (d *lldbDebugger) readUntilPrompt() (string, error) {
	var sb strings.Builder
	buf := make([]byte, 4096)
	prompt := "(lldb) "

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		d.ptyFile.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := d.ptyFile.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
			// Check if we have the prompt at the end
			s := sb.String()
			if strings.HasSuffix(strings.TrimRight(s, " \t"), "(lldb)") {
				// Remove the trailing prompt from the output
				idx := strings.LastIndex(s, prompt)
				if idx >= 0 {
					return strings.TrimSpace(s[:idx]), nil
				}
				// Try just "(lldb)" without trailing space
				idx = strings.LastIndex(s, "(lldb)")
				if idx >= 0 {
					return strings.TrimSpace(s[:idx]), nil
				}
				return strings.TrimSpace(s), nil
			}
		}
		if err != nil {
			if os.IsTimeout(err) {
				continue
			}
			return sb.String(), err
		}
	}
	return sb.String(), errors.New("timeout waiting for lldb prompt")
}

// extractLineNumber looks for a line number in LLDB output.
func extractLineNumber(output string) (int, bool) {
	m := lineRegexp.FindStringSubmatch(output)
	if len(m) >= 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			return n, true
		}
	}
	return 0, false
}

// checkExited returns true if the LLDB output indicates the process has exited.
func checkExited(output string) bool {
	return strings.Contains(output, "Process") && strings.Contains(output, "exited")
}

// Start begins a new debug session using LLDB.
func (d *lldbDebugger) Start(sourceDir, sourceBaseFilename, executableBaseFilename string, lineFunc func(int), doneFunc func()) (string, error) {
	d.End()

	d.lineFunc = lineFunc
	d.doneFunc = doneFunc

	var err error
	originalDirectory, err = os.Getwd()
	if err == nil {
		if err = os.Chdir(sourceDir); err != nil {
			return "", errors.New("could not change directory to " + sourceDir)
		}
	}

	lldbPath := findLLDB()
	if lldbPath == "" {
		return "", errors.New("could not find lldb")
	}

	// Build absolute path to executable
	executablePath := executableBaseFilename
	if !filepath.IsAbs(executablePath) {
		executablePath = filepath.Join(sourceDir, executableBaseFilename)
	}

	d.cmd = exec.Command(lldbPath, "--no-use-colors", executablePath)
	d.cmd.Dir = sourceDir

	// Start LLDB with a PTY so it behaves interactively
	d.ptyFile, err = pty.Start(d.cmd)
	if err != nil {
		return "", fmt.Errorf("could not start lldb with pty: %w", err)
	}

	// Read the initial prompt
	if _, err := d.readUntilPrompt(); err != nil {
		return "", fmt.Errorf("lldb startup: %w", err)
	}

	// Set a breakpoint at main and run to it
	if _, err := d.send("breakpoint set -n main"); err != nil {
		return "", err
	}

	resp, err := d.send("run")
	if err != nil {
		return "", err
	}

	if checkExited(resp) {
		d.running = false
		if d.doneFunc != nil {
			d.doneFunc()
		}
		return "started lldb", nil
	}

	d.running = true

	// Report initial line
	if lineNum, ok := extractLineNumber(resp); ok {
		if d.lineFunc != nil {
			d.lineFunc(lineNum)
		}
	}

	return "started lldb", nil
}

// End terminates the current LLDB session.
func (d *lldbDebugger) End() {
	if d.ptyFile != nil {
		fmt.Fprintf(d.ptyFile, "quit\n")
		d.ptyFile.Close()
		d.ptyFile = nil
	}
	if d.cmd != nil && d.cmd.Process != nil {
		d.cmd.Process.Kill()
		d.cmd.Wait()
		d.cmd = nil
	}
	d.output.Reset()
	d.console.Reset()
	d.lastWatch = ""
	d.running = false
	d.prevRegs = make(map[string]string)
	if originalDirectory != "" {
		os.Chdir(originalDirectory)
	}
	longInstructionPaneWidth = 0
}

// doStep sends a stepping command to LLDB and processes the result.
func (d *lldbDebugger) doStep(command string) error {
	if !d.running {
		return errProgramStopped
	}

	resp, err := d.send(command)
	if err != nil {
		return err
	}

	d.console.WriteString(resp)

	if checkExited(resp) {
		d.running = false
		if d.doneFunc != nil {
			d.doneFunc()
		}
		return errProgramStopped
	}

	if lineNum, ok := extractLineNumber(resp); ok {
		if d.lineFunc != nil {
			d.lineFunc(lineNum)
		}
	}

	// Refresh watches after each step
	d.refreshWatches()

	return nil
}

// refreshWatches re-evaluates all watch expressions.
func (d *lldbDebugger) refreshWatches() {
	for expr := range d.watchMap {
		if value, err := d.EvalExpression(expr); err == nil {
			if value != d.watchMap[expr] {
				d.lastWatch = expr
			}
			d.watchMap[expr] = value
		}
	}
}

// Continue resumes execution to the next breakpoint or end.
func (d *lldbDebugger) Continue() error {
	return d.doStep("continue")
}

// Run starts (or restarts) the program from the beginning.
func (d *lldbDebugger) Run() error {
	return d.doStep("run")
}

// Step performs a single source-level step (into calls).
func (d *lldbDebugger) Step() error {
	return d.doStep("step")
}

// Next steps to the next source line (over calls).
func (d *lldbDebugger) Next() error {
	if d.stepInto {
		return d.doStep("step")
	}
	return d.doStep("next")
}

// NextInstruction steps to the next machine instruction.
func (d *lldbDebugger) NextInstruction() error {
	showInstructionPane = true
	if d.stepInto {
		return d.doStep("si")
	}
	return d.doStep("ni")
}

// Finish runs until the current function returns (step out).
func (d *lldbDebugger) Finish() error {
	return d.doStep("finish")
}

// ActivateBreakpoint sets a breakpoint at the given file and line.
func (d *lldbDebugger) ActivateBreakpoint(file string, line int) error {
	_, err := d.send(fmt.Sprintf("breakpoint set -f %s -l %d", file, line))
	return err
}

// AddWatch adds a watch for the given expression.
func (d *lldbDebugger) AddWatch(expression string) (string, error) {
	// Evaluate the expression to get its current value
	if value, err := d.EvalExpression(expression); err == nil {
		d.watchMap[expression] = value
		d.lastWatch = expression
	} else {
		d.watchMap[expression] = "?"
	}
	return "", nil
}

// EvalExpression evaluates an expression and returns the result.
func (d *lldbDebugger) EvalExpression(expr string) (string, error) {
	resp, err := d.send(fmt.Sprintf("expression -- %s", expr))
	if err != nil {
		return "", err
	}
	// LLDB outputs like: "(int) $0 = 256"
	// Parse the value after the last "="
	resp = strings.TrimSpace(resp)
	// Filter out echo of the command itself (PTY echoes input)
	lines := strings.Split(resp, "\n")
	var filtered []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "expression") {
			continue
		}
		filtered = append(filtered, line)
	}
	resp = strings.Join(filtered, "\n")
	resp = strings.TrimSpace(resp)
	if idx := strings.LastIndex(resp, "= "); idx >= 0 {
		return strings.TrimSpace(resp[idx+2:]), nil
	}
	if resp != "" {
		return resp, nil
	}
	return "", errors.New("could not evaluate expression")
}

// RegisterNames returns all register names.
func (d *lldbDebugger) RegisterNames() ([]string, error) {
	resp, err := d.send("register read")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(resp, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "General") || strings.HasPrefix(line, "Floating") || strings.HasPrefix(line, "register") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			names = append(names, parts[0])
		}
	}
	return names, nil
}

// ChangedRegisters returns indices of registers that changed since last step.
func (d *lldbDebugger) ChangedRegisters() ([]int, error) {
	return nil, nil
}

// ChangedRegisterMap returns a map of changed register names to values.
func (d *lldbDebugger) ChangedRegisterMap() (map[string]string, error) {
	resp, err := d.send("register read")
	if err != nil {
		return nil, err
	}

	current := make(map[string]string)
	for _, line := range strings.Split(resp, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "General") || strings.HasPrefix(line, "Floating") || strings.HasPrefix(line, "register") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			current[name] = value
		}
	}

	changed := make(map[string]string)
	for name, value := range current {
		if prev, ok := d.prevRegs[name]; !ok || prev != value {
			changed[name] = value
		}
	}
	d.prevRegs = current

	delete(changed, "cpsr")
	delete(changed, "eflags")

	return changed, nil
}

// RegisterMap returns a map of all register names to values.
func (d *lldbDebugger) RegisterMap() (map[string]string, error) {
	resp, err := d.send("register read")
	if err != nil {
		return nil, err
	}

	regs := make(map[string]string)
	for _, line := range strings.Split(resp, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "General") || strings.HasPrefix(line, "Floating") || strings.HasPrefix(line, "register") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			regs[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return regs, nil
}

// Disassemble returns the next n assembly instructions.
func (d *lldbDebugger) Disassemble(n int) ([]string, error) {
	resp, err := d.send(fmt.Sprintf("disassemble --pc --count %d", n+1))
	if err != nil {
		return nil, err
	}

	var instructions []string
	for _, line := range strings.Split(resp, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, ":") {
			continue
		}
		// Skip echo and header lines
		if strings.HasPrefix(line, "disassemble") {
			continue
		}
		if !strings.Contains(line, "]") && !strings.Contains(line, "0x") {
			continue
		}
		line = strings.TrimPrefix(line, "->")
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, ": "); idx >= 0 {
			inst := strings.TrimSpace(line[idx+2:])
			if inst != "" {
				instructions = append(instructions, inst)
			}
		}
		if len(instructions) >= n {
			break
		}
	}
	return instructions, nil
}

// Output returns the collected stdout from the debugged program.
func (d *lldbDebugger) Output() string { return d.output.String() }

// OutputLen returns the length of the collected output.
func (d *lldbDebugger) OutputLen() int { return d.output.Len() }

// ConsoleString returns and clears the debugger console output.
func (d *lldbDebugger) ConsoleString() string {
	s := d.console.String()
	d.console.Reset()
	return s
}

// WatchMap returns the current watch variable names and values.
func (d *lldbDebugger) WatchMap() map[string]string { return d.watchMap }

// LastSeenWatch returns the name of the last changed watch variable.
func (d *lldbDebugger) LastSeenWatch() string { return d.lastWatch }

// ProgramRunning returns whether the debugged program is currently running.
func (d *lldbDebugger) ProgramRunning() bool { return d.running }

// SetStepInto controls whether stepping goes into function calls.
func (d *lldbDebugger) SetStepInto(stepInto bool) { d.stepInto = stepInto }

// ReverseStep is not supported by LLDB.
func (d *lldbDebugger) ReverseStep() error {
	return errors.New("reverse stepping is not supported by LLDB")
}

// ReverseNextInstruction is not supported by LLDB.
func (d *lldbDebugger) ReverseNextInstruction() error {
	return errors.New("reverse stepping is not supported by LLDB")
}
