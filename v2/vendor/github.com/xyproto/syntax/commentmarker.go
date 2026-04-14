package syntax

import "github.com/xyproto/mode"

// SingleLineCommentMarker returns the string that starts a single-line
// comment for the given language mode.
func SingleLineCommentMarker(m mode.Mode) string {
	switch m {
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
	case mode.Algol68, mode.Bazel, mode.CMake, mode.Config, mode.Crystal, mode.Docker, mode.FSTAB, mode.GDScript, mode.Ignore, mode.Just, mode.Make, mode.Nim, mode.Nix, mode.Mojo, mode.PolicyLanguage, mode.Python, mode.R, mode.Ruby, mode.Shell, mode.Spec, mode.Starlark:
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
	case mode.Ada, mode.Agda, mode.Elm, mode.Garnet, mode.Haskell, mode.Lua, mode.Nmap, mode.SQL, mode.Teal, mode.Terra:
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
