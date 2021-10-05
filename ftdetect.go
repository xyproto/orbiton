package main

import (
	"path/filepath"
	"strings"
)

const (
	// Mode "enum" values
	modeBlank          = iota
	modeGit            // for git commits and interactive rebases
	modeMarkdown       // for Markdown (and asciidoctor and rst files)
	modeMakefile       // for Makefiles
	modeShell          // for shell scripts and PKGBUILD files
	modeConfig         // for yml, toml, and ini files etc
	modeAssembly       // for Assembly
	modeGoAssembly     // for Go-style Assembly
	modeGo             // for Go
	modeHaskell        // for Haskell
	modeOCaml          // for OCaml
	modeStandardML     // for Standard ML
	modePython         // for Python
	modeText           // for plain text documents
	modeCMake          // for CMake files
	modeVim            // for Vim or NeoVim configuration, or .vim scripts
	modeV              // the V programming language
	modeClojure        // for Clojure
	modeLisp           // for Common Lisp and Emacs Lisp
	modeZig            // for Zig
	modeKotlin         // for Kotlin
	modeJava           // for Java
	modeHIDL           // for the Android-related Hardware Abstraction Layer Interface Definition Language
	modeSQL            // for Structured Query Language
	modeOak            // for Oak
	modeRust           // for Rust
	modeLua            // for Lua
	modeCrystal        // for Crystal
	modeNim            // for Nim
	modeObjectPascal   // for Object Pascal and Delphi
	modeBat            // for DOS batch files
	modeCpp            // for C++
	modeC              // for C
	modeAda            // for Ada
	modeHTML           // for HTML
	modeOdin           // for Odin
	modeXML            // for XML
	modePolicyLanguage // for SE Linux configuration files
	modeNroff          // for editing man pages
	modeScala          // for Scala
	modeJSON           // for JSON and iPython notebooks
	modeBattlestar     // for Battlestar
	modeCS             // for C#
	modeJavaScript     // for JavaScript
	modeTypeScript     // for TypeScript
	modeManPage        // for viewing man pages
	modeAmber          // for Amber templates
	modeBazel          // for Bazel and Starlark
	modeD              // for D
	modePerl           // for Perl
	modeM4             // for M4 macros
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
	case ext == ".yml" || ext == ".toml" || ext == ".ini" || ext == ".bp" || strings.HasSuffix(filename, ".git/config") || (ext == "" && (strings.HasSuffix(baseFilename, "file") || strings.HasSuffix(baseFilename, "rc") || hasS(configFilenames, baseFilename))):
		mode = modeConfig
	case ext == ".sh" || ext == ".ksh" || ext == ".tcsh" || ext == ".bash" || ext == ".zsh" || ext == ".local" || ext == ".profile" || baseFilename == "PKGBUILD" || (strings.HasPrefix(baseFilename, ".") && strings.Contains(baseFilename, "sh")): // This last part covers .bashrc, .zshrc etc
		mode = modeShell
	case ext == ".bzl" || baseFilename == "BUILD" || baseFilename == "WORKSPACE":
		mode = modeBazel
	case baseFilename == "CMakeLists.txt" || ext == ".cmake":
		mode = modeCMake
	default:
		switch ext {
		case ".s", ".S", ".asm", ".inc":
			// Go-style assembly (modeGoAssembly) is enabled if a mid-dot is discovered
			mode = modeAssembly
		//case ".s":
		//mode = modeGoAssembly
		case ".amber":
			mode = modeAmber
		case ".go":
			mode = modeGo
		case ".odin":
			mode = modeOdin
		case ".hs":
			mode = modeHaskell
		case ".sml":
			mode = modeStandardML
		case ".m4":
			mode = modeM4
		case ".ml":
			mode = modeOCaml // or standard ML, if the file does not contain ";;"
		case ".py":
			mode = modePython
		case ".pl":
			mode = modePerl
		case ".md":
			// Markdown mode
			mode = modeMarkdown
		case ".bts":
			mode = modeBattlestar
		case ".cpp", ".cc", ".c++", ".cxx", ".hpp", ".h":
			// C++ mode
			// TODO: Find a way to discover is a .h file is most likely to be C or C++
			mode = modeCpp
		case ".c":
			// C mode
			mode = modeC
		case ".d":
			// D mode
			mode = modeD
		case ".cs":
			// C# mode
			mode = modeCS
		case ".adoc", ".rst", ".scdoc", ".scd":
			// Markdown-like syntax highlighting
			// TODO: Introduce a separate mode for these.
			mode = modeMarkdown
		case ".txt", ".text", ".nfo", ".diz":
			mode = modeText
		case ".clj", ".clojure", "cljs":
			mode = modeClojure
		case ".lsp", ".emacs", ".el", ".elisp", ".lisp", ".cl", ".l":
			mode = modeLisp
		case ".zig", ".zir":
			mode = modeZig
		case ".v":
			mode = modeV
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
		case ".htm", ".html":
			mode = modeHTML
		case ".xml":
			mode = modeXML
		case ".te":
			mode = modePolicyLanguage
		case ".1", ".2", ".3", ".4", ".5", ".6", ".7", ".8":
			mode = modeNroff
		case ".scala":
			mode = modeScala
		case ".json", ".ipynb":
			mode = modeJSON
		case ".js":
			mode = modeJavaScript
		case ".ts":
			mode = modeTypeScript
		default:
			mode = modeBlank
		}
	}

	if mode == modeText {
		mode = modeMarkdown
	}

	// If the mode is not set and the filename is all uppercase and no ".", use modeMarkdown
	if mode == modeBlank && !strings.Contains(baseFilename, ".") && baseFilename == strings.ToUpper(baseFilename) {
		mode = modeMarkdown
	}

	// Check if we should enable syntax highlighting by default
	syntaxHighlightingEnabled := (mode != modeBlank || ext != "") && mode != modeText

	return mode, syntaxHighlightingEnabled
}

