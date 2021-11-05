package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/xyproto/vt100"
)

const konami = "↑↑↓↓←→←→ba"

var (
	errNoLetter = errors.New("no letter")
	smallWords  = []string{"a", "and", "at", "by", "in", "is", "let", "of", "or", "the", "to"}
)

// StringAndPosition is used to store a string and a position within that string
type StringAndPosition struct {
	s   string
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
// Only the first 10 letters of each choice are considered.
func selectionLettersForChoices(choices []string) map[rune]*StringAndPosition {
	// TODO: Find the next item starting with this letter, with wraparound
	// Select the item that starts with this letter, if possible. Try the first, then the second, etc, up to 10
	selectionLetterMap := make(map[rune]*StringAndPosition)
	for index, choice := range choices {
		for pos := 0; pos < 10; pos++ {
			letter, err := getLetter(choice, pos)
			if err == nil {
				_, exists := selectionLetterMap[letter]
				// If the letter is not already stored in the keymap, and it's not q,
				// and it's not in a small word like "at" or "to"
				if !exists && (letter != 'q') && !inSmallWordOrSpace(choice, index) {
					//fmt.Printf("Using %s [%d] for %s\n", string(letter), index, choice)
					// Use this letter!
					selectionLetterMap[letter] = &StringAndPosition{choice, uint(index)}
					// Found a letter for this choice, move on
					break
				}
			}
		}
		// Did not find a letter for this choice, move on
	}
	return selectionLetterMap
}

// Menu starts a loop where keypresses are handled. When a choice is made, a number is returned.
// -1 is "no choice", 0 and up is which choice were selected.
// initialMenuIndex is the choice that should be highlighted when displaying the choices.
func (e *Editor) Menu(status *StatusBar, tty *vt100.TTY, title string, choices []string, titleColor, arrowColor, textColor, highlightColor, selectedColor vt100.AttributeColor, initialMenuIndex int, extraDashes bool) int {

	// Clear the existing handler
	signal.Reset(syscall.SIGWINCH)

	var (
		selectionLetterMap = selectionLettersForChoices(choices)
		selectedDelay      = 100 * time.Millisecond
		c                  = vt100.NewCanvas()
		resizeMut          = &sync.RWMutex{} // used when the terminal is resized
		menu               = NewMenuWidget(title, choices, titleColor, arrowColor, textColor, highlightColor, selectedColor, c.W(), c.H(), extraDashes, selectionLetterMap)
		sigChan            = make(chan os.Signal, 1)
		running            = true
		changed            = true
	)

	// Set up a new resize handler
	signal.Notify(sigChan, syscall.SIGWINCH)

	go func() {
		for range sigChan {
			resizeMut.Lock()
			// Create a new canvas, with the new size
			nc := c.Resized()
			if nc != nil {
				vt100.Clear()
				c = nc
				menu.Draw(c)
				c.Redraw()
				changed = true
			}

			// Inform all elements that the terminal was resized
			resizeMut.Unlock()
		}
	}()

	vt100.Clear()
	vt100.Reset()
	c.Redraw()

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
			c.Draw()
		}

		// Handle events
		key := tty.String()
		switch key {
		case "↑", "←", "c:16": // Up, left or ctrl-p
			resizeMut.Lock()
			menu.Up(c)
			changed = true
			resizeMut.Unlock()
		case "↓", "→", "c:14": // Down, right or ctrl-n
			resizeMut.Lock()
			menu.Down(c)
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
		case "c:27", "q", "c:3", "c:17", "c:15": // ESC, q, ctrl-c, ctrl-q or ctrl-o
			running = false
			changed = true
		case " ", "c:13": // Space or Return
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
			for letter, stringAndPosition := range selectionLetterMap {
				if letter == r {
					resizeMut.Lock()
					menu.SelectIndex(stringAndPosition.pos)
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
			c.Draw()
		}

	}

	if menu.Selected() >= 0 {
		// Draw the selected item in a different color for a very short while
		resizeMut.Lock()
		menu.SelectDraw(c)
		resizeMut.Unlock()
		c.Draw()
		time.Sleep(selectedDelay)
	}

	// Clear the existing handler
	signal.Reset(syscall.SIGWINCH)

	// Restore the resize handler
	e.SetUpResizeHandler(c, tty, status)

	return menu.Selected()
}
