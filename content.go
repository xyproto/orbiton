package main

import "strings"

func (e *Editor) checkContents() {
	// Check if the first line is special
	firstLine := e.Line(0)
	if strings.HasPrefix(firstLine, "#!") { // The line starts with a shebang
		words := strings.Split(firstLine, " ")
		lastWord := words[len(words)-1]
		if strings.Contains(lastWord, "/") {
			words = strings.Split(lastWord, "/")
			lastWord = words[len(words)-1]
		}
		switch lastWord {
		case "python":
			e.mode = modePython
		case "bash", "fish", "zsh", "tcsh", "ksh", "sh", "ash":
			e.mode = modeShell
		}
	} else if strings.HasPrefix(firstLine, "# $") {
		// Most likely a csh script on FreeBSD
		e.mode = modeShell
	} else if strings.HasPrefix(firstLine, "#") {
		e.firstLineHash = true
	} else if strings.HasPrefix(firstLine, "<?xml ") {
		e.mode = modeXML
	} else if strings.Contains(firstLine, "-*- nroff -*-") {
		e.mode = modeNroff
	} else if !strings.HasPrefix(firstLine, "//") && !strings.HasPrefix(firstLine, "#") && strings.Count(strings.TrimSpace(firstLine), " ") > 10 && strings.HasSuffix(firstLine, ")") {
		e.mode = modeManPage
	}
	foundFirstContent := false
	// If more lines start with "# " than "// " or "/* ", and mode is blank,
	// set the mode to modeConfig and enable syntax highlighting.
	if e.mode == modeBlank || e.mode == modeConfig {
		hashComment := 0
		slashComment := 0
		for _, line := range strings.Split(e.String(), "\n") {
			if strings.HasPrefix(line, "# ") {
				hashComment++
			} else if strings.HasPrefix(line, "/") { // Count all lines starting with "/" as a comment, for this purpose
				slashComment++
			}
			if trimmedLine := strings.TrimSpace(line); !foundFirstContent && !strings.HasPrefix(trimmedLine, "//") && len(trimmedLine) > 0 {
				foundFirstContent = true
				if trimmedLine == "{" { // first found content is {, assume JSON
					e.mode = modeJSON
				}
			}
		}
		if hashComment > slashComment {
			e.mode = modeConfig
			e.syntaxHighlight = true
		}
	} else if e.mode == modeAssembly {
		if strings.Contains(e.String(), "·") { // Go-style assembly mid dot
			e.mode = modeGoAssembly
		}
	}
	// If the mode is modeOCaml and there are no ";;" strings, switch to Standard ML
	if e.mode == modeOCaml {
		if !strings.Contains(e.String(), ";;") {
			e.mode = modeStandardML
		}
	}
}
