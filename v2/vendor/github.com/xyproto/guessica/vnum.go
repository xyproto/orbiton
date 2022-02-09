package guessica

// This file is extracted from the "getver" project, for automatically finding the newest
// version number for a given PKGBUILD file, by examining the corresponding web page.
// It has also been modified to fetch the latest git commit for the latest git version tag.
// This code is not particularly pretty and probably needs a good refactoring or two.

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxCollectedWords = 2048
	lookInsideTags    = false
)

// Find a list of likely version numbers, given an URL and a maximum number of results
// TODO: This function needs quite a bit of refactoring
func versionNumbers(url string, maxResults, crawlDepth int, includeFilenames bool) []string {

	const (
		ALLOWED = "0123456789.-+_~ABCDEFGHIJKLNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
		LETTERS = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
		UPPER   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		DIGITS  = "0123456789"
		SPECIAL = ".-+_~"
	)

	// Mutex for storing words while crawling with several gorutines
	wordMut := new(sync.Mutex)

	// Maps from a word to a crawl depth (smaller is further away)
	wordMapDepth := make(map[string]int)
	// Maps from a word to a word index on a page
	wordMapIndex := make(map[string]int)
	// Maps from a word to a char position  on a page
	wordMapCharIndex := make(map[string]int)

	// Find the words
	crawlDomain(url, crawlDepth, func(target, data string, currentDepth int) {
		wordIndex := 0
		//fmt.Println("Finding digits for", target)
		word := ""
		intag := false
		for charIndex, x := range data {
			if !intag && (x == '<') {
				intag = true
			} else if intag && (x == '>') {
				intag = false
			} else if (!intag || lookInsideTags) && strings.Contains(ALLOWED, string(x)) {
				word += string(x)
			} else if (!intag || lookInsideTags) && !strings.Contains(ALLOWED, string(x)) {
				ok := true
				// Check if the word is empty
				if word == "" {
					ok = false
				}
				// Check if the word is at least two letters long
				if ok && (len(word) < 2) {
					ok = false
				}
				// If the word is longer than "100.23.3123-beta" (16-digits),
				// it's unlikely to be a version number
				if ok && (len(word) > 16) {
					ok = false
				}
				// If there is more than one capital letter, skip it
				if ok {
					capCount := 0
					// Count up to 2 capital letters
					for _, letter := range word {
						if strings.Contains(UPPER, string(letter)) {
							capCount++
							if capCount > 1 {
								break
							}
						}
					}
					if capCount > 1 {
						ok = false
					}
					// If there is a capital letter and no dot, skip it
					if ok && (capCount == 1) && !strings.Contains(word, ".") {
						ok = false
					}
				}
				// If the word ends with a dot, remove it
				if ok && strings.HasSuffix(word, ".") {
					word = word[:len(word)-1]
				}
				// Trim space
				if ok {
					word = strings.TrimSpace(word)
				}
				// Check if the word has at least one digit
				if ok {
					found := false
					for _, digit := range DIGITS {
						if strings.Contains(word, string(digit)) {
							found = true
							break
						}
					}
					if !found {
						ok = false
					}
				}
				// If there are more than four dots
				if ok && (strings.Count(word, ".") > 4) {
					ok = false
				}
				// If there are two or more dots, and no other special character
				if ok && (strings.Count(word, ".") > 3) {
					foundOtherSpecial := false
					for _, special := range "-+_" { // Only look for special characters that are not "."
						if strings.Contains(word, string(special)) {
							foundOtherSpecial = true
							break
						}
					}
					if !foundOtherSpecial {
						ok = false
					}
				}
				// Check if the word has two special characters in a row
				if ok {
					for _, special := range SPECIAL {
						if strings.Contains(word, string(special)+string(special)) {
							// Not a version number
							ok = false
							break
						}
					}
				}
				// If the word is at least 4 letters long, check if it could be a filename
				if ok && !includeFilenames && (len(word) >= 4) {
					// If the last letter is not a digit
					if !strings.Contains(DIGITS, string(word[len(word)-1])) {
						// If the '.' leaves three or two letters at the end
						if (word[len(word)-4] == '.') || (word[len(word)-3] == '.') {
							// It's probably a filename
							ok = false
						}
					}
				}
				// If the word starts with a special character, skip it
				if ok && strings.Contains(SPECIAL, string(word[0])) {
					ok = false
				}
				// If the word is digits and two dashes, assume it's a date
				if ok && (strings.Count(word, "-") == 2) {
					onlyDateLetters := true
					for _, letter := range word {
						if !strings.Contains(DIGITS+"-", string(letter)) {
							onlyDateLetters = false
							break
						}
					}
					// More likely to be a date, skip
					if onlyDateLetters {
						ok = false
					}
				}

				// If the word is one dash with one or two digits on either side, assume it's a date
				if ok && (strings.Count(word, "-") == 1) {
					parts := strings.Split(word, "-")
					left, right := parts[0], parts[1]
					if (len(left) <= 2) && (len(right) <= 2) {
						onlyDigits := true
						for _, letter := range left {
							if !strings.Contains(DIGITS, string(letter)) {
								// Not a digit
								onlyDigits = false
								break
							}
						}
						if onlyDigits {
							for _, letter := range right {
								if !strings.Contains(DIGITS, string(letter)) {
									// Not a digit
									onlyDigits = false
									break
								}
							}
						}
						if onlyDigits {
							// Most likely a date
							ok = false
						}
					}
				}

				// Strip away letters. If needed, strip away special characters
				// at the beginning or end too. Don't strip "alpha" and "beta".
				if ok && !noStripLetters && !(strings.Contains(word, "alpha") || strings.Contains(word, "beta")) {
					newWord := ""
					for _, letter := range word {
						if strings.Contains(DIGITS+SPECIAL, string(letter)) {
							newWord += string(letter)
						}
					}
					// If the new word starts with a ".", strip it
					word = strings.TrimPrefix(newWord, ".")
				}

				// If there are only letters in front of the first dot, skip it
				if ok && strings.Contains(word, ".") {
					parts := strings.Split(word, ".")
					foundNonLetter := false
					for _, letter := range parts[0] {
						if !strings.Contains(LETTERS, string(letter)) {
							foundNonLetter = true
						}
					}
					// Only letters before the first dot
					if !foundNonLetter {
						ok = false
					}
				}

				// More than three digits in a row is not likely to be a version number
				if ok {
					streakCount := 0
					maxStreak := 0
					for _, letter := range word {
						if strings.Contains(DIGITS, string(letter)) {
							streakCount++
						} else {
							// Set maxStreak and reset the streakCount
							if streakCount > maxStreak {
								maxStreak = streakCount
							}
							streakCount = 0
						}
					}
					if streakCount > maxStreak {
						maxStreak = streakCount
					}
					if maxStreak > 3 {
						ok = false
					}
				}
				// If the word has no special characters and starts with "0", it's not a version number
				if ok {
					hasSpecial := false
					for _, special := range SPECIAL {
						if strings.Contains(word, string(special)) {
							hasSpecial = true
							break
						}
					}
					if !hasSpecial && strings.HasPrefix(word, "0") {
						ok = false
					}
				}
				// If the first digit is directly preceded by a single letter, skip it
				if ok {
					// Find the first digit
					pos := -1
					for i, letter := range word {
						if strings.Contains(DIGITS, string(letter)) {
							pos = i
							break
						}
					}
					if pos > 0 {
						// Check if the preceding letter contains no special letters
						preceding := word[:pos]
						if (len(preceding) == 1) && !strings.Contains(LETTERS, string(preceding[0])) {
							ok = false
						}
					}
				}
				// If the number is just the digit "0", it's not a version number
				if ok {
					onlyZero := true
					for _, letter := range word {
						if letter != '0' {
							onlyZero = false
							break
						}
					}
					if onlyZero {
						ok = false
					}
				}
				// Some words are usually not part of version numbers (but perhaps filenames)
				if ok {
					for _, unrelatedWord := range []string{"i686", "x86", "x64", "64bit", "32bit", "md5", "sha1", "sha256"} {
						if strings.Contains(word, unrelatedWord) {
							ok = false
							break
						}
					}
				}

				// the word might be a version number, add it to the list
				if ok {
					wordMut.Lock()
					// check if the word already exists
					if oldDepth, ok := wordMapDepth[word]; ok {
						// store the smallest depth
						if currentDepth < oldDepth {
							// save the current crawl depth (smaller is further away) together with the wordindex
							wordMapDepth[word] = currentDepth
							wordMapIndex[word] = wordIndex
							wordMapCharIndex[word] = charIndex
						}
					} else {
						// Save the current crawl depth (smaller is further away) together with the wordIndex
						wordMapDepth[word] = currentDepth
						wordMapIndex[word] = wordIndex
						wordMapCharIndex[word] = charIndex
					}
					wordIndex++
					wordMut.Unlock()
					// If we have enough words, just return
					if len(wordMapDepth) > maxCollectedWords {
						return
					}
				}
				word = ""
				if strings.Contains(ALLOWED, string(x)) {
					word = string(x)
				}
			}
		}
	})

	// Find the maximum number of dots and maximum word index
	maxdots := 0
	count := 0
	maxindex := 0
	index := 0
	for word := range wordMapDepth {
		// Find the maximum dotcount
		count = strings.Count(word, ".")
		if count > maxdots {
			maxdots = count
		}
		// Find the maximum index
		index = wordMapIndex[word]
		if index > maxindex {
			maxindex = index
		}
	}

	// The maximum depth
	maxdepth := crawlDepth

	// Find all char indices
	var charIndexList []int
	for _, charIndex := range wordMapCharIndex {
		charIndexList = append(charIndexList, charIndex)
	}

	// Sort the likely version numbers by a number of criteria

	var sortedWords []string
	resultCounter := 0
OUT:
	for i := maxdots; i >= 0; i-- { // Sort by number of "." in the version number
		for i2 := 0; i2 <= maxindex; i2++ { // Sort by word placement on the page
			for d := maxdepth; d >= 0; d-- { // Sort by crawl depth, highest number first (most shallow)
				for _, charIndex := range charIndexList { // Sort by page character index as well
					for word, depth := range wordMapDepth { // Loop through the gathered words
						if (strings.Count(word, ".") == i) && (depth == d) && (wordMapIndex[word] == i2) && (wordMapCharIndex[word] == charIndex) {
							sortedWords = append(sortedWords, word)
							resultCounter++
							if resultCounter == maxResults {
								break OUT
							}
						}
					}
				}
			}
		}
	}

	return sortedWords
}

