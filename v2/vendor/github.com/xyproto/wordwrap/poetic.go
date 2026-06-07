package wordwrap

import (
	"errors"
	"math"
	"strings"
	"unicode/utf8"
)

// Brokenness penalties; lower is more natural.
const (
	breakSentenceEnd = 0
	breakSemicolon   = 25
	breakColon       = 40
	breakComma       = 60
	breakDash        = 80
	breakMidClause   = 400
)

// breakPenalty scores ending a line on word.
func breakPenalty(word string) float64 {
	trimmed := strings.TrimRight(word, "\"')]"+"’”»")
	if trimmed == "" {
		return breakMidClause
	}
	runes := []rune(trimmed)
	switch runes[len(runes)-1] {
	case '.', '!', '?':
		return breakSentenceEnd
	case ';':
		return breakSemicolon
	case ':':
		return breakColon
	case ',':
		return breakComma
	case '—', '–':
		return breakDash
	}
	return breakMidClause
}

// PoeticWrap wraps text to maxWidth runes, preferring breaks at natural
// punctuation (sentence ends, semicolons, colons, commas, em/en dashes)
// over arbitrary word boundaries. Uses Knuth–Plass-style DP to minimise
// slack² + brokenness globally. Suited to fortunes, quotes and poems.
//
// minWidth is the soft minimum length for non-final lines; 0 picks a
// default of maxWidth*2/5, values above maxWidth disable the guard.
// Newlines are hard paragraph breaks. Honours the NoLineStart rule.
func PoeticWrap(text string, maxWidth, minWidth int) ([]string, error) {
	if maxWidth <= 0 {
		return nil, errors.New("maxWidth must be greater than 0")
	}
	if minWidth <= 0 {
		minWidth = maxWidth * 2 / 5
	}

	paragraphs := strings.Split(text, "\n")
	var result []string
	for _, p := range paragraphs {
		wrapped := poeticWrapParagraph(p, maxWidth, minWidth)
		if len(wrapped) == 0 {
			result = append(result, "")
			continue
		}
		result = append(result, wrapped...)
	}
	return result, nil
}

// poeticWrapParagraph wraps one paragraph (no embedded newlines).
func poeticWrapParagraph(text string, maxWidth, minWidth int) []string {
	words := strings.Fields(text)
	n := len(words)
	if n == 0 {
		return nil
	}
	wordLen := make([]int, n)
	for i, w := range words {
		wordLen[i] = utf8.RuneCountInString(w)
	}

	// cost[i] = min cost to lay out words[0..i-1]; breakAt[i] = first
	// word of the line that ends at words[i-1].
	cost := make([]float64, n+1)
	breakAt := make([]int, n+1)
	for i := 1; i <= n; i++ {
		cost[i] = math.Inf(1)
	}

	for i := 1; i <= n; i++ {
		// Forbid breaks that would start the next line on punctuation.
		if i < n {
			first, _ := utf8.DecodeRuneInString(words[i])
			if NoLineStart(first) {
				continue
			}
		}
		runLen := 0
		for j := i - 1; j >= 0; j-- {
			if j < i-1 {
				runLen++
			}
			runLen += wordLen[j]
			if runLen > maxWidth && j < i-1 {
				break
			}
			if math.IsInf(cost[j], 1) {
				continue
			}
			isLast := i == n
			var lineCost float64
			switch {
			case runLen > maxWidth:
				// Single oversized word: per-rune overflow penalty.
				lineCost = float64(runLen-maxWidth) * 1000
			case isLast:
				lineCost = 0
			default:
				slack := float64(maxWidth - runLen)
				lineCost = slack*slack + breakPenalty(words[i-1])
				if runLen < minWidth {
					lineCost += 10000
				}
			}
			total := cost[j] + lineCost
			if total < cost[i] {
				cost[i] = total
				breakAt[i] = j
			}
		}
	}

	// Walk breakAt chain backwards to recover lines.
	var lines []string
	for i := n; i > 0; {
		j := breakAt[i]
		lines = append(lines, strings.Join(words[j:i], " "))
		i = j
	}
	for l, r := 0, len(lines)-1; l < r; l, r = l+1, r-1 {
		lines[l], lines[r] = lines[r], lines[l]
	}
	return lines
}
