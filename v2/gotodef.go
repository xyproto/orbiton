package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

var backFunctions []func()

// GoToDefinition tries to find the definition of the given string, saves the current location and jumps to the location of the definition.
// Returns true if it was possible to go to the definition.
// This function is currently very experimental and may only work for a few languages, and for a few definitions!
func (e *Editor) GoToDefinition(tty *vt100.TTY, c *vt100.Canvas, status *StatusBar) bool {
	// FuncPrefix may return strings with a leading or trailing blank
	funcPrefix := e.FuncPrefix()

	// Can this language / editor mode support this?
	if funcPrefix == "" {
		return false
	}

	// Do we have a word under the cursor? No need to trim it at this point.
	word := e.WordAtCursor()
	if word == "" {
		return false
	}

	// Is the word a language keyword?
	for kw := range syntax.Keywords {
		if kw == word {
			// Don't go to the definition of keywords
			return false
		}
	}

	currentLine := e.CurrentLine()
	functionCall := strings.Contains(currentLine, word+"(")

	// NOTE: This is extremely rudimentary "go to definition" and may easily end up in the wrong
	// function if two methods have the same name! It also does not go into the standard library.

	// word can be a string like "package.DoSomething" at this point.

	if strings.Count(word, ".") == 1 {
		wordDotWord := strings.Split(word, ".") // often package.function, package.type or object.method
		name := wordDotWord[1]

		ext := filepath.Ext(e.filename)

		filenames, err := filepath.Glob("*" + ext)
		if err == nil { // success
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
						emptyBeforeWord := len(strings.TrimSpace(fields[0])) == 0

						// go to a function definition
						if strings.HasPrefix(trimmedLine, funcPrefix) && strings.Contains(trimmedLine, " "+name+"(") {
							//logf("PROLLY FUNC: %s LINE %d WORD %s NAME %s\n", goFile, i+1, word, name)

							oldFilename := e.filename
							oldLineIndex := e.LineIndex()

							if goFile != oldFilename {
								e.Switch(c, tty, status, fileLock, goFile)
							}
							e.redraw, _ = e.GoTo(LineIndex(i), c, status)

							// Push a function for how to go back
							backFunctions = append(backFunctions, func() {
								oldFilename := oldFilename
								oldLineIndex := oldLineIndex
								goFile := goFile
								if goFile != oldFilename {
									e.Switch(c, tty, status, fileLock, oldFilename)
								}
								e.redraw, _ = e.GoTo(oldLineIndex, c, status)
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
							e.redraw, _ = e.GoTo(LineIndex(i), c, status)

							// Push a function for how to go back
							backFunctions = append(backFunctions, func() {
								oldFilename := oldFilename
								oldLineIndex := oldLineIndex
								goFile := goFile
								if goFile != oldFilename {
									e.Switch(c, tty, status, fileLock, oldFilename)
								}
								e.redraw, _ = e.GoTo(oldLineIndex, c, status)
							})

							return true
						}
					}
				}
			}
		}

	}

	// TODO:
	// * Implement "go to definition"
	// * Go to definition should store the current location in a special kind of bookmark (including filename)
	//   so that another keypress can jump back to where we were.
	// * Implement a special kind of bookmark which also supports storing the filename.

	//bookmark = e.pos.Copy()
	//s := "Bookmarked line " + e.LineNumber().String()
	//status.SetMessage("  " + s + "  ")

	//status.ClearAll(c)

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
	e.SetSearchTerm(c, status, s)

	// Backward search from the current location
	startIndex := e.DataY()
	stopIndex := LineIndex(0)
	foundX, foundY := e.backwardSearch(startIndex, stopIndex)

	if foundY == -1 {
		//status.SetMessage("Could not find " + s)
		//status.Show(c, e)
		return false
	}

	// Go to the found match
	e.redraw, _ = e.GoTo(foundY, c, status)
	if foundX != -1 {
		tabs := strings.Count(e.Line(foundY), "\t")
		e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
		e.HorizontalScrollIfNeeded(c)
	} else {

		// Clear the current search
		//e.SetSearchTerm(c, status, "")

		// Center and prepare to redraw
		e.Center(c)
		e.redraw = true
		e.redrawCursor = e.redraw

		return false
	}

	return true
}
