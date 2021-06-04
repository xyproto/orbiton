package main

import "strings"

var defaultTabsSpaces = map[Mode]TabsSpaces{
	modeJava: TabsSpaces{4, false},
}

// TabsSpaces contains all info needed about tabs and spaces for a file
type TabsSpaces struct {
	spacesPerTab int
	tabs         bool // tabs, or spaces?
}

// String returns the string for one indentation
func (ts TabsSpaces) String() string {
	if ts.tabs {
		return "\t"
	}
	return strings.Repeat(" ", ts.spacesPerTab)
}

func (e *Editor) adjustTabsAndSpaces() {
	// Additional per-mode considerations, before launching the editor
	switch e.mode {
	case modeMakefile, modePython, modeCMake, modeJava, modeKotlin, modeZig, modeBattlestar, modeScala, modeCS:
		e.tabs = TabsSpaces{4, false}
	case modeShell, modeConfig, modeHaskell, modeVim, modeLua, modeObjectPascal, modeJSON, modeHTML, modeXML:
		e.tabs = TabsSpaces{2, false}
	case modeAda:
		e.tabs = TabsSpaces{3, false}
	case modeMarkdown, modeText, modeBlank:
		e.rainbowParenthesis = false
	}
}
