package main

// Debugger is an interface for debugger backends (GDB, Delve, etc.)
type Debugger interface {
	// Start begins a new debug session for the given executable.
	// lineFunc is called when the debugger reports a new source line (e.g. after stepping).
	// doneFunc is called when the program finishes execution.
	Start(sourceDir, sourceBaseFilename, executableBaseFilename string, lineFunc func(int), doneFunc func()) (string, error)

	// End terminates the current debug session.
	End()

	// Continue resumes execution to the next breakpoint or end.
	Continue() error

	// Run starts (or restarts) the program from the beginning.
	Run() error

	// Step performs a single source-level step (into calls).
	Step() error

	// Next steps to the next source line (over calls).
	Next() error

	// NextInstruction steps to the next machine instruction.
	NextInstruction() error

	// Finish runs until the current function returns (step out).
	Finish() error

	// ActivateBreakpoint sets a breakpoint at the given file and line.
	ActivateBreakpoint(file string, line int) error

	// AddWatch adds a watchpoint for the given expression.
	AddWatch(expression string) (string, error)

	// RegisterNames returns all register names.
	RegisterNames() ([]string, error)

	// ChangedRegisters returns indices of registers that changed since last step.
	ChangedRegisters() ([]int, error)

	// ChangedRegisterMap returns a map of changed register names to values.
	ChangedRegisterMap() (map[string]string, error)

	// RegisterMap returns a map of all register names to values.
	RegisterMap() (map[string]string, error)

	// Disassemble returns the next n assembly instructions.
	Disassemble(n int) ([]string, error)

	// EvalExpression evaluates an expression and returns the result string.
	EvalExpression(expr string) (string, error)

	// Output returns the collected stdout from the debugged program.
	Output() string

	// OutputLen returns the length of the collected output (for change detection).
	OutputLen() int

	// ConsoleString returns and clears the debugger console output.
	ConsoleString() string

	// WatchMap returns the current watch variable names and values.
	WatchMap() map[string]string

	// LastSeenWatch returns the name of the last changed watch variable.
	LastSeenWatch() string

	// ProgramRunning returns whether the debugged program is currently running.
	ProgramRunning() bool

	// SetStepInto controls whether stepping goes into function calls.
	SetStepInto(stepInto bool)

	// ReverseStep steps backward one source line (into calls).
	ReverseStep() error

	// ReverseNextInstruction steps backward one machine instruction.
	ReverseNextInstruction() error
}
