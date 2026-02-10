//go:build windows

package vt

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/term"
)

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode      = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode      = kernel32.NewProc("SetConsoleMode")
	procGetStdHandle        = kernel32.NewProc("GetStdHandle")
	procWaitForSingleObject = kernel32.NewProc("WaitForSingleObject")
)

const (
	WAIT_TIMEOUT = 0x00000102
)

const (
	STD_INPUT_HANDLE                   = ^uintptr(10) // -11
	STD_OUTPUT_HANDLE                  = ^uintptr(11) // -12
	ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	ENABLE_VIRTUAL_TERMINAL_INPUT      = 0x0200
	DISABLE_NEWLINE_AUTO_RETURN        = 0x0008
	ENABLE_ECHO_INPUT                  = 0x0004
	ENABLE_LINE_INPUT                  = 0x0002
	ENABLE_PROCESSED_INPUT             = 0x0001
)

var (
	defaultTimeout = 50 * time.Millisecond
	lastKey        int
)

// Key codes for 3-byte sequences (arrows, Home, End)
var keyCodeLookup = map[[3]byte]int{
	{27, 91, 65}:  KeyArrowUp,
	{27, 91, 66}:  KeyArrowDown,
	{27, 91, 67}:  KeyArrowRight,
	{27, 91, 68}:  KeyArrowLeft,
	{27, 91, 'H'}: KeyHome,
	{27, 91, 'F'}: KeyEnd,
}

// Key codes for 4-byte sequences (Page Up, Page Down, Home, End)
var pageNavLookup = map[[4]byte]int{
	{27, 91, 49, 126}: KeyHome,
	{27, 91, 52, 126}: KeyEnd,
	{27, 91, 53, 126}: KeyPageUp,
	{27, 91, 54, 126}: KeyPageDown,
}

// Key codes for 6-byte sequences (Ctrl-Insert)
var ctrlInsertLookup = map[[6]byte]int{
	{27, 91, 50, 59, 53, 126}: KeyCtrlInsert,
}

type TTY struct {
	fd                 int
	originalState      *term.State
	timeout            time.Duration
	originalInputMode  uint32
	originalOutputMode uint32
}

// enableVTMode enables VT100/ANSI escape sequence processing on Windows
// and configures console for raw input mode with VT sequences
func enableVTMode() (uint32, uint32, error) {
	var originalInputMode, originalOutputMode uint32

	// Enable VT processing for stdout
	stdout, _, _ := procGetStdHandle.Call(STD_OUTPUT_HANDLE)
	if stdout != 0 && stdout != ^uintptr(0) {
		var outputMode uint32
		ret, _, _ := procGetConsoleMode.Call(stdout, uintptr(unsafe.Pointer(&outputMode)))
		if ret != 0 {
			originalOutputMode = outputMode
			outputMode |= ENABLE_VIRTUAL_TERMINAL_PROCESSING | DISABLE_NEWLINE_AUTO_RETURN
			procSetConsoleMode.Call(stdout, uintptr(outputMode))
		}
	}

	// Enable VT processing for stdin and set raw mode
	stdin, _, _ := procGetStdHandle.Call(STD_INPUT_HANDLE)
	if stdin == 0 || stdin == ^uintptr(0) {
		return 0, 0, errors.New("could not get stdin handle")
	}

	var inputMode uint32
	ret, _, _ := procGetConsoleMode.Call(stdin, uintptr(unsafe.Pointer(&inputMode)))
	if ret == 0 {
		return 0, 0, errors.New("could not get console input mode")
	}
	originalInputMode = inputMode

	// Set raw mode: disable echo, line input, and processed input
	// but KEEP VT input enabled for arrow keys
	inputMode &^= ENABLE_ECHO_INPUT | ENABLE_LINE_INPUT | ENABLE_PROCESSED_INPUT
	inputMode |= ENABLE_VIRTUAL_TERMINAL_INPUT

	ret, _, _ = procSetConsoleMode.Call(stdin, uintptr(inputMode))
	if ret == 0 {
		return originalInputMode, originalOutputMode, errors.New("could not set console input mode")
	}

	return originalInputMode, originalOutputMode, nil
}

// NewTTY opens stdin/stdout for terminal input/output on Windows
func NewTTY() (*TTY, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, errors.New("not a terminal")
	}

	originalState, err := term.GetState(fd)
	if err != nil {
		return nil, err
	}

	// Try to enable VT100/ANSI mode and set raw mode with proper VT support
	originalInputMode, originalOutputMode, err := enableVTMode()
	if err != nil {
		// If we can't set console modes directly, fall back to term.MakeRaw
		// This might happen in some terminal emulators
		_, err = term.MakeRaw(fd)
		if err != nil {
			return nil, err
		}
	}

	return &TTY{
		fd:                 fd,
		originalState:      originalState,
		timeout:            defaultTimeout,
		originalInputMode:  originalInputMode,
		originalOutputMode: originalOutputMode,
	}, nil
}

