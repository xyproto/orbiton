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
	case ext == ".mk" || strings.HasPrefix(baseFilename, "Makefile") || strings.HasPrefix(baseFilename, "makefile") || baseFilename == "GNUmakefile":
		// NOTE: This one MUST come before the ext == "" check below!
		mode = Makefile
	case strings.HasSuffix(filename, ".git/config") || ext == ".ini" || ext == ".cfg" || ext == ".conf" || ext == ".service" || ext == ".target" || ext == ".socket" || strings.HasPrefix(ext, "rc"):
		fallthrough
	case ext == ".yml" || ext == ".toml" || ext == ".ini" || ext == ".bp" || ext == ".rule" || strings.HasSuffix(filename, ".git/config") || (ext == "" && (strings.HasSuffix(baseFilename, "file") || strings.HasSuffix(baseFilename, "rc") || hasS(configFilenames, baseFilename))):
		mode = Config
	case ext == ".sh" || ext == ".ksh" || ext == ".tcsh" || ext == ".bash" || ext == ".zsh" || ext == ".local" || ext == ".profile" || baseFilename == "PKGBUILD" || (strings.HasPrefix(baseFilename, ".") && strings.Contains(baseFilename, "sh")): // This last part covers .bashrc, .zshrc etc
		mode = Shell
	case ext == ".bzl" || baseFilename == "BUILD" || baseFilename == "WORKSPACE":
		mode = Bazel
	case baseFilename == "CMakeLists.txt" || ext == ".cmake":
		mode = CMake
	case strings.HasPrefix(baseFilename, "man.") && len(ext) > 4: // ex: /tmp/man.0asdfadf
		mode = ManPage
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
		case ".ha":
			mode = Hare
		case ".jakt":
			mode = Jakt
		case ".hs", ".hts":
			mode = Haskell
		case ".agda":
			mode = Agda
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
		case ".java":
			mode = Java
		case ".gradle":
			mode = Gradle
		case ".hal":
			mode = HIDL
		case ".aidl":
			mode = AIDL
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
		case ".bas", ".module", ".frm", ".cls", ".ctl", ".vbp", ".vbg", ".form", ".gambas":
			mode = Basic
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
		case ".log":
			mode = Log
		default:
			mode = Blank
		}
	}

	if mode == Text {
		mode = Markdown
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
