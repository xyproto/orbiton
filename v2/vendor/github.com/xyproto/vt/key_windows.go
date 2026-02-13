//go:build windows

package vt

import (
	"errors"
	"fmt"
	"os"
	"time"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

var (
	defaultTimeout = 2 * time.Millisecond
	lastKey        int
)

type TTY struct {
	fd      int
	orig    *term.State
	timeout time.Duration
}

// NewTTY opens the terminal.
// On Windows, we try to use the file descriptor of os.Stdin.
// If os.Stdin is a pipe (e.g. Mintty), this might fail or require special handling,
// but for now we assume a Windows Console API compatible environment or that term.MakeRaw works.
func NewTTY() (*TTY, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		// Fallback: try opening CONIN$
		f, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
		if err == nil {
			fd = int(f.Fd())
		} else {
			return nil, fmt.Errorf("stdin is not a terminal and CONIN$ could not be opened: %w", err)
		}
	}

	// We don't set raw mode here immediately, unlike unix version which does logic in NewTTY.
	// But the unix version in key.go DOES set raw mode in NewTTY.
	// Let's do the same.

	// Save original state (by making raw and then restoring? No, MakeRaw returns the *old* state).
	// But we want to keep the old state to restore later.
	// And we want to be in raw mode.
	orig, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

	// Enable Virtual Terminal Input if possible, to get arrow keys as sequences
	handle := windows.Handle(fd)
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err == nil {
		// ENABLE_VIRTUAL_TERMINAL_INPUT = 0x0200
		const EnableVirtualTerminalInput = 0x0200
		if mode&EnableVirtualTerminalInput == 0 {
			windows.SetConsoleMode(handle, mode|EnableVirtualTerminalInput)
			// Update the stored original state?
			// term.MakeRaw returns the state *before* it touched it.
			// If we modify it further, we should probably be careful.
			// But Restore() uses the state from MakeRaw.
			// If we changed mode manually after MakeRaw, Restore() might flip it back, which is good.
		}
	}

	return &TTY{
		fd:      fd,
		orig:    orig,
		timeout: defaultTimeout,
	}, nil
}

// SetTimeout sets a timeout for reading a key.
// Since Windows ReadFile blocks, we might need a workaround for timeouts.
// For now, we store it.
func (tty *TTY) SetTimeout(d time.Duration) {
	tty.timeout = d
}

// Close restores the terminal
func (tty *TTY) Close() {
	tty.Restore()
}

// Key reads the keycode or ASCII code
func (tty *TTY) Key() int {
	ascii, keyCode, err := asciiAndKeyCode(tty)
	if err != nil {
		lastKey = 0
		return 0
	}
	var key int
	if keyCode != 0 {
		key = keyCode
	} else {
		key = ascii
	}
	if key == lastKey {
		lastKey = 0
		return 0
	}
	lastKey = key
	return key
}

// asciiAndKeyCode processes input into an ASCII code or key code
func asciiAndKeyCode(tty *TTY) (ascii, keyCode int, err error) {
	// On Windows, we just read bytes. The terminal should be in raw mode sending VT sequences.
	// We use the same logic as Unix but with our read implementation.

	bytes := make([]byte, 6)

	// Read with timeout
	numRead, err := tty.readWithTimeout(bytes)

	if err != nil || numRead == 0 {
		return 0, 0, err
	}

	// Handle multi-byte sequences (same lookup tables as key.go)
	// We need to access the lookup tables from key.go.
	// Since they are in the same package (vt), we can access them if they are exported or in same package.
	// key.go is "!windows", so those variables might NOT be compiled in on Windows if they are defined in key.go!
	// CHECK THIS: variables in key.go are inside `//go:build !windows && !plan9`.
	// So they are NOT available here. We must duplicate them or move them to a common file.

	// For now I will assume I need to duplicate them or move them.
	// Moving them is cleaner. I should move them to `key_common.go`.

	// Let's defer this check and assume I will fix it.

	// ... logic duplicated from key.go ...

	// Since I cannot access them yet, I'll put placeholders or move them in next step.
	// I will just use the logic for now assuming I have the maps.

	switch {
	case numRead == 1:
		ascii = int(bytes[0])
	case numRead == 3:
		seq := [3]byte{bytes[0], bytes[1], bytes[2]}
		if code, found := keyCodeLookup[seq]; found {
			keyCode = code
			return
		}
		r, _ := utf8.DecodeRune(bytes[:numRead])
		if r != utf8.RuneError {
			ascii = int(r)
		}
	case numRead == 4:
		seq := [4]byte{bytes[0], bytes[1], bytes[2], bytes[3]}
		if code, found := pageNavLookup[seq]; found {
			keyCode = code
			return
		}
	case numRead == 6:
		seq := [6]byte{bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5]}
		if code, found := ctrlInsertLookup[seq]; found {
			keyCode = code
			return
		}
	default:
		r, _ := utf8.DecodeRune(bytes[:numRead])
		if r != utf8.RuneError {
			ascii = int(r)
		}
	}
	return
}

