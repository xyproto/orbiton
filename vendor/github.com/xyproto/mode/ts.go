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
	// Languages that use tabs (from the opinionated point of view of this package)
	{4, false}: {AIDL, C, Go, GoAssembly, HIDL, Lisp, M4, Makefile, ManPage, Nroff, OCaml, Odin, Rust}, // Tabs
	// Languages that use spaces (from the opinionated point of view of this package)
	{2, true}: {Agda, Amber, Assembly, Clojure, Config, HTML, Haskell, JSON, Lua, ObjectPascal, Perl, PolicyLanguage, Shell, StandardML, Vim, Vim, XML},
	{3, true}: {Ada}, // Ada is special
	{4, true}: {Bat, Battlestar, CMake, CS, Cpp, Crystal, Git, JSON, Java, JavaScript, Kotlin, Lua, Markdown, Nim, Oak, Python, SQL, Scala, Text, TypeScript, V, Zig},
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
