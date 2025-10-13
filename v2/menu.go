package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/xyproto/vt"
)

const konami = "↑↑↓↓←→←→ba"

var (
	errNoLetter = errors.New("no letter")
	smallWords  = []string{"a", "and", "at", "by", "in", "is", "let", "of", "or", "the", "to"}
)

// RuneAndPosition is used to store a rune and a position within a string
type RuneAndPosition struct {
	r   rune
	pos uint
}

// getLetter returns the Nth letter in a given string, as lowercase. Ignores numbers, special characters, whitespace etc.
func getLetter(s string, pos int) (rune, error) {
	counter := 0
	for _, letter := range s {
		if unicode.IsLetter(letter) {
			if counter == pos {
				return unicode.ToLower(letter), nil
			}
			counter++
		}
	}
	return rune(0), errNoLetter
}

// Check if the position pos points to a letter in a small word like "to", "at" or "the".
func inSmallWordOrSpace(s string, pos int) bool {
	var (
		stopAtNextSpace bool
		wordRunes       []rune
	)
	for i, r := range s {
		if i == pos {
			if r == ' ' {
				// The given position is at a space
				return true
			}
			// We have reached the target position, continue collecting letters of this word until the next space
			stopAtNextSpace = true
		}
		if r == ' ' {
			if stopAtNextSpace {
				// Done collecting the current word
				break
			}
			// Start collecting the next word
			wordRunes = []rune{}
		} else {
			wordRunes = append(wordRunes, r)
		}
	}
	word := strings.ToLower(string(wordRunes))
	for _, smallWord := range smallWords {
		if word == smallWord {
			return true
		}
	}
	return false
}

// selectionLetterForChoices loops through the choices and finds a unique letter (possibly the first one in the choice text),
// that can be used later for selecting that choice. The returned map maps from the letter to the choice index.
// choices where no appropriate letter were found are skipped. q is ignored, because it's reserved for "quit".
// Only the first N letters of each choice are considered.
func selectionLettersForChoices(choices []string) map[string]*RuneAndPosition {
	// TODO: Find the next item starting with this letter, with wraparound
	// Select the item that starts with this letter, if possible. Try the first, then the second, etc, up to N
	N := 16
	selectionLetterMap := make(map[string]*RuneAndPosition)
	for choiceIndex, choiceString := range choices {
		for pos := 0; pos < N; pos++ {
			if pos >= len(choiceString) {
				break
			}
			letter, err := getLetter(choiceString, pos)
			if err != nil {
				continue
			}
			// Check if this letter is already picked
			exists := false
			for _, v := range selectionLetterMap {
				if v.r == letter {
					exists = true
					break
				}
			}
			// If the letter is not already stored in the keymap, and it's not q,
			// and it's not in a small word like "at" or "to"
			if !exists && (letter != 'q') && !inSmallWordOrSpace(choiceString, pos) {
				// fmt.Printf("Using %s [%d] for %s\n", string(letter), index, choice)
				// Use this letter!
				selectionLetterMap[choiceString] = &RuneAndPosition{letter, uint(choiceIndex)}
				// Found a letter for this choice, move on
				break
			}
		}
		// Did not find a letter for this choice, move on
	}
	return selectionLetterMap
}

func ctrlkey2letter(key string) (string, bool) {
	if !strings.HasPrefix(key, "c:") || len(key) < 3 {
		return "", false
	}
	number, err := strconv.Atoi(key[2:])
	if err != nil {
		return "", false
	}
	var r rune = rune((int('a') - 1) + number)
	if r > 'z' {
		return "", false
	}
	return string(r), true
}

