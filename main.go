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

	"github.com/xyproto/termtitle"
	"github.com/xyproto/vt100"
)

const version = "o 2.32.2"

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
ctrl-r     to open a portal so that this text can be pasted into another file with ctrl-v
		   for git interactive rebases, cycle the rebase keywords
ctrl-w     for Zig, Rust, V and Go, format with the "... fmt" command
           for C++, format the current file with "clang-format"
           for markdown, toggle checkboxes
           for git interactive rebases, cycle the rebase keywords
ctrl-a     go to start of line, then start of text and then the previous line
ctrl-e     go to end of line and then the next line
ctrl-n     to scroll down 10 lines or go to the next match if a search is active
ctrl-p     to scroll up 10 lines or go to the previous match
ctrl-k     to delete characters to the end of the line, then delete the line
ctrl-g     to toggle filename/line/column/unicode/word count status display
ctrl-d     to delete a single character
ctrl-o     to open the command menu, where the first option is always "Save and quit"
ctrl-t     to render the current document to PDF
           for C++, toggle between the header and implementation
ctrl-c     to copy the current line, press twice to copy the current block
ctrl-v     to paste one line, press twice to paste the rest
ctrl-x     to cut the current line, press twice to cut the current block
ctrl-b     to toggle a bookmark for the current line, or jump to a bookmark
ctrl-j     to join lines
ctrl-u     to undo (ctrl-z is also possible, but may background the application)
ctrl-l     to jump to a specific line (or press return to jump to the top)
ctrl-f     to find a string
esc        to redraw the screen and clear the last search
ctrl-space to build Go, C++, Zig, V, Rust, Haskell, Markdown, Adoc or Sdoc
ctrl-\     to toggle single-line comments for a block of code
ctrl-~     to jump to matching parenthesis

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

	filename, lineNumber := FilenameAndLineNumber(flag.Arg(0), flag.Arg(1))
	if filename == "" {
		fmt.Fprintln(os.Stderr, "Need a filename.")
		os.Exit(1)
	}

	// If the filename ends with "." and the file does not exist, assume this was a result of tab-completion going wrong.
	if strings.HasSuffix(filename, ".") && !exists(filename) {
		// If there are multiple files that exist that start with the given filename, open the one first in the alphabet (.cpp before .o)
		matches, err := filepath.Glob(filename + "*")
		if err == nil && len(matches) > 0 { // no error and at least 1 match
			sort.Strings(matches)
			filename = matches[0]
		}
	}

	// Set the terminal title, if the current terminal emulator supports it
	if absFilename, err := filepath.Abs(filename); err != nil {
		termtitle.MustSet(filename)
	} else {
		termtitle.MustSet(absFilename)
	}

	// Initialize the VT100 terminal
	tty, err := vt100.NewTTY()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}
	defer tty.Close()

	// Run the main editor loop
	userMessage, err := RunMainLoop(tty, filename, lineNumber, *forceFlag)

	// Remove the terminal title, if the current terminal emulator supports it
	shellName := filepath.Base(os.Getenv("SHELL"))
	termtitle.MustSet(shellName)

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
