package main

import "time"

// Keypress combo time limit
const keypressComboTimeLimit = 120 * time.Millisecond

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

// PrevPrevPrev returns the key pressed before the one before the last one
func (kh *KeyHistory) PrevPrevPrev() string {
	return kh.keys[2]
}

// PrevIsNot checks that the given keypress is not the previous one
func (kh *KeyHistory) PrevIsNot(keyPress string) bool {
	return keyPress != kh.keys[2]
}

// ClearLast clears the previous (and last) keypress in the history
func (kh *KeyHistory) ClearLast() {
	kh.keys[2] = ""
}

// SetLast modifies the previous (and last) keypress in the history
func (kh *KeyHistory) SetLast(keyPress string) {
	kh.keys[2] = keyPress
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

// PrevPrevIs checks if the one before the previous keypress is the given one
func (kh *KeyHistory) PrevPrevIs(keyPresses ...string) bool {
	for _, keyPress := range keyPresses {
		if keyPress == kh.keys[1] {
			return true
		}
	}
	return false
}

// PrevPrevPrevIs checks if the one before the previous keypress is the given one
func (kh *KeyHistory) PrevPrevPrevIs(keyPresses ...string) bool {
	for _, keyPress := range keyPresses {
		if keyPress == kh.keys[0] {
			return true
		}
	}
	return false
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

// SpecialArrowKeypress checks if the last 3 keypresses are all different arrow keys,
// like for instance left, up, right or left, down right, but not left, left, left.
// Also, the keypresses must happen within a fixed amount of time, so that only rapid
// successions are registered.
func (kh *KeyHistory) SpecialArrowKeypress() bool {
	return kh.OnlyInAndAllDiffer("↑", "→", "←", "↓") && kh.AllWithin(keypressComboTimeLimit)
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
