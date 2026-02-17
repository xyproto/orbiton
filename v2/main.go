// main is the main package for the o editor
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/pflag"
	"github.com/xyproto/clip"
	"github.com/xyproto/digraph"
	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/globi"
	"github.com/xyproto/megafile"
	"github.com/xyproto/vt"
)

const (
	versionString = "Orbiton 2.71.0"

	// Timing for slow terminals (vt100, vt220, linux/BSD consoles)
	slowReadTimeout = 50 * time.Millisecond
	slowEscTimeout  = 250 * time.Millisecond

	// Timing for fast terminals (xterm, alacritty, konsole, etc.)
	fastReadTimeout = 2 * time.Millisecond
	fastEscTimeout  = 50 * time.Millisecond
)

var (
	editorLaunchTime = time.Now()

	// The executable name is in arg 0
	editorExecutable     = filepath.Base(os.Args[0]) // using os.Args to get the executable name
	editorExecutablePath string                      // full path to the executable

	// quitMut disallows Exit(1) while a file is being saved
	quitMut sync.Mutex

	// avoid writing to ~/.cache ?
	noWriteToCache bool

	cacheDirForDoc = files.ShortPath(filepath.Join(userCacheDir, "o"))

	// Only for the filename completion, when starting the editor
	probablyDoesNotWantToEditExtensions = []string{".7z", ".a", ".bak", ".core", ".exe", ".gz", ".img", ".lock", ".o", ".out", ".pkg", ".pyc", ".pyo", ".swp", ".tar", ".tmp", ".xz", ".zip"}

	// For when building and running programs with ctrl-space
	inputFileWhenRunning string

	// Check if the parent process is "man"
	parentIsMan *bool

	// Build with release mode instead of debug mode whenever applicable
	releaseBuildFlag bool

	// An empty *Ollama struct
	ollama = NewOllama()

	// Check if TERM is set to vt100 or linux (Linux console) for ASCII fallback
	useASCII = env.Str("TERM") == "vt100" || env.Str("TERM") == "linux"

	// Check if $NO_COLOR is set, or if the terminal is strict VT100 (Linux console supports colors)
	envNoColor = env.Bool("NO_COLOR") || env.Str("TERM") == "vt100"

	// Arguments given on the command line
	globalArgs []string
)

