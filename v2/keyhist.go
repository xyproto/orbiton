package main

import (
	"strings"
	"time"
)

// Keypress combo time limit
const keypressComboTimeLimit = 120 * time.Millisecond

// KeyHistory represents the last 5 keypresses, and when they were pressed
type KeyHistory struct {
	t    [5]time.Time
	keys [5]string
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
	l := len(kh.keys)
	for i := 0; i < (l - 1); i++ {
		kh.keys[i] = kh.keys[i+1]
		kh.t[i] = kh.t[i+1]
	}
	kh.keys[l-1] = key
	kh.t[l-1] = time.Now()
}

// Prev returns the last pressed key
func (kh *KeyHistory) Prev() string {
	return kh.keys[len(kh.keys)-1]
}

// PrevPrev returns the key pressed before the last one
func (kh *KeyHistory) PrevPrev() string {
	return kh.keys[len(kh.keys)-2]
}

// PrevPrevPrev returns the key pressed before the one before the last one
func (kh *KeyHistory) PrevPrevPrev() string {
	return kh.keys[len(kh.keys)-3]
}

// PrevIsNot checks that the given keypress is not the previous one
func (kh *KeyHistory) PrevIsNot(keyPress string) bool {
	return keyPress != kh.keys[len(kh.keys)-1]
}

// ClearLast clears the previous keypress
// (which is also the last keypress in the history)
func (kh *KeyHistory) ClearLast() {
	kh.keys[len(kh.keys)-1] = ""
}

// SetLast modifies the previous keypress,
// (which is also the last keypress in the history)
func (kh *KeyHistory) SetLast(keyPress string) {
	kh.keys[len(kh.keys)-1] = keyPress
}

// PrevIs checks if one of the given strings is the previous keypress
func (kh *KeyHistory) PrevIs(keyPresses ...string) bool {
	l := len(kh.keys)
	for _, keyPress := range keyPresses {
		if keyPress == kh.keys[l-1] {
			return true
		}
	}
	return false
}

// PrevPrevIs checks if the one before the previous keypress is the given one
func (kh *KeyHistory) PrevPrevIs(keyPresses ...string) bool {
	l := len(kh.keys)
	for _, keyPress := range keyPresses {
		if keyPress == kh.keys[l-2] {
			return true
		}
	}
	return false
}

// PrevPrevPrevIs checks if the one before the previous keypress is the given one
func (kh *KeyHistory) PrevPrevPrevIs(keyPresses ...string) bool {
	l := len(kh.keys)
	for _, keyPress := range keyPresses {
		if keyPress == kh.keys[l-3] {
			return true
		}
	}
	return false
}

// Only checks if the key press history only contains the given keypress
func (kh *KeyHistory) Only(keyPress string, n int) bool {
	firstIndex := (len(kh.keys) - 1) - n
	for _, prevKeyPress := range kh.keys[firstIndex:] {
		if prevKeyPress != keyPress {
			return false
		}
	}
	return true
}

// Repeated checks if the given keypress was repeated the N last times
func (kh *KeyHistory) Repeated(keyPress string, n int) bool {
	firstIndex := (len(kh.keys) - 1) - n
	counter := 0
	for i := len(kh.keys) - 1; i >= firstIndex; i-- {
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
func (kh *KeyHistory) OnlyIn(n int, keyPresses ...string) bool {
	firstIndex := (len(kh.keys) - 1) - n
	var found bool
	for _, prevKeyPress := range kh.keys[firstIndex:] {
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
// Only considers the n last keypresses.
func (kh *KeyHistory) OnlyInAndAllDiffer(n int, keyPresses ...string) bool {
	l := len(kh.keys)
	firstIndex := (l - 1) - n
	if 0 > firstIndex || firstIndex >= l {
		firstIndex = 0
	}
	for i := firstIndex; i < l; i++ {
		for j := firstIndex; j < l; j++ {
			if i == j {
				continue
			}
			if kh.keys[i] == kh.keys[j] {
				// found two equal keys
				return false
			}
		}
	}
	return kh.OnlyIn(n, keyPresses...)
}

// AllWithin checks if the entire key history happened within the given duration (to check for rapid successions)
func (kh *KeyHistory) AllWithin(dur time.Duration, n int) bool {
	firstTime := kh.t[0]
	lastIndex := len(kh.keys) - 1
	if 0 <= n && n < lastIndex {
		lastIndex = n
	}
	lastTime := kh.t[lastIndex]
	return lastTime.Sub(firstTime) < dur
}

// SpecialArrowKeypress checks if the last 3 keypresses are all different arrow keys,
// like for instance left, up, right or left, down right, but not left, left, left.
// Also, the keypresses must happen within a fixed amount of time, so that only rapid
// successions are registered.
func (kh *KeyHistory) SpecialArrowKeypress() bool {
	return kh.OnlyInAndAllDiffer(3, "↑", "→", "←", "↓") && kh.AllWithin(keypressComboTimeLimit, 3)
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
	return kh.OnlyInAndAllDiffer(3, "↑", "→", "←", "↓") && kh.AllWithin(keypressComboTimeLimit, 3)
}

// String returns the last keypresses as a string, with the oldest one first
// and the latest one at the end.
func (kh *KeyHistory) String() string {
	var sb strings.Builder
	for i := 0; i < len(kh.keys); i++ {
		sb.WriteString(kh.keys[i])
	}
	return sb.String()
}

// Clear clears the entire history
func (kh *KeyHistory) Clear() {
	for i := 0; i < len(kh.keys); i++ {
		kh.keys[i] = ""
	}
}
