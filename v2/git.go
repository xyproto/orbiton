package main

import (
	"strings"

	"github.com/xyproto/vt"
)

var gitRebasePrefixes = []string{"p", "pick", "f", "fixup", "r", "reword", "d", "drop", "e", "edit", "s", "squash", "x", "exec", "b", "break", "l", "label", "t", "reset", "m", "merge"}

// getNextInCycle returns the next element in the cycle, given the current element and the cycle
func getNextInCycle(current string, cycle []string) string {
	for i, w := range cycle {
		if current == w {
			if i+1 < len(cycle) {
				return cycle[i+1]
			}
			return cycle[0]
		}
	}
	return ""
}

// nextGitRebaseKeyword will use the first word in the given line,
// and replace it with the next git rebase keyword (as ordered in gitRebasePrefixes)
func nextGitRebaseKeyword(line string) string {
	cycle1 := filterS(gitRebasePrefixes, func(s string) bool { return len(s) > 1 })
	cycle2 := filterS(gitRebasePrefixes, func(s string) bool { return len(s) == 1 })

	firstWord := strings.Fields(line)[0]
	next := getNextInCycle(firstWord, cycle1)

	if next == "" {
		next = getNextInCycle(firstWord, cycle2)
	}

	if next == "" {
		// Return the line as it is, no git rebase keyword found
		return line
	}

	// Return the line with the keyword replaced with the next one in cycle1 or cycle2
	return strings.Replace(line, firstWord, next, 1)
}

// syntax highlighting for git commit messages
func (e *Editor) gitHighlight(line string) string {
	// TODO: Refactor
	var coloredString string
	if strings.HasPrefix(line, "#") {
		filenameColor := vt.Red
		renameColor := vt.Magenta
		if strings.HasPrefix(line, "# On branch ") {
			coloredString = vt.DarkGray.Get(line[:12]) + vt.LightCyan.Get(line[12:])
		} else if strings.HasPrefix(line, "# Your branch is up to date with '") && strings.Count(line, "'") == 2 {
			parts := strings.SplitN(line, "'", 3)
			coloredString = vt.DarkGray.Get(parts[0]+"'") + vt.LightGreen.Get(parts[1]) + vt.DarkGray.Get("'"+parts[2])
		} else if line == "# Changes to be committed:" {
			coloredString = vt.DarkGray.Get("# ") + vt.LightBlue.Get("Changes to be committed:")
		} else if line == "# Changes not staged for commit:" {
			coloredString = vt.DarkGray.Get("# ") + vt.LightBlue.Get("Changes not staged for commit:")
		} else if line == "# Untracked files:" {
			coloredString = vt.DarkGray.Get("# ") + vt.LightBlue.Get("Untracked files:")
		} else if strings.Contains(line, "new file:") {
			parts := strings.SplitN(line[1:], ":", 2)
			coloredString = vt.DarkGray.Get("#") + vt.LightYellow.Get(parts[0]) + vt.DarkGray.Get(":") + filenameColor.Get(parts[1])
		} else if strings.Contains(line, "modified:") {
			parts := strings.SplitN(line[1:], ":", 2)
			coloredString = vt.DarkGray.Get("#") + vt.LightYellow.Get(parts[0]) + vt.DarkGray.Get(":") + filenameColor.Get(parts[1])
		} else if strings.Contains(line, "deleted:") {
			parts := strings.SplitN(line[1:], ":", 2)
			coloredString = vt.DarkGray.Get("#") + vt.LightYellow.Get(parts[0]) + vt.DarkGray.Get(":") + filenameColor.Get(parts[1])
		} else if strings.Contains(line, "renamed:") {
			parts := strings.SplitN(line[1:], ":", 2)
			coloredString = vt.DarkGray.Get("#") + vt.LightYellow.Get(parts[0]) + vt.DarkGray.Get(":")
			if strings.Contains(parts[1], "->") {
				filenames := strings.SplitN(parts[1], "->", 2)
				coloredString += renameColor.Get(filenames[0]) + vt.White.Get("->") + renameColor.Get(filenames[1])
			} else {
				coloredString += filenameColor.Get(parts[1])
			}
		} else if fields := strings.Fields(line); strings.HasPrefix(line, "# Rebase ") && len(fields) >= 5 && strings.Contains(fields[2], "..") {
			textColor := vt.LightGray
			commitRange := strings.SplitN(fields[2], "..", 2)
			coloredString = vt.DarkGray.Get("# ") + textColor.Get(fields[1]) + " " + vt.LightBlue.Get(commitRange[0]) + textColor.Get("..") + vt.LightBlue.Get(commitRange[1]) + " " + textColor.Get(fields[3]) + " " + vt.LightBlue.Get(fields[4]) + " " + textColor.Get(strings.Join(fields[5:], " "))
		} else {
			coloredString = vt.DarkGray.Get(line)
		}
	} else if strings.HasPrefix(line, "GIT:") {
		coloredString = vt.DarkGray.Get(line)
	} else if strings.HasPrefix(line, "From ") && strings.Contains(line, "#") {
		parts := strings.SplitN(line, "#", 2)
		// Also syntax highlight the e-mail address
		if strings.Contains(parts[0], "<") && strings.Contains(parts[0], ">") {
			parts1 := strings.SplitN(parts[0], "<", 2)
			parts2 := strings.SplitN(parts1[1], ">", 2)
			coloredString = vt.LightBlue.Get(parts1[0][:5]) + vt.White.Get(parts1[0][5:]) + vt.Red.Get("<") + vt.LightYellow.Get(parts2[0]) + vt.Red.Get(">") + vt.White.Get(parts2[1]) + vt.DarkGray.Get("#"+parts[1])
		} else {
			coloredString = vt.LightCyan.Get(parts[0]) + vt.DarkGray.Get("#"+parts[1])
		}
	} else if hasAnyPrefix(line, []string{"From:", "To:", "Cc:", "Bcc:", "Subject:", "Date:", "Message-Id:", "X-Mailer:", "MIME-Version:", "Content-Type:", "Content-Transfer-Encoding:", "Reply-To:", "In-Reply-To:"}) {
		parts := strings.SplitN(line, ":", 2)
		if strings.Contains(parts[1], "<") && strings.Contains(parts[1], ">") {
			parts1 := strings.SplitN(parts[1], "<", 2)
			parts2 := strings.SplitN(parts1[1], ">", 2)
			coloredString = vt.LightBlue.Get(parts[0]+":") + parts1[0] + vt.Red.Get("<") + vt.LightYellow.Get(parts2[0]) + vt.Red.Get(">") + vt.White.Get(parts2[1])
		} else {
			coloredString = vt.LightBlue.Get(parts[0]+":") + vt.LightYellow.Get(parts[1])
		}
	} else if fields := strings.Fields(line); len(fields) >= 3 && hasAnyPrefixWord(line, []string{"p", "pick", "r", "reword", "e", "edit", "s", "squash", "f", "fixup", "x", "exec", "b", "break", "d", "drop", "l", "label", "t", "reset", "m", "merge"}) {
		coloredString = vt.Red.Get(fields[0]) + " " + vt.LightBlue.Get(fields[1]) + " " + vt.LightGray.Get(strings.Join(fields[2:], " "))
	} else {
		coloredString = e.Git.Get(line)
	}
	return coloredString
}
