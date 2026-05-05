package main

import (
	"strings"

	"github.com/xyproto/mode"
	"github.com/xyproto/syntax"
)

// Keywords is a reference to the syntax package's global keyword map.
var Keywords = syntax.Keywords

func clearKeywords() {
	syntax.ClearKeywords()
	Keywords = syntax.Keywords
}

func addKeywords(kws []string) {
	syntax.AddKeywords(kws)
}

func addKeywordsAsUppercase(xs []string) {
	syntax.AddKeywordsAsUppercase(xs)
}

func removeKeywords(kws []string) {
	syntax.RemoveKeywords(kws)
}

func addAndRemoveKeywords(addAndDelKeywords ...[]string) {
	syntax.AddAndRemoveKeywords(addAndDelKeywords...)
}

func setKeywords(addAndDelKeywords ...[]string) {
	syntax.SetKeywords(addAndDelKeywords...)
	Keywords = syntax.Keywords
}

// adjustSyntaxHighlightingKeywords contains per-language adjustments to highlighting of keywords
func adjustSyntaxHighlightingKeywords(m mode.Mode) {
	switch m {
	case mode.ABC:
		addKeywords([]string{"MIDI"})
	case mode.Ada:
		addKeywords([]string{"constant", "loop", "procedure", "project"})
	case mode.Assembly:
		setKeywords(asmWords)
	case mode.Battlestar:
		setKeywords(battlestarWords)
	case mode.C3:
		setKeywords(c3Words)
	case mode.Vibe67:
		setKeywords(vibe67Words)
		removeKeywords([]string{"double", "true", "false", "True", "False"})
	case mode.Chuck:
		setKeywords(chuckWords)
	case mode.Clojure:
		setKeywords(clojureWords)
	case mode.CMake:
		addAndRemoveKeywords(cmakeWords, []string{"build", "package"})
	case mode.Config, mode.Ini, mode.FSTAB, mode.TOML, mode.YAML:
		removeKeywords([]string{"auto", "build", "def", "default", "for", "from", "get", "install", "int", "local", "no", "not", "package", "return", "super", "type", "var", "with"})
		addKeywords([]string{"bind", "bindsym", "DB_PASSWORD", "exec_always", "PASSWORD", "Password", "password", "POSTGRES_PASSWORD", "PWD", "Pwd", "pwd", "Secret", "SECRET", "secret", "Secrets", "SECRETS", "secrets", "set-option", "set-window-option", "unbind", "uses"})
	case mode.Nix:
		setKeywords(nixWords)
	case mode.CS:
		setKeywords(csWords)
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
		setKeywords(massagedWords)
		//removeKeywords([]string{"flex"}) // flex can be part of the property name and also the value
		addKeywords([]string{"animation", "events", "pointer"})
	case mode.COBOL:
		setKeywords(cobolWords)
	case mode.D:
		setKeywords(dWords)
	case mode.Dart:
		setKeywords(dartWords)
	case mode.Docker:
		removeKeywords([]string{"auto", "default", "from", "install", "int", "local", "no", "not", "pull", "type", "var"})
		addKeywords(dockerWords)
		addKeywordsAsUppercase(dockerWords)
		removeKeywords([]string{"copy", "entrypoint", "env", "from", "pull", "run"}) // remove the lowercase variety of these
	case mode.Erlang:
		setKeywords(erlangWords)
	case mode.Fortran77:
		setKeywords(fortran77Words)
		addKeywordsAsUppercase(fortran77Words)
	case mode.Fortran90:
		setKeywords(fortran90Words)
	case mode.FSharp:
		setKeywords(fsharpWords)
	case mode.GDScript:
		setKeywords(gdscriptWords)
	case mode.Gleam:
		setKeywords(gleamWords)
	case mode.Go, mode.Dingo:
		// TODO: Define goWords and use setKeywords instead
		addKeywords := []string{"chan", "defer", "error", "fallthrough", "func", "go", "import", "package", "print", "println", "range", "rune", "select", "string", "uint", "uint16", "uint32", "uint64", "uint8"}
		delKeywords := []string{"assert", "auto", "build", "char", "class", "def", "def", "del", "die", "dir", "done", "end", "exec", "False", "fi", "final", "finally", "fn", "foreach", "from", "function", "get", "in", "include", "is", "lambda", "last", "let", "match", "mut", "next", "no", "None", "pass", "redo", "rescue", "ret", "retry", "set", "static", "template", "then", "this", "True", "until", "when", "where", "while", "yes"}
		addAndRemoveKeywords(addKeywords, delKeywords)
	case mode.Haskell:
		addKeywords([]string{"data", "deriving", "foreign", "infix", "infixl", "infixr", "instance", "newtype"})
	case mode.Haxe:
		setKeywords(haxeWords)
	case mode.HIDL:
		setKeywords(hidlWords)
	case mode.Inko:
		setKeywords(inkoWords)
	case mode.AIDL:
		addKeywords(append([]string{"interface"}, hidlWords...))
		fallthrough // continue to mode.Java
	case mode.Java:
		addKeywords := []string{"package"}
		delKeywords := []string{"add", "bool", "get", "in", "local", "sub", "until"}
		addAndRemoveKeywords(addKeywords, delKeywords)
	case mode.JavaScript:
		kws := []string{"of", "super"}
		addKeywords(kws)
	case mode.TypeScript:
		addKeywords([]string{"declare", "infer", "keyof", "never", "of", "readonly", "satisfies", "unknown"})
	case mode.JSON:
		removeKeywords([]string{"install", "until"})
	case mode.Koka:
		setKeywords(kokaWords)
	case mode.Kotlin:
		setKeywords(kotlinWords)
	case mode.Lilypond:
		setKeywords(lilypondWords)
	case mode.Lisp:
		setKeywords(emacsWords)
	case mode.Lua, mode.Teal, mode.Terra: // use the Lua mode for Teal and Terra, for now
		setKeywords(luaWords)
	case mode.Nroff:
		addKeywords := []string{"B", "BR", "PP", "SH", "TP", "fB", "fI", "fP", "RB", "TH", "IR", "IP", "fI", "fR"}
		delKeywords := []string{"class"}
		setKeywords(addKeywords, delKeywords)
	case mode.ManPage:
		clearKeywords()
	case mode.ObjectPascal:
		addKeywords(objPasWords)
	case mode.Oak:
		addAndRemoveKeywords([]string{"fn"}, []string{"from", "new", "print"})
	case mode.Python, mode.Mojo, mode.Starlark:
		addAndRemoveKeywords([]string{"type", "class"}, []string{"append", "exit", "fn", "get", "package", "print", "until"})
	case mode.POV:
		addKeywords(povrayWords)
	case mode.Nim:
		addAndRemoveKeywords([]string{"proc", "type"}, []string{"append", "exit", "fn", "get", "package", "print", "until"})
	case mode.Odin:
		setKeywords(odinWords)
	case mode.Ollama:
		removeKeywords([]string{"auto", "default", "from", "install", "int", "local", "no", "not", "type", "var"})
		addKeywords(ollamaWords)
		addKeywordsAsUppercase(ollamaWords)
	case mode.PolicyLanguage: // SE Linux
		setKeywords(policyLanguageWords)
	case mode.Hare:
		addKeywords := []string{"String", "assert_eq", "char", "done", "fn", "i16", "i32", "i64", "i8", "impl", "loop", "mod", "out", "panic", "u16", "u32", "u64", "u8", "usize"}
		// "as" and "mut" are treated as special cases in the syntax package
		delKeywords := []string{"as", "build", "byte", "end", "foreach", "get", "int", "int16", "int32", "int64", "last", "map", "mut", "next", "pass", "print", "uint16", "uint32", "uint64", "until", "var"}
		addAndRemoveKeywords(addKeywords, delKeywords)
	case mode.Garnet, mode.Jakt, mode.Rust: // Originally only for Rust, split up as needed
		addKeywords := []string{"String", "assert_eq", "async", "await", "char", "crate", "dyn", "fn", "i16", "i32", "i64", "i8", "impl", "loop", "mod", "out", "panic", "pub", "u16", "u32", "u64", "u8", "unsafe", "usize"}
		// "as" and "mut" are treated as special cases in the syntax package
		delKeywords := []string{"as", "build", "byte", "done", "foreach", "get", "int", "int16", "int32", "int64", "last", "map", "mut", "next", "pass", "print", "uint16", "uint32", "uint64", "until", "var"}
		if m != mode.Garnet {
			delKeywords = append(delKeywords, "end")
		}
		addAndRemoveKeywords(addKeywords, delKeywords)
	case mode.Spec:
		setKeywords(specWords)
	case mode.Scala:
		setKeywords(scalaWords)
	case mode.OCaml:
		setKeywords(ocamlWords)
	case mode.Elm, mode.StandardML:
		setKeywords(smlWords)
	case mode.SQL:
		addKeywords([]string{"NOT"})
	case mode.Swift:
		setKeywords(swiftWords)
	case mode.Vim:
		addKeywords([]string{"call", "echo", "elseif", "endfunction", "map", "nmap", "redraw"})
	case mode.Zig:
		setKeywords(zigWords, []string{"log"})
	case mode.GoAssembly:
		// Only highlight some words, to make them stand out
		addKeywords := []string{"cap", "close", "complex", "complex128", "complex64", "copy", "db", "dd", "dw", "imag", "int", "len", "panic", "real", "recover", "resb", "resd", "resw", "section", "syscall", "uintptr"}
		setKeywords(addKeywords)
	case mode.Just:
		addKeywords(justWords)
		fallthrough // Continue to Make
	case mode.Make:
		delKeywords := []string{"#else", "#endif", "and", "as", "build", "default", "del", "done", "double", "exec", "export", "finally", "float", "fn", "generic", "get", "install", "local", "long", "new", "no", "package", "pass", "print", "property", "require", "ret", "set", "stop", "super", "super", "template", "type", "var", "with"}
		addAndRemoveKeywords(shellWords, delKeywords)
	case mode.Shell:
		delKeywords := []string{"#else", "#endif", "as", "build", "default", "del", "double", "exec", "false", "finally", "float", "fn", "generic", "get", "install", "long", "namespace", "native", "new", "no", "package", "pass", "print", "property", "require", "ret", "set", "super", "super", "template", "true", "type", "var", "with"}
		addAndRemoveKeywords(shellWords, delKeywords)
	case mode.SuperCollider:
		addKeywords(superColliderWords)
	case mode.Text:
		clearKeywords()
	case mode.Shader:
		addKeywords([]string{"buffer", "bvec2", "bvec3", "bvec4", "coherent", "dvec2", "dvec3", "dvec4", "flat", "in", "inout", "invariant", "ivec2", "ivec3", "ivec4", "layout", "mat", "mat2", "mat3", "mat4", "noperspective", "out", "precision", "readonly", "restrict", "smooth", "uniform", "uvec2", "uvec3", "uvec4", "vec2", "vec3", "vec4", "volatile", "writeonly"})
		fallthrough // Continue to C/C++ and then to the default
	case mode.Arduino, mode.C, mode.Cpp, mode.ObjC:
		addKeywords := []string{"co_await", "co_return", "co_yield", "consteval", "constinit", "int8_t", "uint8_t", "int16_t", "uint16_t", "int32_t", "uint32_t", "int64_t", "uint64_t", "requires", "size_t"}
		delKeywords := []string{"fn", "from", "in", "ret", "static"} // static is treated separately, as a special keyword
		addAndRemoveKeywords(addKeywords, delKeywords)
		fallthrough // Continue to the default
	default:
		addKeywords := []string{"elif", "endif", "ifeq", "ifneq"}
		delKeywords := []string{"build", "done", "package", "require", "set", "super", "type", "when"}
		addAndRemoveKeywords(addKeywords, delKeywords)
	}
}

