package main

import (
	"github.com/xyproto/mode"
	"github.com/xyproto/syntax"
)

// Keywords is a reference to the syntax package's global keyword map.
var Keywords = syntax.Keywords

// adjustSyntaxHighlightingKeywords configures syntax highlighting keywords for the given mode.
func adjustSyntaxHighlightingKeywords(m mode.Mode) {
	syntax.AdjustKeywords(m)
	Keywords = syntax.Keywords
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
		return "//"
	case mode.ReStructured:
		return ".."
	case mode.Vim:
		return "\""
	default:
		return "//"
	}
}