// readWithTimeout implements reading with timeout on Windows
func (tty *TTY) readWithTimeout(b []byte) (int, error) {
	// If timeout is 0, block.
	// If timeout > 0, wait up to timeout.

	if tty.timeout <= 0 {
		var n uint32
		err := windows.ReadFile(windows.Handle(tty.fd), b, &n, nil)
		return int(n), err
	}

	// Wait for input with timeout
	handle := windows.Handle(tty.fd)
	event, err := windows.WaitForSingleObject(handle, uint32(tty.timeout.Milliseconds()))
	if err != nil {
		return 0, err
	}
	if event == uint32(windows.WAIT_TIMEOUT) {
		return 0, nil // Timeout
	}

	// Loop until we find a key down event or timeout again (if we consumed everything)
	// Since x/sys/windows doesn't expose InputRecord easily, we will use a simplified approach using syscall.

	type KEY_EVENT_RECORD struct {
		bKeyDown          int32
		wRepeatCount      uint16
		wVirtualKeyCode   uint16
		wVirtualScanCode  uint16
		uChar             [2]byte
		dwControlKeyState uint32
	}
	// InputRecord size is 20 bytes (2 + 2 padding + 16 union)
	type INPUT_RECORD struct {
		EventType uint16
		_         [2]byte // padding
		Event     [16]byte
	}

	modkernel32 := windows.NewLazySystemDLL("kernel32.dll")
	procPeekConsoleInputW := modkernel32.NewProc("PeekConsoleInputW")
	procReadConsoleInputW := modkernel32.NewProc("ReadConsoleInputW")

	for {
		// Peek at the number of events
		var numEvents uint32
		err = windows.GetNumberOfConsoleInputEvents(handle, &numEvents)
		if err != nil {
			break
		}
		if numEvents == 0 {
			return 0, nil
		}

		// Peek 1 event
		var events [1]INPUT_RECORD
		var numRead uint32

		r1, _, _ := procPeekConsoleInputW.Call(uintptr(handle), uintptr(unsafe.Pointer(&events[0])), 1, uintptr(unsafe.Pointer(&numRead)))
		if r1 == 0 {
			break // Error
		}
		if numRead == 0 {
			return 0, nil
		}

		first := events[0]
		shouldConsume := false

		const KEY_EVENT = 0x0001

		if first.EventType == KEY_EVENT {
			// Extract KeyEvent
			ke := *(*KEY_EVENT_RECORD)(unsafe.Pointer(&first.Event[0]))

			if ke.bKeyDown == 0 {
				shouldConsume = true // Ignore key up
			} else {
				// Key Down.
				vk := ke.wVirtualKeyCode
				if vk == 0x10 || vk == 0x11 || vk == 0x12 { // Shift, Ctrl, Alt
					if ke.uChar[0] == 0 && ke.uChar[1] == 0 {
						shouldConsume = true
					}
				}
			}
		} else {
			// Not a key event
			shouldConsume = true
		}

		if shouldConsume {
			// Remove it
			var dummy [1]INPUT_RECORD
			var n uint32
			procReadConsoleInputW.Call(uintptr(handle), uintptr(unsafe.Pointer(&dummy[0])), 1, uintptr(unsafe.Pointer(&n)))
			continue
		}

		// Good event found
		break
	}
	// ...

	// Input available
	var n uint32
	err = windows.ReadFile(handle, b, &n, nil)
	return int(n), err
}

