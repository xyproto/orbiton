package mode

// Mode is a per-filetype mode, like for Markdown
type Mode int

const (
	Blank          = iota // No file mode found
	AIDL                  // Android-related: Android Interface Definition Language
	Ada                   // Ada
	Agda                  // Agda
	Amber                 // Amber templates
	Assembly              // Assembly
	Basic                 // FreeBasic, Gambas 3
	Bat                   // DOS and Windows batch files
	Battlestar            // Battlestar
	Bazel                 // Bazel and Starlark
	C                     // C
	CMake                 // CMake files
	CS                    // C#
	Clojure               // Clojure
	Config                // Config like yaml, yml, toml, and ini files
	Cpp                   // C++
	Crystal               // Crystal
	D                     // D
	Doc                   // asciidoctor, sdoc etc
	Email                 // For using o with ie. Mutt
	Elm                   // Elm
	Erlang                // Erlang
	Garnet                // Garnet
	GDScript              // Godot Script
	Git                   // Git commits and interactive rebases
	Go                    // Go
	GoAssembly            // Go-style Assembly
	Gradle                // Gradle
	Haxe                  // Haxe: .hx and .hxml files
	HIDL                  // Android-related: Hardware Abstraction Layer Interface Definition Language
	HTML                  // HTML
	Hare                  // Hare
	Haskell               // Haskell
	Ivy                   // Ivy
	JSON                  // JSON and iPython notebooks
	Jakt                  // Jakt
	Java                  // Java
	JavaScript            // JavaScript
	Koka                  // Koka
	Kotlin                // Kotlin
	Lisp                  // Common Lisp and Emacs Lisp
	Log                   // All sorts of log files
	Lua                   // Lua
	M4                    // M4 macros
	Make                  // Makefiles
	ManPage               // viewing man pages
	Markdown              // Markdown
	Nim                   // Nim
	Nroff                 // editing man pages
	OCaml                 // OCaml
	Oak                   // Oak
	ObjectPascal          // Object Pascal and Delphi
	Odin                  // Odin
	Perl                  // Perl
	PolicyLanguage        // SE Linux configuration files
	Prolog                // Prolog
	Python                // Python
	R                     // R
	ReStructured          // reStructuredText
	Rust                  // Rust
	Scala                 // Scala
	Shader                // GLSL Shader
	Shell                 // Shell scripts and PKGBUILD files
	StandardML            // Standard ML
	SQL                   // Structured Query Language
	Teal                  // Teal
	Terra                 // Terra
	Text                  // plain text documents
	TypeScript            // TypeScript
	V                     // V programming language
	Vim                   // Vim or NeoVim configuration, or .vim scripts
	XML                   // XML
	Zig                   // Zig
)

// String will return a short lowercase string representing the given editor mode
func (mode Mode) String() string {
	// TODO: Sort the cases alphabetically
	// TODO: Add a test that makes sure every mode has a string
	switch mode {
	case Ada:
		return "Ada"
	case Agda:
		return "Agda"
	case AIDL:
		return "AIDL"
	case Amber:
		return "Amber"
	case Assembly:
		return "Assembly"
	case Basic:
		return "Basic"
	case Bat:
		return "Bat"
	case Battlestar:
		return "Battlestar"
	case Bazel:
		return "Bazel"
	case Blank:
		return "-"
	case Clojure:
		return "Clojure"
	case CMake:
		return "CMake"
	case Config:
		return "Configuration"
	case Cpp:
		return "C++"
	case C:
		return "C"
	case Crystal:
		return "Crystal"
	case CS:
		return "C#"
	case Doc:
		return "Document"
	case D:
		return "D"
	case Elm:
		return "Elm"
	case Email:
		return "E-mail"
	case Erlang:
		return "Erlang"
	case Garnet:
		return "Garnet"
	case GDScript:
		return "Godot Script"
	case Git:
		return "Git"
	case GoAssembly:
		return "Go-style Assembly"
	case Go:
		return "Go"
	case Gradle:
		return "Gradle"
	case Hare:
		return "Hare"
	case Haskell:
		return "Haskell"
	case Haxe:
		return "Haxe"
	case HIDL:
		return "HIDL"
	case HTML:
		return "HTML"
	case Ivy:
		return "Ivy"
	case Jakt:
		return "Jakt"
	case Java:
		return "Java"
	case JavaScript:
		return "JavaScript"
	case JSON:
		return "JSON"
	case Koka:
		return "Koka"
	case Kotlin:
		return "Kotlin"
	case Lisp:
		return "Lisp"
	case Log:
		return "Log"
	case Lua:
		return "Lua"
	case M4:
		return "M4"
	case Make:
		return "Make"
	case ManPage:
		return "Man"
	case Markdown:
		return "Markdown"
	case Nim:
		return "Nim"
	case Nroff:
		return "Nroff"
	case Oak:
		return "Oak"
	case ObjectPascal:
		return "Pas"
	case OCaml:
		return "Ocaml"
	case Odin:
		return "Odin"
	case Perl:
		return "Perl"
	case PolicyLanguage:
		return "SELinux"
	case Prolog:
		return "Prolog"
	case Python:
		return "Python"
	case R:
		return "R"
	case ReStructured:
		return "reStructuredText"
	case Rust:
		return "Rust"
	case Scala:
		return "Scala"
	case Shader:
		return "Shader"
	case Shell:
		return "Shell"
	case SQL:
		return "SQL"
	case StandardML:
		return "Standard ML"
	case Teal:
		return "Teal"
	case Terra:
		return "Terra"
	case Text:
		return "Text"
	case TypeScript:
		return "TypeScript"
	case Vim:
		return "ViM"
	case V:
		return "V"
	case XML:
		return "XML"
	case Zig:
		return "Zig"
	default:
		return "?"
	}
}
