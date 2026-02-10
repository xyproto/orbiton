package vt

import (
	"strconv"
	"strings"
	"unicode"
)

var keyNameMap = map[int]string{
	KeyTab:        "c:9",
	KeyEnter:      "c:13",
	KeyEsc:        "c:27",
	KeyCtrlA:      "c:1",
	KeyCtrlC:      "c:3",
	KeyCtrlD:      "c:4",
	KeyCtrlE:      "c:5",
	KeyCtrlF:      "c:6",
	KeyCtrlH:      "c:8",
	KeyCtrlL:      "c:12",
	KeyCtrlN:      "c:14",
	KeyCtrlP:      "c:16",
	KeyCtrlQ:      "c:17",
	KeyCtrlS:      "c:19",
	KeyBackspace:  "c:127",
	KeyArrowLeft:  "ArrowLeft",
	KeyArrowRight: "ArrowRight",
	KeyArrowUp:    "ArrowUp",
	KeyArrowDown:  "ArrowDown",
	KeyDelete:     "Delete",
	KeyHome:       "Home",
	KeyEnd:        "End",
	KeyPageUp:     "PageUp",
	KeyPageDown:   "PageDown",
	KeyCtrlInsert: "CtrlInsert",
	KeyF1:         "F1",
	KeyF2:         "F2",
	KeyF3:         "F3",
	KeyF4:         "F4",
	KeyF5:         "F5",
	KeyF6:         "F6",
	KeyF7:         "F7",
	KeyF8:         "F8",
	KeyF9:         "F9",
	KeyF10:        "F10",
	KeyF11:        "F11",
	KeyF12:        "F12",
}

var keySymbolMap = map[int]string{
	KeyArrowUp:    "↑",
	KeyArrowDown:  "↓",
	KeyArrowRight: "→",
	KeyArrowLeft:  "←",
	KeyHome:       "⇱",
	KeyEnd:        "⇲",
	KeyPageUp:     "⇞",
	KeyPageDown:   "⇟",
	KeyCtrlInsert: "⎘",
	KeyTab:        "⇥",
	KeyEnter:      "⏎",
}

var nameToKeyMap map[string]int

func init() {
	nameToKeyMap = make(map[string]int, len(keyNameMap))
	for k, v := range keyNameMap {
		nameToKeyMap[v] = k
	}
}

// KeyName returns a canonical name for a key constant.
// For printable ASCII, returns the character itself.
// For control codes and special keys, returns names like "c:9", "ArrowUp", etc.
func KeyName(key int) string {
	if name, ok := keyNameMap[key]; ok {
		return name
	}
	if key == KeySpace {
		return " "
	}
	if key >= 0 && key < 32 {
		return "c:" + strconv.Itoa(key)
	}
	r := rune(key)
	if unicode.IsPrint(r) {
		return string(r)
	}
	return "c:" + strconv.Itoa(key)
}

// KeySymbol returns a Unicode symbol for a key constant.
// For printable ASCII, returns the character itself.
// For special keys like arrows, returns symbols like "↑", "↓", etc.
func KeySymbol(key int) string {
	if sym, ok := keySymbolMap[key]; ok {
		return sym
	}
	if key == KeySpace {
		return " "
	}
	r := rune(key)
	if unicode.IsPrint(r) {
		return string(r)
	}
	return "c:" + strconv.Itoa(key)
}

// KeyFromName performs a reverse lookup from a key name to a key constant.
// Accepts names like "c:9" for KeyTab, "ArrowUp" for KeyArrowUp, etc.
func KeyFromName(name string) int {
	if key, ok := nameToKeyMap[name]; ok {
		return key
	}
	if strings.HasPrefix(name, "c:") {
		if n, err := strconv.Atoi(name[2:]); err == nil {
			return n
		}
	}
	if len(name) == 1 {
		return int(name[0])
	}
	return 0
}

// KeyRune returns a rune for a key constant.
// For special keys like arrows, returns the symbol rune (↑, ↓, etc.).
// For other keys like Esc, returns rune(key) directly.
func KeyRune(key int) rune {
	if sym, ok := keySymbolMap[key]; ok {
		runes := []rune(sym)
		if len(runes) > 0 {
			return runes[0]
		}
	}
	return rune(key)
}
