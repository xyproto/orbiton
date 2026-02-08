package main

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v3"
	"github.com/xyproto/vt"
)

var (
	keyEventQueue      chan string
	keyInputDone       chan struct{}
	keyScreen          tcell.Screen
	pendingKeyInput    []string
	pendingCSIFragment string
)

const (
	leftArrow  = "←"
	rightArrow = "→"
	upArrow    = "↑"
	downArrow  = "↓"

	pgUpKey = "⇞" // page up
	pgDnKey = "⇟" // page down
	homeKey = "⇱" // home
	endKey  = "⇲" // end
	copyKey = "⎘" // ctrl-insert
)

// configureKeyInput configures keyboard input parsing via tcell/v3.
func configureKeyInput(_ keyInputTTY) error {
	if keyScreen != nil {
		return nil
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	screen.EnablePaste()

	keyScreen = screen
	keyEventQueue = make(chan string, 1024)
	keyInputDone = make(chan struct{})

	go tcellInputLoop()

	return nil
}

func restoreStdinMode() {
	if keyScreen != nil {
		close(keyInputDone)
		keyScreen.Fini()
		keyScreen = nil
	}
}

func startKeyInput(tty keyInputTTY) error {
	_ = tty
	return configureKeyInput(tty)
}

func stopKeyInput(tty keyInputTTY) {
	restoreStdinMode()
	_ = tty
}

func readKeyEvent(_ keyInputTTY) string {
	if keyEventQueue == nil {
		return ""
	}
	for {
		key, ok := nextMappedKey()
		if !ok {
			return ""
		}

		hadPending := pendingCSIFragment != ""
		if hadPending {
			key = pendingCSIFragment + key
			pendingCSIFragment = ""
		}

		if key == "c:27" {
			if mapped, ok := decodeEscPrefixedSequence(); ok {
				return mapped
			}
			return key
		}

		if mapped, complete, incomplete := decodeInlineCSI(key, hadPending); complete {
			return mapped
		} else if incomplete {
			pendingCSIFragment = key
			continue
		}

		if key == "[" {
			if mapped, ok := decodeBracketSequence(); ok {
				return mapped
			}
			return "["
		}

		if mapped, ok := assembleSplitCSI(key); ok {
			return mapped
		}

		return key
	}
}

func tcellInputLoop() {
	evq := keyScreen.EventQ()
	for {
		select {
		case <-keyInputDone:
			close(keyEventQueue)
			return
		case ev, ok := <-evq:
			if !ok {
				close(keyEventQueue)
				return
			}
			switch tev := ev.(type) {
			case *tcell.EventKey:
				key := mapTCellKey(tev)
				if key != "" {
					keyEventQueue <- key
				}
			case *tcell.EventPaste:
				_ = tev
			}
		}
	}
}

func mapTCellKey(ev *tcell.EventKey) string {
	switch ev.Key() {
	case tcell.KeyUp:
		return upArrow
	case tcell.KeyDown:
		return downArrow
	case tcell.KeyRight:
		return rightArrow
	case tcell.KeyLeft:
		return leftArrow
	case tcell.KeyHome:
		return homeKey
	case tcell.KeyEnd:
		return endKey
	case tcell.KeyPgUp:
		return pgUpKey
	case tcell.KeyPgDn:
		return pgDnKey
	case tcell.KeyEsc:
		return "c:27"
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return "c:127"
	case tcell.KeyTab:
		return "c:9"
	case tcell.KeyEnter:
		return "c:13"
	case tcell.KeyRune:
		return mapTCellRuneKey(ev)
	}

	if ev.Key() >= tcell.KeyCtrlA && ev.Key() <= tcell.KeyCtrlZ {
		n := int(ev.Key()-tcell.KeyCtrlA) + 1
		return "c:" + strconv.Itoa(n)
	}

	if name, ok := mapOtherCtrlKeys(ev.Key()); ok {
		return name
	}

	return ""
}

func mapOtherCtrlKeys(k tcell.Key) (string, bool) {
	switch k {
	case tcell.KeyNUL:
		return "c:0", true
	case tcell.KeyETX:
		return "c:3", true
	case tcell.KeyUS:
		return "c:31", true
	default:
		return "", false
	}
}

func mapTCellRuneKey(ev *tcell.EventKey) string {
	s := ev.Str()
	if s == "" {
		return ""
	}

	// Some terminals can surface full escape sequences as rune strings.
	// Decode those directly instead of inserting raw bytes in the buffer.
	if strings.HasPrefix(s, "\x1b") || strings.HasPrefix(s, "[") || strings.HasPrefix(s, "O") {
		if mapped, complete, _ := decodeInlineCSI(s, true); complete {
			return mapped
		}
		// Avoid leaking undecoded escape sequences as text.
		if strings.ContainsRune(s, '\x1b') || strings.HasPrefix(s, "[") || strings.HasPrefix(s, "O") {
			return ""
		}
	}

	r := []rune(s)[0]
	if ev.Modifiers()&tcell.ModCtrl != 0 {
		if r == ' ' {
			return "c:32"
		}
		lr := unicode.ToLower(r)
		if lr >= 'a' && lr <= 'z' {
			return "c:" + strconv.Itoa(int(lr-'a')+1)
		}
		switch lr {
		case '[':
			return "c:27"
		case '\\':
			return "c:28"
		case ']':
			return "c:29"
		case '^':
			return "c:30"
		case '_':
			return "c:31"
		}
	}

	return s
}

func nextMappedKey() (string, bool) {
	if len(pendingKeyInput) > 0 {
		k := pendingKeyInput[0]
		pendingKeyInput = pendingKeyInput[1:]
		return k, true
	}
	k, ok := <-keyEventQueue
	return k, ok
}

func nextMappedKeyTimeout(d time.Duration) (string, bool) {
	if len(pendingKeyInput) > 0 {
		k := pendingKeyInput[0]
		pendingKeyInput = pendingKeyInput[1:]
		return k, true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case k, ok := <-keyEventQueue:
		if !ok {
			return "", false
		}
		return k, true
	case <-timer.C:
		return "", false
	}
}

func unreadMappedKeysFront(keys []string) {
	if len(keys) == 0 {
		return
	}
	merged := make([]string, 0, len(keys)+len(pendingKeyInput))
	merged = append(merged, keys...)
	merged = append(merged, pendingKeyInput...)
	pendingKeyInput = merged
}

func decodeBracketSequence() (string, bool) {
	const maxTokens = 24
	consumed := make([]string, 0, maxTokens)
	for len(consumed) < maxTokens {
		k, ok := nextMappedKeyTimeout(8 * time.Millisecond)
		if !ok {
			unreadMappedKeysFront(consumed)
			return "", false
		}
		consumed = append(consumed, k)
		if len(k) == 1 {
			c := k[0]
			if c == '~' || c == 'u' || (c >= 'A' && c <= 'Z') {
				break
			}
		}
	}

	seq := strings.Join(consumed, "")
	if seq == "" {
		unreadMappedKeysFront(consumed)
		return "", false
	}

	if mapped, ok := decodeCSITail(seq); ok {
		return mapped, true
	}
	if strings.HasSuffix(seq, "u") {
		if mapped, ok := parseKittyCSIuBody(strings.TrimSuffix(seq, "u")); ok {
			return mapped, true
		}
		unreadMappedKeysFront(consumed)
		return "", false
	}
	unreadMappedKeysFront(consumed)
	return "", false
}

func decodeEscPrefixedSequence() (string, bool) {
	k, ok := nextMappedKeyTimeout(12 * time.Millisecond)
	if !ok {
		return "", false
	}
	if k == "[" {
		if mapped, ok := decodeBracketSequence(); ok {
			return mapped, true
		}
		unreadMappedKeysFront([]string{"["})
		return "", false
	}
	if k == "O" {
		k2, ok := nextMappedKeyTimeout(12 * time.Millisecond)
		if !ok {
			unreadMappedKeysFront([]string{"O"})
			return "", false
		}
		switch k2 {
		case "A":
			return upArrow, true
		case "B":
			return downArrow, true
		case "C":
			return rightArrow, true
		case "D":
			return leftArrow, true
		case "H":
			return homeKey, true
		case "F":
			return endKey, true
		default:
			unreadMappedKeysFront([]string{"O", k2})
			return "", false
		}
	}
	unreadMappedKeysFront([]string{k})
	return "", false
}

func decodeInlineCSI(key string, hadPending bool) (mapped string, complete bool, incomplete bool) {
	s := key
	hadPrefix := false

	if strings.HasPrefix(s, "c:27") {
		s = strings.TrimPrefix(s, "c:27")
		hadPrefix = true
	}
	if strings.HasPrefix(s, "\x1b") {
		s = strings.TrimPrefix(s, "\x1b")
		hadPrefix = true
	}
	if strings.HasPrefix(s, "[") {
		s = strings.TrimPrefix(s, "[")
		hadPrefix = true
	} else if strings.HasPrefix(s, "O") {
		s = strings.TrimPrefix(s, "O")
		hadPrefix = true
		if s == "" {
			return "", false, true
		}
		switch s[0] {
		case 'A':
			return upArrow, true, false
		case 'B':
			return downArrow, true, false
		case 'C':
			return rightArrow, true, false
		case 'D':
			return leftArrow, true, false
		case 'H':
			return homeKey, true, false
		case 'F':
			return endKey, true, false
		default:
			return "", false, false
		}
	}

	if s == "" {
		if hadPrefix || hadPending {
			return "", false, true
		}
		return "", false, false
	}

	if isCSINumericTail(s) && (hadPrefix || hadPending) {
		return "", false, true
	}

	last := s[len(s)-1]
	if last == 'u' {
		if mapped, ok := parseKittyCSIuBody(strings.TrimSuffix(s, "u")); ok {
			return mapped, true, false
		}
		return "", false, false
	}

	if last == '~' || (last >= 'A' && last <= 'Z') {
		// Do not map plain text "A"/"B"/... to arrows unless this came from
		// CSI/SS3 context or an existing CSI fragment.
		if len(s) == 1 && !(hadPrefix || hadPending) {
			return "", false, false
		}
		if mapped, ok := decodeCSITail(s); ok {
			return mapped, true, false
		}
	}

	return "", false, false
}

func assembleSplitCSI(first string) (string, bool) {
	if !looksLikeCSIPotentialStart(first) {
		return "", false
	}

	consumed := []string{first}
	seq := first

	if mapped, complete, _ := decodeInlineCSI(seq, true); complete {
		return mapped, true
	}

	for i := 0; i < 8; i++ {
		k, ok := nextMappedKeyTimeout(10 * time.Millisecond)
		if !ok {
			break
		}
		consumed = append(consumed, k)
		seq += k

		if mapped, complete, incomplete := decodeInlineCSI(seq, true); complete {
			return mapped, true
		} else if !incomplete {
			// Not CSI-like anymore, stop trying.
			break
		}
	}

	// Put back all but the first token; caller will emit the first as text.
	if len(consumed) > 1 {
		unreadMappedKeysFront(consumed[1:])
	}
	return "", false
}

func parseKittyCSIuBody(body string) (string, bool) {
	parts := strings.Split(body, ";")
	if len(parts) != 2 {
		return "", false
	}

	codepoint, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", false
	}
	mods, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", false
	}

	modBits := mods - 1
	ctrl := modBits&4 != 0
	if ctrl {
		if codepoint == 32 {
			return "c:32", true
		}
		r := unicode.ToLower(rune(codepoint))
		if r >= 'a' && r <= 'z' {
			return "c:" + strconv.Itoa(int(r-'a')+1), true
		}
		switch r {
		case '[':
			return "c:27", true
		case '\\':
			return "c:28", true
		case ']':
			return "c:29", true
		case '^':
			return "c:30", true
		case '_':
			return "c:31", true
		}
	}

	if codepoint > 0 && codepoint <= unicode.MaxRune {
		return string(rune(codepoint)), true
	}

	return "", false
}