func main() {
	var (
		batFlag                bool
		buildFlag              bool
		catFlag                bool
		clearLocksFlag         bool
		copyFlag               bool
		createDirectoriesFlag  bool
		forceFlag              bool
		formatFlag             bool
		helpFlag               bool
		lastCommandFlag        bool
		listDigraphsFlag       bool
		monitorAndReadOnlyFlag bool
		nanoMode               bool
		noApproxMatchFlag      bool
		noCacheFlag            bool
		ollamaEnabled          bool
		pasteFlag              bool
		quickHelpFlag          bool
		noQuickHelpFlag        bool
		versionFlag            bool
		searchAndOpenFlag      bool
		escToExitFlag          bool
		cycleFilenamesFlag     bool // for internal use
		upsieFlag              bool
	)

	// Available short options: j k

	pflag.BoolVarP(&batFlag, "bat", "B", false, "Cat the file with colors instead of editing it, using bat")
	pflag.BoolVarP(&buildFlag, "build", "b", false, "Try to build the file instead of editing it")
	pflag.BoolVarP(&catFlag, "list", "t", false, "List the file with colors instead of editing it")
	pflag.BoolVarP(&clearLocksFlag, "clear-locks", "e", false, "clear all file locks")
	pflag.BoolVarP(&copyFlag, "copy", "c", false, "copy a file into the clipboard and quit")
	pflag.BoolVarP(&createDirectoriesFlag, "create-dir", "d", false, "create diretories when opening a new file")
	pflag.BoolVarP(&forceFlag, "force", "f", false, "open even if already open")
	pflag.BoolVarP(&formatFlag, "format", "F", false, "Try to build the file instead of editing it")
	pflag.BoolVarP(&helpFlag, "help", "h", false, "quick overview of hotkeys and flags")
	pflag.BoolVarP(&lastCommandFlag, "last-command", "l", false, "output the last build or format command")
	pflag.BoolVarP(&listDigraphsFlag, "digraphs", "s", false, "List digraphs")
	pflag.BoolVarP(&monitorAndReadOnlyFlag, "monitor", "m", false, "open read-only and monitor for changes")
	pflag.BoolVarP(&nanoMode, "nano", "a", false, "Nano/Pico mode")
	pflag.BoolVarP(&noApproxMatchFlag, "noapprox", "x", false, "Disable approximate filename matching")
	pflag.BoolVarP(&noCacheFlag, "no-cache", "n", false, "don't write anything to cache directory")
	pflag.BoolVarP(&ollamaEnabled, "ollama", "o", env.Bool("ORBITON_OLLAMA"), "enable Ollama-specific features")
	pflag.BoolVarP(&pasteFlag, "paste", "p", false, "paste the clipboard into the file and quit")
	pflag.BoolVarP(&releaseBuildFlag, "release", "r", false, "build with release mode instead of debug mode, whenever applicable")
	pflag.BoolVarP(&quickHelpFlag, "quick-help", "q", false, "always display the quick help when starting")
	pflag.BoolVarP(&noQuickHelpFlag, "no-quick-help", "z", false, "never display the quick help when starting")
	pflag.BoolVarP(&versionFlag, "version", "v", false, "version information")
	pflag.StringVarP(&inputFileWhenRunning, "input-file", "i", "input.txt", "input file when building and running programs")
	pflag.BoolVarP(&searchAndOpenFlag, "glob", "g", false, "open the first filename that matches the given glob (recursively)")
	pflag.BoolVarP(&escToExitFlag, "esc", "y", false, "press Esc to exit the program")
	pflag.BoolVarP(&cycleFilenamesFlag, "cycle", "w", false, "cycle files with ctrl-n and ctrl-p (for internal use)")
	pflag.BoolVarP(&upsieFlag, "upsie", "u", false, "show uname+uptime info and exit")

	pflag.CommandLine.MarkHidden("cycle")

	pflag.Parse()

	// Get the full path to the executable for use by megafile
	if absPath, err := filepath.Abs(os.Args[0]); err == nil {
		editorExecutablePath = absPath
	} else {
		editorExecutablePath = os.Args[0]
	}

	if versionFlag {
		fmt.Println(versionString)
		return
	}

	if upsieFlag {
		const fullKernelVersion = false
		if taggedUptimeString, err := megafile.UpsieString(fullKernelVersion); err != nil {
			vt.Println("up")
		} else {
			vt.Println(taggedUptimeString)
		}
		return
	}

	if (ollamaEnabled || helpFlag) && ollama.FindModel() {
		// Used by the --help output, ollamaText is "Use Ollama" before this
		ollamaHelpText += fmt.Sprintf(" and %q", strings.TrimSuffix(ollama.ModelName, ":latest"))
		if env.No("OLLAMA_MODEL") {
			ollamaHelpText += " or $OLLAMA_MODEL"
		}
	}

	if helpFlag {
		Usage()
		return
	}

	if listDigraphsFlag {
		digraph.PrintTable()
		return
	}

	// Output the last used build, export or format command
	if lastCommandFlag {
		lastCommand, err := readLastCommand()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(lastCommand)
		return
	}

	if ollamaEnabled {
		if err := ollama.LoadModel(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	noWriteToCache = noCacheFlag || monitorAndReadOnlyFlag

	var (
		// Get the first rune of the executable name
		firstLetterOfExecutable = []rune(strings.ToLower(editorExecutable))[0]
		args                    = pflag.Args() // using pflag.Args() to get the non-flag arguments
		argsGiven               = len(args) > 0
	)

	globalArgs = args

	// Handle the copy flag / mode - before reading from stdin
	if copyFlag || firstLetterOfExecutable == 'c' {
		// If no filename argument or stdin indicators, copy from stdin
		if !argsGiven || (len(args) == 1 && (args[0] == "-" || args[0] == "/dev/stdin")) {
			// Copy from stdin to clipboard
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading from stdin: %v\n", err)
				os.Exit(1)
			}

			const primaryClipboard = false
			if err := clip.WriteAll(string(data), primaryClipboard); err != nil {
				fmt.Fprintf(os.Stderr, "error writing to clipboard: %v\n", err)
				os.Exit(1)
			}

			n := len(data)
			plural := "s"
			if n == 1 {
				plural = ""
			}
			fmt.Printf("Copied %d byte%s from stdin to the clipboard.\n", n, plural)
			return
		}

		// If filename is provided with copy flag, copy from file
		if argsGiven {
			filename := args[0]
			const primaryClipboard = false
			n, tailString, err := SetClipboardFromFile(filename, primaryClipboard)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			} else if n == 0 {
				fmt.Fprintf(os.Stderr, "Copied 0 bytes from %s\n", filename)
				os.Exit(1)
			}
			plural := "s"
			if n == 1 {
				plural = ""
			}
			if !catFlag && env.Has("ORBITON_BAT") {
				batFlag = true
			}
			if tailString != "" && !batFlag {
				if envNoColor {
					fmt.Printf("Copied %d byte%s from %s to the clipboard. Tail bytes: %s\n", n, plural, filename, strings.TrimSpace(strings.ReplaceAll(tailString, "\n", "\\n")))
				} else {
					fmt.Printf("Copied %s%d%s byte%s from %s to the clipboard. Tail bytes: %s%s%s\n", vt.Yellow, n, vt.Stop(), plural, filename, vt.LightCyan, strings.TrimSpace(strings.ReplaceAll(tailString, "\n", "\\n")), vt.Stop())
				}
			} else {
				fmt.Printf("Copied %d byte%s from %s to the clipboard.\n", n, plural, filename)
			}
			if catFlag {
				// List the file in a colorful way and quit
				quitCat(&FilenameOrData{filename, []byte{}, 0, false})
			} else if batFlag {
				// List the file in a colorful way, using bat, and quit
				quitBat(filename)
			}
			return
		}
	}

	// Handle paste flag
	if pasteFlag || firstLetterOfExecutable == 'p' {
		if !argsGiven {
			fmt.Fprintf(os.Stderr, "paste flag requires a filename\n")
			os.Exit(1)
		}
		filename := args[0]
		const primaryClipboard = false
		n, headString, tailString, err := WriteClipboardToFile(filename, forceFlag, primaryClipboard)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		} else if n == 0 {
			fmt.Fprintf(os.Stderr, "Wrote 0 bytes to %s\n", filename)
			os.Exit(1)
		}
		// chmod +x if this looks like a shell script or is in ie. /usr/bin
		if filepath.Ext(filename) == ".sh" || files.BinDirectory(filename) || strings.HasPrefix(headString, "#!") {
			os.Chmod(filename, 0o755)
		}
		if tailString != "" && !batFlag {
			if envNoColor {
				fmt.Printf("Wrote %d bytes to %s from the clipboard. Tail bytes: %s\n", n, filename, strings.TrimSpace(strings.ReplaceAll(tailString, "\n", "\\n")))
			} else {
				fmt.Printf("Wrote %s%d%s bytes to %s from the clipboard. Tail bytes: %s%s%s\n", vt.Red, n, vt.Stop(), filename, vt.LightBlue, strings.TrimSpace(strings.ReplaceAll(tailString, "\n", "\\n")), vt.Stop())
			}
		} else {
			fmt.Printf("Wrote %d bytes to %s from the clipboard.\n", n, filename)
		}
		if catFlag {
			// List the file in a colorful way and quit
			quitCat(&FilenameOrData{filename, []byte{}, 0, false})
		} else if batFlag {
			// List the file in a colorful way, using bat, and quit
			quitBat(filename)
		}
		return
	}

	// If the -e flag is given, clear all file locks and exit.
	if clearLocksFlag {
		lockErr := os.Remove(defaultLockFile)

		// Also remove the portal file
		portalErr := ClearPortal()

		switch {
		case lockErr == nil && portalErr != nil:
			fmt.Println("All locks clear.")
		case lockErr == nil && portalErr == nil:
			fmt.Println("All locks clear, and the portal has been closed.")
		case lockErr != nil && portalErr == nil:
			fmt.Fprintf(os.Stderr, "Closed the portal, but could not clear locks: %v\n", lockErr)
			os.Exit(1)
		default: // both errors are non-nil
			fmt.Fprintf(os.Stderr, "Could not clear locks: %v\n", lockErr)
			os.Exit(1)
		}
		return
	}

	traceStart() // if building with -tags trace

	var (
		fnord         FilenameOrData
		lineNumber    LineNumber
		colNumber     ColNumber
		stdinFilename = !argsGiven || (len(args) == 1 && (args[0] == "-" || args[0] == "/dev/stdin"))
		osudoMode     = editorExecutable == "osudo" || editorExecutable == "visudo"
		gameMode      = firstLetterOfExecutable == 'f' || firstLetterOfExecutable == 'g'
		err           error
	)

	// Start the game if the executable or symlink starts with 'f' or 'g'
	if gameMode {
		if _, err := Game(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	// If no regular filename is given, check if data is ready at stdin
	if stdinFilename {
		b := parentProcessIs("man")
		parentIsMan = &b
		fnord.stdin = (*parentIsMan || files.DataReadyOnStdin())
	}

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
	} else if osudoMode {
		// osudo may exit the program
		sudoers, err := NewSudoers("/etc/sudoers")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		tempPath := sudoers.TempPath()
		fnord.filename, lineNumber, colNumber = FilenameLineColNumber(tempPath, "", "")
		defer func() {
			if err := sudoers.Finalize(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}()
	} else {
		fnord.filename, lineNumber, colNumber = FilenameLineColNumber(pflag.Arg(0), pflag.Arg(1), pflag.Arg(2))
	}

	if searchAndOpenFlag {
		substring := fnord.filename
		if matches, err := files.FindFile(substring); err == nil && len(matches) > 0 {
			sort.Strings(matches)
			fnord.filename = matches[0]
		}
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
		if !noApproxMatchFlag {
			if !files.IsFileOrSymlink(fnord.filename) {
				if strings.HasSuffix(fnord.filename, ".") {
					// If the filename ends with "." and the file does not exist, assume this was a result of tab-completion going wrong.
					// If there are multiple files that exist that start with the given filename, open the one first in the alphabet (.cpp before .o)
					matches, err := globi.Glob(fnord.filename + "*")
					if err == nil && len(matches) > 0 { // no error and at least 1 match
						// Filter out any binary files
						matches = files.FilterOutBinaryFilesAccurate(matches)
						if len(matches) > 0 {
							sort.Strings(matches)
							// If the matches contains low priority suffixes, such as ".lock", then move it last
							matchesRegular := make([]string, len(matches))
							matchesLowPri := make([]string, len(matches))
							for _, fn := range matches {
								if !hasSuffix(fn, probablyDoesNotWantToEditExtensions) && strings.Contains(fn, ".") {
									matchesRegular = append(matchesRegular, fn)
								} else {
									// Store as a low-priority match
									matchesLowPri = append(matchesLowPri, fn)
								}
							}
							// Combine the regular and the low-priority matches
							matches = append(matchesRegular, matchesLowPri...)
							if len(matches) > 0 && len(matches[0]) > 0 {
								// Use the first filename in the list of matches
								fnord.filename = matches[0]
							}
						}
					}
				} else if !strings.Contains(fnord.filename, ".") && allLower(fnord.filename) {
					// The filename has no ".", is written in lowercase and it does not exist,
					// but more than one file that starts with the filename  exists. Assume tab-completion failed.
					matches, err := globi.Glob(fnord.filename + "*")
					if err == nil && len(matches) > 1 { // no error and more than 1 match
						// Use the first non-binary match of the sorted results
						matches = files.FilterOutBinaryFilesAccurate(matches)
						if len(matches) > 0 {
							sort.Strings(matches)
							fnord.filename = matches[0]
						}
					}
				} else {
					// Also match ie. "PKGBUILD" if just "Pk" was entered
					matches, err := globi.Glob(strings.ToTitle(fnord.filename) + "*")
					if err == nil && len(matches) >= 1 { // no error and at least 1 match
						// Use the first non-binary match of the sorted results
						matches = files.FilterOutBinaryFilesAccurate(matches)
						if len(matches) > 0 {
							sort.Strings(matches)
							fnord.filename = matches[0]
						}
					}
				}
			} // !noApproxMatchFlag
		}
	}

	// Set the terminal title, if the current terminal emulator supports it, and NO_COLOR is not set
	go fnord.SetTitle()

	// If the editor executable has been named "red", use the red/gray theme by default
	theme := NewDefaultTheme()
	syntaxHighlight := true
	if envNoColor {
		theme = NewNoColorDarkBackgroundTheme()
		syntaxHighlight = false
	} else if firstLetterOfExecutable != rune(0) && !osudoMode {
		// Check if the executable starts with a specific letter ('f', 'g', 'p' and 'c' are already checked for)
		specificLetter = true
		switch firstLetterOfExecutable {
		case 'b', 'e': // bo, borland, ed, edit etc.
			theme = NewDarkBlueEditTheme()
			// TODO: Later, when specificLetter is examined, use either NewEditLightTheme or NewEditDarkTheme
			editTheme = true
		case 'l': // lo, light etc
			theme = NewLitmusTheme()
		case 'r': // rb, ro, rt, red etc.
			theme = NewRedBlackTheme()
		case 's': // s, sw, synthwave etc.
			theme = NewSynthwaveTheme()
		case 't': // t, teal
			theme = NewTealTheme()
		case 'n': // nan, nano
			// Check if "Nano mode" should be set
			nanoMode = strings.HasPrefix(editorExecutable, "na")
		case 'v': // vs, vscode etc
			if !strings.HasPrefix(editorExecutable, "vi") { // vi, vim, visudo etc.
				theme = NewDarkVSTheme()
			}
		default:
			specificLetter = false
		}
	}

	if catFlag {
		// List the file in a colorful way and quit
		quitCat(&fnord)
	} else if batFlag { // This should NOT happen if only ORBITON_BAT is set!
		// List the file in a colorful way, using bat, and quit
		quitBat(fnord.filename)
	} else if buildFlag {
		msg, err := OnlyBuild(fnord)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(msg)
		os.Exit(0)
	}

	// Initialize the VT100 terminal
	tty, err := vt.NewTTY()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		quitMut.Lock()
		defer quitMut.Unlock()
		os.Exit(1)
	}
	defer tty.Close()

	// Run the main editor loop
	userMessage, nextAction, err := Loop(tty, fnord, lineNumber, colNumber, forceFlag, theme, syntaxHighlight, monitorAndReadOnlyFlag, nanoMode, createDirectoriesFlag, quickHelpFlag, noQuickHelpFlag, formatFlag, escToExitFlag, cycleFilenamesFlag)

	// SIGQUIT the parent PID. Useful if being opened repeatedly by a find command.
	switch nextAction {
	case megafile.NextFile:
		fmt.Fprintln(os.Stderr, "nextfile")
	case megafile.PreviousFile:
		fmt.Fprintln(os.Stderr, "prevfile")
	case megafile.StopParent:
		defer func() {
			sendParentQuitSignal()
		}()
	}

	// Remove the terminal title, if the current terminal emulator supports it and if NO_COLOR is not set.
	NoTitle()

	// Clear the current color attribute
	if clearOnQuit.Load() {
		fmt.Print(vt.Stop())
	} else {
		fmt.Print("\n" + vt.Stop())
	}

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
