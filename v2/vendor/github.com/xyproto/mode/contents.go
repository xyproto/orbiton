package mode

import (
	"bytes"
	"strings"

	"github.com/xyproto/lookslikegoasm"
)

// SimpleDetectBytes tries to return a Mode given a byte slice of file contents
func SimpleDetectBytes(contents []byte) Mode {
	nl := []byte("\n")
	firstLine := contents
	if bytes.Contains(contents, nl) {
		firstLine = bytes.SplitN(contents, nl, 2)[0]
	}
	if len(firstLine) > 512 { // just look at the first 512, if it's one long line
		firstLine = firstLine[:512]
	}
	if m, found := DetectFromContentBytes(Blank, firstLine, func() []byte { return contents }); found {
		return m
	}
	return Blank
}

// SimpleDetect tries to return a Mode given a string of file contents
func SimpleDetect(contents string) Mode {
	return SimpleDetectBytes([]byte(contents))
}

// DetectFromContentBytes takes the first line of a file as a byte slice,
// and a function that can return the entire contents of the file as a byte slice,
// which will only be called if needed.
// Based on the contents, a Mode is detected and returned.
// Pass inn mode.Blank as the initial Mode if that is the best guess so far.
// Returns true if a mode is found.
func DetectFromContentBytes(initial Mode, firstLine []byte, allBytesFunc func() []byte) (Mode, bool) {
	var found, notConfig bool
	m := initial
	if m == Assembly || m == Blank {
		// Go/Plan9 style Assembly
		data := bytes.TrimSpace(allBytesFunc())
		if bytes.Contains(data, []byte("·")) || (bytes.Contains(data, []byte("TEXT")) && bytes.Contains(data, []byte("RET\n"))) {
			return GoAssembly, true
		}
	}
	if bytes.HasPrefix(firstLine, []byte("#!")) { // The line starts with a shebang
		words := bytes.Split(firstLine, []byte(" "))
		lastWord := words[len(words)-1]
		if bytes.Contains(lastWord, []byte("/")) {
			words = bytes.Split(lastWord, []byte("/"))
			lastWord = words[len(words)-1]
		}
		// Check the two first bytes first, for a tiny bit faster comparison
		if len(lastWord) > 1 {
			if lastWord[0] == 'p' && lastWord[1] == 'e' && string(lastWord) == "perl" { // perl
				return Perl, true
			}
			if lastWord[0] == 'p' && lastWord[1] == 'y' && strings.HasPrefix(string(lastWord), "python") { // check for "python", "python2.7", "python3" etc
				return Python, true
			}
			switch string(lastWord) {
			case "ash", "bash", "fish", "ksh", "oil", "sh", "tcsh", "zsh": // TODO: support Fish and Oil with their own file modes
				return Shell, true
			}

		}
		notConfig = true
	} else if bytes.HasPrefix(firstLine, []byte("# $")) {
		// Most likely a csh script on FreeBSD
		return Shell, true
	} else if bytes.HasPrefix(firstLine, []byte("<?xml ")) {
		return XML, true
	} else if bytes.HasPrefix(firstLine, []byte("{\"")) {
		return JSON, true
	} else if bytes.HasPrefix(bytes.ToLower(firstLine), []byte("<!doctype html")) || bytes.HasPrefix(bytes.ToLower(firstLine), []byte("<html")) {
		return HTML, true
	} else if bytes.Contains(firstLine, []byte("-*- nroff -*-")) {
		return Nroff, true
	} else if !bytes.HasPrefix(firstLine, []byte("//")) && !bytes.HasPrefix(firstLine, []byte("#")) && bytes.Count(bytes.TrimSpace(firstLine), []byte(" ")) > 10 && bytes.HasSuffix(firstLine, []byte(")")) {
		return ManPage, true
	} else if bytes.HasPrefix(firstLine, []byte("\" ")) {
		// The first line starts with '" ', assume ViM script
		return Vim, true
	} else if bytes.HasPrefix(firstLine, []byte("diff -")) {
		return Diff, true
	}
	// If more lines start with "# " than "// " or "/* ", and mode is blank,
	// set the mode to Config and enable syntax highlighting.
	if !notConfig && (m == Blank || m == Config || m == Markdown || m == Nix) {
		foundFirstContent := false
		hashComment := 0
		slashComment := 0
		reStructuredTextMarkers := 0
		byteLines := bytes.Split(allBytesFunc(), []byte("\n"))
		configMarkers := 0
		for _, line := range byteLines {
			if bytes.HasPrefix(line, []byte("# ")) {
				hashComment++
			} else if bytes.HasPrefix(line, []byte("/")) { // Count all lines starting with "/" as a comment, for this purpose
				slashComment++
			}
			trimmedLine := bytes.TrimSpace(line)
			if len(trimmedLine) > 1 && (trimmedLine[0] == byte(':') && trimmedLine[1] == byte(':')) || bytes.HasPrefix(trimmedLine, []byte(".. ")) || bytes.HasPrefix(trimmedLine, []byte("[source,")) {
				reStructuredTextMarkers++
				if reStructuredTextMarkers == 2 {
					return ReStructured, true
				}
			} else if !foundFirstContent && !bytes.HasPrefix(trimmedLine, []byte("//")) && len(trimmedLine) > 0 {
				foundFirstContent = true
				if len(trimmedLine) == 1 && trimmedLine[0] == byte('{') { // first found content is {, assume JSON
					m = JSON
					found = true
				}
			} else if bytes.HasPrefix(trimmedLine, []byte("+++ ")) || bytes.HasPrefix(trimmedLine, []byte("--- ")) {
				return Diff, true
			}
			if bytes.Contains(trimmedLine, []byte("(")) || bytes.Contains(trimmedLine, []byte(")")) || bytes.Contains(trimmedLine, []byte("=")) {
				// Might be a configuration file if most of the lines have (, ) or =
				configMarkers++
			}
		}
		if hashComment > slashComment {
			return Config, true
		}
		// Are "most of the lines" containing (, ) or = ?
		if (float64(configMarkers) / float64(len(byteLines))) > 0.7 {
			return Config, true
		}
	}
	switch m {
	case OCaml:
		// If the mode is modeOCaml and there are no ";;" strings, switch to Standard ML
		if !bytes.Contains(allBytesFunc(), []byte(";;")) {
			return StandardML, true
		}
	case Blank:
		if !notConfig {
			// If it's not a config file and the mode is blank, set it to XML if the first character is "<" and the last is ">"
			// set the mode to modeConfig and enable syntax highlighting.
			data := bytes.TrimSpace(allBytesFunc())
			if bytes.HasPrefix(data, []byte{'<'}) && bytes.HasSuffix(data, []byte{'>'}) {
				return XML, true
			}
		}
	case Assembly:
		// Check if it looks like Go/Plan9-style Assembly or not
		if lookslikegoasm.Consider(string(allBytesFunc())) {
			return GoAssembly, true
		}
	}
	return m, found
}

// DetectFromContents takes the first line of a file as a string,
// and a function that can return the entire contents of the file as a string,
// which will only be called if needed.
// Based on the contents, a Mode is detected and returned.
// Pass inn mode.Blank as the initial Mode if that is the best guess so far.
// Returns true if a mode is found.
func DetectFromContents(initial Mode, firstLine string, allTextFunc func() string) (Mode, bool) {
	allBytesFunc := func() []byte { return []byte(allTextFunc()) }
	return DetectFromContentBytes(initial, []byte(firstLine), allBytesFunc)
}
