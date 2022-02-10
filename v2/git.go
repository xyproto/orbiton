package main

import (
	"strings"

	"github.com/xyproto/vt100"
)

var gitRebasePrefixes = []string{"p", "pick", "f", "fixup", "r", "reword", "d", "drop", "e", "edit", "s", "squash", "x", "exec", "b", "break", "l", "label", "t", "reset", "m", "merge"}

// nextGitRebaseKeywords takes the first word and increase it to the next git rebase keyword
func nextGitRebaseKeyword(line string) string {
	var (
		cycle1 = filterS(gitRebasePrefixes, func(s string) bool { return len(s) > 1 })
		cycle2 = filterS(gitRebasePrefixes, func(s string) bool { return len(s) == 1 })
		first  = strings.Fields(line)[0]
		next   = ""
	)
	// Check if the word is in cycle1, then set "next" to the next one in the cycle
	for i, w := range cycle1 {
		if first == w {
			if i+1 < len(cycle1) {
				next = cycle1[i+1]
				break
			} else {
				next = cycle1[0]
			}
		}
	}
	if next == "" {
		// Check if the word is in cycle2, then set "next" to the next one in the cycle
		for i, w := range cycle2 {
			if first == w {
				if i+1 < len(cycle2) {
					next = cycle2[i+1]
					break
				} else {
					next = cycle2[0]
				}
			}
		}
	}
	if next == "" {
		// Return the line as it is, no git rebase keyword found
		return line
	}
	// Return the line with the keyword replaced with the next one in cycle1 or cycle2
	return strings.Replace(line, first, next, 1)
}

func (e *Editor) gitHighlight(line string) string {
	var coloredString string
	if strings.HasPrefix(line, "#") {
		filenameColor := vt100.Red
		renameColor := vt100.Magenta
		if strings.HasPrefix(line, "# On branch ") {
			coloredString = vt100.DarkGray.Get(line[:12]) + vt100.LightCyan.Get(line[12:])
		} else if strings.HasPrefix(line, "# Your branch is up to date with '") && strings.Count(line, "'") == 2 {
			parts := strings.SplitN(line, "'", 3)
			coloredString = vt100.DarkGray.Get(parts[0]+"'") + vt100.LightGreen.Get(parts[1]) + vt100.DarkGray.Get("'"+parts[2])
		} else if line == "# Changes to be committed:" {
			coloredString = vt100.DarkGray.Get("# ") + vt100.LightBlue.Get("Changes to be committed:")
		} else if line == "# Changes not staged for commit:" {
			coloredString = vt100.DarkGray.Get("# ") + vt100.LightBlue.Get("Changes not staged for commit:")
		} else if line == "# Untracked files:" {
			coloredString = vt100.DarkGray.Get("# ") + vt100.LightBlue.Get("Untracked files:")
		} else if strings.Contains(line, "new file:") {
			parts := strings.SplitN(line[1:], ":", 2)
			coloredString = vt100.DarkGray.Get("#") + vt100.LightYellow.Get(parts[0]) + vt100.DarkGray.Get(":") + filenameColor.Get(parts[1])
		} else if strings.Contains(line, "modified:") {
			parts := strings.SplitN(line[1:], ":", 2)
			coloredString = vt100.DarkGray.Get("#") + vt100.LightYellow.Get(parts[0]) + vt100.DarkGray.Get(":") + filenameColor.Get(parts[1])
		} else if strings.Contains(line, "deleted:") {
			parts := strings.SplitN(line[1:], ":", 2)
			coloredString = vt100.DarkGray.Get("#") + vt100.LightYellow.Get(parts[0]) + vt100.DarkGray.Get(":") + filenameColor.Get(parts[1])
		} else if strings.Contains(line, "renamed:") {
			parts := strings.SplitN(line[1:], ":", 2)
			coloredString = vt100.DarkGray.Get("#") + vt100.LightYellow.Get(parts[0]) + vt100.DarkGray.Get(":")
			if strings.Contains(parts[1], "->") {
				filenames := strings.SplitN(parts[1], "->", 2)
				coloredString += renameColor.Get(filenames[0]) + vt100.White.Get("->") + renameColor.Get(filenames[1])
			} else {
				coloredString += filenameColor.Get(parts[1])
			}
		} else if fields := strings.Fields(line); strings.HasPrefix(line, "# Rebase ") && len(fields) >= 5 && strings.Contains(fields[2], "..") {
			textColor := vt100.LightGray
			commitRange := strings.SplitN(fields[2], "..", 2)
			coloredString = vt100.DarkGray.Get("# ") + textColor.Get(fields[1]) + " " + vt100.LightBlue.Get(commitRange[0]) + textColor.Get("..") + vt100.LightBlue.Get(commitRange[1]) + " " + textColor.Get(fields[3]) + " " + vt100.LightBlue.Get(fields[4]) + " " + textColor.Get(strings.Join(fields[5:], " "))
		} else {
			coloredString = vt100.DarkGray.Get(line)
		}
	} else if fields := strings.Fields(line); len(fields) >= 3 && hasAnyPrefixWord(line, []string{"p", "pick", "r", "reword", "e", "edit", "s", "squash", "f", "fixup", "x", "exec", "b", "break", "d", "drop", "l", "label", "t", "reset", "m", "merge"}) {
		coloredString = vt100.Red.Get(fields[0]) + " " + vt100.LightBlue.Get(fields[1]) + " " + vt100.LightGray.Get(strings.Join(fields[2:], " "))
	} else {
		coloredString = e.Git.Get(line)
	}
	return coloredString
}
