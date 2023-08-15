package main

import (
	"os"
	"runtime"
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
)

var (
	isDarwinCache *bool
	isLinuxCache  *bool
)

// isDarwin checks if the current OS is Darwin, and caches the result
func isDarwin() bool {
	if isDarwinCache != nil {
		return *isDarwinCache
	}
	b := runtime.GOOS == "darwin"
	isDarwinCache = &b
	return b
}

// isLinux checks if the current OS is Linux, and caches the result
func isLinux() bool {
	if isLinuxCache != nil {
		return *isLinuxCache
	}
	b := runtime.GOOS == "linux"
	isLinuxCache = &b
	return b
}

// getFullName tries to find the full name of the current user
func getFullName() (fullName string) {
	// Start out with whatever is in $LOGNAME, then capitalize the words
	fullName = capitalizeWords(env.Str("LOGNAME", "name"))
	// Then look for ~/.gitconfig
	gitConfigFilename := env.ExpandUser("~/.gitconfig")
	if files.Exists(gitConfigFilename) {
		data, err := os.ReadFile(gitConfigFilename)
		if err != nil {
			return fullName
		}
		// Look for a line starting with "name =", in the "[user]" section
		inUserSection := false
		for _, line := range strings.Split(string(data), "\n") {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine == "[user]" {
				inUserSection = true
				continue
			} else if strings.HasPrefix(trimmedLine, "[") {
				inUserSection = false
				continue
			}
			if inUserSection && strings.HasPrefix(trimmedLine, "name =") {
				foundName := strings.TrimSpace(strings.SplitN(trimmedLine, "name =", 2)[1])
				if len(foundName) > len(fullName) {
					fullName = foundName
				}
			}
		}
	}
	return fullName
}
