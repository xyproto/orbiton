package main

import (
	"path/filepath"
	"strings"
)

// Supporting PHP and Perl are non-goals.

const (
	// Mode "enum"
	modeBlank        = iota
	modeGit          // for git commits and interactive rebases
	modeMarkdown     // for Markdown (and asciidoctor and rst files)
	modeMakefile     // for Makefiles
	modeShell        // for shell scripts and PKGBUILD files
	modeConfig       // for yml, toml, and ini files etc
	modeAssembly     // for Assembly files
	modeGo           // for Go
	modeHaskell      // for Haskell
	modeOCaml        // for OCaml
	modeStandardML   // for Standard ML
	modePython       // for Python
	modeText         // for plain text documents
	modeCMake        // for CMake files
	modeVim          // for Vim or NeoVim configuration, or .vim scripts
	modeLisp         // for Common Lisp, Emacs Lisp and Clojure
	modeZig          // for Zig
	modeKotlin       // for Kotlin
	modeJava         // for Java
	modeHIDL         // for the Android-related Hardware Abstraction Layer Interface Definition Language
	modeSQL          // for Structured Query Language
	modeOak          // for Oak
	modeRust         // for Rust
	modeLua          // for Lua
	modeCrystal      // for Crystal
	modeNim          // for Nim
	modeObjectPascal // for Object Pascal and Delphi
	modeBat          // for DOS batch files
	modeCpp          // for C++
	modeC            // for C
	modeAda          // For Ada
)

// Mode is a per-filetype mode, like for Markdown
type Mode int

// detectFileMode looks at the filename and tries to guess what could be an appropriate editor mode.
// This mainly affects syntax highlighting (which can be toggled with ctrl-t) and indentation.
func detectEditorMode(filename string) (Mode, bool) {

	// A list of the most common configuration filenames that does not have an extension
	var (
		configFilenames = []string{"fstab", "config", "BUILD", "WORKSPACE", "passwd", "group", "environment", "shadow", "gshadow", "hostname", "hosts", "issue", "mirrorlist"}
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
	case ext == ".vimrc" || ext == ".vim" || ext == ".nvim":
		mode = modeVim
	case strings.HasPrefix(baseFilename, "Makefile") || strings.HasPrefix(baseFilename, "makefile") || baseFilename == "GNUmakefile":
		// NOTE: This one MUST come before the ext == "" check below!
		mode = modeMakefile
	case strings.HasSuffix(filename, ".git/config") || ext == ".ini" || ext == ".cfg" || ext == ".conf" || ext == ".service" || ext == ".target" || ext == ".socket" || strings.HasPrefix(ext, "rc"):
		fallthrough
	case ext == ".yml" || ext == ".toml" || ext == ".ini" || strings.HasSuffix(filename, ".git/config") || (ext == "" && (strings.HasSuffix(baseFilename, "file") || strings.HasSuffix(baseFilename, "rc") || hasS(configFilenames, baseFilename))):
		mode = modeConfig
	case ext == ".sh" || ext == ".ksh" || ext == ".tcsh" || ext == ".bash" || ext == ".zsh" || ext == ".local" || ext == ".profile" || baseFilename == "PKGBUILD" || (strings.HasPrefix(baseFilename, ".") && strings.Contains(baseFilename, "sh")): // This last part covers .bashrc, .zshrc etc
		mode = modeShell
	case baseFilename == "CMakeLists.txt" || ext == ".cmake":
		mode = modeCMake
	default:
		switch ext {
		case ".asm", ".S", ".s", ".inc":
			mode = modeAssembly
		case ".go":
			mode = modeGo
		case ".hs":
			mode = modeHaskell
		case ".sml":
			mode = modeStandardML
		case ".ml":
			mode = modeOCaml // or standard ML, if the file does not contain ";;"
		case ".py":
			mode = modePython
		case ".md":
			// Markdown mode
			mode = modeMarkdown
		case ".cpp", ".cc", ".c++", ".cxx", ".hpp", ".h":
			// C++ mode
			// TODO: Find a way to discover is a .h file is most likely to be C or C++
			mode = modeCpp
		case ".c":
			// C mode
			mode = modeC
		case ".adoc", ".rst", ".scdoc", ".scd":
			// Markdown-like syntax highlighting
			// TODO: Introduce a separate mode for these.
			mode = modeMarkdown
		case ".txt", ".text", ".nfo", ".diz":
			mode = modeText
		case ".lsp", ".emacs", ".el", ".elisp", ".clojure", ".clj", ".lisp", ".cl", ".l":
			mode = modeLisp
		case ".zig", ".zir":
			mode = modeZig
		case ".kt", ".kts":
			mode = modeKotlin
		case ".java", ".gradle":
			mode = modeJava
		case ".hal":
			mode = modeHIDL
		case ".sql":
			mode = modeSQL
		case ".ok":
			mode = modeOak
		case ".rs":
			mode = modeRust
		case ".lua":
			mode = modeLua
		case ".cr":
			mode = modeCrystal
		case ".nim":
			mode = modeNim
		case ".pas", ".pp", ".lpr":
			mode = modeObjectPascal
		case ".bat":
			mode = modeBat
		case ".adb", ".gpr", ".ads", ".ada":
			mode = modeAda
		default:
			mode = modeBlank
		}
	}

	// TODO: Find all instances that checks if mode if modeBlank in the code, then introduce modeText
	if mode == modeText {
		mode = modeBlank
	}

	// If the mode is not set and the filename is all uppercase and no ".", use modeMarkdown
	if mode == modeBlank && !strings.Contains(baseFilename, ".") && baseFilename == strings.ToUpper(baseFilename) {
		mode = modeMarkdown
	}

	// Check if we should enable syntax highlighting by default
	syntaxHighlightingEnabled := (mode != modeBlank || ext != "") && mode != modeText

	return mode, syntaxHighlightingEnabled
}