func decodeCSITail(seq string) (string, bool) {
	if seq == "" {
		return "", false
	}
	last := seq[len(seq)-1]
	switch last {
	case 'A':
		if seq == "A" || hasCSIMetadataPrefix(seq) {
			return upArrow, true
		}
	case 'B':
		if seq == "B" || hasCSIMetadataPrefix(seq) {
			return downArrow, true
		}
	case 'C':
		if seq == "C" || hasCSIMetadataPrefix(seq) {
			return rightArrow, true
		}
	case 'D':
		if seq == "D" || hasCSIMetadataPrefix(seq) {
			return leftArrow, true
		}
	case 'H':
		return homeKey, true
	case 'F':
		return endKey, true
	case '~':
		param := strings.TrimSuffix(seq, "~")
		switch param {
		case "1", "7":
			return homeKey, true
		case "4", "8":
			return endKey, true
		case "5":
			return pgUpKey, true
		case "6":
			return pgDnKey, true
		case "2;5":
			return copyKey, true
		}
	}
	return "", false
}

func hasCSIMetadataPrefix(seq string) bool {
	seq = strings.TrimSuffix(seq, string(seq[len(seq)-1]))
	if seq == "" {
		return false
	}
	for _, r := range seq {
		if (r < '0' || r > '9') && r != ';' {
			return false
		}
	}
	return true
}