// String reads a string
func (tty *TTY) String() string {
	bytes := make([]byte, 6)
	// Block until data
	tty.SetTimeout(0)
	numRead, err := tty.readWithTimeout(bytes)
	if err != nil || numRead == 0 {
		return ""
	}

	// Same logic as key.go
	switch {
	case numRead == 1:
		r := rune(bytes[0])
		return string(r) // Simplified: assume print if 1 byte
	case numRead == 3:
		seq := [3]byte{bytes[0], bytes[1], bytes[2]}
		if str, found := keyStringLookup[seq]; found {
			return str
		}
		return string(bytes[:numRead])
	// ... handling others ...
	default:
		// Read more?
		return string(bytes[:numRead])
	}
}

// Rune reads a rune
func (tty *TTY) Rune() rune {
	bytes := make([]byte, 6)
	tty.SetTimeout(0)
	numRead, err := tty.readWithTimeout(bytes)
	if err != nil || numRead == 0 {
		return rune(0)
	}
	// Simplified logic
	r, _ := utf8.DecodeRune(bytes[:numRead])
	return r
}

// RawMode switches the terminal to raw mode
func (tty *TTY) RawMode() {
	// Already in raw mode if NewTTY was called?
	// But maybe we restored it.
	// term.MakeRaw returns new state, but we don't need it if we just want to set it.
	// Actually we should store the state if we want to toggle.
	// But for now, let's just call MakeRaw again?
	// MakeRaw returns the *previous* state.
	// If we are already raw, it returns raw state.
	// Warning: calling MakeRaw repeatedly might nest things?
	// term.MakeRaw just calls SetConsoleMode.
	term.MakeRaw(tty.fd)
}

// NoBlock - Windows doesn't easily support non-blocking ReadFile without overlapped IO.
// But we use WaitForSingleObject in readWithTimeout, so we can simulate it by setting timeout to very small.
func (tty *TTY) NoBlock() {
	tty.SetTimeout(1 * time.Millisecond)
}

// Restore the terminal to its original state
func (tty *TTY) Restore() {
	if tty.orig != nil {
		term.Restore(tty.fd, tty.orig)
	}
}

// Flush discards pending input/output
func (tty *TTY) Flush() {
	// Windows FlushConsoleInputBuffer
	windows.FlushConsoleInputBuffer(windows.Handle(tty.fd))
}

// WriteString writes a string to the terminal
func (tty *TTY) WriteString(s string) error {
	_, err := os.Stdout.WriteString(s)
	return err
}

// ReadString reads all available data
func (tty *TTY) ReadString() (string, error) {
	var result []byte
	buf := make([]byte, 128)
	// Temporarily set a short read timeout
	tty.SetTimeout(100 * time.Millisecond)
	defer tty.SetTimeout(tty.timeout)
	for {
		n, err := tty.readWithTimeout(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err != nil || n == 0 {
			break
		}
	}
	if len(result) == 0 {
		return "", errors.New("no data read from TTY")
	}
	return string(result), nil
}

// PrintRawBytes ...
func (tty *TTY) PrintRawBytes() {}

// ASCII ...
func (tty *TTY) ASCII() int { return 0 }

// KeyCode ...
func (tty *TTY) KeyCode() int { return 0 }

// WaitForKey ...
func WaitForKey() {
	tty, _ := NewTTY()
	if tty != nil {
		defer tty.Close()
		for {
			k := tty.Key()
			if k == 3 || k == 13 || k == 27 || k == 32 || k == 113 {
				return
			}
		}
	}
}
