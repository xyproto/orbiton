// Package mode tries to find the correct editor mode, given a filename and/or file data
package mode

// Mode is a per-filetype mode, like for Markdown
type Mode int

const (
	Blank          = iota // Blank is used if no file mode is found
	ABC                   // ABC music notation
	AIDL                  // Android-related: Android Interface Definition Language
	Ada                   // Ada
	Agda                  // Agda
	Algol68               // ALGOL 68
	Amber                 // Amber templates
	Arduino               // Arduino
	ASCIIDoc              // ASCII doc
	Assembly              // Assembly
	Basic                 // FreeBasic, Gambas 3
	Bat                   // DOS and Windows batch files
	Battlestar            // Battlestar
	Bazel                 // Bazel and Starlark
	Beef                  // Beef
	Blueprint             // GNOME Blueprint
	C                     // C
	C3                    // C3
	CMake                 // CMake files
	CS                    // C#
	CSound                // CSound // music
	CSS                   // CSS
	Chuck                 // Chuck // music
	Clojure               // Clojure
	COBOL                 // COBOL
	Config                // Config like yaml, yml, toml, and ini files
	Cpp                   // C++
	Crystal               // Crystal
	D                     // D
	Dart                  // Dart
	Diff                  // Diff / patch
	Dingo                 // Dingo
	Docker                // For Dockerfiles
	Email                 // For using o with ie. Mutt
	Elm                   // Elm
	Erlang                // Erlang
	Faust                 // Faust
	Fortran77             // Fortran 77
	Fortran90             // Fortran 90
	FSharp                // F#
	FSTAB                 // Filesystem table
	Garnet                // Garnet
	GDScript              // Godot Script
	Git                   // Git commits and interactive rebases
	Gleam                 // Gleam
	Go                    // Go
	GoMod                 // go.mod files
	GoAssembly            // Go-style Assembly
	Gradle                // Gradle
	Haxe                  // Haxe: .hx and .hxml files
	HIDL                  // Android-related: Hardware Abstraction Layer Interface Definition Language
	HTML                  // HTML
	HTTP                  // .http files are used by IntelliJ and Visual Studio for testing HTTP services
	Hare                  // Hare
	Haskell               // Haskell
	Ignore                // .gitignore and .ignore files
	Ini                   // INI Configuration
	Inko                  // Inko
	Ivy                   // Ivy
	JSON                  // JSON and iPython notebooks
	Jakt                  // Jakt
	Java                  // Java
	JavaScript            // JavaScript
	Just                  // Just
	Koka                  // Koka
	Kotlin                // Kotlin
	Lilypond              // Lilypond
	Lisp                  // Common Lisp and Emacs Lisp
	Log                   // All sorts of log files
	Lua                   // Lua
	M4                    // M4 macros
	Make                  // Makefiles
	ManPage               // viewing man pages
	Markdown              // Markdown document
	Mojo                  // Mojo
	Nim                   // Nim
	Nix                   // Nix
	Nmap                  // Nmap scripts
	Nroff                 // editing man pages
	OCaml                 // OCaml
	Oak                   // Oak
	ObjC                  // Objective-C
	ObjectPascal          // Object Pascal and Delphi
	Odin                  // Odin
	Ollama                // For Modelfiles
	Perl                  // Perl
	PHP                   // PHP
	PolicyLanguage        // SE Linux configuration files
	POV                   // POV-Ray raytracer
	Prolog                // Prolog
	Python                // Python
	R                     // R
	ReStructured          // reStructuredText
	Ruby                  // Ruby
	Rust                  // Rust
	Scala                 // Scala
	SCDoc                 // SC Doc
	Scheme                // Scheme
	Shader                // GLSL Shader
	Shell                 // Shell scripts and PKGBUILD files
	StandardML            // Standard ML
	Starlark              // Starlark
	SQL                   // Structured Query Language
	Subversion            // Subversion commits
	SuperCollider         // SuperCollider // music
	Swift                 // Swift
	Teal                  // Teal
	Terra                 // Terra
	Text                  // plain text documents
	TypeScript            // TypeScript
	V                     // V programming language
	Vibe67                // Vibe67
	Vim                   // Vim or NeoVim configuration, or .vim scripts
	XML                   // XML
	Zig                   // Zig
)

