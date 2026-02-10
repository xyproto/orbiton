//go:build !windows

package vt

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/pkg/term"
	"github.com/xyproto/env/v2"
)

var (
	defaultTimeout    = 2 * time.Millisecond
	defaultESCTimeout = 100 * time.Millisecond
	lastKey           int
)

// Key codes for CSI (ESC [) sequences and SS3 (ESC O) sequences.
var csiKeyLookup = map[string]int{
	"[A":    KeyArrowUp,
	"[B":    KeyArrowDown,
	"[C":    KeyArrowRight,
	"[D":    KeyArrowLeft,
	"[H":    KeyHome,
	"[F":    KeyEnd,
	"[1~":   KeyHome,
	"[4~":   KeyEnd,
	"[5~":   KeyPageUp,
	"[6~":   KeyPageDown,
	"[2;5~": KeyCtrlInsert,
}

var ss3KeyLookup = map[byte]int{
	'A': KeyArrowUp,
	'B': KeyArrowDown,
	'C': KeyArrowRight,
	'D': KeyArrowLeft,
	'H': KeyHome,
	'F': KeyEnd,
}

// Legacy lookup maps for basic terminals (Linux console, VT100)
// These use fixed-size byte arrays which work better with blocking reads.
var legacyKeyStringLookup = map[[3]byte]string{
	{27, 91, 65}:  "↑", // Up Arrow
	{27, 91, 66}:  "↓", // Down Arrow
	{27, 91, 67}:  "→", // Right Arrow
	{27, 91, 68}:  "←", // Left Arrow
	{27, 91, 'H'}: "⇱", // Home
	{27, 91, 'F'}: "⇲", // End
}

var legacyPageStringLookup = map[[4]byte]string{
	{27, 91, 49, 126}: "⇱", // Home
	{27, 91, 52, 126}: "⇲", // End
	{27, 91, 53, 126}: "⇞", // Page Up
	{27, 91, 54, 126}: "⇟", // Page Down
}

var legacyCtrlInsertStringLookup = map[[6]byte]string{
	{27, 91, 50, 59, 53, 126}: "⎘", // Ctrl-Insert (Copy)
}

// Legacy key code lookup maps (for Key/KeyRaw functions)
var legacyKeyCodeLookup = map[[3]byte]int{
	{27, 91, 65}:  KeyArrowUp,
	{27, 91, 66}:  KeyArrowDown,
	{27, 91, 67}:  KeyArrowRight,
	{27, 91, 68}:  KeyArrowLeft,
	{27, 91, 'H'}: KeyHome,
	{27, 91, 'F'}: KeyEnd,
}

var legacyPageCodeLookup = map[[4]byte]int{
	{27, 91, 49, 126}: KeyHome,
	{27, 91, 52, 126}: KeyEnd,
	{27, 91, 53, 126}: KeyPageUp,
	{27, 91, 54, 126}: KeyPageDown,
}

var legacyCtrlInsertCodeLookup = map[[6]byte]int{
	{27, 91, 50, 59, 53, 126}: KeyCtrlInsert,
}

// isBasicTerminal returns true for terminals that need the legacy
// blocking read approach (Linux console, VT100, etc.)
func isBasicTerminal() bool {
	term := env.Str("TERM")
	if term == "linux" || strings.HasPrefix(term, "vt") {
		return true
	}
	// Detect Linux console even if TERM is set incorrectly.
	// Virtual consoles are /dev/tty[1-9] or /dev/tty[1-9][0-9], not /dev/pts/*.
	if ttyName, err := os.Readlink("/proc/self/fd/0"); err == nil {
		if strings.HasPrefix(ttyName, "/dev/tty") && !strings.HasPrefix(ttyName, "/dev/tty/") {
			// /dev/tty followed by a digit means virtual console
			rest := strings.TrimPrefix(ttyName, "/dev/tty")
			if len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
				return true
			}
		}
	}
	return false
}

// legacyInputEnabled returns true if the old blocking input parser should be used.
func legacyInputEnabled() bool {
	return env.Bool("VT_LEGACY_INPUT") || isBasicTerminal()
}

