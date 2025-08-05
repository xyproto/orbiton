package main

import (
	"sync"
	"time"
)

const (
	// Keypress combo time limit
	keypressComboTimeLimit = 100 * time.Millisecond

	// Double tap time limit
	doubleTapTimeLimit = 300 * time.Millisecond
)

// KeyHistory represents the last 3 keypresses, and when they were pressed
type KeyHistory struct {
	mu   sync.RWMutex // instance-level mutex instead of global
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
	kh.mu.Lock()
	defer kh.mu.Unlock()

	kh.keys[0] = kh.keys[1]
	kh.t[0] = kh.t[1]
	kh.keys[1] = kh.keys[2]
	kh.t[1] = kh.t[2]
	kh.keys[2] = key
	kh.t[2] = time.Now()
}

// Prev returns the last pressed key
func (kh *KeyHistory) Prev() string {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	return kh.keys[2]
}

// PrevIs checks if the last pressed key is the given string
func (kh *KeyHistory) PrevIs(s string) bool {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	return kh.keys[2] == s
}

// PrevPrev returns the key pressed before the last one
func (kh *KeyHistory) PrevPrev() string {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	return kh.keys[1]
}

// PrevHas checks if one of the given strings is the previous keypress
func (kh *KeyHistory) PrevHas(keyPresses ...string) bool {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	for _, keyPress := range keyPresses {
		if keyPress == kh.keys[2] {
			return true
		}
	}
	return false
}

// PrevIsWithin checks if one of the given strings is the previous keypress, within the given duration
func (kh *KeyHistory) PrevIsWithin(duration time.Duration, keyPresses ...string) bool {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	now := time.Now()
	for _, keyPress := range keyPresses {
		if keyPress == kh.keys[2] && now.Sub(kh.t[2]) < duration {
			return true
		}
	}
	return false
}

// TwoLastAre checks if the two previous keypresses are the given keypress
func (kh *KeyHistory) TwoLastAre(keyPress string) bool {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	return kh.Prev() == keyPress && kh.PrevPrev() == keyPress
}

// Repeated checks if the given keypress was repeated the N last times
func (kh *KeyHistory) Repeated(keyPress string, n int) bool {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

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
	kh.mu.RLock()
	defer kh.mu.RUnlock()

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
	kh.mu.RLock()
	allDiffer := kh.keys[0] != kh.keys[1] && kh.keys[0] != kh.keys[2] && kh.keys[1] != kh.keys[2]
	kh.mu.RUnlock()
	return kh.OnlyIn(keyPresses...) && allDiffer
}

// AllWithin checks if the entire key history happened within the given duration (to check for rapid successions)
func (kh *KeyHistory) AllWithin(dur time.Duration) bool {
	kh.mu.RLock()
	defer kh.mu.RUnlock()
	firstTime := kh.t[0]
	lastTime := kh.t[2]
	return lastTime.Sub(firstTime) < dur
}

// LastChanged checks if a key was added to the key history for longer since than the given duration, or not
func (kh *KeyHistory) LastChanged(dur time.Duration) bool {
	kh.mu.RLock()
	defer kh.mu.RUnlock()
	lastTime := kh.t[2]
	return time.Since(lastTime) < dur
}

// PrevWithin checks if the previous keypress happened within the given duration (to check for rapid successions)
func (kh *KeyHistory) PrevWithin(dur time.Duration) bool {
	kh.mu.RLock()
	defer kh.mu.RUnlock()
	prevTime := kh.t[2]
	return time.Since(prevTime) < dur
}

// SpecialArrowKeypressWith is like SpecialArrowKeypress, but also considers
// the given extraKeypress as if it was the last one pressed.
func (kh *KeyHistory) SpecialArrowKeypressWith(extraKeypress string) bool {
	// Push the extra keypress temporarily
	kh.mu.RLock()
	khb := *kh
	kh.mu.RUnlock()
	kh.Push(extraKeypress)
	defer func() {
		kh.mu.Lock()
		*kh = khb
		kh.mu.Unlock()
	}()
	// Check if the special keypress was pressed (3 arrow keys in a row, any arrow key goes)
	return kh.OnlyInAndAllDiffer(upArrow, rightArrow, leftArrow, downArrow) && kh.AllWithin(keypressComboTimeLimit)
}

// DoubleTapped checks if the given key was pressed twice within a short period of time
func (kh *KeyHistory) DoubleTapped(keypress string) bool {
	// Check if the previous keypress was the same as this one and within the time limit for double taps
	return kh.Prev() == keypress && kh.PrevWithin(doubleTapTimeLimit)
}

// String returns the last keypresses as a string, with the oldest one first
// and the latest one at the end.
func (kh *KeyHistory) String() string {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	return kh.keys[0] + kh.keys[1] + kh.keys[2]
}

// Clear clears the entire history
func (kh *KeyHistory) Clear() {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	kh.keys[0] = ""
	kh.keys[1] = ""
	kh.keys[2] = ""
}
