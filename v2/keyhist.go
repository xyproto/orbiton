package main

import (
	"slices"
	"sync/atomic"
	"time"
)

const (
	// Keypress combo time limit
	keypressComboTimeLimit = 100 * time.Millisecond

	// Double tap time limit
	doubleTapTimeLimit = 300 * time.Millisecond
)

// keyHistoryState holds an immutable snapshot of the last 3 keypresses.
// Swapped atomically so readers never block writers.
type keyHistoryState struct {
	t    [3]time.Time
	keys [3]string
}

// KeyHistory represents the last 3 keypresses, and when they were pressed.
// Uses atomic pointer swaps instead of mutexes for lock-free access.
type KeyHistory struct {
	state atomic.Pointer[keyHistoryState]
}

// NewKeyHistory creates a new KeyHistory struct
func NewKeyHistory() *KeyHistory {
	kh := &KeyHistory{}
	kh.state.Store(&keyHistoryState{})
	return kh
}

// load returns the current snapshot
func (kh *KeyHistory) load() *keyHistoryState {
	return kh.state.Load()
}

// Push adds another key to the key history,
// at the end of the history.
// The oldest keypress is pushed out.
// The time is also registered.
func (kh *KeyHistory) Push(key string) {
	old := kh.load()
	kh.state.Store(&keyHistoryState{
		keys: [3]string{old.keys[1], old.keys[2], key},
		t:    [3]time.Time{old.t[1], old.t[2], time.Now()},
	})
}

// Prev returns the last pressed key
func (kh *KeyHistory) Prev() string {
	return kh.load().keys[2]
}

// PrevIs checks if the last pressed key is the given string
func (kh *KeyHistory) PrevIs(s string) bool {
	return kh.load().keys[2] == s
}

// PrevPrev returns the key pressed before the last one
func (kh *KeyHistory) PrevPrev() string {
	return kh.load().keys[1]
}

// PrevHas checks if one of the given strings is the previous keypress
func (kh *KeyHistory) PrevHas(keyPresses ...string) bool {
	return slices.Contains(keyPresses, kh.load().keys[2])
}

// PrevIsWithin checks if one of the given strings is the previous keypress, within the given duration
func (kh *KeyHistory) PrevIsWithin(duration time.Duration, keyPresses ...string) bool {
	s := kh.load()
	now := time.Now()
	for _, keyPress := range keyPresses {
		if keyPress == s.keys[2] && now.Sub(s.t[2]) < duration {
			return true
		}
	}
	return false
}

// TwoLastAre checks if the two previous keypresses are the given keypress
func (kh *KeyHistory) TwoLastAre(keyPress string) bool {
	s := kh.load()
	return s.keys[2] == keyPress && s.keys[1] == keyPress
}

// Repeated checks if the given keypress was repeated the N last times
func (kh *KeyHistory) Repeated(keyPress string, n int) bool {
	s := kh.load()
	counter := 0
	for i := len(s.keys) - 1; i >= 0; i-- {
		if s.keys[i] == keyPress {
			counter++
		} else {
			break
		}
	}
	return counter >= n
}

// OnlyIn checks if the key press history only contains the given
// keypresses and no other keypresses.
func (kh *KeyHistory) OnlyIn(keyPresses ...string) bool {
	s := kh.load()
	for _, prevKeyPress := range s.keys {
		if !slices.Contains(keyPresses, prevKeyPress) {
			return false
		}
	}
	return true
}

// OnlyInAndAllDiffer checks if the key press history only contains the given
// keypresses and no other keypresses, and that all are different.
func (kh *KeyHistory) OnlyInAndAllDiffer(keyPresses ...string) bool {
	s := kh.load()
	if s.keys[0] == s.keys[1] || s.keys[0] == s.keys[2] || s.keys[1] == s.keys[2] {
		return false
	}
	for _, prevKeyPress := range s.keys {
		if !slices.Contains(keyPresses, prevKeyPress) {
			return false
		}
	}
	return true
}

// AllWithin checks if the entire key history happened within the given duration (to check for rapid successions)
func (kh *KeyHistory) AllWithin(dur time.Duration) bool {
	s := kh.load()
	return s.t[2].Sub(s.t[0]) < dur
}

// LastChanged checks if a key was added to the key history for longer since than the given duration, or not
func (kh *KeyHistory) LastChanged(dur time.Duration) bool {
	return time.Since(kh.load().t[2]) < dur
}

// PrevWithin checks if the previous keypress happened within the given duration (to check for rapid successions)
func (kh *KeyHistory) PrevWithin(dur time.Duration) bool {
	return time.Since(kh.load().t[2]) < dur
}

// SpecialArrowKeypressWith is like SpecialArrowKeypress, but also considers
// the given extraKeypress as if it was the last one pressed.
func (kh *KeyHistory) SpecialArrowKeypressWith(extraKeypress string) bool {
	old := kh.load()
	// Build a temporary history with the extra keypress shifted in
	tmp := keyHistoryState{
		keys: [3]string{old.keys[1], old.keys[2], extraKeypress},
		t:    [3]time.Time{old.t[1], old.t[2], time.Now()},
	}
	if tmp.keys[0] == tmp.keys[1] || tmp.keys[0] == tmp.keys[2] || tmp.keys[1] == tmp.keys[2] {
		return false
	}
	arrowKeys := []string{upArrow, rightArrow, leftArrow, downArrow}
	for _, k := range tmp.keys {
		if !slices.Contains(arrowKeys, k) {
			return false
		}
	}
	return tmp.t[2].Sub(tmp.t[0]) < keypressComboTimeLimit
}

// DoubleTapped checks if the given key was pressed twice within a short period of time
func (kh *KeyHistory) DoubleTapped(keypress string) bool {
	s := kh.load()
	return s.keys[2] == keypress && time.Since(s.t[2]) < doubleTapTimeLimit
}

// String returns the last keypresses as a string, with the oldest one first
// and the latest one at the end.
func (kh *KeyHistory) String() string {
	s := kh.load()
	return s.keys[0] + s.keys[1] + s.keys[2]
}

// Clear clears the entire history
func (kh *KeyHistory) Clear() {
	kh.state.Store(&keyHistoryState{})
}
