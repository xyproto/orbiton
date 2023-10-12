package mode

import (
	"path/filepath"
	"strconv"
	"strings"
)

// Detect looks at the filename and tries to guess what could be an appropriate editor mode.
func Detect(filename string) Mode {
	// A list of the most common configuration filenames that does not have an extension
	var (
		configFilenames = []string{"BUILD", "WORKSPACE", "config", "environment", "fstab", "group", "gshadow", "hostname", "hosts", "issue", "mirrorlist", "passwd", "shadow"}
		mode            Mode
	)

	baseFilename := filepath.Base(filename)
	ext := filepath.Ext(baseFilename)

	// Check if we should be in a particular mode for a particular type of file
	// TODO: Create a hash map to look up many of the extensions
	switch {
	case baseFilename == "COMMIT_EDITMSG" ||
		baseFilename == "MERGE_MSG" ||
		(strings.HasPrefix(baseFilename, "git-") &&
			!strings.Contains(baseFilename, ".") &&
			strings.Count(baseFilename, "-") >= 2):
		// Git mode
		mode = Git
	case baseFilename == "Dockerfile" || baseFilename == "dockerfile":
		mode = Docker
	case baseFilename == "Modelfile" || baseFilename == "modelfile":
		mode = Ollama
	case baseFilename == "svn-commit.tmp":
		mode = Subversion
	case ext == ".vimrc" || ext == ".vim" || ext == ".nvim":
		mode = Vim
	case ext == ".mk" || strings.HasPrefix(baseFilename, "Make") || strings.HasPrefix(baseFilename, "makefile") || baseFilename == "GNUmakefile":
		// NOTE: This one MUST come before the ext == "" check below!
		mode = Make
	case ext == ".just" || ext == ".justfile" || baseFilename == "justfile":
		// NOTE: This one MUST come before the ext == "" check below!
		mode = Just
	case strings.HasSuffix(filename, ".git/config") || ext == ".ini" || ext == ".cfg" || ext == ".conf" || ext == ".service" || ext == ".target" || ext == ".socket" || strings.HasPrefix(ext, "rc"):
		fallthrough
	case ext == ".yml" || ext == ".toml" || ext == ".ini" || ext == ".bp" || ext == ".rule" || strings.HasSuffix(filename, ".git/config") || (ext == "" && (strings.HasSuffix(baseFilename, "file") || strings.HasSuffix(baseFilename, "rc") || hasS(configFilenames, baseFilename))):
		mode = Config
	case ext == ".sh" || ext == ".install" || ext == ".ksh" || ext == ".tcsh" || ext == ".bash" || ext == ".zsh" || ext == ".local" || ext == ".profile" || baseFilename == "PKGBUILD" || baseFilename == "APKBUILD" || (strings.HasPrefix(baseFilename, ".") && strings.Contains(baseFilename, "sh")): // This last part covers .bashrc, .zshrc etc
		mode = Shell
	case ext == ".bzl" || baseFilename == "BUILD" || baseFilename == "WORKSPACE":
		mode = Bazel
	case baseFilename == "CMakeLists.txt" || ext == ".cmake":
		mode = CMake
	case strings.HasPrefix(baseFilename, "man.") && len(ext) > 4: // ie.: /tmp/man.0asdfadf
		mode = ManPage
	case strings.HasPrefix(baseFilename, "mutt-"): // ie.: /tmp/mutt-hostname-0000-0000-00000000000000000
		mode = Email
	case strings.HasSuffix(baseFilename, "Log.txt"): // ie. MinecraftLog.txt
		mode = Log
	default:
		switch ext {
		case ".1", ".2", ".3", ".4", ".5", ".6", ".7", ".8": // not .9
			mode = Nroff
		case ".a68":
			mode = Algol68
		case ".adb", ".gpr", ".ads", ".ada":
			mode = Ada
		case ".adoc":
			mode = ASCIIDoc
		case ".scdoc", ".scd":
			mode = SCDoc
		case ".aidl":
			mode = AIDL
		case ".agda":
			mode = Agda
		case ".amber":
			mode = Amber
		case ".bas", ".module", ".frm", ".cls", ".ctl", ".vbp", ".vbg", ".form", ".gambas":
			mode = Basic
		case ".bat":
			mode = Bat
		case ".bts":
			mode = Battlestar
		case ".c":
			// C mode
			mode = C
		case ".cm":
			// Standard ML project file
			mode = StandardML
		case ".cpp", ".cc", ".c++", ".cxx", ".hpp", ".h": // C++ mode
			// TODO: Find a way to discover is a .h file is most likely to be C or C++
			mode = Cpp
		case ".clj", ".clojure", "cljs":
			mode = Clojure
		case ".cs": // C#
			mode = CS
		case ".csproj": // C# projects
			mode = XML
		case ".cl", ".el", ".elisp", ".emacs", ".l", ".lisp", ".lsp":
			mode = Lisp
		case ".cr":
			mode = Crystal
		case ".d":
			mode = D
		case ".dart":
			mode = Dart
		case ".elm":
			mode = Elm
		case ".eml":
			mode = Email
		case ".erl":
			mode = Erlang
		case ".f":
			mode = Fortran77
		case ".f90":
			mode = Fortran90
		case ".fs":
			mode = FSharp
		case ".gd":
			mode = GDScript
		case ".gt":
			mode = Garnet
		case ".go":
			mode = Go
		case ".glsl":
			mode = Shader
		case ".gradle":
			mode = Gradle
		case ".ha":
			mode = Hare
		case ".hal":
			mode = HIDL
		case ".hs", ".hts", ".cabal":
			mode = Haskell
		case ".htm", ".html":
			mode = HTML
		case ".hx", ".hxml":
			mode = Haxe
		case ".ino":
			mode = Arduino
		case ".ivy":
			mode = Ivy
		case ".jakt":
			mode = Jakt
		case ".java":
			mode = Java
		case ".js":
			mode = JavaScript
		case ".json", ".ipynb":
			mode = JSON
		case ".kk":
			mode = Koka
		case ".kt", ".kts":
			mode = Kotlin
		case ".log":
			mode = Log
		case ".lua":
			mode = Lua
		case ".ly":
			mode = Lilypond
		case ".m4":
			mode = M4
		case ".md":
			// Markdown mode
			mode = Markdown
		case ".ml":
			mode = OCaml // or standard ML, if the file does not contain ";;"
		case ".nim":
			mode = Nim
		case ".odin":
			mode = Odin
		case ".ok":
			mode = Oak
		case ".pas", ".pp", ".lpr":
			mode = ObjectPascal
		case ".pl", ".pro":
			mode = Prolog
		case ".py":
			mode = Python
		case ".mojo", "." + fireEmoji:
			mode = Mojo
		case ".r":
			mode = R
		case ".razor":
			mode = XML
		case ".rs":
			mode = Rust
		case ".rst":
			mode = ReStructured // reStructuredText
		case ".s", ".S", ".asm", ".inc":
			// Go-style assembly (modeGoAssembly) is enabled if a mid-dot is discovered
			mode = Assembly
		case ".scala":
			mode = Scala
		case ".fun", ".sml":
			mode = StandardML
		case ".sql":
			mode = SQL
		case ".t":
			mode = Terra
		case ".te":
			mode = PolicyLanguage
		case ".tl":
			mode = Teal
		case ".ts":
			mode = TypeScript
		case ".txt", ".text", ".nfo", ".diz":
			mode = Text
		case ".v":
			mode = V
		case ".xml":
			mode = XML
		case ".zig", ".zir":
			mode = Zig
		default:
			mode = Blank
		}
	}

	// If the mode is not set, and there is no extensions
	if mode == Blank && !strings.Contains(baseFilename, ".") {
		if baseFilename == strings.ToUpper(baseFilename) {
			// If the filename is all uppercase and no ".", use mode.Markdown
			mode = Markdown
		} else if len(baseFilename) > 2 && baseFilename[2] == '-' {
			// Could it be a rule-file, that starts with ie. "90-" ?
			if _, err := strconv.Atoi(baseFilename[:2]); err == nil { // success
				// Yes, assume this is a shell-like configuration file
				mode = Config
			}
		}
	}

	return mode
}
