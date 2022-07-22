package mode

import (
	"bytes"
	"strings"
)

// SimpleDetect tries to return a Mode given a string of file contents
func SimpleDetect(contents string) Mode {
	firstLine := contents
	if strings.Contains(contents, "\n") {
		firstLine = strings.SplitN(contents, "\n", 2)[0]
	}
	if len(firstLine) > 100 { // just look at the first 100, if it's one long line
		firstLine = firstLine[:100]
	}
	if m, found := DetectFromContents(Blank, firstLine, func() string { return contents }); found {
		return m
	}
	return Blank
}

// SimpleDetectBytes tries to return a Mode given a byte slice of file contents
func SimpleDetectBytes(contents []byte) Mode {
	var nl = []byte("\n")
	firstLine := contents
	if bytes.Contains(contents, nl) {
		firstLine = bytes.SplitN(contents, nl, 2)[0]
	}
	if len(firstLine) > 100 { // just look at the first 100, if it's one long line
		firstLine = firstLine[:100]
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
		switch lastWord {
		case "perl":
			return Perl, true
		case "python":
			return Python, true
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
	} else if !strings.HasPrefix(firstLine, "//") && !strings.HasPrefix(firstLine, "#") && strings.Count(strings.TrimSpace(firstLine), " ") > 10 && strings.HasSuffix(firstLine, ")") {
		return ManPage, true
	} else if strings.HasPrefix(firstLine, "From ") && strings.HasSuffix(firstLine, "# This line is ignored.") {
		return Email, true
	}
	// If more lines start with "# " than "// " or "/* ", and mode is blank,
	// set the mode to modeConfig and enable syntax highlighting.
	// If more than one line is just "::"  or starts with "[source" and ends with "]", assume reStructuredText
	if !notConfig && (m == Blank || m == Config || m == Markdown) {
		foundFirstContent := false
		hashComment := 0
		slashComment := 0
		reStructuredTextMarkers := 0
		for _, line := range strings.Split(allTextFunc(), "\n") {
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
			}
		}
		if hashComment > slashComment {
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
			} else if lastWord[0] == 'p' && lastWord[1] == 'y' && string(lastWord) == "python" { // python
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
	}
	// If more lines start with "# " than "// " or "/* ", and mode is blank,
	// set the mode to modeConfig and enable syntax highlighting.
	if !notConfig && (m == Blank || m == Config || m == Markdown) {
		foundFirstContent := false
		hashComment := 0
		slashComment := 0
		reStructuredTextMarkers := 0
		for _, line := range bytes.Split(allBytesFunc(), []byte("\n")) {
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
		}
		if hashComment > slashComment {
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
