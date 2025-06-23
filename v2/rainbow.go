package main

import (
	"errors"
	"github.com/xyproto/vt"
)

var (
	colorSlice = make([]vt.AttributeColor, 0) // to be pushed to and popped from

	errUnmatchedParenthesis = errors.New("unmatched parenthesis")
)

// rainbowParen implements "rainbow parenthesis" which colors "(" and ")" according to how deep they are nested
// pCount is the existing parenthesis count when reaching the start of this line
func (e *Editor) rainbowParen(parCount, braCount *int, chars *[]vt.CharAttribute, singleLineCommentMarker string, ignoreSingleQuotes bool) (err error) {
	var (
		prevPrevRune = '\n'

		// CharAttribute has a rune "R" and a AttributeColor "A"
		nextChar = vt.CharAttribute{R: '\n', A: e.Background}
		prevChar = vt.CharAttribute{R: '\n', A: e.Background}

		// the first color in this slice will normally not be used until the parenthesis are many levels deep,
		// the second one will be used for the regular case which is 1 level deep
		rainbowParenColors = e.Theme.RainbowParenColors

		lastColor = rainbowParenColors[len(rainbowParenColors)-1]
	)

	q, qerr := NewQuoteState(singleLineCommentMarker, e.mode, ignoreSingleQuotes)
	if qerr != nil {
		return qerr
	}

	// Initialize the quote state parenthesis count with the one that is for the beginning of this line, in the current document
	q.parCount = *parCount // parenthesis count
	q.braCount = *braCount // bracket count

	unmatchedParenColor := e.UnmatchedParenColor

	for i, char := range *chars {

		q.ProcessRune(char.R, prevChar.R, prevPrevRune)
		prevPrevRune = prevChar.R

		if !q.None() {
			// Skip comments and strings
			continue
		}

		// Get the next rune and attribute
		if (i + 1) < len(*chars) {
			nextChar.R = (*chars)[i+1].R
			nextChar.A = (*chars)[i+1].A
		}

		// Get the previous rune and attribute
		if i > 0 {
			prevChar.R = (*chars)[i-1].R
			prevChar.A = (*chars)[i-1].A
		}

		// Count parenthesis
		*parCount = q.parCount
		// Count square brackets
		*braCount = q.braCount

		openingP := false // parenthesis
		openingB := false // bracket
		switch char.R {
		case '(':
			openingP = true
		case '[':
			openingB = true
		case ')':
		// openingP is already set to false, for this case
		// openingP = false
		case ']':
			// Don't continue the loop, continue below
		default:
			// Not an opening or closing parenthesis or square bracket
			continue
		}

		if *parCount < 0 || *braCount < 0 {
			// Too many closing parenthesis or brackets!
			char.A = unmatchedParenColor
			err = errUnmatchedParenthesis
		} else if openingB || openingP {
			// Select a color, using modulo
			// Select another color if it's the same as the text that follows
			selected := (*braCount + *parCount) % len(rainbowParenColors)
			char.A = rainbowParenColors[selected]
			// If the character before ( or ) are ' ' or '\t' OR the index is 0, color it with the last color in rainbowParenColors
			if prevChar.R == ' ' || prevChar.R == '\t' || i == 0 {
				char.A = lastColor
			} else {
				// Loop until a color that is not the same as the color of the next character is selected
				// (and the next rune is not blank or end of line)
				for (char.A.Equal(nextChar.A) && nextChar.R != ' ' && nextChar.R != '\t' && nextChar.R != '(' && nextChar.R != ')' && nextChar.R != '[' && nextChar.R != ']') || (char.A.Equal(prevChar.A) && prevChar.R != ' ' && prevChar.R != '\t' && prevChar.R != '(' && prevChar.R != ')' && prevChar.R != '[' && prevChar.R != ']') {
					selected++
					if selected >= len(rainbowParenColors) {
						selected = 0
					}
					char.A = rainbowParenColors[selected]
				}
			}
			// Push the color to the color stack
			colorSlice = append(colorSlice, char.A)
		} else {
			if len(colorSlice) > 0 {
				// pop the color from the color stack
				lastIndex := len(colorSlice) - 1
				char.A = colorSlice[lastIndex]
				colorSlice = colorSlice[:lastIndex]
			} else {
				char.A = lastColor
			}
		}

		// For debugging
		//s := strconv.Itoa(*parCount)
		//if len(s) == 1 {
		//	char.R = []rune(s)[0]
		//} else {
		//	char.R = []rune(s)[1]
		//}

		// Keep the rune, but use the new AttributeColor
		(*chars)[i] = char
	}
	return
}