const (
	esc                   = 0x1b
	bracketedPasteStart   = "\x1b[200~"
	bracketedPasteEnd     = "\x1b[201~"
	enableBracketedPaste  = "\x1b[?2004h"
	disableBracketedPaste = "\x1b[?2004l"
)

type inputReader struct {
	escDeadline time.Time
	buf         []byte
	escSeqLen   int
	inPaste     bool
}

type TTY struct {
	t          *term.Term
	reader     *inputReader
	fastFile   *os.File
	timeout    time.Duration
	escTimeout time.Duration
	fastBuf    [256]byte
	noBlock    bool
	fastRead   bool
}

// NewTTY opens /dev/tty in raw and cbreak mode as a term.Term
func NewTTY() (*TTY, error) {
	// Apply raw mode last to avoid cbreak overriding raw settings.
	t, err := term.Open("/dev/tty", term.CBreakMode, term.RawMode, term.ReadTimeout(defaultTimeout))
	if err != nil {
		return nil, err
	}
	tty := &TTY{
		t:          t,
		timeout:    defaultTimeout,
		escTimeout: defaultESCTimeout,
		reader:     &inputReader{},
	}
	// Best-effort enable bracketed paste for terminals that support it.
	_, _ = tty.t.Write([]byte(enableBracketedPaste))
	return tty, nil
}

// SetTimeout sets a timeout for reading a key
func (tty *TTY) SetTimeout(d time.Duration) {
	tty.timeout = d
	tty.t.SetReadTimeout(tty.timeout)
}

// SetEscTimeout sets the timeout used to decide if ESC is a standalone key.
func (tty *TTY) SetEscTimeout(d time.Duration) {
	tty.escTimeout = d
}

// FastInput enables or disables low-latency input for game loops and other real-time uses.
func (tty *TTY) FastInput(enable bool) {
	if enable {
		if tty.fastRead {
			return
		}
		f, err := os.OpenFile("/dev/tty", os.O_RDWR|syscall.O_NONBLOCK, 0)
		if err == nil {
			tty.fastFile = f
			tty.fastRead = true
			tty.SetTimeout(1 * time.Millisecond)
			tty.SetEscTimeout(5 * time.Millisecond)
		}
	} else {
		if !tty.fastRead {
			return
		}
		if tty.fastFile != nil {
			_ = tty.fastFile.Close()
			tty.fastFile = nil
		}
		tty.fastRead = false
		tty.SetTimeout(defaultTimeout)
		tty.SetEscTimeout(defaultESCTimeout)
	}
}

// Close will restore and close the raw terminal
func (tty *TTY) Close() {
	// Best-effort disable bracketed paste before restoring the terminal.
	_, _ = tty.t.Write([]byte(disableBracketedPaste))
	tty.t.Restore()
	tty.t.Close()
	if tty.fastFile != nil {
		_ = tty.fastFile.Close()
		tty.fastFile = nil
	}
}

// ReadEvent reads and parses a single input event (key, rune, or paste).
// It is designed to feel non-blocking while still assembling escape sequences.
func (tty *TTY) ReadEvent() (Event, error) {
	return tty.readEvent(tty.timeout, tty.escTimeout)
}

// ReadEventBlocking waits until a full input event is available.
func (tty *TTY) ReadEventBlocking() (Event, error) {
	for {
		ev, err := tty.readEvent(0, tty.escTimeout)
		if err != nil {
			return ev, err
		}
		if ev.Kind != EventNone {
			return ev, nil
		}
	}
}

