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

	"github.com/xyproto/env"
	"github.com/xyproto/termtitle"
	"github.com/xyproto/vt100"
)

const (
	version = "o 2.37.0"

	defaultTheme Theme = iota
	redBlackTheme
	lightTheme
)

// Theme is an "enum" type
type Theme int

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
		fmt.Println(version)
		return
	}

	if *helpFlag {
		fmt.Println(version + " - simple and limited text editor")
		fmt.Print(`
Hotkeys

ctrl-s     to save
ctrl-q     to quit
ctrl-r     to open a portal so that this text can be pasted into another file
ctrl-space to build Go, C++, Zig, V, Rust, Haskell, Markdown, Adoc or Sdoc
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
ctrl-t     to render the current document to PDF
           for C++, toggle between the header and implementation
ctrl-c     to copy the current line, press twice to copy the current block
ctrl-v     to paste one line, press twice to paste the rest
ctrl-x     to cut the current line, press twice to cut the current block
ctrl-b     to toggle a bookmark for the current line, or jump to a bookmark
ctrl-j     to join lines
ctrl-u     to undo (ctrl-z is also possible, but may background the application)
ctrl-l     to jump to a specific line (press return to jump to the top or bottom)
ctrl-f     to find a string
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

	filename, lineNumber, colNumber := FilenameAndLineNumberAndColNumber(flag.Arg(0), flag.Arg(1), flag.Arg(2))
	if filename == "" {
		fmt.Fprintln(os.Stderr, "please provide a filename")
		os.Exit(1)
	}

	// If the filename ends with "." and the file does not exist, assume this was a result of tab-completion going wrong.
	if strings.HasSuffix(filename, ".") && !exists(filename) {
		// If there are multiple files that exist that start with the given filename, open the one first in the alphabet (.cpp before .o)
		matches, err := filepath.Glob(filename + "*")
		if err == nil && len(matches) > 0 { // no error and at least 1 match
			// Use the first match of the sorted results
			sort.Strings(matches)
			filename = matches[0]
		}
	} else if !strings.Contains(filename, ".") && isLower(filename) && !exists(filename) {
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
	if !hasE("NO_COLOR") {
		termtitle.MustSet(generateTitle(filename))
	}

	// Initialize the VT100 terminal
	tty, err := vt100.NewTTY()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}
	defer tty.Close()

	// If the editor executable has been named "red", use the red/gray theme by default
	// Also use the red/gray theme if $SHELL is /bin/csh (typically BSD)
	useTheme := defaultTheme
	if filepath.Base(os.Args[0]) == "red" {
		useTheme = redBlackTheme
	} else if filepath.Base(os.Args[0]) == "light" {
		useTheme = lightTheme
	} else if filepath.Base(os.Args[0]) == "default" {
		useTheme = defaultTheme
	}

	// Run the main editor loop
	userMessage, err := Loop(tty, filename, lineNumber, colNumber, *forceFlag, useTheme)

	// Remove the terminal title, if the current terminal emulator supports it
	// and if NO_COLOR is not set.
	if !hasE("NO_COLOR") {
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
