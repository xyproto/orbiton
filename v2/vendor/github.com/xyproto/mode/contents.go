package mode

import (
	"bytes"
	"path/filepath"
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
		if (bytes.Contains(data, []byte("·")) || (bytes.Contains(data, []byte("TEXT")) && bytes.Contains(data, []byte("RET\n")))) && !lookslikegoasm.SeveralSemicolonComments(string(data)) {
			return GoAssembly, true
		}
	}
	if bytes.HasPrefix(firstLine, []byte("#!")) { // The line starts with a shebang
		if m, ok := shebangMode(shebangInterpreter(firstLine)); ok {
			return m, true
		}
		notConfig = true
	} else if bytes.HasPrefix(firstLine, []byte("# $")) {
		// Most likely a csh script on FreeBSD
		return Shell, true
	} else if bytes.HasPrefix(firstLine, []byte("<?xml ")) {
		return XML, true
	} else if bytes.HasPrefix(firstLine, []byte("{\"")) {
		return JSON, true
	} else if bytes.HasPrefix(firstLine, []byte("{\\rtf")) {
		return RTF, true
	} else if bytes.HasPrefix(firstLine, []byte("WordGrinder")) {
		return WordGrinder, true
	} else if bytes.Contains(firstLine, []byte("<abiword")) {
		return Abiword, true
	} else if bytes.HasPrefix(firstLine, []byte("PK\x03\x04")) && bytes.Contains(firstLine, []byte("mimetypeapplication/vnd.oasis.opendocument")) {
		return LibreOffice, true
	} else if bytes.HasPrefix(bytes.ToLower(firstLine), []byte("<!doctype html")) || bytes.HasPrefix(bytes.ToLower(firstLine), []byte("<html")) {
		return HTML, true
	} else if bytes.Contains(firstLine, []byte("-*- nroff -*-")) {
		return Nroff, true
	} else if !bytes.HasPrefix(firstLine, []byte("//")) && !bytes.HasPrefix(firstLine, []byte("#")) && bytes.Count(bytes.TrimSpace(firstLine), []byte(" ")) > 10 && bytes.HasSuffix(firstLine, []byte(")")) && looksLikeManPageHeader(firstLine) {
		return ManPage, true
	} else if bytes.HasPrefix(firstLine, []byte("\" ")) {
		// The first line starts with '" ', assume ViM script
		return Vim, true
	} else if bytes.HasPrefix(firstLine, []byte("diff -")) {
		return Diff, true
	} else if bytes.HasPrefix(firstLine, []byte("%YAML ")) {
		return YAML, true
	} else if bytes.Equal(bytes.TrimSpace(firstLine), []byte("---")) && (m == YAML || m == Blank) {
		return YAML, true
	} else if bytes.HasPrefix(firstLine, []byte(`syntax = "proto`)) {
		return Protobuf, true
	} else if bytes.HasPrefix(firstLine, []byte("terraform {")) || bytes.HasPrefix(firstLine, []byte(`resource "`)) || bytes.HasPrefix(firstLine, []byte(`variable "`)) || bytes.HasPrefix(firstLine, []byte(`provider "`)) {
		return HCL, true
	} else if bytes.HasPrefix(firstLine, []byte(`amends "`)) {
		return Pkl, true
	} else if bytes.HasPrefix(firstLine, []byte("@vertex")) || bytes.HasPrefix(firstLine, []byte("@fragment")) || bytes.HasPrefix(firstLine, []byte("@compute")) {
		return WGSL, true
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
		prevLineWasText := false
		for _, line := range byteLines {
			if bytes.HasPrefix(line, []byte("# ")) {
				hashComment++
			} else if bytes.HasPrefix(line, []byte("/")) { // Count all lines starting with "/" as a comment, for this purpose
				slashComment++
			}
			trimmedLine := bytes.TrimSpace(line)
			if len(trimmedLine) > 1 && ((trimmedLine[0] == byte(':') && trimmedLine[1] == byte(':')) || bytes.HasPrefix(trimmedLine, []byte(".. ")) || bytes.HasPrefix(trimmedLine, []byte("[source,"))) {
				reStructuredTextMarkers++
				if reStructuredTextMarkers == 2 {
					return ReStructured, true
				}
			} else if rstAdornment(trimmedLine) {
				reStructuredTextMarkers++
				// An adornment line following a text line is a strong RST title indicator
				if prevLineWasText {
					reStructuredTextMarkers++
				}
				if reStructuredTextMarkers >= 2 {
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
			prevLineWasText = len(trimmedLine) > 0 && !rstAdornment(trimmedLine)
		}
		// If non-comment lines look like shell commands, prefer Shell over Config.
		// Handles the common case of typing "# comment" + "ls -al" into a buffer
		// without a shebang or filename extension.
		if (m == Blank || m == Config) && looksLikeShell(byteLines) {
			return Shell, true
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

// shebangInterpreter returns the name of the interpreter in a shebang line,
// like "python3" for "#!/usr/bin/env -S python3 -u" or "bash" for "#!/bin/bash -e"
func shebangInterpreter(firstLine []byte) string {
	fields := strings.Fields(string(bytes.TrimPrefix(firstLine, []byte("#!"))))
	if len(fields) == 0 {
		return ""
	}
	interpreter := filepath.Base(fields[0])
	if interpreter == "env" {
		interpreter = ""
		for _, field := range fields[1:] {
			if !strings.HasPrefix(field, "-") && !strings.Contains(field, "=") {
				interpreter = filepath.Base(field)
				break
			}
		}
	}
	return interpreter
}

// shebangMode returns the Mode for the given shebang interpreter name, if it is known
func shebangMode(interpreter string) (Mode, bool) {
	switch interpreter {
	case "ash", "bash", "csh", "dash", "ksh", "mksh", "osh", "sh", "tcsh", "zsh":
		return Shell, true
	case "fish":
		return Fish, true
	case "oil", "ysh":
		return Oil, true
	case "nu":
		return Nushell, true
	case "node", "nodejs":
		return JavaScript, true
	}
	for _, prefix := range []struct {
		name string
		mode Mode
	}{{"perl", Perl}, {"python", Python}, {"ruby", Ruby}, {"lua", Lua}, {"php", PHP}} {
		if strings.HasPrefix(interpreter, prefix.name) {
			return prefix.mode, true
		}
	}
	return Blank, false
}

// looksLikeManPageHeader checks if the first word of the given line looks like "LS(1)" or "printf(3p)"
func looksLikeManPageHeader(firstLine []byte) bool {
	fields := bytes.Fields(firstLine)
	if len(fields) == 0 {
		return false
	}
	name, section, ok := bytes.Cut(fields[0], []byte("("))
	return ok && len(name) > 0 && len(section) > 1 && bytes.IndexByte(section, ')') == len(section)-1
}

// looksLikeShell returns true if the content has the typical shape of a shell
// script even without a shebang: at least one '#' comment line followed (after
// any blank lines) by a line whose first word is made up of [a-zA-Z0-9./] and
// is separated from the rest by a space — the "command arg ..." pattern.
// Lines that look like "key = value" assignments are rejected as config-like.
func looksLikeShell(byteLines [][]byte) bool {
	sawHashComment := false
	for _, line := range byteLines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		if trimmed[0] == '#' {
			sawHashComment = true
			continue
		}
		if !sawHashComment {
			return false
		}
		spaceIdx := bytes.IndexByte(trimmed, ' ')
		if spaceIdx <= 0 {
			return false
		}
		for _, c := range trimmed[:spaceIdx] {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '/') {
				return false
			}
		}
		rest := bytes.TrimLeft(trimmed[spaceIdx+1:], " \t")
		if len(rest) == 0 || rest[0] == '=' {
			return false
		}
		return true
	}
	return false
}

// rstAdornment checks if a line consists entirely of one repeated RST
// adornment character and is at least 3 characters long. This helps detect
// reStructuredText section title underlines/overlines.
func rstAdornment(trimmedLine []byte) bool {
	if len(trimmedLine) < 3 {
		return false
	}
	ch := trimmedLine[0]
	if !strings.ContainsRune("=-~^\"'`#*+._", rune(ch)) {
		return false
	}
	for i := 1; i < len(trimmedLine); i++ {
		if trimmedLine[i] != ch {
			return false
		}
	}
	return true
}
