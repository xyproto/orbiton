package syntax

import (
	"strings"
	"text/scanner"
	"unicode"
	"unicode/utf8"

	"github.com/xyproto/mode"
)

// tokenKind determines the Kind of a token for syntax highlighting.
func tokenKind(tok rune, tokText string, inComment *bool, m mode.Mode) Kind {

	// Check if we are in a single line comment
	if (m == mode.Assembly && tok == ';') ||
		(m != mode.Assembly && m != mode.GoAssembly && m != mode.Clojure && m != mode.Lisp && m != mode.C && m != mode.Cpp && m != mode.Lua && tok == '#') ||
		((m == mode.ABC || m == mode.Lilypond || m == mode.Perl || m == mode.Prolog) && tok == '%') {
		*inComment = true
	} else if tok == '\n' {
		*inComment = false
	}

	// Check if this is #include or #define
	if (m == mode.C || m == mode.Cpp) && (tokText == "include" || tokText == "define" || tokText == "ifdef" || tokText == "ifndef" || tokText == "endif" || tokText == "else" || tokText == "elif") {
		*inComment = false
		return Keyword
	}

	// If we are in a comment, return the Comment kind
	if *inComment {
		return Comment
	}

	// Check if this is the "as" or "mut" keyword, for Rust
	if m == mode.Rust {
		switch tokText {
		case "as":
			return Type // re-use color
		case "mut":
			return Mut
		}
	}

	// Check if this is the "self" keyword, for Python
	if m == mode.Python && tokText == "self" {
		return Self
	}

	// If not, do the regular switch
	switch tok {
	case scanner.Ident:
		if _, ok := Keywords[tokText]; ok {
			return Keyword
		}
		switch tokText {
		case "private":
			return Private
		case "public":
			return Public
		case "protected":
			return Protected
		case "class":
			return Class
		case "static":
			return Static
		case "JMP", "jmp", "LEAVE", "leave", "RET", "ret", "CALL", "call":
			if m == mode.Assembly || m == mode.GoAssembly {
				return AssemblyEnd
			}
		}
		if r, _ := utf8.DecodeRuneInString(tokText); unicode.IsUpper(r) {
			return Type
		}
		return Plaintext

	case scanner.Float, scanner.Int:
		return Decimal
	case scanner.Char, scanner.String, scanner.RawString:
		return String
	case scanner.Comment:
		// In Nix, // is the attribute set update operator, not a comment
		if m == mode.Nix && strings.HasPrefix(tokText, "//") {
			return AndOr
		}
		return Comment
	}

	if tok == '&' || tok == '|' {
		return AndOr
	} else if tok == '*' {
		return Star
	} else if tok == '$' {
		return Dollar
	} else if tok == '<' || tok == '>' {
		return AngleBracket
	}

	if unicode.IsSpace(tok) {
		return Whitespace
	}

	return Punctuation
}

// ClearKeywords resets the global Keywords map.
func ClearKeywords() {
	Keywords = make(map[string]struct{})
}

// AddKeywords adds the given keywords so that they will be syntax highlighted.
func AddKeywords(kws []string) {
	for _, kw := range kws {
		Keywords[kw] = struct{}{}
	}
}

// AddKeywordsAsUppercase adds uppercased versions of the given keywords.
func AddKeywordsAsUppercase(xs []string) {
	uppercase := make([]string, 0, len(xs))
	for _, word := range xs {
		uppercase = append(uppercase, strings.ToUpper(word))
	}
	AddKeywords(uppercase)
}

// RemoveKeywords removes keywords that should not be syntax highlighted.
func RemoveKeywords(kws []string) {
	for _, kw := range kws {
		delete(Keywords, kw)
	}
}

// AddAndRemoveKeywords first adds and then removes keywords.
func AddAndRemoveKeywords(addAndDel ...[]string) {
	l := len(addAndDel)
	if l > 0 {
		AddKeywords(addAndDel[0])
	}
	if l > 1 {
		RemoveKeywords(addAndDel[1])
	}
}

// SetKeywords clears, then adds/removes keywords.
func SetKeywords(addAndDel ...[]string) {
	ClearKeywords()
	AddAndRemoveKeywords(addAndDel...)
}