// String will return a short lowercase string representing the given editor mode
func (mode Mode) String() string {
	switch mode {
	case modeBlank:
		return "-"
	case modeGit:
		return "Git"
	case modeMarkdown:
		return "Markdown"
	case modeMakefile:
		return "Make"
	case modeShell:
		return "Shell"
	case modeConfig:
		return "Configuration"
	case modeAssembly:
		return "Assembly"
	case modeGoAssembly:
		return "Go-style Assembly"
	case modeGo:
		return "Go"
	case modeHaskell:
		return "Haskell"
	case modeOCaml:
		return "Ocaml"
	case modeStandardML:
		return "Standard ML"
	case modePython:
		return "Python"
	case modeText:
		return "Text"
	case modeCMake:
		return "Cmake"
	case modeVim:
		return "ViM"
	case modeClojure:
		return "Clojure"
	case modeLisp:
		return "Lisp"
	case modeZig:
		return "Zig"
	case modeKotlin:
		return "Kotlin"
	case modeJava:
		return "Java"
	case modeHIDL:
		return "HIDL"
	case modeSQL:
		return "SQL"
	case modeOak:
		return "Oak"
	case modeRust:
		return "Rust"
	case modeLua:
		return "Lua"
	case modeCrystal:
		return "Crystal"
	case modeNim:
		return "Nim"
	case modeObjectPascal:
		return "Pas"
	case modeBat:
		return "Bat"
	case modeCpp:
		return "C++"
	case modeC:
		return "C"
	case modeAda:
		return "Ada"
	case modeHTML:
		return "HTML"
	case modeOdin:
		return "Odin"
	case modePerl:
		return "Perl"
	case modeXML:
		return "XML"
	case modePolicyLanguage:
		return "SELinux"
	case modeNroff:
		return "Nroff"
	case modeScala:
		return "Scala"
	case modeJSON:
		return "JSON"
	case modeBattlestar:
		return "Battlestar"
	case modeCS:
		return "C#"
	case modeTypeScript:
		return "TypeScript"
	case modeJavaScript:
		return "JavaScript"
	case modeManPage:
		return "Man"
	case modeAmber:
		return "Amber"
	case modeBazel:
		return "Bazel"
	case modeD:
		return "D"
	case modeV:
		return "V"
	case modeM4:
		return "M4"
	default:
		return "?"
	}
}
