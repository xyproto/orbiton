package main

import (
	"github.com/xyproto/textoutput"
	"github.com/xyproto/vt100"
)

var (
	colorSlice = make([]vt100.AttributeColor, 0) // to be pushed to and popped from
)

// rainbowParen implements "rainbow parenthesis" which colors "(" and ")" according to how deep they are nested
// pCount is the existing parenthesis count when reaching the start of this line
func rainbowParen(pCount *int, chars *[]textoutput.CharAttribute, singleLineCommentMarker string) {
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
		q.ProcessRune(r, prevRune, prevPrevRune)
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

		// Select a color, using modulo 6 of the parenthesis count
		// Select another color if it's the same as the text that follows
		if opening {
			switch *pCount % 9 {
			case 1: // the first one because of modulo (and a parenthesis has already been counted)
				charAttr.A = vt100.LightRed
				if charAttr.A.Equal(nextColor) {
					charAttr.A = vt100.White
				}
				colorSlice = append(colorSlice, charAttr.A)
			case 2:
				charAttr.A = vt100.Yellow
				if charAttr.A.Equal(nextColor) {
					charAttr.A = vt100.White
				}
				colorSlice = append(colorSlice, charAttr.A)
			case 3:
				charAttr.A = vt100.LightYellow
				if charAttr.A.Equal(nextColor) {
					charAttr.A = vt100.White
				}
				colorSlice = append(colorSlice, charAttr.A)
			case 4:
				charAttr.A = vt100.LightGreen
				if charAttr.A.Equal(nextColor) {
					charAttr.A = vt100.White
				}
				colorSlice = append(colorSlice, charAttr.A)
			case 5:
				charAttr.A = vt100.Green
				if charAttr.A.Equal(nextColor) {
					charAttr.A = vt100.White
				}
				colorSlice = append(colorSlice, charAttr.A)
			case 6:
				charAttr.A = vt100.LightBlue
				if charAttr.A.Equal(nextColor) {
					charAttr.A = vt100.White
				}
				colorSlice = append(colorSlice, charAttr.A)
			case 7:
				charAttr.A = vt100.Blue
				if charAttr.A.Equal(nextColor) {
					charAttr.A = vt100.White
				}
				colorSlice = append(colorSlice, charAttr.A)
			case 0: // the last one because of modulo
				charAttr.A = vt100.Magenta
				if charAttr.A.Equal(nextColor) {
					charAttr.A = vt100.White
				}
				colorSlice = append(colorSlice, charAttr.A)
			}
		} else {
			if len(colorSlice) > 0 {
				lastIndex = len(colorSlice) - 1
				charAttr.A = colorSlice[lastIndex]
				colorSlice = colorSlice[:lastIndex]
			} else {
				charAttr.A = vt100.Red
			}
		}

		//*pCount -= parens

		// For debugging
		//charAttr.R = []rune(strconv.Itoa(*pCount))[0]

		// Keep the rune, but use the new AttributeColor
		(*chars)[i] = charAttr
	}
}
