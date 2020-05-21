package main

import (
	"github.com/xyproto/textoutput"
	"github.com/xyproto/vt100"
)

var (
	colorSlice         = make([]vt100.AttributeColor, 0) // to be pushed to and popped from
	rainbowParenColors = []vt100.AttributeColor{vt100.LightMagenta, vt100.LightRed, vt100.Yellow, vt100.LightYellow, vt100.LightGreen, vt100.Green, vt100.LightBlue, vt100.Blue}
	parenErrorColor    = vt100.White // this color is meant to stand out, for unbalanced parenthesis
)

// rainbowParen implements "rainbow parenthesis" which colors "(" and ")" according to how deep they are nested
// pCount is the existing parenthesis count when reaching the start of this line
func rainbowParen(pCount *int, chars *[]textoutput.CharAttribute, singleLineCommentMarker string, ignoreSingleQuotes bool) {
	var (
		q = NewQuoteState(singleLineCommentMarker) // TODO: Use the proper single line comment
		//decrement              = 0
		prevRune, prevPrevRune = '\n', '\n'

		// CharAttribute has a rune "R" and a vt100.AttributeColor "A"
		nextColor = defaultEditorForeground // Used for smarter color selection for parenthesis
		lastIndex = len(*chars) - 1
	)
	//q.parCount = *pCount
	for i, charAttr := range *chars {
		r := charAttr.R
		q.ProcessRune(r, prevRune, prevPrevRune, ignoreSingleQuotes)
		prevPrevRune = prevRune
		prevRune = r

		if !q.None() {
			// Skip comments and strings
			continue
		}

		if (i + 1) < len(*chars) {
			nextColor = (*chars)[i+1].A
		}

		opening := false

		*pCount = q.parCount
		//parens := 0

		// Count parenthesis
		if r == '(' {
			opening = true
			//*pCount++
			//parens++
		} else if r == ')' {
			opening = false
			//parens--
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
			charAttr.A = rainbowParenColors[selected]
			// Loop until a color that is not the same as the color of the next character is selected
			for charAttr.A.Equal(nextColor) {
				selected++
				if selected >= len(rainbowParenColors) {
					selected = 0
				}
				charAttr.A = rainbowParenColors[selected]
			}
			// Push the color to the color stack
			colorSlice = append(colorSlice, charAttr.A)
		} else {
			if len(colorSlice) > 0 {
				// pop the color from the color stack
				lastIndex = len(colorSlice) - 1
				charAttr.A = colorSlice[lastIndex]
				colorSlice = colorSlice[:lastIndex]
			} else {
				charAttr.A = parenErrorColor
			}
		}

		// For debugging
		//charAttr.R = []rune(strconv.Itoa(*pCount))[0]

		// Keep the rune, but use the new AttributeColor
		(*chars)[i] = charAttr
	}
}