func (tty *TTY) readEvent(poll, escWait time.Duration) (Event, error) {
	for {
		// Try to parse what's already in the buffer.
		ev, ready, needMore := tty.reader.parse(time.Now(), escWait)
		if ready {
			return ev, nil
		} else if !needMore && len(tty.reader.buf) == 0 {
			// No buffered input; read from terminal.
			if poll > 0 {
				// Kilo-style: just read with a timeout and parse whatever arrived.
				if err := tty.readIntoBuffer(poll); err != nil {
					return Event{Kind: EventNone}, err
				}
			} else {
				readTimeout := poll
				if poll == 0 && len(tty.reader.buf) > 0 {
					readTimeout = tty.timeout
				}
				if err := tty.readIntoBuffer(readTimeout); err != nil {
					return Event{Kind: EventNone}, err
				}
			}
			if len(tty.reader.buf) == 0 {
				return Event{Kind: EventNone}, nil
			}
			continue
		}

		// Kilo-style: after ESC, wait a little for the rest of the sequence.
		if needMore && len(tty.reader.buf) > 0 && tty.reader.buf[0] == esc {
			if err := tty.readIntoBuffer(escWait); err != nil {
				return Event{Kind: EventNone}, err
			}
			continue
		}

		readTimeout := poll
		if poll == 0 && len(tty.reader.buf) > 0 {
			readTimeout = tty.timeout
		}
		if err := tty.readIntoBuffer(readTimeout); err != nil {
			return Event{Kind: EventNone}, err
		}
	}
}

func (tty *TTY) readIntoBuffer(timeout time.Duration) error {
	if tty.fastRead {
		return tty.readIntoBufferFast(timeout)
	}
	_ = tty.t.SetReadTimeout(timeout)
	tmp := make([]byte, 256)
	n, err := tty.t.Read(tmp)
	if n > 0 {
		tty.reader.buf = append(tty.reader.buf, tmp[:n]...)
	}
	if err != nil && !errors.Is(err, io.EOF) {
		if isTimeoutErr(err) {
			return nil
		}
		return err
	}
	for {
		avail, err := tty.t.Available()
		if err != nil || avail <= 0 {
			break
		}
		if avail > len(tmp) {
			if avail > 4096 {
				avail = 4096
			}
			tmp = make([]byte, avail)
		}
		n, err = tty.t.Read(tmp[:avail])
		if n > 0 {
			tty.reader.buf = append(tty.reader.buf, tmp[:n]...)
		}
		if err != nil && !errors.Is(err, io.EOF) {
			if isTimeoutErr(err) {
				return nil
			}
			return err
		}
		if n == 0 {
			break
		}
	}
	return nil
}

func (tty *TTY) readIntoBufferFast(timeout time.Duration) error {
	if tty.fastFile == nil {
		return nil
	}
	deadline := time.Time{}
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}
	tmp := tty.fastBuf[:]
	for {
		n, err := syscall.Read(int(tty.fastFile.Fd()), tmp)
		if n > 0 {
			tty.reader.buf = append(tty.reader.buf, tmp[:n]...)
		}
		if err != nil {
			if isTimeoutErr(err) {
				if deadline.IsZero() {
					time.Sleep(1 * time.Millisecond)
					continue
				}
				return nil
			}
			return err
		}
		if n == 0 {
			return nil
		}
		if deadline.IsZero() || time.Now().After(deadline) {
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func isTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EINTR) {
		return true
	}
	if perr, ok := err.(*os.PathError); ok {
		return errors.Is(perr.Err, os.ErrDeadlineExceeded) ||
			errors.Is(perr.Err, syscall.EAGAIN) ||
			errors.Is(perr.Err, syscall.EWOULDBLOCK) ||
			errors.Is(perr.Err, syscall.EINTR)
	}
	return false
}

