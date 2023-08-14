package autoimport

import (
	"errors"
	"path/filepath"

	"github.com/xyproto/env/v2"
)

const (
	archJavaPath    = "/usr/lib/jvm/default"
	debianJavaPath  = "/usr/lib/jvm/default-java"
	freeBSDJavaPath = "/usr/local/share/java/classes"
)

// FindJava finds the most likely location of a Java installation
// (with subfolders with .jar files) on the system.
func FindJava() (string, error) {
	// Respect $JAVA_HOME, if it's set
	if javaHomePath := env.Str("JAVA_HOME"); javaHomePath != "" && isDir(javaHomePath) {
		if isDir(javaHomePath) {
			return javaHomePath, nil
		}
	}
	// Find out if "java" is in the $PATH
	if javaExecutablePath := which("java"); javaExecutablePath != "" {
		// Follow the symlink up to three times, if it's a symlink
		followedSymlink := false
		if isSymlink(javaExecutablePath) {
			javaExecutablePath = followSymlink(javaExecutablePath)
			followedSymlink = true
		}
		if isSymlink(javaExecutablePath) {
			javaExecutablePath = followSymlink(javaExecutablePath)
			followedSymlink = true
		}
		if followedSymlink {
			parentDirectory := filepath.Dir(javaExecutablePath)
			if isDir(parentDirectory) {
				return parentDirectory, nil
			}
		}
		// Return the grandparent directory of the java executable (since it's typically in the "bin" directory")
		if grandParentDirectory := filepath.Dir(filepath.Dir(javaExecutablePath)); isDir(grandParentDirectory) {
			// If the directory is "x64", do another ".."
			if filepath.Base(grandParentDirectory) == "x64" {
				grandParentDirectory = filepath.Dir(grandParentDirectory)
			}
			return grandParentDirectory, nil
		}
	}
	// Check if JAVA_HOME is defined in /etc/environment
	javaPath, err := env.EtcEnvironment("JAVA_HOME")
	if err == nil && isDir(javaPath) {
		javaPathParent := filepath.Dir(javaPath)
		if isDir(javaPathParent) {
			return javaPathParent, nil
		}
		return javaPath, nil
	}
	// Consider typical paths, for Arch Linux, Debian/Ubuntu and FreeBSD
	if isDir(archJavaPath) {
		return archJavaPath, nil
	} else if isDir(debianJavaPath) {
		return debianJavaPath, nil
	} else if isDir(freeBSDJavaPath) {
		return freeBSDJavaPath, nil
	}
	return "", errors.New("could not find an installation of Java")
}