// SetTimeout sets a timeout for reading a key
func (tty *TTY) SetTimeout(d time.Duration) {
	tty.timeout = d
}

// SetEscTimeout is a no-op on Windows.
func (tty *TTY) SetEscTimeout(d time.Duration) {}

// FastInput is a no-op on Windows.
func (tty *TTY) FastInput(enable bool) {}

// Close will restore the terminal state
func (tty *TTY) Close() {
	if tty.originalState != nil {
		term.Restore(tty.fd, tty.originalState)
	}
	// Also restore console modes
	stdin, _, _ := procGetStdHandle.Call(STD_INPUT_HANDLE)
	if stdin != 0 {
		procSetConsoleMode.Call(stdin, uintptr(tty.originalInputMode))
	}
	stdout, _, _ := procGetStdHandle.Call(STD_OUTPUT_HANDLE)
	if stdout != 0 {
		procSetConsoleMode.Call(stdout, uintptr(tty.originalOutputMode))
	}
}

// waitForInput waits for console input using WaitForSingleObject with timeout
func waitForInput(timeout time.Duration) bool {
	stdin, _, _ := procGetStdHandle.Call(STD_INPUT_HANDLE)
	if stdin == 0 {
		return false
	}
	// Convert timeout to milliseconds
	timeoutMs := uint32(timeout / time.Millisecond)
	ret, _, _ := procWaitForSingleObject.Call(stdin, uintptr(timeoutMs))
	// ret == 0 (WAIT_OBJECT_0) means input is available
	// ret == WAIT_TIMEOUT means no input
	return ret == 0
}

// readRawBytes reads raw bytes from stdin, returning the number read and an error.
func readRawBytes(tty *TTY, buf []byte) (int, error) {
	if !waitForInput(tty.timeout) {
		return 0, nil
	}
	return os.Stdin.Read(buf)
}

// parseRawInput converts raw byte input to an Event.
func parseRawInput(buf []byte, numRead int) Event {
	switch {
	case numRead == 1:
		b := buf[0]
		if b == 127 {
			return Event{Kind: EventKey, Key: KeyBackspace}
		}
		if b < 32 {
			return Event{Kind: EventKey, Key: int(b)}
		}
		return Event{Kind: EventRune, Rune: rune(b)}
	case numRead == 3:
		seq := [3]byte{buf[0], buf[1], buf[2]}
		if code, found := keyCodeLookup[seq]; found {
			return Event{Kind: EventKey, Key: code}
		}
		r, _ := utf8.DecodeRune(buf[:numRead])
		if unicode.IsPrint(r) {
			return Event{Kind: EventRune, Rune: r}
		}
		return Event{Kind: EventText, Text: string(buf[:numRead])}
	case numRead == 4:
		seq := [4]byte{buf[0], buf[1], buf[2], buf[3]}
		if code, found := pageNavLookup[seq]; found {
			return Event{Kind: EventKey, Key: code}
		}
		return Event{Kind: EventText, Text: string(buf[:numRead])}
	case numRead == 6:
		seq := [6]byte{buf[0], buf[1], buf[2], buf[3], buf[4], buf[5]}
		if code, found := ctrlInsertLookup[seq]; found {
			return Event{Kind: EventKey, Key: code}
		}
		return Event{Kind: EventText, Text: string(buf[:numRead])}
	default:
		r, _ := utf8.DecodeRune(buf[:numRead])
		if unicode.IsPrint(r) {
			return Event{Kind: EventRune, Rune: r}
		}
		return Event{Kind: EventText, Text: string(buf[:numRead])}
	}
}

// ReadEvent reads and parses a single input event (key, rune, or text).
func (tty *TTY) ReadEvent() (Event, error) {
	buf := make([]byte, 6)
	numRead, err := readRawBytes(tty, buf)
	if err != nil {
		return Event{Kind: EventNone}, err
	}
	if numRead == 0 {
		return Event{Kind: EventNone}, nil
	}
	return parseRawInput(buf, numRead), nil
}

// ReadEventBlocking waits until a full input event is available.
func (tty *TTY) ReadEventBlocking() (Event, error) {
	buf := make([]byte, 6)
	for {
		numRead, err := os.Stdin.Read(buf)
		if err != nil {
			return Event{Kind: EventNone}, err
		}
		if numRead > 0 {
			return parseRawInput(buf, numRead), nil
		}
	}
}

// asciiAndKeyCode processes input into an ASCII code or key code, handling multi-byte sequences like Ctrl-Insert
func asciiAndKeyCode(tty *TTY) (ascii, keyCode int, err error) {
	ev, err := tty.ReadEvent()
	if err != nil {
		return 0, 0, err
	}
	switch ev.Kind {
	case EventKey:
		return 0, ev.Key, nil
	case EventRune:
		return int(ev.Rune), 0, nil
	default:
		return 0, 0, nil
	}
}

// Key reads the keycode or ASCII code and avoids repeated keys
func (tty *TTY) Key() int {
	ascii, keyCode, err := asciiAndKeyCode(tty)
	if err != nil {
		return 0
	}
	var key int
	if keyCode != 0 {
		key = keyCode
	} else {
		key = ascii
	}
	// Don't filter repeated keys - let the application handle key repeats
	return key
}