func (r *inputReader) parse(now time.Time, escWait time.Duration) (Event, bool, bool) {
	if len(r.buf) == 0 {
		r.escDeadline = time.Time{}
		r.escSeqLen = 0
		return Event{Kind: EventNone}, false, false
	}

	if r.inPaste {
		if idx := bytes.Index(r.buf, []byte(bracketedPasteEnd)); idx >= 0 {
			text := string(r.buf[:idx])
			r.buf = r.buf[idx+len(bracketedPasteEnd):]
			r.inPaste = false
			r.escSeqLen = 0
			return Event{Kind: EventPaste, Text: text}, true, false
		}
		return Event{Kind: EventNone}, false, true
	}

	if r.buf[0] == esc || r.buf[0] == 0x9b || r.buf[0] == 0x8f {
		if r.buf[0] == esc && len(r.buf) >= 3 && r.buf[1] == '[' {
			if r.buf[2] == '[' {
				copy(r.buf[2:], r.buf[3:])
				r.buf = r.buf[:len(r.buf)-1]
			} else if r.buf[2] == 'O' {
				r.buf[1] = 'O'
				copy(r.buf[2:], r.buf[3:])
				r.buf = r.buf[:len(r.buf)-1]
			}
		}
		if bytes.HasPrefix(r.buf, []byte(bracketedPasteStart)) {
			r.buf = r.buf[len(bracketedPasteStart):]
			r.inPaste = true
			if idx := bytes.Index(r.buf, []byte(bracketedPasteEnd)); idx >= 0 {
				text := string(r.buf[:idx])
				r.buf = r.buf[idx+len(bracketedPasteEnd):]
				r.inPaste = false
				r.escSeqLen = 0
				return Event{Kind: EventPaste, Text: text}, true, false
			}
			return Event{Kind: EventNone}, false, true
		}

		if ev, consumed, ok, complete := parseCSI(r.buf); ok {
			r.buf = r.buf[consumed:]
			r.escDeadline = time.Time{}
			r.escSeqLen = 0
			return ev, true, false
		} else if complete && consumed > 0 {
			seq := string(r.buf[:consumed])
			r.buf = r.buf[consumed:]
			r.escDeadline = time.Time{}
			r.escSeqLen = 0
			return Event{Kind: EventText, Text: seq}, true, false
		}

		if ev, consumed, ok, complete := parseSS3(r.buf); ok {
			r.buf = r.buf[consumed:]
			r.escDeadline = time.Time{}
			r.escSeqLen = 0
			return ev, true, false
		} else if complete && consumed > 0 {
			seq := string(r.buf[:consumed])
			r.buf = r.buf[consumed:]
			r.escDeadline = time.Time{}
			r.escSeqLen = 0
			return Event{Kind: EventText, Text: seq}, true, false
		}

		if r.escDeadline.IsZero() || len(r.buf) > r.escSeqLen {
			r.escDeadline = now.Add(escWait)
			r.escSeqLen = len(r.buf)
		}
		if now.Before(r.escDeadline) {
			return Event{Kind: EventNone}, false, true
		}

		r.buf = r.buf[1:]
		r.escDeadline = time.Time{}
		r.escSeqLen = 0
		return Event{Kind: EventKey, Key: int(esc)}, true, false
	}

	r.escDeadline = time.Time{}
	r.escSeqLen = 0
	r0, size := utf8.DecodeRune(r.buf)
	if r0 == utf8.RuneError && size == 1 {
		return Event{Kind: EventNone}, false, true
	}
	r.buf = r.buf[size:]
	return Event{Kind: EventRune, Rune: r0}, true, false
}

func parseCSI(buf []byte) (Event, int, bool, bool) {
	if len(buf) == 0 {
		return Event{}, 0, false, false
	}
	prefixLen := 0
	if buf[0] == esc {
		if len(buf) < 2 || buf[1] != '[' {
			return Event{}, 0, false, false
		}
		prefixLen = 2
	} else if buf[0] == 0x9b {
		prefixLen = 1
	} else {
		return Event{}, 0, false, false
	}
	for i := prefixLen; i < len(buf); i++ {
		b := buf[i]
		if b >= 0x40 && b <= 0x7e {
			if prefixLen == 2 && i == 2 && (b == '[' || b == 'O') {
				return Event{}, 0, false, false
			}
			var lookup string
			if prefixLen == 2 {
				lookup = string(buf[1 : i+1])
			} else {
				lookup = "[" + string(buf[1:i+1])
			}
			if code, ok := csiKeyLookup[lookup]; ok {
				return Event{Kind: EventKey, Key: code}, i + 1, true, true
			}
			seqStart := prefixLen
			if prefixLen == 2 {
				seqStart = 2
			}
			if ev, ok := parseCSIFallback(buf[seqStart:i], b); ok {
				return ev, i + 1, true, true
			}
			return Event{}, i + 1, false, true
		}
	}
	return Event{}, 0, false, false
}

