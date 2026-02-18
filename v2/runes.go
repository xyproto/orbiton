package main

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
)

// Check if the given string only consists of the given rune,
// ignoring the other given runes.
func consistsOf(s string, e rune, ignore []rune) bool {
OUTER_LOOP:
	for _, r := range s {
		for _, x := range ignore {
			if r == x {
				continue OUTER_LOOP
			}
		}
		if r != e {
			return false
		}
	}
	return true
}

// hexDigit checks if the given rune is 0-9, a-f, A-F or x
func hexDigit(r rune) bool {
	switch r {
	case 'x', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'a', 'B', 'b', 'C', 'c', 'D', 'd', 'E', 'e', 'F', 'f':
		return true
	}
	return false
}

// runeCount counts the instances of r in the given string
func runeCount(s string, r rune) int {
	counter := 0
	for _, e := range s {
		if e == r {
			counter++
		}
	}
	return counter
}

// runeFromUBytes returns a rune from a byte slice on the form "U+0000"
func runeFromUBytes(bs []byte) (rune, error) {
	if !bytes.HasPrefix(bs, []byte("U+")) && !bytes.HasPrefix(bs, []byte("u+")) {
		return rune(0), errors.New("not a rune on the form U+0000 or u+0000")
	}
	numberString := string(bs[2:])
	unicodeNumber, err := strconv.ParseUint(numberString, 16, 64)
	if err != nil {
		return rune(0), err
	}
	return rune(unicodeNumber), nil
}

// repeatRune can repeat a rune, n number of times.
// Returns an empty string if memory cannot be allocated within append.
func repeatRune(r rune, n uint) string {
	var sb strings.Builder
	for range n {
		_, err := sb.WriteRune(r)
		if err != nil {
			// In the unlikely event that append inside WriteRune won't work
			return ""
		}
	}
	return sb.String()
}
