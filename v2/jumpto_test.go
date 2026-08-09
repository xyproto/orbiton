package main

import "testing"

// If only "W" is highlighted on screen, pressing "w" should jump to it,
// but "w" must still be available as a jump letter of its own.
func TestJumpLetterCaseFallback(t *testing.T) {
	e := &Editor{}
	e.ClearJumpLetters()
	defer e.ClearJumpLetters()

	if !e.RegisterJumpLetter('W', 3, 7) {
		t.Fatal("could not register W")
	}
	if !e.CanJumpTo('w') {
		t.Error("w should be able to jump to the registered W")
	}
	if x, y := e.GetJumpX('w'), e.GetJumpY('w'); x != 3 || y != 7 {
		t.Errorf("w jumped to %d,%d, expected 3,7", x, y)
	}
	if e.HasJumpLetter('w') {
		t.Error("HasJumpLetter should stay exact, so that w can still be registered on its own")
	}

	// Once "w" is registered on its own, each letter keeps its own position
	if !e.RegisterJumpLetter('w', 11, 12) {
		t.Fatal("could not register w")
	}
	if x, y := e.GetJumpX('w'), e.GetJumpY('w'); x != 11 || y != 12 {
		t.Errorf("w jumped to %d,%d, expected 11,12", x, y)
	}
	if x, y := e.GetJumpX('W'), e.GetJumpY('W'); x != 3 || y != 7 {
		t.Errorf("W jumped to %d,%d, expected 3,7", x, y)
	}

	e.ClearJumpLetters()
	if e.CanJumpTo('w') || e.CanJumpTo('W') {
		t.Error("the jump letters should have been cleared")
	}
}
