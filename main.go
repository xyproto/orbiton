package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"syscall"

	"github.com/xyproto/env"
	"github.com/xyproto/termtitle"
	"github.com/xyproto/vt100"
)

const (
	versionString = "o 2.48.1"
)

func main() {
	var (
		versionFlag = flag.Bool("version", false, "version information")
		helpFlag    = flag.Bool("help", false, "quick overview of hotkeys")
		forceFlag   = flag.Bool("f", false, "open even if already open")
		cpuProfile  = flag.String("cpuprofile", "", "write CPU profile to `file`")
		memProfile  = flag.String("memprofile", "", "write memory profile to `file`")
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

ctrl-s     to save
ctrl-q     to quit
ctrl-r     to open a portal so that this text can be pasted into another file
ctrl-space to compile programs, render MD to PDF or export adoc/sdoc as man
ctrl-w     for Zig, Rust, V and Go, format with the "... fmt" command
           for C++, format the current file with "clang-format"
           for HTML, format the file with "tidy", for Python: "autopep8"
           for Markdown, toggle checkboxes
           for git interactive rebases, cycle the rebase keywords
ctrl-a     go to start of line, then start of text and then the previous line
ctrl-e     go to end of line and then the next line
ctrl-n     to scroll down 10 lines or go to the next match if a search is active
ctrl-p     to scroll up 10 lines or go to the previous match
ctrl-k     to delete characters to the end of the line, then delete the line
ctrl-g     to toggle filename/line/column/unicode/word count status display
ctrl-d     to delete a single character
ctrl-o     to open the command menu, where the first option is always
           "Save and quit"
ctrl-t     for C and C++, toggle between the header and implementation,
           for Agda, insert a symbol,
           for the rest, record and play back macros.
ctrl-c     to copy the current line, press twice to copy the current block
ctrl-v     to paste one line, press twice to paste the rest
ctrl-x     to cut the current line, press twice to cut the current block
ctrl-b     to toggle a bookmark for the current line, or jump to a bookmark
ctrl-j     to join lines
ctrl-u     to undo (ctrl-z is also possible, but may background the application)
ctrl-l     to jump to a specific line (press return to jump to the top or bottom)
ctrl-f     to find a string, press tab after the text to search and replace
ctrl-\     to toggle single-line comments for a block of code
ctrl-~     to jump to matching parenthesis
esc        to redraw the screen and clear the last search

See the man page for more information.

Set NO_COLOR=1 to disable colors.

`)
		return
	}

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	// Check if the executable starts with "g" or "f"
	var executableName string
	if len(os.Args) > 0 {
		executableName = filepath.Base(os.Args[0])
		if len(executableName) > 0 {
			switch executableName[0] {
			case 'f', 'g':
				// Start the game
				if _, err := Game(); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				} else {
					return
				}
			}
		}
	}

	filename, lineNumber, colNumber := FilenameAndLineNumberAndColNumber(flag.Arg(0), flag.Arg(1), flag.Arg(2))
	if filename == "" {
		fmt.Fprintln(os.Stderr, "please provide a filename")
		os.Exit(1)
	}

	if strings.HasSuffix(filename, ".") && !exists(filename) {
		// If the filename ends with "." and the file does not exist, assume this was a result of tab-completion going wrong.
		// If there are multiple files that exist that start with the given filename, open the one first in the alphabet (.cpp before .o)
		matches, err := filepath.Glob(filename + "*")
		if err == nil && len(matches) > 0 { // no error and at least 1 match
			// Use the first match of the sorted results
			sort.Strings(matches)
			filename = matches[0]
		}
	} else if !strings.Contains(filename, ".") && allLower(filename) && !exists(filename) {
		// The filename has no ".", is written in lowercase and it does not exist,
		// but more than one file that starts with the filename  exists. Assume tab-completion failed.
		matches, err := filepath.Glob(filename + "*")
		if err == nil && len(matches) > 1 { // no error and more than 1 match
			// Use the first match of the sorted results
			sort.Strings(matches)
			filename = matches[0]
		}
	} else if !exists(filename) {
		// Also match "PKGBUILD" if just "Pk" was entered
		matches, err := filepath.Glob(strings.ToTitle(filename) + "*")
		if err == nil && len(matches) >= 1 { // no error and at least 1 match
			// Use the first match of the sorted results
			sort.Strings(matches)
			filename = matches[0]
		}
	}

	// Set the terminal title, if the current terminal emulator supports it, and NO_COLOR is not set
	if !envNoColor {
		termtitle.MustSet(termtitle.GenerateTitle(filename))
	}

	// If the editor executable has been named "red", use the red/gray theme by default
	// Also use the red/gray theme if $SHELL is /bin/csh (typically BSD)
	theme := NewDefaultTheme()
	syntaxHighlight := true
	if envNoColor {
		theme = NewNoColorTheme()
		syntaxHighlight = false
	} else {
		// Check if the executable starts with "r" or "l"
		if len(executableName) > 0 {
			switch executableName[0] {
			case 'r': // red, ro, rb, rt etc
				theme = NewRedBlackTheme()
			case 'l': // light, lo etc
				theme = NewLightTheme()
			}
		}
	}

	// Initialize the VT100 terminal
	tty, err := vt100.NewTTY()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}
	defer tty.Close()

	// Run the main editor loop
	userMessage, err, stopParent := Loop(tty, filename, lineNumber, colNumber, *forceFlag, theme, syntaxHighlight)

	// SIGQUIT the parent PID. Useful if being opened repeatedly by a find command.
	if stopParent {
		defer func() {
			syscall.Kill(os.Getppid(), syscall.SIGQUIT)
		}()
	}

	// Remove the terminal title, if the current terminal emulator supports it
	// and if NO_COLOR is not set.
	if !envNoColor {
		shellName := filepath.Base(env.Str("SHELL", "/bin/sh"))
		termtitle.MustSet(shellName)
	}

	// Clear the current color attribute
	fmt.Print(vt100.Stop())

	// Respond to the error returned from the main loop, if any
	if err != nil {
		if userMessage != "" {
			quitMessage(tty, userMessage)
		} else {
			quitError(tty, err)
		}
	}

	// Output memory profile information, if the flag is given
	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}
