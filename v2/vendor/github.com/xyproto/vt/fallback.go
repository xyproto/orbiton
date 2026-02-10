//go:build !windows

package vt

// parseCSIFallback handles common CSI sequences that include parameters.
// seq is the parameter bytes between ESC[ and the final byte.
func parseCSIFallback(seq []byte, final byte) (Event, bool) {
	switch final {
	case 'A':
		return Event{Kind: EventKey, Key: KeyArrowUp}, true
	case 'B':
		return Event{Kind: EventKey, Key: KeyArrowDown}, true
	case 'C':
		return Event{Kind: EventKey, Key: KeyArrowRight}, true
	case 'D':
		return Event{Kind: EventKey, Key: KeyArrowLeft}, true
	case 'H':
		return Event{Kind: EventKey, Key: KeyHome}, true
	case 'F':
		return Event{Kind: EventKey, Key: KeyEnd}, true
	case '~':
		params, ok := parseCSIParams(seq)
		if !ok || len(params) == 0 {
			return Event{}, false
		}
		switch params[0] {
		case 1, 7:
			return Event{Kind: EventKey, Key: KeyHome}, true
		case 4, 8:
			return Event{Kind: EventKey, Key: KeyEnd}, true
		case 5:
			return Event{Kind: EventKey, Key: KeyPageUp}, true
		case 6:
			return Event{Kind: EventKey, Key: KeyPageDown}, true
		}
	}
	return Event{}, false
}

func parseCSIParams(seq []byte) ([]int, bool) {
	if len(seq) == 0 {
		return nil, true
	}
	params := make([]int, 0, 2)
	value := 0
	hasDigit := false
	for _, b := range seq {
		switch {
		case b >= '0' && b <= '9':
			value = value*10 + int(b-'0')
			hasDigit = true
		case b == ';':
			if hasDigit {
				params = append(params, value)
			} else {
				params = append(params, 0)
			}
			value = 0
			hasDigit = false
		default:
			return nil, false
		}
	}
	if hasDigit {
		params = append(params, value)
	} else if len(seq) > 0 {
		// Trailing ';' implies an empty parameter.
		params = append(params, 0)
	}
	return params, true
}