func getver(v string) (string, error) {
	retrieve := 1
	selection := -1
	crawlDepth := 1
	timeout := 10000
	sortResults := false
	nostripped := false
	includeFilenames := false

	// If a specific result is wanted, make sure to retrieve just enough results
	// This also makes x=0 work, even though 1 is the minimum specified index for x.
	if selection != -1 {
		retrieve = selection + 1
	}

	if crawlDepth > 3 {
		return "", errors.New("maximum crawl depth is 3")
	}

	url := v

	// Set a default protocol (for crawling relative links)
	if strings.HasPrefix(url, "https") {
		defaultProtocol = "https"
	} else if !strings.Contains(url, "://") {
		// Add a default protocol if "*://" is mising
		url = defaultProtocol + "://" + url
	}

	clientTimeout = time.Duration(timeout) * time.Millisecond
	noStripLetters = nostripped

	// Retrieve the results

	foundVersionNumbers := versionNumbers(url, retrieve, crawlDepth, includeFilenames)
	if sortResults {
		sort.Strings(foundVersionNumbers)
		var reversed []string
		maxnum := len(foundVersionNumbers) - 1
		for i := 0; i <= maxnum; i++ {
			reversed = append(reversed, foundVersionNumbers[maxnum-i])
		}
		foundVersionNumbers = reversed
	}

	// Output the results

	if (selection > 0) && (selection <= len(foundVersionNumbers)) {
		return foundVersionNumbers[selection-1], nil
	} else if selection >= len(foundVersionNumbers) {
		return "", fmt.Errorf("not enough results to retrieve result number %d", selection)
	} else {
		// Regular non-numbered output of the results
		for _, word := range foundVersionNumbers {
			return word, nil
		}
		return "", errors.New("no results, no errors, no output")
	}
}