// KeyRaw reads a key without suppressing repeats.
func (tty *TTY) KeyRaw() int {
	return tty.Key()
}

// String reads a string, handling key sequences and printable characters
func (tty *TTY) String() string {
	ev, err := tty.ReadEvent()
	if err != nil {
		return ""
	}
	switch ev.Kind {
	case EventNone:
		return ""
	case EventKey:
		return KeySymbol(ev.Key)
	case EventRune:
		if unicode.IsPrint(ev.Rune) {
			return string(ev.Rune)
		}
		return "c:" + strconv.Itoa(int(ev.Rune))
	case EventText:
		return ev.Text
	default:
		return ""
	}
}

// StringRaw reads a string without suppressing repeats.
func (tty *TTY) StringRaw() string {
	return tty.String()
}

// Rune reads a rune, handling special sequences for arrows, Home, End, etc.
func (tty *TTY) Rune() rune {
	ev, err := tty.ReadEvent()
	if err != nil {
		return rune(0)
	}
	switch ev.Kind {
	case EventNone:
		return rune(0)
	case EventKey:
		s := KeySymbol(ev.Key)
		if len(s) > 0 {
			return []rune(s)[0]
		}
	case EventRune:
		return ev.Rune
	case EventText:
		if ev.Text != "" {
			return []rune(ev.Text)[0]
		}
	}
	return rune(0)
}

// RuneRaw reads a rune without suppressing repeats.
func (tty *TTY) RuneRaw() rune {
	return tty.Rune()
}

// RawMode switches the terminal to raw mode
func (tty *TTY) RawMode() {
	_, _ = term.MakeRaw(tty.fd)
}

// NoBlock sets the terminal to cbreak mode (no-op for golang.org/x/term)
func (tty *TTY) NoBlock() {
	// No-op for golang.org/x/term - raw mode handles this
}

// Restore the terminal to its original state
func (tty *TTY) Restore() {
	if tty.originalState != nil {
		term.Restore(tty.fd, tty.originalState)
	}
}

// Flush flushes the terminal output (no-op)
func (tty *TTY) Flush() {
	// No-op for golang.org/x/term
}

// WriteString writes a string to stdout
func (tty *TTY) WriteString(s string) error {
	_, err := os.Stdout.WriteString(s)
	return err
}

// ReadString reads a string from the TTY with timeout
func (tty *TTY) ReadString() (string, error) {
	// Set up a timeout channel
	timeout := time.After(100 * time.Millisecond) // Short timeout for terminal responses
	resultChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		// Set raw mode temporarily
		_, err := term.MakeRaw(tty.fd)
		if err != nil {
			errorChan <- err
			return
		}
		defer term.Restore(tty.fd, tty.originalState)

		var result []byte
		buffer := make([]byte, 1)

		for {
			n, err := os.Stdin.Read(buffer)
			if err != nil {
				errorChan <- err
				return
			}
			if n > 0 {
				// For terminal responses, look for bell character (0x07) which terminates OSC sequences
				if buffer[0] == 0x07 || buffer[0] == '\a' {
					resultChan <- string(result)
					return
				}
				// Also break on ESC sequence end for some terminals
				if len(result) > 0 && buffer[0] == '\\' && result[len(result)-1] == 0x1b {
					resultChan <- string(result)
					return
				}
				result = append(result, buffer[0])

				// Prevent infinite reading - limit response size
				if len(result) > 512 {
					resultChan <- string(result)
					return
				}
			}
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return "", err
	case <-timeout:
		// Timeout - return empty string (no error, just no response from terminal)
		return "", nil
	}
}

// PrintRawBytes for debugging raw byte sequences
func (tty *TTY) PrintRawBytes() {
	bytes := make([]byte, 6)
	var numRead int

	// Set the terminal into raw mode
	_, err := term.MakeRaw(tty.fd)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer term.Restore(tty.fd, tty.originalState)

	// Read bytes from stdin
	numRead, err = os.Stdin.Read(bytes)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Raw bytes: %v\n", bytes[:numRead])
}

// ASCII returns the ASCII code of the key pressed
func (tty *TTY) ASCII() int {
	ascii, _, err := asciiAndKeyCode(tty)
	if err != nil {
		return 0
	}
	return ascii
}

// KeyCode returns the key code of the key pressed
func (tty *TTY) KeyCode() int {
	_, keyCode, err := asciiAndKeyCode(tty)
	if err != nil {
		return 0
	}
	return keyCode
}

// WaitForKey waits for ctrl-c, Return, Esc, Space, or 'q' to be pressed
func WaitForKey() {
	// Get a new TTY and start reading keypresses in a loop
	r, err := NewTTY()
	if err != nil {
		panic(err)
	}
	defer r.Close()
	for {
		switch r.Key() {
		case KeyCtrlC, KeyEnter, KeyEsc, KeySpace, 'q':
			return
		}
	}
}
