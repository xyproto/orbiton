package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/xyproto/files"
)

var dlvPathCached *string

func findDlv() string {
	if dlvPathCached == nil {
		path := files.WhichCached("dlv")
		dlvPathCached = &path
	}
	return *dlvPathCached
}

// compile-time check that delveDebugger implements Debugger
var _ Debugger = (*delveDebugger)(nil)

type delveDebugger struct {
	cmd           *exec.Cmd
	conn          net.Conn
	enc           *json.Encoder
	dec           *json.Decoder
	mu            sync.Mutex
	seq           int64
	output        bytes.Buffer
	watchMap      map[string]string
	lastWatch     string
	prevRegisters map[string]string
	lineFunc      func(int)
	doneFunc      func()
	running       bool
	stepInto      bool
}

func newDelveDebugger() *delveDebugger {
	return &delveDebugger{
		watchMap:      make(map[string]string),
		prevRegisters: make(map[string]string),
	}
}

// --- JSON-RPC wire types ---

type dlvRequest struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
	ID     int64         `json:"id"`
}

type dlvResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *string         `json:"error"`
}

// --- Delve API input types ---
// Fields without json tags use their Go name (capital) as the JSON key.

type dlvCommandIn struct {
	Name string `json:"Name"`
}

type dlvStateIn struct {
	NonBlocking bool `json:"NonBlocking"`
}

type dlvBreakpoint struct {
	FunctionName string `json:"functionName,omitempty"`
	File         string `json:"file,omitempty"`
	Line         int    `json:"line,omitempty"`
}

type dlvCreateBreakpointIn struct {
	Breakpoint dlvBreakpoint `json:"Breakpoint"`
}

type dlvEvalScope struct {
	GoroutineID int64 `json:"goroutineID"`
	Frame       int   `json:"frame"`
}

type dlvLoadConfig struct {
	FollowPointers     bool `json:"followPointers"`
	MaxVariableRecurse int  `json:"maxVariableRecurse"`
	MaxStringLen       int  `json:"maxStringLen"`
	MaxArrayValues     int  `json:"maxArrayValues"`
	MaxStructFields    int  `json:"maxStructFields"`
}

type dlvEvalIn struct {
	Scope dlvEvalScope  `json:"Scope"`
	Expr  string        `json:"Expr"`
	Cfg   dlvLoadConfig `json:"Cfg"`
}

type dlvListRegistersIn struct {
	ThreadID  int  `json:"ThreadID"`
	IncludeFP bool `json:"IncludeFP"`
}

type dlvDisassembleIn struct {
	Scope   dlvEvalScope `json:"Scope"`
	StartPC uint64       `json:"StartPC"`
	EndPC   uint64       `json:"EndPC"`
	Flavour int          `json:"Flavour"` // 0 = Intel
}

type dlvDetachIn struct {
	Kill bool `json:"Kill"`
}

type dlvRestartIn struct {
	Position string `json:"Position"`
}

// --- Delve API output types ---
// These mirror Delve's api package, which uses lowercase json tags.

type dlvThread struct {
	ID   int    `json:"id"`
	PC   uint64 `json:"pc"`
	File string `json:"file"`
	Line int    `json:"line"`
}

type dlvState struct {
	CurrentThread *dlvThread `json:"currentThread"`
	Exited        bool       `json:"exited"`
	ExitStatus    int        `json:"exitStatus"`
	Err           *string    `json:"err,omitempty"`
}

// Result wrappers: top-level field names match Go struct field names (capital).
type dlvCommandOut struct {
	State dlvState `json:"State"`
}

type dlvStateOut struct {
	State dlvState `json:"State"`
}

type dlvRegister struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type dlvListRegistersOut struct {
	Regs []dlvRegister `json:"Regs"`
}

type dlvVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type dlvEvalOut struct {
	Variable dlvVariable `json:"Variable"`
}

type dlvAsmInstruction struct {
	Loc struct {
		PC   uint64 `json:"pc"`
		File string `json:"file"`
		Line int    `json:"line"`
	} `json:"loc"`
	Text string `json:"text"`
	AtPC bool   `json:"atPC"`
}

type dlvDisassembleOut struct {
	Disassemble []dlvAsmInstruction `json:"Disassemble"`
}

// call sends a JSON-RPC 2.0 request to Delve and decodes the result into out.
func (d *delveDebugger) call(method string, in interface{}, out interface{}) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seq++
	if err := d.enc.Encode(dlvRequest{
		Method: "RPCServer." + method,
		Params: []interface{}{in},
		ID:     d.seq,
	}); err != nil {
		return fmt.Errorf("dlv send: %w", err)
	}
	var resp dlvResponse
	if err := d.dec.Decode(&resp); err != nil {
		return fmt.Errorf("dlv recv: %w", err)
	}
	if resp.Error != nil {
		return errors.New(*resp.Error)
	}
	if out != nil && len(resp.Result) > 0 {
		return json.Unmarshal(resp.Result, out)
	}
	return nil
}

