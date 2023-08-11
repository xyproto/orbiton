package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xyproto/binary"
	"github.com/xyproto/env/v2"
)

// exists checks if the given path exists
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isFile checks if the given path exists and is a regular file
func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

// isDir checks if the given path exists and is a directory
func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsDir()
}

// isFileOrSymlink checks if the given path exists and is a regular file or a symbolic link
func isFileOrSymlink(path string) bool {
	// use Lstat instead of Stat to avoid following the symlink
	fi, err := os.Lstat(path)
	return err == nil && (fi.Mode().IsRegular() || (fi.Mode()&os.ModeSymlink != 0))
}

// which tries to find the given executable name in the $PATH
// Returns an empty string if not found.
func which(executable string) string {
	if p, err := exec.LookPath(executable); err == nil { // success
		return p
	}
	return ""
}

// aBinDirectory will check if the given filename is in one of these directories:
// /bin, /sbin, /usr/bin, /usr/sbin, /usr/local/bin, /usr/local/sbin, ~/.bin, ~/bin, ~/.local/bin
func aBinDirectory(filename string) bool {
	p, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		return false
	}
	switch p {
	case "/bin", "/sbin", "/usr/bin", "/usr/sbin", "/usr/local/bin", "/usr/local/sbin":
		return true
	}
	homeDir := env.HomeDir()
	switch p {
	case filepath.Join(homeDir, ".bin"), filepath.Join(homeDir, "bin"), filepath.Join("local", "bin"):
		return true
	}
	return false
}

// dataReadyOnStdin checks if data is ready on stdin
func dataReadyOnStdin() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return !(fileInfo.Mode()&os.ModeNamedPipe == 0)
}

// removeBinaryFiles filters out files that are either binary or can not be read from the given slice
func removeBinaryFiles(filenames []string) []string {
	var nonBinaryFilenames []string
	for _, filename := range filenames {
		if isBinary, err := binary.File(filename); !isBinary && err == nil {
			nonBinaryFilenames = append(nonBinaryFilenames, filename)
		}
	}
	return nonBinaryFilenames
}

// timestampedFilename prefixes the given filename with a timestamp
func timestampedFilename(filename string) string {
	now := time.Now()
	year, month, day := now.Date()
	hour, minute, second := now.Clock()
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d-%s", year, int(month), day, hour, minute, second, filename)
}

// shortPath replaces the home directory with ~ in a given path
func shortPath(path string) string {
	homeDir, _ := os.UserHomeDir()
	if strings.HasPrefix(path, homeDir) {
		return strings.Replace(path, homeDir, "~", 1)
	}
	return path
}

// fileHas checks if the given file exists and contains the given string
func fileHas(path, what string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte(what))
}
