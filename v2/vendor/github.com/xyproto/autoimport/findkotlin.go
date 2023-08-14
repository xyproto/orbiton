package autoimport

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/env/v2"
)

const kotlinPath = "/usr/share/kotlin/lib"

// FindKotlin finds the most likely location of a Kotlin installation
// (with subfolders with .jar files) on the system.
func FindKotlin() (string, error) {
	// Find out if "kotlinc" is in the $PATH
	if kotlinExecutablePath := which("kotlinc"); kotlinExecutablePath != "" {
		// Follow the symlink up to three times, if it's a symlink
		followedSymlink := false
		if isSymlink(kotlinExecutablePath) {
			kotlinExecutablePath = followSymlink(kotlinExecutablePath)
			followedSymlink = true
		}
		if isSymlink(kotlinExecutablePath) {
			kotlinExecutablePath = followSymlink(kotlinExecutablePath)
			followedSymlink = true
		}
		if followedSymlink {
			parentDirectory := filepath.Dir(kotlinExecutablePath)
			if isDir(parentDirectory) {
				return parentDirectory, nil
			}
		}
		// Find the definition of KOTLIN_HOME within the kotlinc script
		data, err := os.ReadFile(kotlinExecutablePath)
		if err != nil {
			return "", err
		}
		lines := bytes.Split(data, []byte{'\n'})
		for _, line := range lines {
			if bytes.Contains(line, []byte("KOTLIN_HOME")) && bytes.Count(line, []byte("=")) == 1 {
				fields := bytes.SplitN(line, []byte("="), 2)
				kotlinPath := strings.TrimSpace(string(fields[1]))
				if !isDir(kotlinPath) {
					continue
				}
				return kotlinPath, nil
			}
		}
	}
	// Check if KOTLIN_HOME is defined in /etc/environment
	kotlinPath, err := env.EtcEnvironment("KOTLIN_HOME")
	if err == nil && isDir(kotlinPath) {
		kotlinPathParent := filepath.Dir(kotlinPath)
		if isDir(kotlinPathParent) {
			return kotlinPathParent, nil
		}
		return kotlinPath, nil
	}
	// Consider typical path, for Arch Linux
	if isDir(kotlinPath) {
		return kotlinPath, nil
	}
	return "", errors.New("could not find an installation of Kotlin")
}