// command sends a named debugger command and returns the resulting state.
func (d *delveDebugger) command(name string) (dlvState, error) {
	var out dlvCommandOut
	err := d.call("Command", dlvCommandIn{Name: name}, &out)
	return out.State, err
}

// doStep is the common implementation for all stepping/running methods.
func (d *delveDebugger) doStep(commandName string) error {
	if !d.running {
		return errProgramStopped
	}
	state, err := d.command(commandName)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "exited") || strings.Contains(msg, "has exited") || strings.Contains(msg, "process exited") {
			d.running = false
			if d.doneFunc != nil {
				d.doneFunc()
			}
			return nil
		}
		return err
	}
	if state.Exited {
		d.running = false
		if d.doneFunc != nil {
			d.doneFunc()
		}
		return nil
	}
	if state.CurrentThread != nil && state.CurrentThread.Line > 0 {
		if d.lineFunc != nil {
			d.lineFunc(state.CurrentThread.Line)
		}
	}
	// Re-evaluate watches after each step
	for expr := range d.watchMap {
		if val, evalErr := d.evalExpr(expr); evalErr == nil && d.watchMap[expr] != val {
			d.watchMap[expr] = val
			d.lastWatch = expr
		}
	}
	return nil
}

// evalExpr evaluates an expression without acquiring the watch lock.
func (d *delveDebugger) evalExpr(expr string) (string, error) {
	var out dlvEvalOut
	err := d.call("Eval", dlvEvalIn{
		Scope: dlvEvalScope{GoroutineID: -1, Frame: 0},
		Expr:  expr,
		Cfg: dlvLoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 1,
			MaxStringLen:       64,
			MaxArrayValues:     64,
			MaxStructFields:    -1,
		},
	}, &out)
	if err != nil {
		return "", err
	}
	return out.Variable.Value, nil
}

// Start launches dlv in headless mode, connects to it, and runs to main.main.
func (d *delveDebugger) Start(sourceDir, sourceBaseFilename, executableBaseFilename string, lineFunc func(int), doneFunc func()) (string, error) {
	d.End()
	d.lineFunc = lineFunc
	d.doneFunc = doneFunc

	var err error
	originalDirectory, err = os.Getwd()
	if err == nil {
		if err = os.Chdir(sourceDir); err != nil {
			return "", fmt.Errorf("could not change directory to %s", sourceDir)
		}
	}

	dlv := findDlv()
	if dlv == "" {
		return "", errors.New("dlv not found, install with: go install github.com/go-delve/delve/cmd/dlv@latest")
	}

	// Let the OS pick a free port, then release it for dlv to bind.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("could not find a free port: %w", err)
	}
	addr := l.Addr().String()
	l.Close()

	d.cmd = exec.Command(dlv, "exec", executableBaseFilename,
		"--headless", "--api-version=2", "--accept-multiclient",
		"--listen="+addr, "--log-dest=/dev/null",
	)
	d.cmd.Dir = sourceDir

	stdout, err := d.cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	go io.Copy(&d.output, stdout)

	if err = d.cmd.Start(); err != nil {
		return "", fmt.Errorf("could not start dlv: %w", err)
	}

	// Retry connecting until dlv is ready (up to 3 seconds).
	var conn net.Conn
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if conn, err = net.Dial("tcp", addr); err == nil {
			break
		}
	}
	if err != nil {
		d.cmd.Process.Kill()
		return "", fmt.Errorf("could not connect to dlv at %s: %w", addr, err)
	}
	d.conn = conn
	d.enc = json.NewEncoder(conn)
	d.dec = json.NewDecoder(bufio.NewReader(conn))
	d.running = true

	// Set a breakpoint at main.main, then continue to it.
	_ = d.call("CreateBreakpoint", dlvCreateBreakpointIn{
		Breakpoint: dlvBreakpoint{FunctionName: "main.main"},
	}, nil)

	state, err := d.command("continue")
	if err != nil {
		if strings.Contains(err.Error(), "exited") {
			d.running = false
			if doneFunc != nil {
				doneFunc()
			}
			return "started dlv", nil
		}
		d.running = false
		return "", fmt.Errorf("dlv continue to main failed: %w", err)
	}
	if state.Exited {
		d.running = false
		if doneFunc != nil {
			doneFunc()
		}
		return "started dlv", nil
	}
	if state.CurrentThread != nil && state.CurrentThread.Line > 0 {
		lineFunc(state.CurrentThread.Line)
	}
	return "started dlv", nil
}

// End terminates the dlv session and restores the working directory.
func (d *delveDebugger) End() {
	if d.conn != nil {
		_ = d.call("Detach", dlvDetachIn{Kill: true}, nil)
		d.conn.Close()
		d.conn = nil
	}
	if d.cmd != nil && d.cmd.Process != nil {
		d.cmd.Process.Kill()
		d.cmd.Wait()
		d.cmd = nil
	}
	if originalDirectory != "" {
		os.Chdir(originalDirectory)
	}
	d.running = false
}

