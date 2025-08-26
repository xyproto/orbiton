package main

// Macro represents a series of keypresses that can be played back later
type Macro struct {
	KeyPresses []string
	index      int // current position, when playing back
	Recording  bool
}

// NewMacro creates a new Macro struct
func NewMacro() *Macro {
	return &Macro{KeyPresses: make([]string, 0, 16)}
}

// Add adds a keypress to this macro, when recording
func (m *Macro) Add(keyPress string) {
	m.KeyPresses = append(m.KeyPresses, keyPress)
}

// Next returns the next keypress when playing back a macro, or an empty string
func (m *Macro) Next() string {
	if kc := len(m.KeyPresses); kc == 0 || m.index >= kc {
		return ""
	}
	defer func() { m.index++ }()
	return m.KeyPresses[m.index]
}

// Home moves the current keypress index back to 0, for playing back the macro again
func (m *Macro) Home() {
	m.index = 0
}

// Len returns the number of keypresses in this macro
func (m *Macro) Len() int {
	return len(m.KeyPresses)
}
