package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/files"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

var backFunctions []func()

// GoToDefinition tries to find the definition of the given string, saves the current location and jumps to the location of the definition.
// Returns true if it was possible to go to the definition.
// First tries LSP if available, then falls back to text search.
func (e *Editor) GoToDefinition(tty *vt.TTY, c *vt.Canvas, status *StatusBar) bool {

	// Try LSP first if it's already running and initialized
	if lspLocation := e.tryLSPDefinition(); lspLocation != nil {
		return e.jumpToLSPLocation(lspLocation, tty, c, status)
	}

	// Fallback to text-based search (existing implementation below)
	return e.textSearchDefinition(tty, c, status)
}

// tryLSPDefinition attempts to get definition from LSP if ready (non-blocking)
// Returns nil if LSP is not available or not ready
func (e *Editor) tryLSPDefinition() *LSPLocation {
	// Check if LSP is supported for this mode
	config, ok := lspConfigs[e.mode]
	if !ok {
		return nil
	}

	// Get workspace root and absolute path
	absPath, err := filepath.Abs(e.filename)
	if err != nil {
		return nil
	}
	workspaceRoot := findWorkspaceRoot(absPath, config.RootMarkerFiles)

	// For Rust standalone files, create a temporary workspace
	// and get the mapped file path that rust-analyzer will understand
	lspFilePath := absPath
	if e.mode == mode.Rust {
		workspaceRoot, lspFilePath = ensureRustWorkspace(workspaceRoot, absPath)
	}

	// Trigger background LSP initialization if not already running
	// This won't help the current call but will help future ones
	TriggerLSPInitialization(e.mode, workspaceRoot)

	// Check if LSP is already ready (non-blocking)
	client := GetReadyLSPClient(e.mode, workspaceRoot)
	if client == nil {
		return nil // LSP not ready yet, use fallback
	}

	// Ensure document is synced with LSP
	var buf bytes.Buffer
	for i := 0; i < len(e.lines); i++ {
		if lineContent, ok := e.lines[i]; ok {
			buf.WriteString(string(lineContent))
		}
		buf.WriteRune('\n')
	}

	// Use the LSP file path (which might be in temp workspace for Rust)
	uri := "file://" + lspFilePath
	if lastOpenedURI != uri {
		if err := client.DidOpen(uri, config.LanguageID, buf.String()); err != nil {
			return nil
		}
		lastOpenedURI = uri
		lastOpenedVersion = 1
	} else {
		lastOpenedVersion++
		if err := client.DidChange(uri, buf.String(), lastOpenedVersion); err != nil {
			return nil
		}
	}

	// Get current cursor position
	line := int(e.DataY())
	x, err := e.DataX()
	if err != nil {
		x = 0
	}

	// Request definition with short timeout
	location, err := client.GetDefinition(uri, line, x, lspDefinitionTimeout)
	if err != nil {
		return nil
	}

	return location
}

// jumpToLSPLocation jumps to a location returned by LSP
func (e *Editor) jumpToLSPLocation(location *LSPLocation, tty *vt.TTY, c *vt.Canvas, status *StatusBar) bool {
	// Parse URI (format: "file:///path/to/file")
	targetPath := strings.TrimPrefix(location.URI, "file://")

	// Capture current state for back navigation
	oldFilename := e.filename
	oldLineIndex := e.LineIndex()

	// Switch to target file if different
	if targetPath != e.filename {
		if err := e.Switch(c, tty, status, fileLock, targetPath); err != nil {
			return false
		}
	}

	// Jump to line
	targetLine := LineIndex(location.Range.Start.Line)
	redraw, _ := e.GoTo(targetLine, c, status)
	e.redraw.Store(redraw)

	// Set horizontal position
	targetChar := location.Range.Start.Character
	tabs := strings.Count(e.Line(targetLine), "\t")
	e.pos.sx = targetChar + (tabs * (e.indentation.PerTab - 1))
	e.HorizontalScrollIfNeeded(c)

	// Push back function
	backFunctions = append(backFunctions, func() {
		if e.filename != oldFilename {
			e.Switch(c, tty, status, fileLock, oldFilename)
		}
		redraw, _ := e.GoTo(oldLineIndex, c, status)
		e.redraw.Store(redraw)
	})

	return true
}

