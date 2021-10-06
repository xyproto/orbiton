package mode

import (
	"strings"
)

// DetectFromContents takes the first line of a file as a string,
// and a function that can return the entire contents of the file as a string,
// which will only be called if needed.
// Based on the contents, a Mode is detected and returned.
func DetectFromContents(firstLine string, getAllText func() string) (Mode, bool) {
	var m Mode
	found := false
	if strings.HasPrefix(firstLine, "#!") { // The line starts with a shebang
		words := strings.Split(firstLine, " ")
		lastWord := words[len(words)-1]
		if strings.Contains(lastWord, "/") {
			words = strings.Split(lastWord, "/")
			lastWord = words[len(words)-1]
		}
		switch lastWord {
		case "python":
			m = Python
			found = true
		case "bash", "fish", "zsh", "tcsh", "ksh", "sh", "ash":
			m = Shell
			found = true
		}
	} else if strings.HasPrefix(firstLine, "# $") {
		// Most likely a csh script on FreeBSD
		m = Shell
		found = true
	} else if strings.HasPrefix(firstLine, "<?xml ") {
		m = XML
		found = true
	} else if strings.Contains(firstLine, "-*- nroff -*-") {
		m = Nroff
		found = true
	} else if !strings.HasPrefix(firstLine, "//") && !strings.HasPrefix(firstLine, "#") && strings.Count(strings.TrimSpace(firstLine), " ") > 10 && strings.HasSuffix(firstLine, ")") {
		m = ManPage
		found = true
	}
	foundFirstContent := false
	// If more lines start with "# " than "// " or "/* ", and mode is blank,
	// set the mode to modeConfig and enable syntax highlighting.
	if m == Blank || m == Config {
		hashComment := 0
		slashComment := 0
		for _, line := range strings.Split(getAllText(), "\n") {
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
			m = Config
			found = true
		}
	} else if m == Assembly {
		if strings.Contains(getAllText(), "·") { // Go-style assembly mid dot
			m = GoAssembly
			found = true
		}
	}
	// If the mode is modeOCaml and there are no ";;" strings, switch to Standard ML
	if m == OCaml {
		if !strings.Contains(getAllText(), ";;") {
			m = StandardML
			found = true
		}
	}
	return m, found
}
