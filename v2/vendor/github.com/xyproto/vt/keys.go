package vt

// Key constants
const (
	KeyCtrlA     = 1
	KeyCtrlC     = 3
	KeyCtrlD     = 4
	KeyCtrlE     = 5
	KeyCtrlF     = 6
	KeyCtrlH     = 8
	KeyTab       = 9
	KeyCtrlL     = 12
	KeyEnter     = 13
	KeyCtrlN     = 14
	KeyCtrlP     = 16
	KeyCtrlQ     = 17
	KeyCtrlS     = 19
	KeyEsc       = 27
	KeySpace     = 32
	KeyBackspace = 127

	KeyArrowLeft  = 1000
	KeyArrowRight = 1001
	KeyArrowUp    = 1002
	KeyArrowDown  = 1003
	KeyDelete     = 1004
	KeyHome       = 1005
	KeyEnd        = 1006
	KeyPageUp     = 1007
	KeyPageDown   = 1008

	// Function keys
	KeyF1  = 1009
	KeyF2  = 1010
	KeyF3  = 1011
	KeyF4  = 1012
	KeyF5  = 1013
	KeyF6  = 1014
	KeyF7  = 1015
	KeyF8  = 1016
	KeyF9  = 1017
	KeyF10 = 1018
	KeyF11 = 1019
	KeyF12 = 1020

	KeyCtrlInsert = 1021
)

// Modifiers
const (
	ModNone  = 0
	ModCtrl  = 1 << 0
	ModAlt   = 1 << 1
	ModShift = 1 << 2
)

// KeyEvent represents a keyboard event
type KeyEvent struct {
	Name      string // Human readable name for debugging
	Key       int    // One of the key constants or a rune value
	Modifiers int    // Bitmask of modifiers
	Rune      rune   // The actual rune if applicable
}

type EventKind int

const (
	EventNone EventKind = iota
	EventKey
	EventRune
	EventText
	EventPaste
)

type Event struct {
	Text string
	Kind EventKind
	Key  int
	Rune rune
}