// textSearchDefinition is the original text-search based implementation
func (e *Editor) textSearchDefinition(tty *vt.TTY, c *vt.Canvas, status *StatusBar) bool {

	// FuncPrefix may return strings with a leading or trailing blank
	funcPrefix := e.FuncPrefix()

	// Can this language / editor mode support this?
	if funcPrefix == "" && cLikeness(e.mode) == 0 {
		return false
	}

	// Do we have a word under the cursor? No need to trim it at this point.
	word := e.CurrentWord()
	if word == "" {
		return false
	}

	// Is the word a language keyword?
	for kw := range Keywords {
		if kw == word {
			// Don't go to the definition of keywords
			return false
		}
	}

	// The search string we will use for searching for functions within this file
	s := funcPrefix + word

	// Or should one search for a method instead?
	if strings.Contains(word, ".") {
		fields := strings.SplitN(word, ".", 2)
		methodName := fields[1]
		if strings.Contains(methodName, "[") {
			fields := strings.SplitN(methodName, "[", 2)
			arrayOrMapName := fields[0]
			// TODO: Also look for const and in "var"-blocks
			s = "var " + arrayOrMapName
		} else {
			s = ") " + methodName + "("
		}
	}

	// TODO: Search for variables, constants etc
	// Go to definition, but only of functions defined within the same Go file, for now

	var (
		// Capture state before jumping
		oldFilename  = e.filename
		oldLineIndex = e.LineIndex()

		foundX           = -1
		foundY LineIndex = -1
	)

	if funcPrefix == "" && cLikeness(e.mode) > 0 && s == word { // Special handling for C-like languages, if it's a simple function name
		startIndex := e.DataY()
		for y := startIndex; y >= 0; y-- {
			line := e.Line(y)
			if e.LooksLikeFunctionDef(line, "") && e.FunctionName(line) == word {
				foundY = y
				foundX = max(
					// Position cursor at the function name
					strings.Index(line, word), 0)
				break
			}
		}
	} else {
		e.SetSearchTerm(c, status, s, false)

		// Backward search from the current location
		startIndex := e.DataY()
		stopIndex := LineIndex(0)
		foundX, foundY = e.backwardSearch(startIndex, stopIndex)
	}
	if foundY != -1 {
		// Go to the found match
		redraw, _ := e.GoTo(foundY, c, status)
		e.redraw.Store(redraw)
		if foundX == -1 {
			// Center and prepare to redraw
			e.Center(c)
			e.redraw.Store(true)
			e.redrawCursor.Store(e.redraw.Load())
			return false
		}
		tabs := strings.Count(e.Line(foundY), "\t")
		e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
		e.HorizontalScrollIfNeeded(c)

		// Push a function for how to go back
		backFunctions = append(backFunctions, func() {
			oldFilename := oldFilename
			oldLineIndex := oldLineIndex
			if e.filename != oldFilename {
				e.Switch(c, tty, status, fileLock, oldFilename)
			}
			redraw, _ := e.GoTo(oldLineIndex, c, status)
			e.redraw.Store(redraw)
		})

		return true
	}

	currentLine := e.CurrentLine()
	functionCall := strings.Contains(currentLine, word+"(")

	// NOTE: This is extremely rudimentary "go to definition" and may easily end up in the wrong
	// function if two methods have the same name! It also does not go into the standard library.

	// word can be a string like "package.DoSomething" at this point.

	name := strings.TrimSpace(word)
	if strings.Count(word, ".") == 1 {
		wordDotWord := strings.Split(word, ".") // often package.function, package.type or object.method
		name = strings.TrimSpace(wordDotWord[1])
	}

	if len(name) > 0 {
		// Determine which file extensions to search
		// For C-like languages, search across all C-family extensions (.c, .h, .cpp, etc.)
		// For other languages, search only files with the same extension
		var extensions []string
		if cLikeness(e.mode) > 0 {
			extensions = cExtensions
		} else {
			extensions = []string{filepath.Ext(e.filename)}
		}

		// Use a map to deduplicate files (avoid processing same file multiple times)
		fileSet := make(map[string]bool)
		var filenames []string

		for _, ext := range extensions {
			if curDirFilenames, err := filepath.Glob("*" + ext); err == nil {
				for _, fn := range curDirFilenames {
					if absFn, err := filepath.Abs(fn); err == nil {
						if !fileSet[absFn] {
							fileSet[absFn] = true
							filenames = append(filenames, fn)
						}
					}
				}
			}
			if absFilename, err := filepath.Abs(e.filename); err == nil {
				sourceDir := filepath.Join(filepath.Dir(absFilename), "*"+ext)
				if sourceDirFiles, err := filepath.Glob(sourceDir); err == nil {
					for _, fn := range sourceDirFiles {
						if absFn, err := filepath.Abs(fn); err == nil {
							if !fileSet[absFn] {
								fileSet[absFn] = true
								filenames = append(filenames, fn)
							}
						}
					}
				}
			}
			if filenamesParent, err := filepath.Glob("../*" + ext); err == nil {
				for _, fn := range filenamesParent {
					if absFn, err := filepath.Abs(fn); err == nil {
						if !fileSet[absFn] {
							fileSet[absFn] = true
							filenames = append(filenames, fn)
						}
					}
				}
			}
		}

		if len(filenames) > 0 { // success, found source files to examine
			// Prioritize searching current file first, then others
			var orderedFilenames []string
			var otherFilenames []string
			currentFileAbs, _ := filepath.Abs(e.filename)
			for _, fn := range filenames {
				if absFn, err := filepath.Abs(fn); err == nil && absFn == currentFileAbs {
					orderedFilenames = append([]string{fn}, orderedFilenames...) // prepend current file
				} else {
					otherFilenames = append(otherFilenames, fn)
				}
			}
			orderedFilenames = append(orderedFilenames, otherFilenames...)

			for _, goFile := range orderedFilenames {
				// Normalize path to absolute for proper comparison
				absGoFile, err := filepath.Abs(goFile)
				if err != nil {
					absGoFile = goFile // fallback to original if Abs fails
				}

				data, err := os.ReadFile(goFile)
				if err != nil {
					continue
				}
				singleLineCommentMarker := e.SingleLineCommentMarker()
				for i, line := range strings.Split(string(data), "\n") {
					trimmedLine := strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, singleLineCommentMarker) {
						continue
					}
					if strings.Contains(trimmedLine, name) {
						fields := strings.SplitN(trimmedLine, name, 2)
						emptyBeforeWord := strings.TrimSpace(fields[0]) == ""

						// go to a function definition
						if e.LooksLikeFunctionDef(line, funcPrefix) && e.FunctionName(line) == name {
							//logf("FOUND FUNC: %s LINE %d WORD %s NAME %s\n", goFile, i+1, word, name)

							if absGoFile != oldFilename {
								if err := e.Switch(c, tty, status, fileLock, goFile); err != nil {
									return false // could not switch
								}
							}
							redraw, _ := e.GoTo(LineIndex(i), c, status)
							e.redraw.Store(redraw || (absGoFile != oldFilename))

							// Push a function for how to go back
							backFunctions = append(backFunctions, func() {
								oldFilename := oldFilename
								oldLineIndex := oldLineIndex
								absGoFile := absGoFile
								if absGoFile != oldFilename {
									e.Switch(c, tty, status, fileLock, oldFilename)
								}
								redraw, _ := e.GoTo(oldLineIndex, c, status)
								e.redraw.Store(redraw)
							})

							return true
						}

						// go to a type definition
						if !functionCall && emptyBeforeWord && !strings.Contains(trimmedLine, ":") && !strings.Contains(trimmedLine, "=") && !strings.Contains(trimmedLine, ",") {
							//logf("PROLLY TYPE: %s LINE %d WORD %s NAME %s\n", goFile, i+1, word, name)

							if absGoFile != oldFilename {
								if err := e.Switch(c, tty, status, fileLock, goFile); err != nil {
									return false // could not switch
								}
							}
							redraw, _ := e.GoTo(LineIndex(i), c, status)
							e.redraw.Store(redraw)

							// Push a function for how to go back
							backFunctions = append(backFunctions, func() {
								oldFilename := oldFilename
								oldLineIndex := oldLineIndex
								absGoFile := absGoFile
								if absGoFile != oldFilename {
									e.Switch(c, tty, status, fileLock, oldFilename)
								}
								redraw, _ := e.GoTo(oldLineIndex, c, status)
								e.redraw.Store(redraw)
							})

							return true
						}
					}
				}
			}
		}

	}

	return false
}

