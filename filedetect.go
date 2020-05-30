package main

import (
	"path/filepath"
	"strings"
)

const (
	// Mode "enum"
	modeBlank      = iota
	modeGit        // for git commits and interactive rebases
	modeMarkdown   // for Markdown (and asciidoctor and rst files)
	modeMakefile   // for Makefiles
	modeShell      // for shell scripts and PKGBUILD files
	modeConfig     // for yml, toml, and ini files etc
	modeAssembly   // for Assembly files
	modeGo         // for Go source files
	modeHaskell    // for Haskell source files
	modeOCaml      // for OCaml source files
	modeStandardML // for Standard ML source files
	modePython     // for Python source files
	modeText       // for plain text documents
	modeCMake      // for CMake files
	modeVim        // for Vim or NeoVim configuration, or .vim scripts
	modeLisp       // for Common Lisp, Emacs Lisp and Clojure
	modeZig        // for Zig
)

// Mode is a per-filetype mode, like for Markdown
type Mode int

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
	case ext == ".vimrc" || ext == ".vim" || ext == ".nvim":
		mode = modeVim
	case strings.HasSuffix(filename, ".git/config") || ext == "ini" || ext == "cfg" || ext == "conf" || strings.HasPrefix(ext, "rc"):
		mode = modeConfig
	case ext == ".sh" || ext == ".ksh" || ext == ".tcsh" || ext == ".bash" || ext == ".zsh" || baseFilename == "PKGBUILD" || (strings.HasPrefix(baseFilename, ".") && strings.Contains(baseFilename, "sh")): // This last part covers .bashrc, .zshrc etc
		mode = modeShell
	case ext == ".yml" || ext == ".toml" || ext == ".ini" || strings.HasSuffix(filename, ".git/config") || (ext == "" && (strings.HasSuffix(baseFilename, "file") || strings.HasSuffix(baseFilename, "rc") || hasS(configFilenames, baseFilename))):
		mode = modeConfig
	case baseFilename == "Makefile" || baseFilename == "makefile" || baseFilename == "GNUmakefile":
		mode = modeMakefile
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
