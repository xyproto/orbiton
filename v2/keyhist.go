package main

import "time"

// Keypress combo time limit
const keypressComboTimeLimit = 100 * time.Millisecond

// Double tap time limit
const doubleTapTimeLimit = 300 * time.Millisecond

// KeyHistory represents the last 3 keypresses, and when they were pressed
type KeyHistory struct {
	t    [3]time.Time
	keys [3]string
}

// NewKeyHistory creates a new KeyHistory struct
func NewKeyHistory() *KeyHistory {
	return &KeyHistory{}
}

// Push adds another key to the key history,
// at the end of the history.
// The oldest keypress is pushed out.
// The time is also registered.
func (kh *KeyHistory) Push(key string) {
	kh.keys[0] = kh.keys[1]
	kh.t[0] = kh.t[1]
	kh.keys[1] = kh.keys[2]
	kh.t[1] = kh.t[2]
	kh.keys[2] = key
	kh.t[2] = time.Now()
}

// Prev returns the last pressed key
func (kh *KeyHistory) Prev() string {
	return kh.keys[2]
}

// PrevPrev returns the key pressed before the last one
func (kh *KeyHistory) PrevPrev() string {
	return kh.keys[1]
}

// PrevIs checks if one of the given strings is the previous keypress
func (kh *KeyHistory) PrevIs(keyPresses ...string) bool {
	for _, keyPress := range keyPresses {
		if keyPress == kh.keys[2] {
			return true
		}
	}
	return false
}

// TwoLastAre checks if the two previous keypresses are the given keypress
func (kh *KeyHistory) TwoLastAre(keyPress string) bool {
	return kh.Prev() == keyPress && kh.PrevPrev() == keyPress
}

// Repeated checks if the given keypress was repeated the N last times
func (kh *KeyHistory) Repeated(keyPress string, n int) bool {
	counter := 0
	for i := len(kh.keys) - 1; i >= 0; i-- {
		if kh.keys[i] == keyPress {
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
	var found bool
	for _, prevKeyPress := range kh.keys {
		found = false
		for _, keyPress := range keyPresses {
			if prevKeyPress == keyPress {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// OnlyInAndAllDiffer checks if the key press history only contains the given
// keypresses and no other keypresses, and that all are different.
func (kh *KeyHistory) OnlyInAndAllDiffer(keyPresses ...string) bool {
	allDiffer := kh.keys[0] != kh.keys[1] && kh.keys[0] != kh.keys[2] && kh.keys[1] != kh.keys[2]
	return kh.OnlyIn(keyPresses...) && allDiffer
}

// AllWithin checks if the entire key history happened within the given duration (to check for rapid successions)
func (kh *KeyHistory) AllWithin(dur time.Duration) bool {
	firstTime := kh.t[0]
	lastTime := kh.t[2]
	return lastTime.Sub(firstTime) < dur
}

// PrevWithin checks if the previous keypress happened within the given duration (to check for rapid successions)
func (kh *KeyHistory) PrevWithin(dur time.Duration) bool {
	prevTime := kh.t[2]
	return time.Since(prevTime) < dur
}

// SpecialArrowKeypressWith is like SpecialArrowKeypress, but also considers
// the given extraKeypress as if it was the last one pressed.
func (kh *KeyHistory) SpecialArrowKeypressWith(extraKeypress string) bool {
	// Push the extra keypress temporarily
	khb := *kh
	kh.Push(extraKeypress)
	defer func() {
		*kh = khb
	}()
	// Check if the special keypress was pressed (3 arrow keys in a row, any arrow key goes)
	return kh.OnlyInAndAllDiffer("↑", "→", "←", "↓") && kh.AllWithin(keypressComboTimeLimit)
}

// DoubleTapped checks if the given key was pressed twice within a short period of time
func (kh *KeyHistory) DoubleTapped(keypress string) bool {
	// Check if the previous keypress was the same as this one and within the time limit for double taps
	return kh.Prev() == keypress && kh.PrevWithin(doubleTapTimeLimit)
}

// String returns the last keypresses as a string, with the oldest one first
// and the latest one at the end.
func (kh *KeyHistory) String() string {
	return kh.keys[0] + kh.keys[1] + kh.keys[2]
}

// Clear clears the entire history
func (kh *KeyHistory) Clear() {
	kh.keys[0] = ""
	kh.keys[1] = ""
	kh.keys[2] = ""
}