func isCSINumericTail(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && r != ';' {
			return false
		}
	}
	return true
}

func looksLikeCSIPotentialStart(s string) bool {
	if s == "" {
		return false
	}
	if s == "[" || s == "O" {
		return true
	}
	if strings.HasPrefix(s, "c:27") || strings.HasPrefix(s, "\x1b") {
		return true
	}
	// Fragments commonly seen when CSI is split across events.
	if isCSINumericTail(s) {
		return true
	}
	return false
}

func decodeKittyCSIu() (string, bool) {
	const maxTokens = 18
	consumed := make([]string, 0, maxTokens)
	for len(consumed) < maxTokens {
		k, ok := nextMappedKeyTimeout(8 * time.Millisecond)
		if !ok {
			unreadMappedKeysFront(consumed)
			return "", false
		}
		consumed = append(consumed, k)
		if k == "u" {
			break
		}
	}

	seq := strings.Join(consumed, "")
	if !strings.HasSuffix(seq, "u") {
		unreadMappedKeysFront(consumed)
		return "", false
	}
	seq = strings.TrimSuffix(seq, "u")
	if mapped, ok := parseKittyCSIuBody(seq); ok {
		return mapped, true
	}

	unreadMappedKeysFront(consumed)
	return "", false
}

type keyInputTTY interface {
	RawMode()
	Restore()
}

var _ keyInputTTY = (*vt.TTY)(nil)
