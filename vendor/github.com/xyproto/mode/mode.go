package mode

import (
	"path/filepath"
	"strings"
)

// Mode is a per-filetype mode, like for Markdown
type Mode int

const (
	// Mode "enum" values
	Blank          = iota
	Git            // Git commits and interactive rebases
	Markdown       // Markdown (and asciidoctor and rst files)
	Makefile       // Makefiles
	Shell          // Shell scripts and PKGBUILD files
	Config         // Config like yaml, yml, toml, and ini files
	Assembly       // Assembly
	GoAssembly     // Go-style Assembly
	Go             // Go
	Haskell        // Haskell
	OCaml          // OCaml
	StandardML     // Standard ML
	Python         // Python
	Text           // plain text documents
	CMake          // CMake files
	Vim            // Vim or NeoVim configuration, or .vim scripts
	V              // V programming language
	Clojure        // Clojure
	Lisp           // Common Lisp and Emacs Lisp
	Zig            // Zig
	Kotlin         // Kotlin
	Java           // Java
	HIDL           // the Android-related Hardware Abstraction Layer Interface Definition Language
	SQL            // Structured Query Language
	Oak            // Oak
	Rust           // Rust
	Lua            // Lua
	Crystal        // Crystal
	Nim            // Nim
	ObjectPascal   // Object Pascal and Delphi
	Bat            // DOS batch files
	Cpp            // C++
	C              // C
	Ada            // Ada
	HTML           // HTML
	Odin           // Odin
	XML            // XML
	PolicyLanguage // SE Linux configuration files
	Nroff          // editing man pages
	Scala          // Scala
	JSON           // JSON and iPython notebooks
	Battlestar     // Battlestar
	CS             // C#
	JavaScript     // JavaScript
	TypeScript     // TypeScript
	ManPage        // viewing man pages
	Amber          // Amber templates
	Bazel          // Bazel and Starlark
	D              // D
	Perl           // Perl
	M4             // M4 macros
)

// Detect looks at the filename and tries to guess what could be an appropriate editor mode.
// This mainly affects syntax highlighting (which can be toggled with ctrl-t) and indentation.
func Detect(filename string) (Mode, bool) {

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
		mode = Git
	case ext == ".vimrc" || ext == ".vim" || ext == ".nvim":
		mode = Vim
	case strings.HasPrefix(baseFilename, "Makefile") || strings.HasPrefix(baseFilename, "makefile") || baseFilename == "GNUmakefile":
		// NOTE: This one MUST come before the ext == "" check below!
		mode = Makefile
	case strings.HasSuffix(filename, ".git/config") || ext == ".ini" || ext == ".cfg" || ext == ".conf" || ext == ".service" || ext == ".target" || ext == ".socket" || strings.HasPrefix(ext, "rc"):
		fallthrough
	case ext == ".yml" || ext == ".toml" || ext == ".ini" || ext == ".bp" || strings.HasSuffix(filename, ".git/config") || (ext == "" && (strings.HasSuffix(baseFilename, "file") || strings.HasSuffix(baseFilename, "rc") || hasS(configFilenames, baseFilename))):
		mode = Config
	case ext == ".sh" || ext == ".ksh" || ext == ".tcsh" || ext == ".bash" || ext == ".zsh" || ext == ".local" || ext == ".profile" || baseFilename == "PKGBUILD" || (strings.HasPrefix(baseFilename, ".") && strings.Contains(baseFilename, "sh")): // This last part covers .bashrc, .zshrc etc
		mode = Shell
	case ext == ".bzl" || baseFilename == "BUILD" || baseFilename == "WORKSPACE":
		mode = Bazel
	case baseFilename == "CMakeLists.txt" || ext == ".cmake":
		mode = CMake
	default:
		switch ext {
		case ".s", ".S", ".asm", ".inc":
			// Go-style assembly (modeGoAssembly) is enabled if a mid-dot is discovered
			mode = Assembly
		//case ".s":
		//mode = GoAssembly
		case ".amber":
			mode = Amber
		case ".go":
			mode = Go
		case ".odin":
			mode = Odin
		case ".hs":
			mode = Haskell
		case ".sml":
			mode = StandardML
		case ".m4":
			mode = M4
		case ".ml":
			mode = OCaml // or standard ML, if the file does not contain ";;"
		case ".py":
			mode = Python
		case ".pl":
			mode = Perl
		case ".md":
			// Markdown mode
			mode = Markdown
		case ".bts":
			mode = Battlestar
		case ".cpp", ".cc", ".c++", ".cxx", ".hpp", ".h":
			// C++ mode
			// TODO: Find a way to discover is a .h file is most likely to be C or C++
			mode = Cpp
		case ".c":
			// C mode
			mode = C
		case ".d":
			// D mode
			mode = D
		case ".cs":
			// C# mode
			mode = CS
		case ".adoc", ".rst", ".scdoc", ".scd":
			// Markdown-like syntax highlighting
			// TODO: Introduce a separate mode for these.
			mode = Markdown
		case ".txt", ".text", ".nfo", ".diz":
			mode = Text
		case ".clj", ".clojure", "cljs":
			mode = Clojure
		case ".lsp", ".emacs", ".el", ".elisp", ".lisp", ".cl", ".l":
			mode = Lisp
		case ".zig", ".zir":
			mode = Zig
		case ".v":
			mode = V
		case ".kt", ".kts":
			mode = Kotlin
		case ".java", ".gradle":
			mode = Java
		case ".hal":
			mode = HIDL
		case ".sql":
			mode = SQL
		case ".ok":
			mode = Oak
		case ".rs":
			mode = Rust
		case ".lua":
			mode = Lua
		case ".cr":
			mode = Crystal
		case ".nim":
			mode = Nim
		case ".pas", ".pp", ".lpr":
			mode = ObjectPascal
		case ".bat":
			mode = Bat
		case ".adb", ".gpr", ".ads", ".ada":
			mode = Ada
		case ".htm", ".html":
			mode = HTML
		case ".xml":
			mode = XML
		case ".te":
			mode = PolicyLanguage
		case ".1", ".2", ".3", ".4", ".5", ".6", ".7", ".8":
			mode = Nroff
		case ".scala":
			mode = Scala
		case ".json", ".ipynb":
			mode = JSON
		case ".js":
			mode = JavaScript
		case ".ts":
			mode = TypeScript
		default:
			mode = Blank
		}
	}

	if mode == Text {
		mode = Markdown
	}

	// If the mode is not set and the filename is all uppercase and no ".", use modeMarkdown
	if mode == Blank && !strings.Contains(baseFilename, ".") && baseFilename == strings.ToUpper(baseFilename) {
		mode = Markdown
	}

	// Check if we should enable syntax highlighting by default
	syntaxHighlightingEnabled := (mode != Blank || ext != "") && mode != Text

	return mode, syntaxHighlightingEnabled
}

