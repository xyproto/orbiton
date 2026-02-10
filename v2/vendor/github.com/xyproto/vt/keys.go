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

	// Arrow keys (legacy codes for backwards compatibility)
	KeyArrowLeft  = 252
	KeyArrowRight = 254
	KeyArrowUp    = 253
	KeyArrowDown  = 255

	// Navigation keys
	KeyHome     = 1
	KeyEnd      = 5
	KeyPageUp   = 251
	KeyPageDown = 250
	KeyDelete   = 127

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

	KeyCtrlInsert = 258
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