// GoToInclude looks for an #include filename and jumps to it, or returns false.
// returns the include filename (if found) and then true if a jump/switch was made.
func (e *Editor) GoToInclude(tty *vt.TTY, c *vt.Canvas, status *StatusBar) (string, bool) {
	var goFile string
	// First check if we are jumping to an #include
	trimmedLine := e.TrimmedLine()
	if strings.HasPrefix(trimmedLine, "#include ") {
		if fn := strings.TrimSpace(between(trimmedLine, "\"", "\"")); fn != "" {
			goFile = fn
		} else if fn := between(trimmedLine, "<", ">"); fn != "" {
			goFile = fn
		}
		systemInclude := filepath.Join("/usr/include", goFile)
		if !files.Exists(goFile) && files.Exists(systemInclude) {
			goFile = systemInclude
		}
		if goFile != "" && files.Exists(goFile) {
			oldFilename := e.filename
			if goFile != oldFilename {
				if err := e.Switch(c, tty, status, fileLock, goFile); err != nil {
					return goFile, false // could not switch
				}
			}
			// Push a function for how to go back
			backFunctions = append(backFunctions, func() {
				oldFilename := oldFilename
				goFile := goFile
				if goFile != oldFilename {
					e.Switch(c, tty, status, fileLock, oldFilename)
				}
			})
			return goFile, true // jumped
		}
	}
	return goFile, false // did not jump, not an #include statement
}
