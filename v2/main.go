// main is the main package for the o editor
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/xyproto/files"
	"github.com/xyproto/vt100"
)

const versionString = "Orbiton 2.64.2"

var (
	// quitMut disallows Exit(1) while a file is being saved
	quitMut sync.Mutex

	// avoid writing to ~/.cache ?
	noWriteToCache bool

	cacheDirForDoc = files.ShortPath(filepath.Join(userCacheDir, "o"))

	probablyDoesNotWantToEditExtensions = []string{".7z", ".a", ".bak", ".core", ".gz", ".lock", ".o", ".out", ".pyc", ".pyo", ".swp", ".tar", ".tmp", ".zip"}
)

func main() {
	var (
		copyFlag               bool
		forceFlag              bool
		helpFlag               bool
		monitorAndReadOnlyFlag bool
		noCacheFlag            bool
		pasteFlag              bool
		clearLocksFlag         bool
		lastCommandFlag        bool
		versionFlag            bool
	)

	pflag.BoolVarP(&copyFlag, "copy", "c", false, "copy a file into the clipboard and quit")
	pflag.BoolVarP(&forceFlag, "force", "f", false, "open even if already open")
	pflag.BoolVarP(&helpFlag, "help", "h", false, "quick overview of hotkeys and flags")
	pflag.BoolVarP(&monitorAndReadOnlyFlag, "monitor", "m", false, "open read-only and monitor for changes")
	pflag.BoolVarP(&noCacheFlag, "no-cache", "n", false, "don't write anything to cache directory")
	pflag.BoolVarP(&pasteFlag, "paste", "p", false, "paste the clipboard into the file and quit")
	pflag.BoolVarP(&clearLocksFlag, "clear-locks", "r", false, "clear all file locks")
	pflag.BoolVarP(&lastCommandFlag, "last-command", "l", false, "output the last build or format command")
	pflag.BoolVarP(&versionFlag, "version", "v", false, "version information")

	pflag.Parse()

	if versionFlag {
		fmt.Println(versionString)
		return
	}
	if helpFlag {
		Usage()
		return
	}

	// Output the last used build, export or format command
	if lastCommandFlag {
		data, err := os.ReadFile(lastCommandFile)
		if err != nil {
			fmt.Println("no available last command")
			return
		}
		// Remove the shebang
		firstLineAndRest := strings.SplitN(string(data), "\n", 2)
		if len(firstLineAndRest) != 2 || !strings.HasPrefix(firstLineAndRest[0], "#") {
			fmt.Fprintf(os.Stderr, "unrecognized contents in %s\n", lastCommandFile)
			os.Exit(1)
		}
		theRest := strings.TrimSpace(firstLineAndRest[1])
		replaced := regexp.MustCompile(`/tmp/o\..*$`).ReplaceAllString(theRest, "")
		fmt.Println(replaced)
		return
	}

	noWriteToCache = noCacheFlag || monitorAndReadOnlyFlag

	// If the -p flag is given, just paste the clipboard to the given filename and exit
	if filename := flag.Arg(0); filename != "" && pasteFlag {
		const primaryClipboard = false
		n, headString, tailString, err := WriteClipboardToFile(filename, forceFlag, primaryClipboard)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			quitMut.Lock()
			defer quitMut.Unlock()
			os.Exit(1)
		} else if n == 0 {
			fmt.Fprintf(os.Stderr, "Wrote 0 bytes to %s\n", filename)
			quitMut.Lock()
			defer quitMut.Unlock()
			os.Exit(1)
		}
		// chmod +x if this looks like a shell script or is in ie. /usr/bin
		if filepath.Ext(filename) == ".sh" || files.BinDirectory(filename) || strings.HasPrefix(headString, "#!") {
			os.Chmod(filename, 0o755)
		}
		if tailString != "" {
			fmt.Printf("Wrote %d bytes to %s from the clipboard. Tail bytes: %s\n", n, filename, strings.TrimSpace(strings.ReplaceAll(tailString, "\n", "\\n")))
		} else {
			fmt.Printf("Wrote %d bytes to %s from the clipboard.\n", n, filename)
		}
		return
	}

	// If the -c flag is given, just copy the given filename to the clipboard and exit
	if filename := flag.Arg(0); filename != "" && copyFlag {
		const primaryClipboard = false
		n, tailString, err := SetClipboardFromFile(filename, primaryClipboard)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			quitMut.Lock()
			defer quitMut.Unlock()
			os.Exit(1)
		} else if n == 0 {
			fmt.Fprintf(os.Stderr, "Wrote 0 bytes to %s\n", filename)
			quitMut.Lock()
			defer quitMut.Unlock()
			os.Exit(1)
		}
		if tailString != "" {
			fmt.Printf("Copied %d bytes from %s to the clipboard. Tail bytes: %s\n", n, filename, strings.TrimSpace(strings.ReplaceAll(tailString, "\n", "\\n")))
		} else {
			fmt.Printf("Copied %d bytes from %s to the clipboard.\n", n, filename)
		}
		return
	}

	// If the -r flag is given, clear all file locks and exit.
	if clearLocksFlag {
		lockErr := os.Remove(defaultLockFile)

		// Also remove the portal file
		portalErr := ClearPortal()

		if lockErr == nil && portalErr == nil {
			fmt.Println("Locks cleared and portal closed.")
		} else if lockErr != nil && portalErr == nil {
			fmt.Fprintf(os.Stderr, "Closed the portal, but could not clear locks: %v\n", lockErr)
			os.Exit(1)
		} else if lockErr == nil && portalErr != nil {
			fmt.Fprintf(os.Stderr, "Cleared all locks, but could not close the portal: %v\n", portalErr)
			os.Exit(1)
		} else {
			fmt.Fprintf(os.Stderr, "Could not clear locks and not close the portal, got these errors: %v %v\n", lockErr, portalErr)
			os.Exit(1)
		}
		return
	}

	traceStart() // if building with -tags trace

	// Check if the executable starts with "g" or "f"
	var executableName string
	if len(os.Args) > 0 {
		executableName = filepath.Base(os.Args[0]) // if os.Args[0] is empty, executableName will be "."
		switch executableName[0] {
		case 'f', 'g':
			// Start the game
			if _, err := Game(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				quitMut.Lock()
				defer quitMut.Unlock()
				os.Exit(1)
			}
			return
		}
	}

	var (
		err        error
		fnord      FilenameOrData
		lineNumber LineNumber
		colNumber  ColNumber
	)

	stdinFilename := len(os.Args) == 1 || (len(os.Args) == 2 && (os.Args[1] == "-" || os.Args[1] == "/dev/stdin"))
	// If no regular filename is given, check if data is ready at stdin
	fnord.stdin = stdinFilename && (files.DataReadyOnStdin() || manIsParent())
	if fnord.stdin {
		// TODO: Use a spinner?
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not read from stdin")
			quitMut.Lock()
			defer quitMut.Unlock()
			os.Exit(1)
		}
		// Now stop reading further from stdin
		os.Stdin.Close()

		if lendata := len(data); lendata > 0 {
			fnord.filename = "-"
			fnord.data = data
			fnord.length = lendata
		}
	} else {
		fnord.filename, lineNumber, colNumber = FilenameAndLineNumberAndColNumber(flag.Arg(0), flag.Arg(1), flag.Arg(2))
	}
	// Check if the given filename contains something
	if fnord.Empty() {
		if fnord.filename == "" {
			fmt.Fprintln(os.Stderr, "please provide a filename")
			quitMut.Lock()
			defer quitMut.Unlock()
			os.Exit(1)
		}

		// If the filename starts with "~", then expand it
		fnord.ExpandUser()

		// Check if the given filename is not a file or a symlink
		if !files.IsFileOrSymlink(fnord.filename) {
			if strings.HasSuffix(fnord.filename, ".") {
				// If the filename ends with "." and the file does not exist, assume this was a result of tab-completion going wrong.
				// If there are multiple files that exist that start with the given filename, open the one first in the alphabet (.cpp before .o)
				matches, err := filepath.Glob(fnord.filename + "*")
				if err == nil && len(matches) > 0 { // no error and at least 1 match
					// Filter out any binary files
					matches = files.FilterOutBinaryFiles(matches)
					if len(matches) > 0 {
						sort.Strings(matches)
						// If the matches contains low priority suffixes, such as ".lock", then move it last
						for i, fn := range matches {
							if hasSuffix(fn, probablyDoesNotWantToEditExtensions) {
								// Move this filename last
								matches = append(matches[:i], matches[i+1:]...)
								matches = append(matches, fn)
								break
							}
						}
						// Use the first filename in the list of matches
						fnord.filename = matches[0]
					}
				}
			} else if !strings.Contains(fnord.filename, ".") && allLower(fnord.filename) {
				// The filename has no ".", is written in lowercase and it does not exist,
				// but more than one file that starts with the filename  exists. Assume tab-completion failed.
				matches, err := filepath.Glob(fnord.filename + "*")
				if err == nil && len(matches) > 1 { // no error and more than 1 match
					// Use the first non-binary match of the sorted results
					matches = files.FilterOutBinaryFiles(matches)
					if len(matches) > 0 {
						sort.Strings(matches)
						fnord.filename = matches[0]
					}
				}
			} else {
				// Also match ie. "PKGBUILD" if just "Pk" was entered
				matches, err := filepath.Glob(strings.ToTitle(fnord.filename) + "*")
				if err == nil && len(matches) >= 1 { // no error and at least 1 match
					// Use the first non-binary match of the sorted results
					matches = files.FilterOutBinaryFiles(matches)
					if len(matches) > 0 {
						sort.Strings(matches)
						fnord.filename = matches[0]
					}
				}
			}
		}
	}

	// Set the terminal title, if the current terminal emulator supports it, and NO_COLOR is not set
	fnord.SetTitle()

	// If the editor executable has been named "red", use the red/gray theme by default
	// Also use the red/gray theme if $SHELL is /bin/csh (typically BSD)
	theme := NewDefaultTheme()
	syntaxHighlight := true
	nanoMode := false
	if envNoColor {
		theme = NewNoColorDarkBackgroundTheme()
		syntaxHighlight = false
	} else if len(executableName) > 0 {
		// Check if the executable name is a specific word
		if executableName == "nano" || executableName == "nan" {
			nanoMode = true
		}
		// Check if the executable starts with a specific letter
		specificLetter = true
		switch executableName[0] {
		case 'b', 'e': // bo, borland, ed, edit etc.
			theme = NewDarkBlueEditTheme()
			// TODO: Later, when specificLetter is examined, use either NewEditLightTheme or NewEditDarkTheme
			editTheme = true
		case 'l': // lo, light etc
			theme = NewLitmusTheme()
		case 'v': // vs, vscode etc.
			theme = NewDarkVSTheme()
		case 'r': // rb, ro, rt, red etc.
			theme = NewRedBlackTheme()
		case 's': // s, sw, synthwave etc.
			theme = NewSynthwaveTheme()
		default:
			specificLetter = false
		}
	}

	// TODO: Move this to themes.go
	if nanoMode { // make the status bar stand out
		theme.StatusBackground = theme.DebugInstructionsBackground
		theme.StatusErrorBackground = theme.DebugInstructionsBackground
	}

	// Initialize the VT100 terminal
	tty, err := vt100.NewTTY()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		quitMut.Lock()
		defer quitMut.Unlock()
		os.Exit(1)
	}
	defer tty.Close()

	// Run the main editor loop
	userMessage, stopParent, err := Loop(tty, fnord, lineNumber, colNumber, forceFlag, theme, syntaxHighlight, monitorAndReadOnlyFlag, nanoMode)

	// SIGQUIT the parent PID. Useful if being opened repeatedly by a find command.
	if stopParent {
		defer func() {
			syscall.Kill(os.Getppid(), syscall.SIGQUIT)
		}()
	}

	// Remove the terminal title, if the current terminal emulator supports it
	// and if NO_COLOR is not set.
	NoTitle()

	// Clear the current color attribute
	fmt.Print(vt100.Stop())

	traceComplete() // if building with -tags trace

	// Respond to the error returned from the main loop, if any
	if err != nil {
		if userMessage != "" {
			quitMessage(tty, userMessage)
		} else {
			quitError(tty, err)
		}
	}
}
