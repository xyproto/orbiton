// main is the main package for the o editor
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/xyproto/vt100"
)

const versionString = "Orbiton 2.62.8"

var (
	// quitMut disallows Exit(1) while a file is being saved
	quitMut sync.Mutex

	// avoid writing to ~/.cache ?
	noWriteToCache bool

	cacheDirForDoc = shortPath(filepath.Join(userCacheDir, "o"))
)

func main() {
	var (
		copyFlag       = flag.Bool("c", false, "copy a file into the clipboard and quit")
		forceFlag      = flag.Bool("f", false, "open even if already open")
		helpFlag       = flag.Bool("help", false, "quick overview of hotkeys and flags")
		pasteFlag      = flag.Bool("p", false, "paste the clipboard into the file and quit")
		clearLocksFlag = flag.Bool("r", false, "clear all file locks")
		noCacheFlag    = flag.Bool("n", false, "don't write anything to "+cacheDirForDoc)
		versionFlag    = flag.Bool("version", false, "version information")
	)

	flag.Parse()

	if *versionFlag {
		fmt.Println(versionString)
		return
	}

	if *helpFlag {
		fmt.Println(versionString + " - simple and limited text editor")
		fmt.Print(`
Hotkeys

ctrl-s      to save
ctrl-q      to quit
ctrl-o      to open the command menu
ctrl-r      to open a portal so that text can be pasted into another file with ctrl-v
ctrl-space  to compile programs, render MD to PDF or export adoc/sdoc as man
ctrl-w      for Zig, Rust, V and Go, format with the "... fmt" command
            for C++, format the current file with "clang-format"
            for HTML, format the file with "tidy", for Python: "autopep8"
            for Markdown, toggle checkboxes or re-format tables
            for git interactive rebases, cycle the rebase keywords
ctrl-g      to display simple help 2 times, then toggle the status bar
            can jump to definition (experimental feature), and back with ctrl-t
ctrl-_      insert a symbol by typing in a two letter ViM-style digraph
            see https://raw.githubusercontent.com/xyproto/digraph/main/digraphs.txt
ctrl-a      go to start of line, then start of text and then the previous line
ctrl-e      go to end of line and then the next line
ctrl-n      to scroll down 10 lines or go to the next match if a search is active
            insert a column when in the Markdown table editor
ctrl-p      to scroll up 10 lines or go to the previous match
            or jump to a matching parenthesis or bracket, if on one
            remove an empty column when in the Markdown table editor
ctrl-k      to delete characters to the end of the line, then delete the line
ctrl-j      to join lines
ctrl-d      to delete a single character
ctrl-t      for C and C++, toggle between the header and implementation,
            for Markdown, launch the Markdown table editor if the cursor is on a table
            for Agda, insert a symbol,
            for the rest, record and then play back a macro
ctrl-c      to copy the current line, press twice to copy the current block
ctrl-v      to paste one line, press twice to paste the rest
ctrl-x      to cut the current line, press twice to cut the current block
ctrl-b      to toggle a bookmark for the current line, or jump to a bookmark
ctrl-u      to undo (ctrl-z is also possible, but may background the application)
ctrl-l      to jump to a specific line (press return to jump to the top or bottom)
ctrl-f      to find a string, press Tab after the text to search and replace
ctrl-\      to toggle single-line comments for a block of code
ctrl-~      to jump to matching parenthesis
esc         to redraw the screen and clear the last search

Set NO_COLOR=1 to disable colors.

Flags:
  -r                         - clear all file locks
  -c FILENAME                - just copy a file into the clipboard
  -p FILENAME                - just paste the contents of the clipboard into a file
  -f                         - force, ignore file locks or combine with -p to overwrite files
  -n                         - avoid writing the location history, search history, highscore,
                               compilation and format command to ` + cacheDirForDoc + `
  --version                  - show the current version

See the man page for more information.

`)
		return
	}

	noWriteToCache = *noCacheFlag

	// If the -p flag is given, just paste the clipboard to the given filename and exit
	if filename := flag.Arg(0); filename != "" && *pasteFlag {
		n, headString, tailString, err := WriteClipboardToFile(filename, *forceFlag)
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
		if filepath.Ext(filename) == ".sh" || aBinDirectory(filename) || strings.HasPrefix(headString, "#!") {
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
	if filename := flag.Arg(0); filename != "" && *copyFlag {
		n, tailString, err := SetClipboardFromFile(filename)
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
	if *clearLocksFlag {
		// If the -n flag is also given (to avoid writing to ~/.cache), then ignore it.
		if err := os.Remove(defaultLockFile); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			fmt.Println("Locks cleared")
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
	fnord.stdin = stdinFilename && (dataReadyOnStdin() || manIsParent())
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

		// Check if the given filename exists
		if !exists(fnord.filename) {
			if strings.HasSuffix(fnord.filename, ".") {
				// If the filename ends with "." and the file does not exist, assume this was a result of tab-completion going wrong.
				// If there are multiple files that exist that start with the given filename, open the one first in the alphabet (.cpp before .o)
				matches, err := filepath.Glob(fnord.filename + "*")
				if err == nil && len(matches) > 0 { // no error and at least 1 match
					// Use the first non-binary match of the sorted results
					matches = removeBinaryFiles(matches)
					if len(matches) > 0 {
						sort.Strings(matches)
						fnord.filename = matches[0]
					}
				}
			} else if !strings.Contains(fnord.filename, ".") && allLower(fnord.filename) {
				// The filename has no ".", is written in lowercase and it does not exist,
				// but more than one file that starts with the filename  exists. Assume tab-completion failed.
				matches, err := filepath.Glob(fnord.filename + "*")
				if err == nil && len(matches) > 1 { // no error and more than 1 match
					// Use the first non-binary match of the sorted results
					matches = removeBinaryFiles(matches)
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
					matches = removeBinaryFiles(matches)
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
	if envNoColor {
		theme = NewNoColorDarkBackgroundTheme()
		syntaxHighlight = false
	} else {
		// Check if the executable starts with a specific letter
		if len(executableName) > 0 {
			switch executableName[0] {
			case 'b', 'e': // bo, borland, ed, edit etc.
				theme = NewDarkBlueEditTheme()
				// TODO: Later, when specificLetter is examined, use either NewEditLightTheme or NewEditDarkTheme
				specificLetter = true
				editTheme = true
			case 'l', 'v': // lo, light, vs, vscode etc.
				theme = NewDarkVSTheme()
				specificLetter = true
			case 'r': // rb, ro, rt, red etc.
				theme = NewRedBlackTheme()
				specificLetter = true
			case 's': // s, sw, synthwave etc.
				theme = NewSynthwaveTheme()
				specificLetter = true
			}
		}
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
	userMessage, stopParent, err := Loop(tty, fnord, lineNumber, colNumber, *forceFlag, theme, syntaxHighlight)

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