// SingleLineCommentMarker will return the string that starts a single-line
// comment for the current language mode the editor is in.
func (e *Editor) SingleLineCommentMarker() string {
	switch e.mode {
	case mode.ABC, mode.Lilypond, mode.Perl, mode.Prolog:
		return "%"
	case mode.Amber:
		return "!!"
	case mode.Assembly, mode.Ini:
		return ";"
	case mode.Basic:
		return "'"
	case mode.Bat:
		return "@rem" // or rem or just ":" ...
	case mode.Algol68, mode.Bazel, mode.CMake, mode.Config, mode.Crystal, mode.Docker, mode.FSTAB, mode.GDScript, mode.HCL, mode.Ignore, mode.Janet, mode.Just, mode.Make, mode.Nim, mode.Nix, mode.Mojo, mode.Nushell, mode.PolicyLanguage, mode.Python, mode.R, mode.Ruby, mode.Shell, mode.Spec, mode.Starlark, mode.TOML, mode.YAML:
		return "#"
	case mode.Clojure, mode.Lisp:
		return ";;"
	case mode.Email:
		return "GIT:"
	case mode.Fortran77:
		return "*" // TODO: Also add "C", "c" and all the others
	case mode.Fortran90:
		return "!" // TODO: Only at the start of lines
	case mode.OCaml, mode.StandardML:
		// Not applicable, just return the multiline comment start marker
		return "(*"
	case mode.Ada, mode.Agda, mode.Dhall, mode.Elm, mode.Garnet, mode.Haskell, mode.Lua, mode.Nmap, mode.SQL, mode.Teal, mode.Terra:
		return "--"
	case mode.M4:
		return "dnl"
	case mode.Nroff:
		return `.\"`
	case mode.ObjectPascal:
		return "{"
	case mode.ReStructured:
		return "["
	case mode.Vim:
		return "\""
	default:
		return "//"
	}
}
