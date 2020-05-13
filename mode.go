package main

import (
	"path/filepath"
	"strings"
)

// detectFileMode looks at the filename and tries to guess what could be an appropriate editor mode.
// This mainly affects syntax highlighting (which can be toggled with ctrl-t) and indentation.
func detectEditorMode(filename string) (Mode, bool) {

	// A list of the most common configuration filenames that does not have an extension
	var (
		configFilenames = []string{"fstab", "config", "BUILD", "WORKSPACE", "passwd", "group", "environment", "shadow", "gshadow", "hostname", "hosts", "issue"}
		mode            Mode
	)

	baseFilename := filepath.Base(filename)
	ext := filepath.Ext(baseFilename)

	// Check if we should be in a particular mode for a particular type of file
	switch {
	case baseFilename == "COMMIT_EDITMSG" ||
		baseFilename == "MERGE_MSG" ||
		(strings.HasPrefix(baseFilename, "git-") &&
			!strings.Contains(baseFilename, ".") &&
			strings.Count(baseFilename, "-") >= 2):
		// Git mode
		mode = modeGit
	case strings.HasSuffix(filename, ".git/config") || ext == "ini":
		mode = modeConfig
	case ext == ".sh" || ext == ".ksh" || ext == ".tcsh" || ext == ".bash" || ext == ".zsh" || baseFilename == "PKGBUILD" || (strings.HasPrefix(baseFilename, ".") && strings.Contains(baseFilename, "sh")): // This last part covers .bashrc, .zshrc etc
		mode = modeShell
	case ext == ".yml" || ext == ".toml" || ext == ".ini" || strings.HasSuffix(filename, ".git/config") || (ext == "" && (strings.HasSuffix(baseFilename, "file") || strings.HasSuffix(baseFilename, "rc") || hasS(configFilenames, baseFilename))):
		mode = modeConfig
	case baseFilename == "Makefile" || baseFilename == "makefile" || baseFilename == "GNUmakefile":
		mode = modeMakefile
	default:
		switch ext {
		case ".asm", ".S", ".s", ".inc":
			mode = modeAssembly
		case ".go":
			mode = modeGo
		case ".hs":
			mode = modeHaskell
		case ".ml":
			mode = modeOCaml
		case ".py":
			mode = modePython
		case ".md":
			// Markdown mode
			mode = modeMarkdown
		case ".adoc", ".rst", ".scdoc", ".scd":
			// Markdown-like syntax highlighting
			// TODO: Introduce a separate mode for these.
			mode = modeMarkdown
		case ".txt", ".text", ".nfo", ".diz":
			mode = modeText
		default:
			mode = modeBlank
		}
	}
	// Check if we should enable syntax highlighting by default
	syntaxHighlightingEnabled := (mode != modeBlank || ext != "") && mode != modeText

	return mode, syntaxHighlightingEnabled
}
