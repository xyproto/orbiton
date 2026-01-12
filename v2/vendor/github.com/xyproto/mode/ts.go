package mode

import (
	"strings"
)

// TabsSpaces contains all info needed about tabs and spaces for a file
type TabsSpaces struct {
	PerTab int  // number of spaces per tab/indentation
	Spaces bool // use spaces, or tabs?
}

// DefaultTabsSpaces is the default setting: 4 spaces
var DefaultTabsSpaces = TabsSpaces{4, true}

var languageIndentation = map[TabsSpaces][]Mode{
	// Languages that use spaces (from the opinionated point of view of this package)
	{1, true}: {ABC},
	{2, true}: {Agda, Algol68, Amber, Arduino, Assembly, Blueprint, C3, Clojure, Config, CSS, CSound, Dart, Diff, Erlang, Fortran90, FSTAB, Gleam, HTML, Haskell, Ignore, Ini, Inko, JSON, Koka, Lilypond, Lua, Nmap, Nix, ObjC, ObjectPascal, OCaml, Perl, PolicyLanguage, POV, ReStructured, Ruby, Scala, Scheme, Shell, StandardML, Teal, Vim, Vim, XML},
	{3, true}: {Ada, Prolog}, // Ada and Prolog are special
	{4, true}: {ASCIIDoc, Basic, Bat, Battlestar, Beef, Vibe67, CMake, Chuck, CS, Cpp, COBOL, Crystal, Docker, Elm, Email, Faust, FSharp, GDScript, Garnet, Git, Haxe, JSON, Jakt, Java, JavaScript, Kotlin, Markdown, Mojo, Nim, Oak, Ollama, PHP, Python, R, Rust, SCDoc, SQL, Starlark, Subversion, Swift, Terra, Text, TypeScript, V, Zig},
	{7, true}: {Fortran77},        // Fortran77 is weird
	{8, true}: {GoMod, Hare, Ivy}, // go.mod files, Hare and Ivy are special
	// Languages that use tabs (from the opinionated point of view of this package)
	{4, false}: {AIDL, C, Dingo, Go, GoAssembly, HIDL, Just, Lisp, M4, Make, ManPage, Nroff, Odin, Shader, SuperCollider}, // Tabs
}

// Spaces returns true if spaces should be used for the current mode
func (m Mode) Spaces() bool {
	for k, vs := range languageIndentation {
		for _, v := range vs {
			if v == m {
				return k.Spaces
			}
		}
	}
	return DefaultTabsSpaces.Spaces
}

// TabsSpaces tries to return the appropriate settings for tabs and spaces as a TabsSpaces struct
func (m Mode) TabsSpaces() TabsSpaces {
	for k, vs := range languageIndentation {
		for _, v := range vs {
			if v == m {
				return k
			}
		}
	}
	return DefaultTabsSpaces
}

// String returns the string for one indentation
func (ts TabsSpaces) String() string {
	if !ts.Spaces {
		return "\t"
	}
	return strings.Repeat(" ", ts.PerTab)
}

// WSLen will count the length of the given whitespace string, in terms of spaces
func (ts TabsSpaces) WSLen(whitespaceString string) int {
	return strings.Count(whitespaceString, "\t")*ts.PerTab + strings.Count(whitespaceString, " ")
}