func parseSS3(buf []byte) (Event, int, bool, bool) {
	if len(buf) == 0 {
		return Event{}, 0, false, false
	}
	if buf[0] == esc {
		if len(buf) < 2 || buf[1] != 'O' {
			return Event{}, 0, false, false
		}
		if len(buf) < 3 {
			return Event{}, 0, false, false
		}
		if code, ok := ss3KeyLookup[buf[2]]; ok {
			return Event{Kind: EventKey, Key: code}, 3, true, true
		}
		return Event{}, 3, false, true
	}
	if buf[0] == 0x8f {
		if len(buf) < 2 {
			return Event{}, 0, false, false
		}
		if code, ok := ss3KeyLookup[buf[1]]; ok {
			return Event{Kind: EventKey, Key: code}, 2, true, true
		}
		return Event{}, 2, false, true
	}
	return Event{}, 0, false, false
}

// Key reads the keycode or ASCII code and avoids repeated keys
func (tty *TTY) Key() int {
	if !tty.noBlock {
		tty.RawMode()
	}

	var key int

	// Use legacy blocking read for basic terminals (Linux console, VT100)
	if legacyInputEnabled() {
		key = tty.keyRawLegacy()
	} else {
		ev, err := tty.ReadEvent()
		if ev.Kind != EventNone {
			tty.t.Flush()
		}
		if err != nil {
			if !tty.noBlock {
				tty.Restore()
			}
			lastKey = 0
			return 0
		}
		switch ev.Kind {
		case EventKey:
			key = ev.Key
		case EventRune:
			key = int(ev.Rune)
		default:
			key = 0
		}
	}

	if !tty.noBlock {
		tty.Restore()
	}

	if key == lastKey {
		lastKey = 0
		return 0
	}
	lastKey = key
	return key
}

// KeyRaw reads a key without toggling raw mode or flushing input.
// Callers should manage tty.RawMode() / tty.Restore() themselves.
func (tty *TTY) KeyRaw() int {
	// Ensure raw mode is active to avoid echoing escape sequences.
	tty.RawMode()

	// Use legacy blocking read for basic terminals (Linux console, VT100)
	if legacyInputEnabled() {
		return tty.keyRawLegacy()
	}

	ev, err := tty.ReadEvent()
	if err != nil {
		return 0
	}
	var key int
	switch ev.Kind {
	case EventKey:
		key = ev.Key
	case EventRune:
		key = int(ev.Rune)
	default:
		key = 0
	}
	return key
}

// keyRawLegacy uses the old blocking read approach for key codes.
func (tty *TTY) keyRawLegacy() int {
	bytes := make([]byte, 6)
	tty.SetTimeout(tty.timeout)
	numRead, err := tty.t.Read(bytes)
	if err != nil || numRead == 0 {
		return 0
	}
	switch {
	case numRead == 1:
		return int(bytes[0])
	case numRead == 3:
		seq := [3]byte{bytes[0], bytes[1], bytes[2]}
		if code, found := legacyKeyCodeLookup[seq]; found {
			return code
		}
		r, _ := utf8.DecodeRune(bytes[:numRead])
		if unicode.IsPrint(r) {
			return int(r)
		}
	case numRead == 4:
		seq := [4]byte{bytes[0], bytes[1], bytes[2], bytes[3]}
		if code, found := legacyPageCodeLookup[seq]; found {
			return code
		}
	case numRead == 6:
		seq := [6]byte{bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5]}
		if code, found := legacyCtrlInsertCodeLookup[seq]; found {
			return code
		}
	default:
		r, _ := utf8.DecodeRune(bytes[:numRead])
		if unicode.IsPrint(r) {
			return int(r)
		}
	}
	return 0
}

