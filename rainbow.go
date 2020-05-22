package main

import (
	"github.com/xyproto/textoutput"
	"github.com/xyproto/vt100"
)

var (
	colorSlice = make([]vt100.AttributeColor, 0) // to be pushed to and popped from

	// the first color in this slice will normally not be used until the paranthesis are many levels deep,
	// the second one will be used for the regular case which is 1 level deep
	rainbowParenColors = []vt100.AttributeColor{vt100.LightMagenta, vt100.LightRed, vt100.Yellow, vt100.LightYellow, vt100.LightGreen, vt100.Green, vt100.LightBlue}
	parenErrorColor    = vt100.White // this color is meant to stand out, for unbalanced parenthesis
)

// rainbowParen implements "rainbow parenthesis" which colors "(" and ")" according to how deep they are nested
// pCount is the existing parenthesis count when reaching the start of this line
func rainbowParen(pCount *int, chars *[]textoutput.CharAttribute, singleLineCommentMarker string, ignoreSingleQuotes bool) {
	var (
		q            = NewQuoteState(singleLineCommentMarker) // TODO: Use the proper single line comment
		prevPrevRune = '\n'

		// CharAttribute has a rune "R" and a vt100.AttributeColor "A"
		nextChar = textoutput.CharAttribute{'\n', defaultEditorBackground}
		prevChar = textoutput.CharAttribute{'\n', defaultEditorBackground}

		lastIndex = len(*chars) - 1
		lastColor = rainbowParenColors[len(rainbowParenColors)-1]
	)

	for i, char := range *chars {
		q.ProcessRune(char.R, prevChar.R, prevPrevRune, ignoreSingleQuotes)
		prevPrevRune = prevChar.R

		if !q.None() {
			// Skip comments and strings
			continue
		}

		// TODO: Just use nextChar.A and nextChar.R instead of having nextColor and nextRune

		if (i + 1) < len(*chars) {
			nextChar.R = (*chars)[i+1].R
			nextChar.A = (*chars)[i+1].A
		}

		if i > 0 {
			prevChar.R = (*chars)[i-1].R
			prevChar.A = (*chars)[i-1].A
		}

		opening := false

		*pCount = q.parCount

		// Count parenthesis
		if char.R == '(' {
			opening = true
		} else if char.R == ')' {
			opening = false
		} else {
			// Not an opening or closing parenthesis
			continue
		}

		if *pCount < 0 {
			*pCount = 0
		}

		// Select a color, using modulo
		// Select another color if it's the same as the text that follows
		if opening {
			selected := (*pCount) % len(rainbowParenColors)
			char.A = rainbowParenColors[selected]
			// Loop until a color that is not the same as the color of the next character is selected
			// (and the next rune is not blank or end of line)

			// TODO: If the character before ( or ) are ' ' or '\t' OR the index is 0, color it blue (last color in rainbowParenColors)
			if prevChar.R == ' ' || prevChar.R == '\t' || i == 0 {
				char.A = lastColor
			} else {
				for char.A.Equal(nextChar.A) && (nextChar.R != ' ' && nextChar.R != '\n') {
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
				lastIndex = len(colorSlice) - 1
				char.A = colorSlice[lastIndex]
				colorSlice = colorSlice[:lastIndex]
			} else {
				char.A = parenErrorColor
			}
		}

		// For debugging
		//char.R = []rune(strconv.Itoa(*pCount))[0]

		// Keep the rune, but use the new AttributeColor
		(*chars)[i] = char
	}
}