func (d *delveDebugger) Continue() error { return d.doStep("continue") }
func (d *delveDebugger) Finish() error   { return d.doStep("stepout") }

func (d *delveDebugger) Next() error {
	if d.stepInto {
		return d.doStep("step")
	}
	return d.doStep("next")
}

func (d *delveDebugger) Step() error { return d.doStep("step") }

func (d *delveDebugger) NextInstruction() error {
	if d.stepInto {
		return d.doStep("stepInstruction")
	}
	return d.doStep("nextInstruction")
}

func (d *delveDebugger) Run() error {
	if !d.running {
		return errProgramStopped
	}
	return d.call("Restart", dlvRestartIn{}, nil)
}

func (d *delveDebugger) ActivateBreakpoint(file string, line int) error {
	return d.call("CreateBreakpoint", dlvCreateBreakpointIn{
		Breakpoint: dlvBreakpoint{File: file, Line: line},
	}, nil)
}

func (d *delveDebugger) AddWatch(expression string) (string, error) {
	val, err := d.evalExpr(expression)
	if err != nil {
		// Store with placeholder so it shows up in the watch list
		d.watchMap[expression] = "?"
		return "", err
	}
	d.watchMap[expression] = val
	d.lastWatch = expression
	return val, nil
}

func (d *delveDebugger) EvalExpression(expr string) (string, error) {
	return d.evalExpr(expr)
}

func (d *delveDebugger) listRegisters() ([]dlvRegister, error) {
	var out dlvListRegistersOut
	if err := d.call("ListRegisters", dlvListRegistersIn{ThreadID: 0, IncludeFP: false}, &out); err != nil {
		return nil, err
	}
	return out.Regs, nil
}

func (d *delveDebugger) RegisterNames() ([]string, error) {
	regs, err := d.listRegisters()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(regs))
	for i, r := range regs {
		names[i] = r.Name
	}
	return names, nil
}

func (d *delveDebugger) RegisterMap() (map[string]string, error) {
	regs, err := d.listRegisters()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(regs))
	for _, r := range regs {
		m[r.Name] = r.Value
	}
	return m, nil
}

func (d *delveDebugger) ChangedRegisterMap() (map[string]string, error) {
	regs, err := d.listRegisters()
	if err != nil {
		return nil, err
	}
	changed := make(map[string]string)
	for _, r := range regs {
		if prev, ok := d.prevRegisters[r.Name]; !ok || prev != r.Value {
			changed[r.Name] = r.Value
		}
	}
	for _, r := range regs {
		d.prevRegisters[r.Name] = r.Value
	}
	return changed, nil
}

func (d *delveDebugger) ChangedRegisters() ([]int, error) {
	regs, err := d.listRegisters()
	if err != nil {
		return nil, err
	}
	var indices []int
	for i, r := range regs {
		if prev, ok := d.prevRegisters[r.Name]; !ok || prev != r.Value {
			indices = append(indices, i)
		}
	}
	return indices, nil
}

func (d *delveDebugger) Disassemble(n int) ([]string, error) {
	var stateOut dlvStateOut
	if err := d.call("State", dlvStateIn{NonBlocking: true}, &stateOut); err != nil {
		return nil, err
	}
	if stateOut.State.CurrentThread == nil {
		return nil, errors.New("no current thread")
	}
	pc := stateOut.State.CurrentThread.PC
	// Estimate end PC: ~8 bytes per x86-64 instruction on average
	endPC := pc + uint64(n*8)
	var out dlvDisassembleOut
	if err := d.call("Disassemble", dlvDisassembleIn{
		Scope:   dlvEvalScope{GoroutineID: -1, Frame: 0},
		StartPC: pc,
		EndPC:   endPC,
		Flavour: 0,
	}, &out); err != nil {
		return nil, err
	}
	result := make([]string, 0, len(out.Disassemble))
	for i, instr := range out.Disassemble {
		if i >= n {
			break
		}
		result = append(result, instr.Text)
	}
	return result, nil
}

func (d *delveDebugger) Output() string        { return d.output.String() }
func (d *delveDebugger) OutputLen() int        { return d.output.Len() }
func (d *delveDebugger) ConsoleString() string { return "" }

func (d *delveDebugger) WatchMap() map[string]string { return d.watchMap }
func (d *delveDebugger) LastSeenWatch() string       { return d.lastWatch }
func (d *delveDebugger) ProgramRunning() bool        { return d.running }
func (d *delveDebugger) SetStepInto(stepInto bool)   { d.stepInto = stepInto }
func (d *delveDebugger) ReverseStep() error {
	return errors.New("reverse stepping is not supported by Delve")
}
func (d *delveDebugger) ReverseNextInstruction() error {
	return errors.New("reverse stepping is not supported by Delve")
}
