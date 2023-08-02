package mode

import (
	"bytes"
	"strconv"
	"strings"
)

// SimpleDetect tries to return a Mode given a string of file contents
func SimpleDetect(contents string) Mode {
	firstLine := contents
	if strings.Contains(contents, "\n") {
		firstLine = strings.SplitN(contents, "\n", 2)[0]
	}
	if len(firstLine) > 512 { // just look at the first 512, if it's one long line
		firstLine = firstLine[:512]
	}
	if m, found := DetectFromContents(Blank, firstLine, func() string { return contents }); found {
		return m
	}
	return Blank
}

// SimpleDetectBytes tries to return a Mode given a byte slice of file contents
func SimpleDetectBytes(contents []byte) Mode {
	nl := []byte("\n")
	firstLine := contents
	if bytes.Contains(contents, nl) {
		firstLine = bytes.SplitN(contents, nl, 2)[0]
	}
	if len(firstLine) > 512 { // just look at the first 255, if it's one long line
		firstLine = firstLine[:512]
	}
	if m, found := DetectFromContentBytes(Blank, firstLine, func() []byte { return contents }); found {
		return m
	}
	return Blank
}

// DetectFromContents takes the first line of a file as a string,
// and a function that can return the entire contents of the file as a string,
// which will only be called if needed.
// Based on the contents, a Mode is detected and returned.
// Pass inn mode.Blank as the initial Mode if that is the best guess so far.
// Returns true if a mode is found.
// TODO: Create a generic function that handles both strings and bytes instead of maintaining both.
func DetectFromContents(initial Mode, firstLine string, allTextFunc func() string) (Mode, bool) {
	var found, notConfig bool
	m := initial
	if strings.HasPrefix(firstLine, "#!") { // The line starts with a shebang
		words := strings.Split(firstLine, " ")
		lastWord := words[len(words)-1]
		if strings.Contains(lastWord, "/") {
			words = strings.Split(lastWord, "/")
			lastWord = words[len(words)-1]
		}
		// check for "python", "python2.7", "python3" etc
		if strings.HasPrefix(lastWord, "python") {
			return Python, true
		}
		switch lastWord {
		case "perl":
			return Perl, true
		case "ash", "bash", "fish", "ksh", "oil", "sh", "tcsh", "zsh": // TODO: support Fish and Oil with their own file modes
			return Shell, true
		}
		notConfig = true
	} else if strings.HasPrefix(firstLine, "# $") {
		// Most likely a csh script on FreeBSD
		return Shell, true
	} else if strings.HasPrefix(firstLine, "<?xml ") {
		return XML, true
	} else if strings.Contains(firstLine, "-*- nroff -*-") {
		return Nroff, true
	} else if strings.HasPrefix(firstLine, "From ") && strings.HasSuffix(firstLine, "# This line is ignored.") {
		return Email, true
	} else if strings.HasPrefix(firstLine, "\" ") {
		// The first line starts with '" ', assume ViM script
		return Vim, true
	}
	// Man page detection (two equal words at the start and end of the line, and both have "(" and ")")
	// Also, the line does not start with a number and does not contain "//"
	if !strings.Contains(firstLine, "//") && !strings.HasPrefix(firstLine, "#") && strings.Count(strings.TrimSpace(firstLine), " ") > 10 && strings.HasSuffix(firstLine, ")") {
		fields := strings.Fields(strings.TrimSpace(firstLine))
		if len(fields) > 2 {
			firstWord := fields[0]
			if _, err := strconv.Atoi(firstWord); err != nil { // the first word is not a number
				return ManPage, true
			}
		}
	}
	if m == Blank {
		fields := strings.Fields(strings.TrimSpace(firstLine))
		if len(fields) > 2 {
			firstWord := fields[0]
			lastWord := fields[len(fields)-1]
			if firstWord == lastWord && strings.Count(firstWord, "(") == 1 && strings.Count(firstWord, ")") == 1 {
				if _, err := strconv.Atoi(firstWord); err != nil { // the first word is not a number
					return ManPage, true
				}
			}
		}
	}
	// If more lines start with "# " than "// " or "/* ", and mode is blank,
	// set the mode to modeConfig and enable syntax highlighting.
	// If more than one line is just "::"  or starts with "[source" and ends with "]", assume reStructuredText
	if !notConfig && (m == Blank || m == Config || m == Markdown) {
		foundFirstContent := false
		hashComment := 0
		slashComment := 0
		reStructuredTextMarkers := 0
		configMarkers := 0
		markdownMarkers := 0
		lines := strings.Split(allTextFunc(), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "# ") {
				hashComment++
			} else if strings.HasPrefix(line, "/") { // Count all lines starting with "/" as a comment, for this purpose
				slashComment++
			}
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine == "::" || strings.HasPrefix(trimmedLine, ".. ") || strings.HasPrefix(trimmedLine, "[source,") {
				reStructuredTextMarkers++
				if reStructuredTextMarkers == 2 {
					return ReStructured, true
				}
			} else if !foundFirstContent && !strings.HasPrefix(trimmedLine, "//") && len(trimmedLine) > 0 {
				foundFirstContent = true
				if trimmedLine == "{" { // first found content is {, assume JSON
					m = JSON
					found = true
				}
			} else if strings.Contains(trimmedLine, "====") || strings.Contains(trimmedLine, "----") {
				markdownMarkers++
			}
			if strings.Contains(trimmedLine, "(") || strings.Contains(trimmedLine, ")") || strings.Contains(trimmedLine, "=") {
				// Might be a configuration file if most of the lines have (, ) or =
				configMarkers++
			}
		}
		if markdownMarkers > 3 {
			return Markdown, true
		}
		if hashComment > slashComment {
			return Config, true
		}
		// Are "most of the lines" containing (, ) or = ?
		if (float64(configMarkers) / float64(len(lines))) > 0.7 {
			return Config, true
		}
	}
	// If the mode is modeOCaml and there are no ";;" strings, switch to Standard ML
	if m == OCaml {
		if !strings.Contains(allTextFunc(), ";;") {
			return StandardML, true
		}
	} else if m == Assembly {
		if strings.Contains(allTextFunc(), "·") { // Go-style assembly mid dot
			return GoAssembly, true
		}
	}
	return m, found
}