// AdjustKeywords contains per-language adjustments to highlighting of keywords
func AdjustKeywords(m mode.Mode) {
	switch m {
	case mode.ABC:
		AddKeywords([]string{"MIDI"})
	case mode.Ada:
		AddKeywords([]string{"constant", "loop", "procedure", "project"})
	case mode.Assembly:
		SetKeywords(asmWords)
	case mode.Battlestar:
		SetKeywords(battlestarWords)
	case mode.C3:
		SetKeywords(c3Words)
	case mode.Vibe67:
		SetKeywords(vibe67Words)
		RemoveKeywords([]string{"double", "true", "false", "True", "False"})
	case mode.Chuck:
		SetKeywords(chuckWords)
	case mode.Clojure:
		SetKeywords(clojureWords)
	case mode.CMake:
		AddAndRemoveKeywords(cmakeWords, []string{"build", "package"})
	case mode.Config, mode.Ini, mode.FSTAB:
		RemoveKeywords([]string{"auto", "build", "def", "default", "for", "from", "get", "install", "int", "local", "no", "not", "package", "return", "super", "type", "var", "with"})
		AddKeywords([]string{"bind", "bindsym", "DB_PASSWORD", "exec_always", "PASSWORD", "Password", "password", "POSTGRES_PASSWORD", "PWD", "Pwd", "pwd", "Secret", "SECRET", "secret", "Secrets", "SECRETS", "secrets", "set-option", "set-window-option", "unbind", "uses"})
	case mode.Nix:
		SetKeywords(nixWords)
	case mode.CS:
		SetKeywords(csWords)
	case mode.CSS:
		var massagedWords []string
		for _, word := range cssWords {
			if strings.Contains(word, "-") {
				fields := strings.Split(word, "-")
				massagedWords = append(massagedWords, fields...)
			} else {
				massagedWords = append(massagedWords, word)
			}
		}
		SetKeywords(massagedWords)
		AddKeywords([]string{"animation", "events", "pointer"})
	case mode.COBOL:
		SetKeywords(cobolWords)
	case mode.D:
		SetKeywords(dWords)
	case mode.Dart:
		SetKeywords(dartWords)
	case mode.Docker:
		RemoveKeywords([]string{"auto", "default", "from", "install", "int", "local", "no", "not", "pull", "type", "var"})
		AddKeywords(dockerWords)
		AddKeywordsAsUppercase(dockerWords)
		RemoveKeywords([]string{"copy", "entrypoint", "env", "from", "pull", "run"})
	case mode.Erlang:
		SetKeywords(erlangWords)
	case mode.Fortran77:
		SetKeywords(fortran77Words)
		AddKeywordsAsUppercase(fortran77Words)
	case mode.Fortran90:
		SetKeywords(fortran90Words)
	case mode.FSharp:
		SetKeywords(fsharpWords)
	case mode.GDScript:
		SetKeywords(gdscriptWords)
	case mode.Gleam:
		SetKeywords(gleamWords)
	case mode.Go, mode.Dingo:
		addKws := []string{"chan", "defer", "error", "fallthrough", "func", "go", "import", "package", "print", "println", "range", "rune", "select", "string", "uint", "uint16", "uint32", "uint64", "uint8"}
		delKws := []string{"assert", "auto", "build", "char", "class", "def", "def", "del", "die", "dir", "done", "end", "exec", "False", "fi", "final", "finally", "fn", "foreach", "from", "function", "get", "in", "include", "is", "lambda", "last", "let", "match", "mut", "next", "no", "None", "pass", "redo", "rescue", "ret", "retry", "set", "static", "template", "then", "this", "True", "until", "when", "where", "while", "yes"}
		AddAndRemoveKeywords(addKws, delKws)
	case mode.Haskell:
		AddKeywords([]string{"data", "deriving", "foreign", "infix", "infixl", "infixr", "instance", "newtype"})
	case mode.Haxe:
		SetKeywords(haxeWords)
	case mode.HIDL:
		SetKeywords(hidlWords)
	case mode.Inko:
		SetKeywords(inkoWords)
	case mode.AIDL:
		AddKeywords(append([]string{"interface"}, hidlWords...))
		fallthrough
	case mode.Java:
		addKws := []string{"package"}
		delKws := []string{"add", "bool", "get", "in", "local", "sub", "until"}
		AddAndRemoveKeywords(addKws, delKws)
	case mode.JavaScript:
		AddKeywords([]string{"of", "super"})
	case mode.TypeScript:
		AddKeywords([]string{"declare", "infer", "keyof", "never", "of", "readonly", "satisfies", "unknown"})
	case mode.JSON:
		RemoveKeywords([]string{"install", "until"})
	case mode.Koka:
		SetKeywords(kokaWords)
	case mode.Kotlin:
		SetKeywords(kotlinWords)
	case mode.Lilypond:
		SetKeywords(lilypondWords)
	case mode.Lisp:
		SetKeywords(emacsWords)
	case mode.Lua, mode.Teal, mode.Terra:
		SetKeywords(luaWords)
	case mode.Nroff:
		addKws := []string{"B", "BR", "PP", "SH", "TP", "fB", "fI", "fP", "RB", "TH", "IR", "IP", "fI", "fR"}
		delKws := []string{"class"}
		SetKeywords(addKws, delKws)
	case mode.ManPage:
		ClearKeywords()
	case mode.ObjectPascal:
		AddKeywords(objPasWords)
	case mode.Oak:
		AddAndRemoveKeywords([]string{"fn"}, []string{"from", "new", "print"})
	case mode.Python, mode.Mojo, mode.Starlark:
		AddAndRemoveKeywords([]string{"type", "class"}, []string{"append", "exit", "fn", "get", "package", "print", "until"})
	case mode.POV:
		AddKeywords(povrayWords)
	case mode.Nim:
		AddAndRemoveKeywords([]string{"proc", "type"}, []string{"append", "exit", "fn", "get", "package", "print", "until"})
	case mode.Odin:
		SetKeywords(odinWords)
	case mode.Ollama:
		RemoveKeywords([]string{"auto", "default", "from", "install", "int", "local", "no", "not", "type", "var"})
		AddKeywords(ollamaWords)
		AddKeywordsAsUppercase(ollamaWords)
	case mode.PolicyLanguage:
		SetKeywords(policyLanguageWords)
	case mode.Hare:
		addKws := []string{"String", "assert_eq", "char", "done", "fn", "i16", "i32", "i64", "i8", "impl", "loop", "mod", "out", "panic", "u16", "u32", "u64", "u8", "usize"}
		delKws := []string{"as", "build", "byte", "end", "foreach", "get", "int", "int16", "int32", "int64", "last", "map", "mut", "next", "pass", "print", "uint16", "uint32", "uint64", "until", "var"}
		AddAndRemoveKeywords(addKws, delKws)
	case mode.Garnet, mode.Jakt, mode.Rust:
		addKws := []string{"String", "assert_eq", "async", "await", "char", "crate", "dyn", "fn", "i16", "i32", "i64", "i8", "impl", "loop", "mod", "out", "panic", "pub", "u16", "u32", "u64", "u8", "unsafe", "usize"}
		delKws := []string{"as", "build", "byte", "done", "foreach", "get", "int", "int16", "int32", "int64", "last", "map", "mut", "next", "pass", "print", "uint16", "uint32", "uint64", "until", "var"}
		if m != mode.Garnet {
			delKws = append(delKws, "end")
		}
		AddAndRemoveKeywords(addKws, delKws)
	case mode.Scala:
		SetKeywords(scalaWords)
	case mode.OCaml:
		SetKeywords(ocamlWords)
	case mode.Elm, mode.StandardML:
		SetKeywords(smlWords)
	case mode.SQL:
		AddKeywords([]string{"NOT"})
	case mode.Swift:
		SetKeywords(swiftWords)
	case mode.Vim:
		AddKeywords([]string{"call", "echo", "elseif", "endfunction", "map", "nmap", "redraw"})
	case mode.Zig:
		SetKeywords(zigWords, []string{"log"})
	case mode.GoAssembly:
		SetKeywords([]string{"cap", "close", "complex", "complex128", "complex64", "copy", "db", "dd", "dw", "imag", "int", "len", "panic", "real", "recover", "resb", "resd", "resw", "section", "syscall", "uintptr"})
	case mode.Just:
		AddKeywords(justWords)
		fallthrough
	case mode.Make:
		delKws := []string{"#else", "#endif", "and", "as", "build", "default", "del", "done", "double", "exec", "export", "finally", "float", "fn", "generic", "get", "install", "local", "long", "new", "no", "package", "pass", "print", "property", "require", "ret", "set", "stop", "super", "super", "template", "type", "var", "with"}
		AddAndRemoveKeywords(shellWords, delKws)
	case mode.Shell:
		delKws := []string{"#else", "#endif", "as", "build", "default", "del", "double", "exec", "false", "finally", "float", "fn", "generic", "get", "install", "long", "namespace", "native", "new", "no", "package", "pass", "print", "property", "require", "ret", "set", "super", "super", "template", "true", "type", "var", "with"}
		AddAndRemoveKeywords(shellWords, delKws)
	case mode.SuperCollider:
		AddKeywords(superColliderWords)
	case mode.Text:
		ClearKeywords()
	case mode.Shader:
		AddKeywords([]string{"buffer", "bvec2", "bvec3", "bvec4", "coherent", "dvec2", "dvec3", "dvec4", "flat", "in", "inout", "invariant", "ivec2", "ivec3", "ivec4", "layout", "mat", "mat2", "mat3", "mat4", "noperspective", "out", "precision", "readonly", "restrict", "smooth", "uniform", "uvec2", "uvec3", "uvec4", "vec2", "vec3", "vec4", "volatile", "writeonly"})
		fallthrough
	case mode.Arduino, mode.C, mode.Cpp, mode.ObjC:
		addKws := []string{"co_await", "co_return", "co_yield", "consteval", "constinit", "int8_t", "uint8_t", "int16_t", "uint16_t", "int32_t", "uint32_t", "int64_t", "uint64_t", "requires", "size_t"}
		delKws := []string{"fn", "from", "in", "ret", "static"}
		AddAndRemoveKeywords(addKws, delKws)
		fallthrough
	default:
		addKws := []string{"elif", "endif", "ifeq", "ifneq"}
		delKws := []string{"build", "done", "package", "require", "set", "super", "type", "when"}
		AddAndRemoveKeywords(addKws, delKws)
	}
}