// String reads a string, handling key sequences and printable characters
func (tty *TTY) String() string {
	tty.RawMode()
	ev, err := tty.ReadEventBlocking()
	tty.Restore()
	if ev.Kind != EventNone {
		tty.t.Flush()
	}
	if err != nil {
		return ""
	}
	switch ev.Kind {
	case EventPaste:
		return ev.Text
	case EventText:
		return ev.Text
	case EventKey:
		return KeySymbol(ev.Key)
	case EventRune:
		if unicode.IsPrint(ev.Rune) {
			return string(ev.Rune)
		}
		return "c:" + strconv.Itoa(int(ev.Rune))
	default:
		return ""
	}
}

// StringRaw reads a string without toggling raw mode or flushing input.
// Callers should manage tty.RawMode() / tty.Restore() themselves.
func (tty *TTY) StringRaw() string {
	// Ensure raw mode is active to avoid echoing escape sequences.
	tty.RawMode()

	// Use legacy blocking read for basic terminals (Linux console, VT100)
	// where termios timeout-based escape sequence assembly doesn't work reliably.
	if legacyInputEnabled() {
		return tty.stringRawLegacy()
	}

	ev, err := tty.ReadEventBlocking()
	if err != nil {
		return ""
	}
	switch ev.Kind {
	case EventPaste:
		return ev.Text
	case EventText:
		return ev.Text
	case EventKey:
		return KeySymbol(ev.Key)
	case EventRune:
		if unicode.IsPrint(ev.Rune) {
			return string(ev.Rune)
		}
		return "c:" + strconv.Itoa(int(ev.Rune))
	default:
		return ""
	}
}

// stringRawLegacy uses the old blocking read approach that works reliably
// on the Linux console and VT100 terminals.
func (tty *TTY) stringRawLegacy() string {
	bytes := make([]byte, 6)
	tty.SetTimeout(0) // Block until input
	numRead, err := tty.t.Read(bytes)
	if err != nil || numRead == 0 {
		return ""
	}
	switch {
	case numRead == 1:
		r := rune(bytes[0])
		if unicode.IsPrint(r) {
			return string(r)
		}
		return "c:" + strconv.Itoa(int(r))
	case numRead == 3:
		seq := [3]byte{bytes[0], bytes[1], bytes[2]}
		if str, found := legacyKeyStringLookup[seq]; found {
			return str
		}
		// Attempt to interpret as UTF-8 string
		return string(bytes[:numRead])
	case numRead == 4:
		seq := [4]byte{bytes[0], bytes[1], bytes[2], bytes[3]}
		if str, found := legacyPageStringLookup[seq]; found {
			return str
		}
		return string(bytes[:numRead])
	case numRead == 6:
		seq := [6]byte{bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5]}
		if str, found := legacyCtrlInsertStringLookup[seq]; found {
			return str
		}
		fallthrough
	default:
		bytesLeftToRead, err := tty.t.Available()
		if err == nil && bytesLeftToRead > 0 {
			bytes2 := make([]byte, bytesLeftToRead)
			numRead2, err := tty.t.Read(bytes2)
			if err == nil && numRead2 > 0 {
				return string(append(bytes[:numRead], bytes2[:numRead2]...))
			}
		}
	}
	return string(bytes[:numRead])
}

// ReadStringEvent reads a string, entering raw mode and flushing after events.
// It does not call Restore; the caller is responsible for restoring the terminal.
func (tty *TTY) ReadStringEvent() string {
	tty.RawMode()

	// Use legacy blocking read for basic terminals (Linux console, VT100)
	if legacyInputEnabled() {
		return tty.stringRawLegacy()
	}

	ev, err := tty.ReadEventBlocking()
	if ev.Kind != EventNone {
		tty.t.Flush()
	}
	if err != nil {
		return ""
	}
	switch ev.Kind {
	case EventPaste:
		return ev.Text
	case EventText:
		return ev.Text
	case EventKey:
		return KeySymbol(ev.Key)
	case EventRune:
		if unicode.IsPrint(ev.Rune) {
			return string(ev.Rune)
		}
		return "c:" + strconv.Itoa(int(ev.Rune))
	default:
		return ""
	}
}