// Menu starts a loop where keypresses are handled. When a choice is made, a number is returned.
// -1 is "no choice", 0 and up is which choice were selected.
// initialMenuIndex is the choice that should be highlighted when displaying the choices.
// returns -1, true if space was pressed
func (e *Editor) Menu(status *StatusBar, tty *vt.TTY, title string, choices []string, bgColor, titleColor, arrowColor, textColor, highlightColor, selectedColor vt.AttributeColor, initialMenuIndex int, extraDashes bool) (int, bool) {
	notRegularEditingRightNow.Store(true)
	defer func() {
		notRegularEditingRightNow.Store(false)
	}()

	// Clear the existing handler
	resetResizeSignal()

	var (
		selectionLetterMap = selectionLettersForChoices(choices)
		selectedDelay      = 100 * time.Millisecond
		c                  = vt.NewCanvas()
		menu               = NewMenuWidget(title, choices, titleColor, arrowColor, textColor, highlightColor, selectedColor, c.W(), c.H(), extraDashes, selectionLetterMap)
		sigChan            = make(chan os.Signal, 1)
		running            = true
		changed            = true
	)

	// Set up a new resize handler
	setupResizeSignal(sigChan)

	ctx, cancelFunc := context.WithCancel(context.Background())

	// Cleanup function to be called on function exit
	defer func() {
		cancelFunc()
		resetResizeSignal()
	}()

	go func() {
		for {
			select {
			case <-sigChan:
				resizeMut.Lock()
				nc := c.Resized()
				if nc != nil {
					c.Clear()
					vt.Clear()
					c = nc
					menu.Draw(c)
					c.HideCursorAndRedraw()
					changed = true
				}
				resizeMut.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	vt.Clear()
	vt.Reset()
	c.FillBackground(bgColor)
	c.HideCursorAndRedraw()

	// Set the initial menu index
	menu.SelectIndex(uint(initialMenuIndex))

	var collectedString string

	for running {

		// Draw elements in their new positions

		if changed {
			resizeMut.RLock()
			menu.Draw(c)
			resizeMut.RUnlock()

			// Update the canvas
			c.HideCursorAndDraw()
		}

		// Handle events
		key := tty.String()
		switch key {
		case upArrow, leftArrow, "c:16": // Up, left or ctrl-p
			resizeMut.Lock()
			menu.Up()
			changed = true
			resizeMut.Unlock()
		case downArrow, rightArrow, "c:14": // Down, right or ctrl-n
			resizeMut.Lock()
			menu.Down()
			changed = true
			resizeMut.Unlock()
		case "c:1": // Top, ctrl-a
			resizeMut.Lock()
			menu.SelectFirst()
			changed = true
			resizeMut.Unlock()
		case "c:5": // Bottom, ctrl-e
			resizeMut.Lock()
			menu.SelectLast()
			changed = true
			resizeMut.Unlock()
		case "c:27", "q", "c:17", "c:15": // ESC, q, ctrl-q or ctrl-o
			running = false
			changed = true
		case " ", "c:0": // space or ctrl-space
			return -1, true
		case "c:13": // return
			resizeMut.Lock()
			menu.Select()
			resizeMut.Unlock()
			running = false
			changed = true
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9": // 0 .. 9
			number, err := strconv.Atoi(key)
			if err != nil {
				break
			}
			resizeMut.Lock()
			menu.SelectIndex(uint(number))
			changed = true
			resizeMut.Unlock()
		default:
			if letter, ok := ctrlkey2letter(key); ok {
				key = letter
			}
			if len([]rune(key)) == 0 {
				// this happens if pgup or pgdn is pressed
				break
			}
			// Check if the key matches the first letter (A-Z, a-z) in the choices
			r := []rune(key)[0]
			if !(65 <= r && r <= 90) && !(97 <= r && r <= 122) {
				break
			}
			// Choose the index for the letter that was pressed and found in the keymap, if found
			for _, runeAndPosition := range selectionLetterMap {
				if r == runeAndPosition.r {
					resizeMut.Lock()
					menu.SelectIndex(runeAndPosition.pos)
					changed = true
					resizeMut.Unlock()
				}
			}
		}

		// Konami code collector
		if len([]rune(key)) == 1 {
			collectedString += key
			// pop a letter in front of the collected string if it's too long
			if len(collectedString) > len(konami) {
				runes := []rune(collectedString)
				collectedString = string(runes[1:])
			}
		}
		// Was it the konami code?
		if collectedString == konami {
			collectedString = ""
			// Start the game
			if ctrlq, err := Game(); err != nil {
				// This should never happen
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			} else if ctrlq {
				// ctrl-q was pressed, quit entirely
				running = false
				e.quit = true
			} else {
				// The game ended, return from the menu
				running = false
				changed = true
			}
		}

		// If the menu was changed, draw the canvas
		if changed {
			c.HideCursorAndDraw()
		}

	}

	if menu.Selected() >= 0 {
		// Draw the selected item in a different color for a very short while
		resizeMut.Lock()
		menu.SelectDraw(c)
		resizeMut.Unlock()
		c.HideCursorAndDraw()
		time.Sleep(selectedDelay)
	}

	// Restore the resize handler
	e.SetUpSignalHandlers(c, tty, status, false) // do not only clear the signals

	return menu.Selected(), false
}