// String will return a short lowercase string representing the given editor mode
func (mode Mode) String() string {
	// TODO: Sort the cases alphabetically
	// TODO: Add a test that makes sure every mode has a string
	switch mode {
	case ABC:
		return "ABC"
	case Ada:
		return "Ada"
	case Agda:
		return "Agda"
	case Algol68:
		return "ALGOL 68"
	case AIDL:
		return "AIDL"
	case Amber:
		return "Amber"
	case Arduino:
		return "Arduino"
	case ASCIIDoc:
		return "ASCII Doc"
	case Assembly:
		return "Assembly"
	case Basic:
		return "Basic"
	case Bat:
		return "Batch"
	case Battlestar:
		return "Battlestar"
	case Bazel:
		return "Bazel"
	case Beef:
		return "Beef"
	case Blueprint:
		return "Blueprint"
	case Blank:
		return "-"
	case C:
		return "C"
	case C3:
		return "C3"
	case Clojure:
		return "Clojure"
	case Chuck:
		return "Chuck"
	case CMake:
		return "CMake"
	case COBOL:
		return "COBOL"
	case Config:
		return "Configuration"
	case Cpp:
		return "C++"
	case Crystal:
		return "Crystal"
	case CS:
		return "C#"
	case CSound:
		return "Csound"
	case CSS:
		return "CSS"
	case D:
		return "D"
	case Dart:
		return "Dart"
	case Diff:
		return "Diff / patch"
	case Dingo:
		return "Dingo"
	case Docker:
		return "Docker"
	case Elm:
		return "Elm"
	case Email:
		return "E-mail"
	case Erlang:
		return "Erlang"
	case Faust:
		return "Faust"
	case Vibe67:
		return "Vibe67"
	case Fortran77:
		return "Fortran 77"
	case Fortran90:
		return "Fortran 90"
	case FSharp:
		return "F#"
	case FSTAB:
		return "Filesystem Table"
	case Garnet:
		return "Garnet"
	case GDScript:
		return "Godot Script"
	case Git:
		return "Git"
	case Gleam:
		return "Gleam"
	case GoAssembly:
		return "Go-style Assembly"
	case Go:
		return "Go"
	case GoMod:
		return "Go Module"
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
	case HTTP:
		return "HTTP Tests"
	case Ignore:
		return "Ignore"
	case Ini:
		return "INI Configuration"
	case Inko:
		return "Inko"
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
	case Just:
		return "Just"
	case Koka:
		return "Koka"
	case Kotlin:
		return "Kotlin"
	case Lilypond:
		return "Lilypond"
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
	case Mojo:
		return "Mojo"
	case Nim:
		return "Nim"
	case Nix:
		return "Nix"
	case Nmap:
		return "Nmap"
	case Nroff:
		return "Nroff"
	case Oak:
		return "Oak"
	case ObjC:
		return "Objective-C"
	case Ollama:
		return "Ollama"
	case ObjectPascal:
		return "Pas"
	case OCaml:
		return "Ocaml"
	case Odin:
		return "Odin"
	case Perl:
		return "Perl"
	case PHP:
		return "PHP"
	case PolicyLanguage:
		return "SELinux"
	case POV:
		return "POV-Ray"
	case Prolog:
		return "Prolog"
	case Python:
		return "Python"
	case R:
		return "R"
	case ReStructured:
		return "reStructuredText"
	case Ruby:
		return "Ruby"
	case Rust:
		return "Rust"
	case Scala:
		return "Scala"
	case SCDoc:
		return "SCDoc"
	case Scheme:
		return "Scheme"
	case Shader:
		return "Shader"
	case Shell:
		return "Shell"
	case SQL:
		return "SQL"
	case StandardML:
		return "Standard ML"
	case Starlark:
		return "Starlark"
	case Subversion:
		return "Subversion"
	case SuperCollider:
		return "SuperCollider"
	case Swift:
		return "Swift"
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