// Rune reads a rune, handling special sequences for arrows, Home, End, etc.
func (tty *TTY) Rune() rune {
	tty.RawMode()
	ev, err := tty.ReadEventBlocking()
	tty.Restore()
	if ev.Kind != EventNone {
		tty.t.Flush()
	}
	if err != nil {
		return rune(0)
	}
	switch ev.Kind {
	case EventRune:
		return ev.Rune
	case EventKey:
		return KeyRune(ev.Key)
	case EventText:
		if ev.Text != "" {
			return []rune(ev.Text)[0]
		}
	case EventPaste:
		if ev.Text != "" {
			return []rune(ev.Text)[0]
		}
	}
	return rune(0)
}

// RuneRaw reads a rune without toggling raw mode or flushing input.
// Callers should manage tty.RawMode() / tty.Restore() themselves.
func (tty *TTY) RuneRaw() rune {
	// Ensure raw mode is active to avoid echoing escape sequences.
	tty.RawMode()
	ev, err := tty.ReadEventBlocking()
	if err != nil {
		return rune(0)
	}
	switch ev.Kind {
	case EventRune:
		return ev.Rune
	case EventKey:
		return KeyRune(ev.Key)
	case EventText:
		if ev.Text != "" {
			return []rune(ev.Text)[0]
		}
	case EventPaste:
		if ev.Text != "" {
			return []rune(ev.Text)[0]
		}
	}
	return rune(0)
}

// RawMode switches the terminal to raw mode
func (tty *TTY) RawMode() {
	tty.t.SetRaw()
}

// NoBlock prevents Key() from toggling terminal modes.
// Use this in game loops to prevent escape sequence characters from being echoed.
func (tty *TTY) NoBlock() {
	tty.noBlock = true
	tty.RawMode()
}

// Restore the terminal to its original state
func (tty *TTY) Restore() {
	tty.t.Restore()
}

// Flush flushes the terminal output
func (tty *TTY) Flush() {
	tty.t.Flush()
}

// WriteString writes a string to the terminal
func (tty *TTY) WriteString(s string) error {
	if n, err := tty.t.Write([]byte(s)); err != nil || n == 0 {
		return errors.New("no bytes written to the TTY")
	}
	return nil
}

// ReadString reads a string from the TTY with timeout
func (tty *TTY) ReadString() (string, error) {
	// Set up a timeout channel
	timeout := time.After(100 * time.Millisecond)
	resultChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		// Set raw mode temporarily
		tty.RawMode()
		defer tty.Restore()
		defer tty.Flush()

		var result []byte
		buffer := make([]byte, 1)

		for {
			n, err := tty.t.Read(buffer)
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

	// Set the terminal into raw mode with a timeout
	tty.RawMode()
	tty.SetTimeout(0)
	// Read bytes from the terminal
	numRead, err := tty.t.Read(bytes)
	// Restore the terminal settings
	tty.Restore()
	tty.t.Flush()

	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Raw bytes: %v\n", bytes[:numRead])
}

// Term will return the underlying term.Term
func (tty *TTY) Term() *term.Term {
	return tty.t
}

// ASCII returns the ASCII code of the key pressed
func (tty *TTY) ASCII() int {
	tty.RawMode()
	defer func() {
		tty.Restore()
		tty.t.Flush()
	}()
	ev, err := tty.ReadEvent()
	if err != nil {
		return 0
	}
	if ev.Kind == EventRune {
		return int(ev.Rune)
	}
	return 0
}

// KeyCode returns the key code of the key pressed
func (tty *TTY) KeyCode() int {
	tty.RawMode()
	defer func() {
		tty.Restore()
		tty.t.Flush()
	}()
	ev, err := tty.ReadEvent()
	if err != nil {
		return 0
	}
	if ev.Kind == EventKey {
		return ev.Key
	}
	return 0
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
