package main

import "strings"

// TabsSpaces contains all info needed about tabs and spaces for a file
type TabsSpaces struct {
	perTab int  // number of spaces per tab/indentation
	spaces bool // use spaces, or tabs?
}

var defaultTabsSpaces = TabsSpaces{4, true}

// modeBlank
var languageIndentation = map[TabsSpaces][]Mode{
	// Languages that use tabs (from the opinionated point of view of o)
	{4, false}: {modeC, modeGo, modeGoAssembly, modeHIDL, modeLisp, modeMakefile, modeManPage, modeNroff, modeOCaml, modeOdin, modeRust, modeStandardML}, // Tabs
	// Languages that use spaces (from the opinionated point of view of o)
	{2, true}: {modeAmber, modeAssembly, modeClojure, modeConfig, modeHTML, modeHaskell, modeJSON, modeLua, modeObjectPascal, modePerl, modePolicyLanguage, modeShell, modeVim, modeVim, modeXML},
	{3, true}: {modeAda}, // Ada is special
	{4, true}: {modeBat, modeBattlestar, modeCMake, modeCS, modeCpp, modeCrystal, modeGit, modeJSON, modeJava, modeJavaScript, modeKotlin, modeLua, modeMarkdown, modeNim, modeOak, modePython, modeSQL, modeScala, modeText, modeTypeScript, modeV, modeZig},
}

// Spaces checks if the given mode should use tabs or spaces.
// Returns true for spaces.
func Spaces(mode Mode) bool {
	for k, vs := range languageIndentation {
		for _, v := range vs {
			if v == mode {
				return k.spaces
			}
		}
	}
	return defaultTabsSpaces.spaces
}

// TabsSpacesFromMode takes a mode, like modeJava, and tries to return the appropriate
// settings for tabs and spaces, as a TabsSpaces struct.
func TabsSpacesFromMode(mode Mode) TabsSpaces {
	// Given e.mode, find the matching TabsSpaces struct and set that to e.tabs
	for k, vs := range languageIndentation {
		for _, v := range vs {
			if v == mode {
				return k
			}
		}
	}
	return defaultTabsSpaces
}

// String returns the string for one indentation
func (ts TabsSpaces) String() string {
	if !ts.spaces {
		return "\t"
	}
	return strings.Repeat(" ", ts.perTab)
}

// WSLen will count the length of the given whitespace string, in terms of spaces
func (ts TabsSpaces) WSLen(whitespaceString string) int {
	return strings.Count(whitespaceString, "\t")*ts.perTab + strings.Count(whitespaceString, " ")
}
