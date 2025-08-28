package main

import (
	"strings"
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
		filenameColor := Red
		renameColor := Magenta
		if strings.HasPrefix(line, "# On branch ") {
			coloredString = DarkGray.Get(line[:12]) + LightCyan.Get(line[12:])
		} else if strings.HasPrefix(line, "# Your branch is up to date with '") && strings.Count(line, "'") == 2 {
			parts := strings.SplitN(line, "'", 3)
			coloredString = DarkGray.Get(parts[0]+"'") + LightGreen.Get(parts[1]) + DarkGray.Get("'"+parts[2])
		} else if line == "# Changes to be committed:" {
			coloredString = DarkGray.Get("# ") + LightBlue.Get("Changes to be committed:")
		} else if line == "# Changes not staged for commit:" {
			coloredString = DarkGray.Get("# ") + LightBlue.Get("Changes not staged for commit:")
		} else if line == "# Untracked files:" {
			coloredString = DarkGray.Get("# ") + LightBlue.Get("Untracked files:")
		} else if strings.Contains(line, "new file:") {
			parts := strings.SplitN(line[1:], ":", 2)
			coloredString = DarkGray.Get("#") + LightYellow.Get(parts[0]) + DarkGray.Get(":") + filenameColor.Get(parts[1])
		} else if strings.Contains(line, "modified:") {
			parts := strings.SplitN(line[1:], ":", 2)
			coloredString = DarkGray.Get("#") + LightYellow.Get(parts[0]) + DarkGray.Get(":") + filenameColor.Get(parts[1])
		} else if strings.Contains(line, "deleted:") {
			parts := strings.SplitN(line[1:], ":", 2)
			coloredString = DarkGray.Get("#") + LightYellow.Get(parts[0]) + DarkGray.Get(":") + filenameColor.Get(parts[1])
		} else if strings.Contains(line, "renamed:") {
			parts := strings.SplitN(line[1:], ":", 2)
			coloredString = DarkGray.Get("#") + LightYellow.Get(parts[0]) + DarkGray.Get(":")
			if strings.Contains(parts[1], "->") {
				filenames := strings.SplitN(parts[1], "->", 2)
				coloredString += renameColor.Get(filenames[0]) + White.Get("->") + renameColor.Get(filenames[1])
			} else {
				coloredString += filenameColor.Get(parts[1])
			}
		} else if fields := strings.Fields(line); strings.HasPrefix(line, "# Rebase ") && len(fields) >= 5 && strings.Contains(fields[2], "..") {
			textColor := LightGray
			commitRange := strings.SplitN(fields[2], "..", 2)
			coloredString = DarkGray.Get("# ") + textColor.Get(fields[1]) + " " + LightBlue.Get(commitRange[0]) + textColor.Get("..") + LightBlue.Get(commitRange[1]) + " " + textColor.Get(fields[3]) + " " + LightBlue.Get(fields[4]) + " " + textColor.Get(strings.Join(fields[5:], " "))
		} else {
			coloredString = DarkGray.Get(line)
		}
	} else if strings.HasPrefix(line, "GIT:") {
		coloredString = DarkGray.Get(line)
	} else if strings.HasPrefix(line, "From ") && strings.Contains(line, "#") {
		parts := strings.SplitN(line, "#", 2)
		// Also syntax highlight the e-mail address
		if strings.Contains(parts[0], "<") && strings.Contains(parts[0], ">") {
			parts1 := strings.SplitN(parts[0], "<", 2)
			parts2 := strings.SplitN(parts1[1], ">", 2)
			coloredString = LightBlue.Get(parts1[0][:5]) + White.Get(parts1[0][5:]) + Red.Get("<") + LightYellow.Get(parts2[0]) + Red.Get(">") + White.Get(parts2[1]) + DarkGray.Get("#"+parts[1])
		} else {
			coloredString = LightCyan.Get(parts[0]) + DarkGray.Get("#"+parts[1])
		}
	} else if hasAnyPrefix(line, []string{"From:", "To:", "Cc:", "Bcc:", "Subject:", "Date:", "Message-Id:", "X-Mailer:", "MIME-Version:", "Content-Type:", "Content-Transfer-Encoding:", "Reply-To:", "In-Reply-To:"}) {
		parts := strings.SplitN(line, ":", 2)
		if strings.Contains(parts[1], "<") && strings.Contains(parts[1], ">") {
			parts1 := strings.SplitN(parts[1], "<", 2)
			parts2 := strings.SplitN(parts1[1], ">", 2)
			coloredString = LightBlue.Get(parts[0]+":") + parts1[0] + Red.Get("<") + LightYellow.Get(parts2[0]) + Red.Get(">") + White.Get(parts2[1])
		} else {
			coloredString = LightBlue.Get(parts[0]+":") + LightYellow.Get(parts[1])
		}
	} else if fields := strings.Fields(line); len(fields) >= 3 && hasAnyPrefixWord(line, []string{"p", "pick", "r", "reword", "e", "edit", "s", "squash", "f", "fixup", "x", "exec", "b", "break", "d", "drop", "l", "label", "t", "reset", "m", "merge"}) {
		coloredString = Red.Get(fields[0]) + " " + LightBlue.Get(fields[1]) + " " + LightGray.Get(strings.Join(fields[2:], " "))
	} else {
		coloredString = e.Git.Get(line)
	}
	return coloredString
}