// String will return a short lowercase string representing the given editor mode
func (mode Mode) String() string {
	switch mode {
	case Blank:
		return "-"
	case Git:
		return "Git"
	case Markdown:
		return "Markdown"
	case Makefile:
		return "Make"
	case Shell:
		return "Shell"
	case Config:
		return "Configuration"
	case Assembly:
		return "Assembly"
	case GoAssembly:
		return "Go-style Assembly"
	case Go:
		return "Go"
	case Haskell:
		return "Haskell"
	case OCaml:
		return "Ocaml"
	case StandardML:
		return "Standard ML"
	case Python:
		return "Python"
	case Text:
		return "Text"
	case CMake:
		return "Cmake"
	case Vim:
		return "ViM"
	case Clojure:
		return "Clojure"
	case Lisp:
		return "Lisp"
	case Zig:
		return "Zig"
	case Kotlin:
		return "Kotlin"
	case Java:
		return "Java"
	case HIDL:
		return "HIDL"
	case SQL:
		return "SQL"
	case Oak:
		return "Oak"
	case Rust:
		return "Rust"
	case Lua:
		return "Lua"
	case Crystal:
		return "Crystal"
	case Nim:
		return "Nim"
	case ObjectPascal:
		return "Pas"
	case Bat:
		return "Bat"
	case Cpp:
		return "C++"
	case C:
		return "C"
	case Ada:
		return "Ada"
	case HTML:
		return "HTML"
	case Odin:
		return "Odin"
	case Perl:
		return "Perl"
	case XML:
		return "XML"
	case PolicyLanguage:
		return "SELinux"
	case Nroff:
		return "Nroff"
	case Scala:
		return "Scala"
	case JSON:
		return "JSON"
	case Battlestar:
		return "Battlestar"
	case CS:
		return "C#"
	case TypeScript:
		return "TypeScript"
	case JavaScript:
		return "JavaScript"
	case ManPage:
		return "Man"
	case Amber:
		return "Amber"
	case Bazel:
		return "Bazel"
	case D:
		return "D"
	case V:
		return "V"
	case M4:
		return "M4"
	default:
		return "?"
	}
}
