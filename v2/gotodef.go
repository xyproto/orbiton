package main

import (
	"os"
	"path/filepath"
	"strings"
)

var backFunctions []func()

// GoToDefinition tries to find the definition of the given string, saves the current location and jumps to the location of the definition.
// Returns true if it was possible to go to the definition.
// This function is currently very experimental and may only work for a few languages, and for a few definitions!
// TODO: Parse some programming languages before jumping.
func (e *Editor) GoToDefinition(tty *TTY, c *Canvas, status *StatusBar) bool {
	// FuncPrefix may return strings with a leading or trailing blank
	funcPrefix := e.FuncPrefix()

	// Can this language / editor mode support this?
	if funcPrefix == "" {
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
	e.SetSearchTerm(c, status, s, false)

	// Backward search from the current location
	startIndex := e.DataY()
	stopIndex := LineIndex(0)
	foundX, foundY := e.backwardSearch(startIndex, stopIndex)
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
		ext := filepath.Ext(e.filename)

		var filenames []string

		if curDirFilenames, err := filepath.Glob("*" + ext); err == nil { // success
			filenames = append(filenames, curDirFilenames...)
		}
		if absFilename, err := filepath.Abs(e.filename); err == nil { // success
			sourceDir := filepath.Join(filepath.Dir(absFilename), "*"+ext)
			if sourceDirFiles, err := filepath.Glob(sourceDir); err == nil { // success
				filenames = append(filenames, sourceDirFiles...)
			}
		}
		if filenamesParent, err := filepath.Glob("../*" + ext); err == nil { // success
			filenames = append(filenames, filenamesParent...)
		}

		if len(filenames) > 0 { // success, found source files to examine
			for _, goFile := range filenames {
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
							//logf("PROLLY FUNC: %s LINE %d WORD %s NAME %s\n", goFile, i+1, word, name)

							oldFilename := e.filename
							oldLineIndex := e.LineIndex()

							if goFile != oldFilename {
								e.Switch(c, tty, status, fileLock, goFile)
							}
							redraw, _ := e.GoTo(LineIndex(i), c, status)
							e.redraw.Store(redraw)

							// Push a function for how to go back
							backFunctions = append(backFunctions, func() {
								oldFilename := oldFilename
								oldLineIndex := oldLineIndex
								goFile := goFile
								if goFile != oldFilename {
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

							oldFilename := e.filename
							oldLineIndex := e.LineIndex()

							if goFile != oldFilename {
								e.Switch(c, tty, status, fileLock, goFile)
							}
							redraw, _ := e.GoTo(LineIndex(i), c, status)
							e.redraw.Store(redraw)

							// Push a function for how to go back
							backFunctions = append(backFunctions, func() {
								oldFilename := oldFilename
								oldLineIndex := oldLineIndex
								goFile := goFile
								if goFile != oldFilename {
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
