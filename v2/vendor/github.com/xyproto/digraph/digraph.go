// Package digraph provides functions for looking up ViM-style digraphs
package digraph

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"
)

var (
	digraphMap     map[string]rune
	descriptionMap map[string]string
)

//go:embed digraphs.txt.gz
var digraphsCompressed []byte

// Used to avoid memory allocations while filling the maps, and for testing
const digraphsInDigraphFile = 1305

func init() {

	// Start out by decompressing the embedded digraphCompressed bytes to a digraphs string

	reader, err := gzip.NewReader(bytes.NewReader(digraphsCompressed))
	if err != nil {
		panic("could not create a gzip reader in init: " + err.Error())
	}
	defer reader.Close()
	var decompressed bytes.Buffer
	_, err = io.Copy(&decompressed, reader)
	if err != nil {
		panic("could not decompress digraphs in init: " + err.Error())
	}
	digraphs := decompressed.String()

	// Then initialize the global maps

	digraphMap = make(map[string]rune, digraphsInDigraphFile)
	descriptionMap = make(map[string]string, digraphsInDigraphFile)

	lines := strings.Split(digraphs, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		fields := strings.Fields(line)
		digraph := fields[0]
		//hexvalue := fields[1]
		decvalue := fields[2]

		num, err := strconv.Atoi(decvalue)
		if err != nil {
			continue
		}

		digraphMap[digraph] = rune(num)

		description := strings.Join(fields[3:], " ")
		descriptionMap[digraph] = description
	}
}

// MustLookup tries to look up a digraph and return a rune,
// but does not return any bool or error if the digraph string is not found.
func MustLookup(digraph string) rune {
	return digraphMap[digraph]
}

// Lookup a digraph and return a rune. Returns false if the digraph could not be found.
func Lookup(digraph string) (rune, bool) {
	r, ok := digraphMap[digraph]
	return r, ok
}

// MustLookupDescription tries to look up a digraph and return a description,
// but it edoes not return any bool or error if the digraph string is not found.
func MustLookupDescription(digraph string) string {
	return descriptionMap[digraph]
}

// LookupDescription tries to look up a digraph and return a description.
// Returns false if the digraph could not be found.
func LookupDescription(digraph string) (string, bool) {
	description, ok := descriptionMap[digraph]
	return description, ok
}

// All returns a string slice of all available digraphs
func All() []string {
	allDigraphs := make([]string, len(digraphMap))
	i := 0
	for k := range digraphMap {
		allDigraphs[i] = k
		i++
	}
	return allDigraphs
}

// PrintTable outputs a table of all available digraphs
func PrintTable() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	defer w.Flush()
	for _, twoLetters := range All() {
		symbol := MustLookup(twoLetters)
		description := MustLookupDescription(twoLetters)
		symbolStr := "N/A"
		if unicode.IsPrint(symbol) {
			symbolStr = fmt.Sprintf("%c", symbol)
		}
		// print hex code, two letters, description and symbol
		fmt.Fprintf(w, "%04X\t%s\t%s\t%s\n", symbol, twoLetters, description, symbolStr)
	}
}
