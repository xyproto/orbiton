package mode

import (
	"strings"
)

// SimpleDetect tries to return a Mode given a string of file contents
func SimpleDetect(contents string) Mode {
	firstLine := contents
	if strings.Contains(contents, "\n") {
		firstLine = strings.Split(contents, "\n")[0]
	}
	if len(firstLine) > 100 { // just look at the first 100, if it's one long line
		firstLine = firstLine[:100]
	}
	if m, found := DetectFromContents(Blank, firstLine, func() string { return contents }); found {
		return m
	}
	return Blank
}

// DetectFromContents takes the first line of a file as a string,
// and a function that can return the entire contents of the file as a string,
// which will only be called if needed.
// Based on the contents, a Mode is detected and returned.
// Pass inn mode.Blank as the initial Mode if that is the best guess so far.
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
		case "python":
			return Python, true
		case "ash", "bash", "fish", "ksh", "sh", "tcsh", "zsh":
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
	}
	// If more lines start with "# " than "// " or "/* ", and mode is blank,
	// set the mode to modeConfig and enable syntax highlighting.
	if !notConfig && (m == Blank || m == Config) {
		foundFirstContent := false
		hashComment := 0
		slashComment := 0
		for _, line := range strings.Split(allTextFunc(), "\n") {
			if strings.HasPrefix(line, "# ") {
				hashComment++
			} else if strings.HasPrefix(line, "/") { // Count all lines starting with "/" as a comment, for this purpose
				slashComment++
			}
			if trimmedLine := strings.TrimSpace(line); !foundFirstContent && !strings.HasPrefix(trimmedLine, "//") && len(trimmedLine) > 0 {
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