// DetectFromContentBytes takes the first line of a file as a byte slice,
// and a function that can return the entire contents of the file as a byte slice,
// which will only be called if needed.
// Based on the contents, a Mode is detected and returned.
// Pass inn mode.Blank as the initial Mode if that is the best guess so far.
// Returns true if a mode is found.
// TODO: Create a generic function that handles both strings and bytes instead of maintaining both.
func DetectFromContentBytes(initial Mode, firstLine []byte, allBytesFunc func() []byte) (Mode, bool) {
	var found, notConfig bool
	m := initial
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
			} else if lastWord[0] == 'p' && lastWord[1] == 'y' && strings.HasPrefix(string(lastWord), "python") { // check for "python", "python2.7", "python3" etc
				return Python, true
			} else {
				switch string(lastWord) {
				case "ash", "bash", "fish", "ksh", "oil", "sh", "tcsh", "zsh": // TODO: support Fish and Oil with their own file modes
					return Shell, true
				}
			}
		}
		notConfig = true
	} else if bytes.HasPrefix(firstLine, []byte("# $")) {
		// Most likely a csh script on FreeBSD
		return Shell, true
	} else if bytes.HasPrefix(firstLine, []byte("<?xml ")) {
		return XML, true
	} else if bytes.Contains(firstLine, []byte("-*- nroff -*-")) {
		return Nroff, true
	} else if !bytes.HasPrefix(firstLine, []byte("//")) && !bytes.HasPrefix(firstLine, []byte("#")) && bytes.Count(bytes.TrimSpace(firstLine), []byte(" ")) > 10 && bytes.HasSuffix(firstLine, []byte(")")) {
		return ManPage, true
	} else if bytes.HasPrefix(firstLine, []byte("\" ")) {
		// The first line starts with '" ', assume ViM script
		return Vim, true
	}
	// If more lines start with "# " than "// " or "/* ", and mode is blank,
	// set the mode to modeConfig and enable syntax highlighting.
	if !notConfig && (m == Blank || m == Config || m == Markdown) {
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
	// If the mode is modeOCaml and there are no ";;" strings, switch to Standard ML
	if m == OCaml {
		if !bytes.Contains(allBytesFunc(), []byte(";;")) {
			return StandardML, true
		}
	} else if m == Assembly {
		if bytes.Contains(allBytesFunc(), []byte("·")) { // Go-style assembly mid dot
			return GoAssembly, true
		}
	}
	return m, found
}
