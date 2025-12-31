// Package megafile provides functionality for a simple TUI for browsing files and directories
package megafile

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/vt"
)

const (
	leftArrow  = "←"
	rightArrow = "→"
	upArrow    = "↑"
	downArrow  = "↓"

	pgUpKey = "⇞" // page up
	pgDnKey = "⇟" // page down
	homeKey = "⇱" // home
	endKey  = "⇲" // end

	topLine = uint(1)
)

// FileEntry represents a file entry with position and name information
type FileEntry struct {
	x           uint
	y           uint
	realName    string
	displayName string
}

// State holds the current state of the shell, then canvas and the directory structures
type State struct {
	canvas              *vt.Canvas
	tty                 *vt.TTY
	dirIndex            uint
	quit                bool
	startx              uint
	starty              uint
	promptLength        uint
	written             []rune
	prevdir             []string
	fileEntries         []FileEntry
	selectedIndex       int
	selectionMoved      bool
	filterPattern       string
	editor              string // typically $EDITOR
	ShowHidden          bool
	Directories         []string
	StartMessage        string // title/header
	AngleColor          vt.AttributeColor
	PromptColor         vt.AttributeColor
	TitleColor          vt.AttributeColor
	HighlightBackground vt.AttributeColor
	Background          vt.AttributeColor
	EdgeBackground      vt.AttributeColor
	WrittenTextColor    vt.AttributeColor
}

// ErrExit is the error that is returned if the user appeared to want to exit
var ErrExit = errors.New("exit")

func ulen[T string | []rune | []string](xs T) uint {
	return uint(len(xs))
}

func (s *State) drawOutput(text string, tty *vt.TTY) {
	lines := strings.Split(text, "\n")
	x := s.startx
	y := s.starty + 1
	for _, line := range lines {
		vt.SetXY(x, y)
		s.canvas.Write(x, y, vt.Default, s.Background, strings.TrimSpace(line))
		y++
	}
	s.canvas.Draw()
	// Wait for a key press before continuing
	tty.String()
}

func (s *State) drawError(text string) {
	lines := strings.Split(text, "\n")
	x := s.startx
	y := s.starty + 1
	for _, line := range lines {
		vt.SetXY(x, y)
		s.canvas.Write(x, y, vt.Red, s.Background, line)
		y++
	}
}

func (s *State) highlightSelection() {
	if len(s.fileEntries) == 0 || s.selectedIndex < 0 {
		return
	}
	if s.selectedIndex >= len(s.fileEntries) {
		s.selectedIndex = len(s.fileEntries) - 1
	}

	entry := s.fileEntries[s.selectedIndex]
	s.canvas.Write(entry.x, entry.y, vt.Black, s.HighlightBackground, entry.displayName)
}

func (s *State) clearHighlight() {
	if s.selectedIndex >= 0 && s.selectedIndex < len(s.fileEntries) {
		entry := s.fileEntries[s.selectedIndex]

		// Clear only the area that was actually highlighted (displayName + suffix)
		clearWidth := ulen(entry.displayName) + 2 // +2 for suffix and safety margin
		for i := uint(0); i < clearWidth; i++ {
			s.canvas.WriteRune(entry.x+i, entry.y, vt.Default, s.Background, ' ')
		}

		// Redraw with original colors
		path := filepath.Join(s.Directories[s.dirIndex], entry.realName)
		var color vt.AttributeColor
		var suffix string

		if files.IsDir(path) && files.IsSymlink(path) {
			color = vt.Blue
			suffix = ">"
		} else if files.IsDir(path) {
			color = vt.Blue
			suffix = "/"
		} else if files.IsExecutableCached(path) {
			color = vt.LightGreen
			suffix = "*"
		} else if files.IsSymlink(path) {
			color = vt.LightRed
			suffix = "^"
		} else if files.IsBinary(path) {
			color = vt.LightMagenta
			suffix = "¤"
		} else {
			color = vt.Default
			suffix = ""
		}

		s.canvas.Write(entry.x, entry.y, color, s.Background, entry.displayName)
		if suffix != "" {
			s.canvas.Write(entry.x+ulen(entry.displayName), entry.y, vt.White, s.Background, suffix)
		}
	}
}

func (s *State) ls(dir string) (int, error) {
	const (
		margin      = 1
		columnWidth = 25
	)
	var (
		x            = s.startx
		y            = s.starty + 1
		w            = s.canvas.W()
		longestSoFar = uint(0)
	)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	// Clear file entries for new listing
	s.fileEntries = []FileEntry{}

	for _, e := range entries {
		name := e.Name()
		if !s.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}

		// Filter by pattern if one is set
		if s.filterPattern != "" {
			// Check if pattern contains glob special characters
			hasGlobChars := strings.ContainsAny(s.filterPattern, "*?[]")
			matched := false
			if hasGlobChars {
				// Use glob pattern matching
				var err error
				matched, err = filepath.Match(s.filterPattern, name)
				if err != nil {
					matched = false
				}
			} else {
				// Use simple prefix matching for plain text
				matched = strings.HasPrefix(strings.ToLower(name), strings.ToLower(s.filterPattern))
			}
			if !matched {
				continue
			}
		}

		// Determine display name (truncate if needed)
		displayName := name
		if ulen(name) > columnWidth-2 {
			displayName = string([]rune(name)[:columnWidth-5]) + "..."
		}

		// Store file entry with position info
		s.fileEntries = append(s.fileEntries, FileEntry{
			x:           x,
			y:           y,
			realName:    name,
			displayName: displayName,
		})

		if ulen(name) > longestSoFar {
			longestSoFar = ulen(name)
		}
		if longestSoFar > columnWidth {
			longestSoFar = columnWidth
		}

		path := filepath.Join(dir, name)
		var color vt.AttributeColor
		var suffix string

		if files.IsDir(path) && files.IsSymlink(path) {
			color = vt.Blue
			suffix = ">"
		} else if files.IsDir(path) {
			color = vt.Blue
			suffix = "/"
		} else if files.IsExecutableCached(path) {
			color = vt.LightGreen
			suffix = "*"
		} else if files.IsSymlink(path) { // not a directory symlink
			color = vt.LightRed
			suffix = "^"
		} else if files.IsBinary(path) {
			color = vt.LightMagenta
			suffix = "¤"
		} else {
			color = vt.Default
			suffix = ""
		}

		s.canvas.Write(x, y, color, s.Background, displayName)
		if suffix != "" {
			s.canvas.Write(x+ulen(displayName), y, vt.White, s.Background, suffix)
		}

		y++
		if y >= s.canvas.H() {
			x += longestSoFar + margin
			y = s.starty + 1
		}
		if x+longestSoFar > w {
			break
		}
	}

	// Reset selection if out of bounds
	if s.selectedIndex >= len(s.fileEntries) {
		s.selectedIndex = 0
	}

	return len(s.fileEntries), nil
}

func (s *State) confirmBinaryEdit(tty *vt.TTY, filename string) bool {
	c := s.canvas
	w := c.W()
	h := c.H()

	// Calculate dialog box dimensions
	boxWidth := uint(60)
	boxHeight := uint(9)
	if boxWidth > w-4 {
		boxWidth = w - 4
	}
	startX := (w - boxWidth) / 2
	startY := (h - boxHeight) / 2

	// Draw fancy ASCII art dialog box
	// Top border
	c.Write(startX, startY, vt.LightCyan, s.EdgeBackground, "╔")
	for i := uint(1); i < boxWidth-1; i++ {
		c.Write(startX+i, startY, vt.LightCyan, s.EdgeBackground, "═")
	}
	c.Write(startX+boxWidth-1, startY, vt.LightCyan, s.EdgeBackground, "╗")

	// Middle rows
	for i := uint(1); i < boxHeight-1; i++ {
		c.Write(startX, startY+i, vt.LightCyan, s.EdgeBackground, "║")
		// Clear the middle
		for j := uint(1); j < boxWidth-1; j++ {
			c.WriteRune(startX+j, startY+i, vt.Default, s.EdgeBackground, ' ')
		}
		c.Write(startX+boxWidth-1, startY+i, vt.LightCyan, s.EdgeBackground, "║")
	}

	// Bottom border
	c.Write(startX, startY+boxHeight-1, vt.LightCyan, s.EdgeBackground, "╚")
	for i := uint(1); i < boxWidth-1; i++ {
		c.Write(startX+i, startY+boxHeight-1, vt.LightCyan, s.EdgeBackground, "═")
	}
	c.Write(startX+boxWidth-1, startY+boxHeight-1, vt.LightCyan, s.EdgeBackground, "╝")

	// First line: filename is a binary file
	maxNameLen := int(boxWidth - 20) // Leave room for " is a binary file"
	displayName := filename
	if len(filename) > maxNameLen {
		displayName = filename[:maxNameLen-3] + "..."
	}
	line1 := displayName + " is binary and executable"
	line1X := startX + (boxWidth-uint(len(line1)))/2
	c.Write(line1X, startY+2, vt.LightYellow, s.Background, line1)

	// Second line: do you really want to edit it?
	line2 := "Do you really want to edit it?"
	line2X := startX + (boxWidth-uint(len(line2)))/2
	c.Write(line2X, startY+4, vt.Default, s.Background, line2)

	// Third line: instruction
	line3 := "Press y or return to edit or any other key to cancel."
	line3X := startX + (boxWidth-uint(len(line3)))/2
	c.Write(line3X, startY+6, vt.LightGreen, s.Background, line3)

	c.Draw()

	// Wait for key press
	for {
		switch tty.String() {
		case "c13", "y": // return/enter, y
			return true
		default:
			return false
		}
	}
}

func (s *State) edit(filename, path string) error {
	executableName := s.editor
	var args []string
	if strings.Contains(executableName, " ") {
		fields := strings.Split(s.editor, " ")
		executableName = fields[0]
		args = fields[1:]
	}
	editorPath, err := exec.LookPath(executableName)
	if err != nil {
		return err
	}
	// Add -y flag for "o" editor
	if filepath.Base(editorPath) == "o" {
		args = append(args, "-y")
	}
	args = append(args, filename)
	command := exec.Command(editorPath, args...)
	command.Dir = path
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin
	return command.Run()
}

func run(executableName string, args []string, path string) error {
	executablePath, err := exec.LookPath(executableName)
	if err != nil {
		return err
	}
	command := exec.Command(executablePath, args...)
	command.Dir = path
	command.Env = env.Environ()
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin
	return command.Run()
}

func run2(executableName string, args []string, path string) (string, error) {
	command := exec.Command(executableName, args...)
	command.Dir = path
	command.Env = env.Environ()
	outBytes, err := command.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(outBytes), nil
}

func (s *State) setPath(path string) {
	absPath, err := filepath.Abs(path)
	if err == nil { // success
		s.prevdir[s.dirIndex] = s.Directories[s.dirIndex]
		s.Directories[s.dirIndex] = absPath
	} else {
		s.prevdir[s.dirIndex] = s.Directories[s.dirIndex]
		s.Directories[s.dirIndex] = path
	}
}

// execute tries to execute the given command in the given directory,
// and returns true if the directory was changed
// and returns true if a file was edited
// and returns an error if something went wrong
func (s *State) execute(cmd, path string, tty *vt.TTY) (bool, bool, error) {
	// Common for non-bash and bash mode
	if cmd == "exit" || cmd == "quit" || cmd == "q" || cmd == "bye" {
		s.quit = true
		return false, false, nil
	}
	if cmd == "cd" || cmd == "-" || strings.HasPrefix(cmd, "cd ") {
		possibleDirectory := ""
		rest := ""
		if len(cmd) > 3 {
			rest = strings.TrimSpace(cmd[3:])
			possibleDirectory = filepath.Join(s.Directories[s.dirIndex], rest)
		}
		if cmd == "-" || rest == "-" {
			if s.Directories[s.dirIndex] != s.prevdir[s.dirIndex] {
				s.prevdir[s.dirIndex], s.Directories[s.dirIndex] = s.Directories[s.dirIndex], s.prevdir[s.dirIndex]
				return true, false, nil
			}
			return false, false, errors.New("OLDPWD not set")
		} else if possibleDirectory == "" {
			homedir := env.HomeDir()
			if s.Directories[s.dirIndex] != homedir {
				s.setPath(homedir)
				return true, false, nil
			}
			return false, false, nil
		} else if files.IsDir(possibleDirectory) {
			if s.Directories[s.dirIndex] != possibleDirectory {
				s.setPath(possibleDirectory)
				return true, false, nil
			}
			return false, false, nil
		} else if files.IsDir(rest) {
			if s.Directories[s.dirIndex] != rest {
				s.setPath(rest)
				return true, false, nil
			}
			return false, false, nil
		}
		return false, false, errors.New("cd WHAT?")
	}
	if files.IsDir(filepath.Join(path, cmd)) { // relative path
		newPath := filepath.Join(path, cmd)
		if s.Directories[s.dirIndex] != newPath {
			s.setPath(newPath)
			return true, false, nil
		}
		return false, false, nil
	}
	if files.IsDir(cmd) { // absolute path
		if s.Directories[s.dirIndex] != cmd {
			s.setPath(cmd)
			return true, false, nil
		}
		return false, false, nil
	}
	if files.IsFile(filepath.Join(path, cmd)) { // relative path
		if strings.HasPrefix(cmd, "./") && files.IsExecutableCached(filepath.Join(path, cmd)) {
			args := []string{}
			if strings.Contains(cmd, " ") {
				fields := strings.Split(cmd, " ")
				args = fields[1:]
			}
			output, err := run2(cmd, args, path)
			if err == nil {
				s.drawOutput(output, tty)
			}
			return false, false, err
		}
		// Check if file is both binary and executable
		fullPath := filepath.Join(path, cmd)
		if files.IsBinary(fullPath) && files.IsExecutable(fullPath) {
			if !s.confirmBinaryEdit(tty, cmd) {
				return false, false, nil // User cancelled
			}
		}
		return false, true, s.edit(cmd, path)
	}
	if files.IsFile(cmd) { // abs absolute path
		// Check if file is binary (but allow .gz files as they can be edited)
		if files.IsBinary(cmd) && !strings.HasSuffix(cmd, ".gz") {
			if !s.confirmBinaryEdit(tty, filepath.Base(cmd)) {
				return false, false, nil // User cancelled
			}
		}
		return false, true, s.edit(cmd, path)
	}
	if cmd == "l" || cmd == "ls" || cmd == "dir" {
		_, err := s.ls(path)
		return false, false, err
	}
	if strings.HasPrefix(cmd, "which ") {
		rest := ""
		if len(cmd) > 6 {
			rest = cmd[6:]
			found := files.WhichCached(rest)
			s.drawOutput(found, tty)
		}
		return false, false, nil
	}
	if cmd == "echo" {
		return false, false, nil
	}
	if strings.HasPrefix(cmd, "echo ") {
		s.drawOutput(cmd[5:], tty)
		return false, false, nil
	}
	if cmd == filepath.Base(env.Str("EDITOR")) {
		return false, true, s.edit("", path)
	}
	if strings.HasPrefix(cmd, filepath.Base(env.Str("EDITOR"))+" ") {
		spaceIndex := strings.Index(cmd, " ")
		rest := ""
		if spaceIndex+1 < len(cmd) {
			rest = cmd[spaceIndex+1:]
		}
		return false, true, s.edit(rest, path)
	}
	if strings.Contains(cmd, " ") {
		fields := strings.Split(cmd, " ")
		program := fields[0]
		arguments := fields[1:]
		output, err := run2(program, arguments, s.Directories[s.dirIndex])
		if err == nil {
			s.drawOutput(output, tty)
		}
		return false, false, err
	} else if foundExecutableInPath := files.WhichCached(cmd); foundExecutableInPath != "" {
		return false, false, run(foundExecutableInPath, []string{}, s.Directories[s.dirIndex])
	}

	return false, false, fmt.Errorf("WHAT DO YOU MEAN, %s?", cmd)
}

func (s *State) currentAbsDir() string {
	path := s.Directories[s.dirIndex]
	if absPath, err := filepath.Abs(path); err == nil { // success
		return absPath
	}
	return path
}

// Cleanup tries to set everything right in the terminal emulator before returning
func Cleanup(c *vt.Canvas) {
	vt.SetXY(0, c.H()-1)
	c.Clear()
	vt.SetLineWrap(true)
	vt.ShowCursor(true)
}

func dupli(xs []string) []string {
	tmp := make([]string, len(xs))
	copy(tmp, xs)
	return tmp
}

// New creates a new MegaFile State
// c and tty is a canvas and TTY, initiated with the vt package
// startdirs is a slice of directories to browse (toggle with tab)
// startMessage is the string to display at the top of the screen
// the function returns the absolute path to the directory the user ended up in,
// and an error if something went wrong
func New(c *vt.Canvas, tty *vt.TTY, startdirs []string, startMessage, editor string) *State {
	return &State{
		canvas:              c,
		tty:                 tty,
		prevdir:             dupli(startdirs),
		dirIndex:            0,
		quit:                false,
		startx:              uint(5),
		starty:              topLine + uint(4),
		fileEntries:         []FileEntry{},
		selectedIndex:       -1,
		selectionMoved:      false,
		filterPattern:       "",
		editor:              editor,
		ShowHidden:          false,
		Directories:         startdirs,
		StartMessage:        startMessage,
		AngleColor:          vt.LightRed,
		PromptColor:         vt.LightGreen,
		TitleColor:          vt.LightMagenta,
		Background:          vt.BackgroundDefault,
		HighlightBackground: vt.BackgroundWhite,
		EdgeBackground:      vt.BackgroundDefault,
		WrittenTextColor:    vt.LightYellow,
	}
}

// Run launches a file browser
func (s *State) Run() (string, error) {
	var x, y uint
	c := s.canvas
	drawPrompt := func() {
		prompt := ""
		if absPath, err := filepath.Abs(s.Directories[s.dirIndex]); err == nil { // success
			prompt = absPath //+ "> "
		} else {
			prompt = s.Directories[s.dirIndex] //+ "> "
		}
		prompt = strings.Replace(prompt, env.HomeDir(), "~", 1)
		c.Write(s.startx, s.starty, s.PromptColor, s.Background, prompt)
		s.promptLength = ulen([]rune(prompt)) + 2 // +2 for > and " "
		c.WriteRune(s.startx+s.promptLength-2, s.starty, s.AngleColor, s.Background, '>')
		c.WriteRune(s.startx+s.promptLength-1, s.starty, vt.Default, s.Background, ' ')
	}

	// The rune index for the text that has been written
	index := uint(0)

	drawWritten := func() {
		x = s.startx + s.promptLength
		y = s.starty
		c.Write(x, y, s.WrittenTextColor, s.Background, string(s.written))
		r := rune(' ')
		if index < ulen(s.written) {
			r = s.written[index]
		}
		c.WriteRune(x+index, y, vt.Black, vt.BackgroundGreen, r)
		vt.SetXY(x, y)
	}

	clearWritten := func() {
		y := s.starty
		for x := s.startx + s.promptLength; x < c.W(); x++ {
			c.WriteRune(x, y, vt.Default, s.Background, ' ')
		}
		vt.SetXY(x, y)
	}

	clearAndPrepare := func() {
		c.Clear()

		y := topLine

		// the title
		c.Write(5, y, s.TitleColor, s.Background, s.StartMessage)
		y++

		// the directory number
		c.Write(5, y, vt.LightYellow, s.Background, fmt.Sprintf("%d [%s]", s.dirIndex, s.Directories[s.dirIndex]))
		y++

		// if files are hidden or not
		if s.ShowHidden {
			c.Write(5, y, vt.Default, s.Background, ".")
		} else {
			c.Write(5, y, vt.Default, s.Background, " ")
		}

		// the prompt and written text (if any)
		drawPrompt()
		//x = s.startx + s.promptLength
		//y = s.starty
		drawWritten()
	}

	listDirectory := func() {
		s.clearHighlight() // Clear old highlight before clearing entries
		s.fileEntries = []FileEntry{}
		s.selectedIndex = -1
		s.selectionMoved = false // Reset selection moved flag
		s.filterPattern = ""     // Clear filter when changing directories
		clearAndPrepare()
		s.ls(s.Directories[s.dirIndex])
		s.written = []rune{}
		index = 0
		clearWritten()
		drawWritten()
	}

	clearAndPrepare()
	s.ls(s.Directories[s.dirIndex])
	c.Draw()

	for !s.quit {
		key := s.tty.String()
		switch key {
		case "c:27": // esc
			if s.selectedIndex >= 0 {
				// If a file selection is active, clear it
				s.clearHighlight()
				s.selectedIndex = -1
				c.Draw()
				break
			}
			if s.filterPattern != "" || len(s.written) > 0 {
				// If a file filter is active, clear it
				s.filterPattern = ""
				// Clear the written text
				s.written = []rune{}
				index = 0
				// Clear and redraw everything
				clearWritten()
				c.Clear()
				clearAndPrepare()
				s.ls(s.Directories[s.dirIndex])
				c.Draw()
			} else {
				// Quit the program
				s.quit = true
			}
		case "c:17": // ctrl-q
			s.quit = true
		case "c:13": // return
			// If a file is selected (via arrow keys), execute it regardless of text
			if s.selectedIndex >= 0 && s.selectedIndex < len(s.fileEntries) {
				s.clearHighlight()
				selectedFile := s.fileEntries[s.selectedIndex].realName
				savedFilename := selectedFile // Save the filename before editing
				if changedDirectory, editedFile, err := s.execute(selectedFile, s.Directories[s.dirIndex], s.tty); err != nil {
					clearAndPrepare()
					s.ls(s.Directories[s.dirIndex])
					s.drawError(err.Error())
					s.highlightSelection()
				} else if changedDirectory {
					listDirectory()
				} else if editedFile {
					// File was edited, restore selection by finding the filename
					listDirectory()
					// Search for the file by name
					for i, entry := range s.fileEntries {
						if entry.realName == savedFilename {
							s.selectedIndex = i
							s.highlightSelection()
							break
						}
					}
				} else {
					// User cancelled or nothing happened, redraw screen
					clearAndPrepare()
					s.ls(s.Directories[s.dirIndex])
					// Search for the file by name
					for i, entry := range s.fileEntries {
						if entry.realName == savedFilename {
							s.selectedIndex = i
							s.highlightSelection()
							break
						}
					}
				}
				s.written = []rune{}
				index = 0
				s.filterPattern = ""
				clearWritten()
				drawWritten()
				break
			}
			// No file selected, check if text was written
			if len(s.written) == 0 { // nothing was written
				homedir := env.HomeDir()
				if s.Directories[s.dirIndex] != homedir {
					s.setPath(homedir)
				}
				listDirectory()
				clearWritten()
				drawWritten()
				break
			}
			// Text has been written - execute it as a command
			commandText := string(s.written)
			s.written = []rune{}
			index = 0
			clearAndPrepare()
			clearWritten()
			c.Draw()
			if changedDirectory, editedFile, err := s.execute(commandText, s.Directories[s.dirIndex], s.tty); err != nil {
				s.drawError(err.Error())
			} else if changedDirectory || editedFile {
				listDirectory()
			} else {
				// Command output was shown, clear screen and redraw
				clearAndPrepare()
				s.ls(s.Directories[s.dirIndex])
			}
			drawWritten() // for the cursor
		case "c:11": // ctrl-k
			clearWritten()
			if len(s.written) > 0 {
				s.written = s.written[:index]
			}
			// Update filter pattern and redraw
			s.clearHighlight()
			s.filterPattern = string(s.written)
			clearAndPrepare()
			count, _ := s.ls(s.Directories[s.dirIndex])
			// If no matches, redraw without filter
			if count == 0 && s.filterPattern != "" {
				s.filterPattern = ""
				clearAndPrepare()
				s.ls(s.Directories[s.dirIndex])
			}
			s.selectedIndex = -1
			clearWritten()
			drawWritten()
		case "c:4": // ctrl-d
			if len(s.written) == 0 {
				Cleanup(c)
				return s.currentAbsDir(), ErrExit
			}
			clearWritten()
			s.written = append(s.written[:index], s.written[index+1:]...)
			// Update filter pattern and redraw
			s.clearHighlight()
			s.filterPattern = string(s.written)
			clearAndPrepare()
			count, _ := s.ls(s.Directories[s.dirIndex])
			// If no matches, redraw without filter
			if count == 0 && s.filterPattern != "" {
				s.filterPattern = ""
				clearAndPrepare()
				s.ls(s.Directories[s.dirIndex])
			}
			s.selectedIndex = -1
			clearWritten()
			drawWritten()
		case pgUpKey: // page up
			if len(s.fileEntries) > 0 && s.selectedIndex >= 0 {
				s.selectionMoved = true
				s.clearHighlight()
				// Find the first entry in the current column (same x, lowest y)
				currentX := s.fileEntries[s.selectedIndex].x
				for i := 0; i < len(s.fileEntries); i++ {
					if s.fileEntries[i].x == currentX {
						s.selectedIndex = i
						break
					}
				}
				s.highlightSelection()
			}
		case pgDnKey: // page down
			if len(s.fileEntries) > 0 && s.selectedIndex >= 0 {
				s.selectionMoved = true
				s.clearHighlight()
				// Find the last entry in the current column (same x, highest y)
				currentX := s.fileEntries[s.selectedIndex].x
				lastInColumn := s.selectedIndex
				for i := s.selectedIndex; i < len(s.fileEntries); i++ {
					if s.fileEntries[i].x == currentX {
						lastInColumn = i
					} else if s.fileEntries[i].x > currentX {
						break
					}
				}
				s.selectedIndex = lastInColumn
				s.highlightSelection()
			}
		case "c:1", homeKey: // ctrl-a, home
			if len(s.written) > 0 {
				clearWritten()
				index = 0
				drawWritten()
			} else if len(s.fileEntries) > 0 {
				s.selectionMoved = true
				s.clearHighlight()
				// Jump to first file
				s.selectedIndex = 0
				s.highlightSelection()
			}
		case "c:5", endKey: // ctrl-e, end
			if len(s.written) > 0 {
				clearWritten()
				index = ulen(s.written) // one after the text
				drawWritten()
			} else if len(s.fileEntries) > 0 {
				s.selectionMoved = true
				s.clearHighlight()
				// Jump to last file
				s.selectedIndex = len(s.fileEntries) - 1
				s.highlightSelection()
			}
		case upArrow:
			if len(s.written) > 0 && len(s.fileEntries) == 0 {
				// No files listed, move cursor to start of text
				clearWritten()
				index = 0
				drawWritten()
			} else if len(s.fileEntries) > 0 {
				// Files listed, navigate files
				s.selectionMoved = true
				s.clearHighlight()
				// Move selection up
				if s.selectedIndex < 0 {
					s.selectedIndex = 0
				} else if s.selectedIndex > 0 {
					s.selectedIndex--
				}
				s.highlightSelection()
			}
		case downArrow:
			if len(s.written) > 0 && len(s.fileEntries) == 0 {
				// No files listed, move cursor to end of text
				clearWritten()
				index = ulen(s.written) // one after the text
				drawWritten()
			} else if len(s.fileEntries) > 0 {
				// Files listed, navigate files
				s.selectionMoved = true
				s.clearHighlight()
				// Move selection down
				if s.selectedIndex < 0 {
					s.selectedIndex = 0
				} else if s.selectedIndex < len(s.fileEntries)-1 {
					s.selectedIndex++
				}
				s.highlightSelection()
			}
		case leftArrow:
			if len(s.written) > 0 {
				clearWritten()
				if index > 0 {
					index--
				}
				drawWritten()
			} else if len(s.fileEntries) > 0 && s.selectedIndex >= 0 {
				s.selectionMoved = true
				s.clearHighlight()
				// Move to previous column (with wraparound)
				currentEntry := s.fileEntries[s.selectedIndex]
				currentY := currentEntry.y

				found := false
				// 1. Try to find exact Y match in previous column
				for i := s.selectedIndex - 1; i >= 0; i-- {
					if s.fileEntries[i].y == currentY && s.fileEntries[i].x < currentEntry.x {
						s.selectedIndex = i
						found = true
						break
					}
				}

				// 2. If not found, find closest Y in previous column (or wrap to last)
				if !found {
					targetX := uint(0)
					targetXFound := false

					// Check if there IS a previous column
					for i := s.selectedIndex - 1; i >= 0; i-- {
						if s.fileEntries[i].x < currentEntry.x {
							targetX = s.fileEntries[i].x
							targetXFound = true
							break
						}
					}

					// If not found, wrap to last column
					if !targetXFound {
						targetX = s.fileEntries[len(s.fileEntries)-1].x
					}

					// Find closest Y in target column
					bestIndex := -1
					minDist := uint(10000)
					for i := 0; i < len(s.fileEntries); i++ {
						if s.fileEntries[i].x == targetX {
							dist := uint(0)
							if s.fileEntries[i].y > currentY {
								dist = s.fileEntries[i].y - currentY
							} else {
								dist = currentY - s.fileEntries[i].y
							}
							if dist < minDist {
								minDist = dist
								bestIndex = i
							}
						}
					}
					if bestIndex != -1 {
						s.selectedIndex = bestIndex
					}
				}
				s.highlightSelection()
			}
		case rightArrow:
			if len(s.written) > 0 {
				clearWritten()
				if index < ulen(s.written) {
					index++
				}
				drawWritten()
			} else if len(s.fileEntries) > 0 && s.selectedIndex >= 0 {
				s.selectionMoved = true
				s.clearHighlight()
				// Move to next column (with wraparound)
				currentEntry := s.fileEntries[s.selectedIndex]
				currentY := currentEntry.y

				found := false
				// 1. Try to find exact Y match in next column
				for i := s.selectedIndex + 1; i < len(s.fileEntries); i++ {
					if s.fileEntries[i].y == currentY && s.fileEntries[i].x > currentEntry.x {
						s.selectedIndex = i
						found = true
						break
					}
				}

				// 2. If not found, find closest Y in next column (or wrap to first)
				if !found {
					targetX := uint(0)
					targetXFound := false

					// Check if there IS a next column
					for i := s.selectedIndex + 1; i < len(s.fileEntries); i++ {
						if s.fileEntries[i].x > currentEntry.x {
							targetX = s.fileEntries[i].x
							targetXFound = true
							break
						}
					}

					// If not found, wrap to first column
					if !targetXFound {
						targetX = s.fileEntries[0].x
					}

					// Find closest Y in target column
					bestIndex := -1
					minDist := uint(10000)
					for i := 0; i < len(s.fileEntries); i++ {
						if s.fileEntries[i].x == targetX {
							dist := uint(0)
							if s.fileEntries[i].y > currentY {
								dist = s.fileEntries[i].y - currentY
							} else {
								dist = currentY - s.fileEntries[i].y
							}
							if dist < minDist {
								minDist = dist
								bestIndex = i
							}
						}
					}
					if bestIndex != -1 {
						s.selectedIndex = bestIndex
					}
				}
				s.highlightSelection()
			}
		case "c:15": // ctrl-o, toggle hidden files
			s.ShowHidden = !s.ShowHidden
			listDirectory()
		case "c:8": // ctrl-h, either toggle hidden files or delete text
			if index == 0 {
				s.ShowHidden = !s.ShowHidden
				listDirectory()
				break
			}
			clearWritten()
			if len(s.written) > 0 && index > 0 {
				s.written = append(s.written[:index-1], s.written[index:]...)
				index--
			}
			drawWritten()
		case "c:127": // backspace, either go one directory up or delete text
			if index == 0 { // cursor is at the start of the line, nothing to delete
				// go one directory up
				if absPath, err := filepath.Abs(filepath.Join(s.Directories[s.dirIndex], "..")); err == nil { // success
					s.setPath(absPath)
					listDirectory()
				}
				break
			}
			clearWritten()
			if len(s.written) > 0 && index > 0 {
				s.written = append(s.written[:index-1], s.written[index:]...)
				index--
			}
			// Update filter pattern and redraw
			s.clearHighlight()
			s.filterPattern = string(s.written)
			clearAndPrepare()
			count, _ := s.ls(s.Directories[s.dirIndex])
			// If no matches, redraw without filter
			if count == 0 && s.filterPattern != "" {
				s.filterPattern = ""
				clearAndPrepare()
				s.ls(s.Directories[s.dirIndex])
			}
			s.selectedIndex = -1
			clearWritten()
			drawWritten()
		case "c:14": // ctrl-n : cycle directory index forward
			s.dirIndex++
			if s.dirIndex >= ulen(s.Directories) {
				s.dirIndex = 0
			}
			listDirectory()
		case "c:0": // ctrl-space : enter the most recent directory
			if entries, err := os.ReadDir(s.Directories[s.dirIndex]); err == nil { // success
				var youngestTime time.Time
				var youngestName string
				for _, entry := range entries {
					if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
						fi, err := entry.Info()
						if err != nil {
							continue
						}
						if fi.ModTime().After(youngestTime) {
							youngestTime = fi.ModTime()
							youngestName = entry.Name()
						}
					}
				}
				if youngestName != "" {
					s.setPath(filepath.Join(s.Directories[s.dirIndex], youngestName))
					listDirectory()
				}
			}
		case "c:9": // tab : behave like right arrow or tab complete
			if len(s.written) == 0 && len(s.fileEntries) > 1 {
				// No text written and more than 1 file, cycle through files
				if len(s.fileEntries) > 0 && s.selectedIndex >= 0 {
					s.selectionMoved = true
					s.clearHighlight()
					currentEntry := s.fileEntries[s.selectedIndex]
					currentY := currentEntry.y

					// Find an entry with larger x at the same y position
					found := false
					for i := s.selectedIndex + 1; i < len(s.fileEntries); i++ {
						if s.fileEntries[i].y == currentY && s.fileEntries[i].x > currentEntry.x {
							s.selectedIndex = i
							found = true
							break
						}
					}

					// If not found at same row, move to first column of next row
					if !found {
						var nextY uint
						nextRowFound := false
						// Find the y position of the next row
						for i := s.selectedIndex + 1; i < len(s.fileEntries); i++ {
							if s.fileEntries[i].y > currentY {
								nextY = s.fileEntries[i].y
								nextRowFound = true
								break
							}
						}
						// Find the first entry (smallest x) on that next row
						if nextRowFound {
							minX := ^uint(0) // max uint value
							for i := 0; i < len(s.fileEntries); i++ {
								if s.fileEntries[i].y == nextY && s.fileEntries[i].x < minX {
									s.selectedIndex = i
									minX = s.fileEntries[i].x
									found = true
								}
							}
						}
					}

					// If still not found, wrap to the very first entry
					if !found {
						s.selectedIndex = 0
					}
					s.highlightSelection()
				}
				break
			}
			// Text has been written or only 1 file, do tab completion
			if len(s.written) == 0 {
				break
			}
			clearWritten()
			lastWordWrittenSoFar := strings.TrimPrefix(string(s.written), "./")
			if fields := strings.Fields(lastWordWrittenSoFar); len(fields) > 1 {
				lastWordWrittenSoFar = fields[len(fields)-1]
			}
			found := false
			if entries, err := os.ReadDir(s.Directories[s.dirIndex]); err == nil { // success
				for _, entry := range entries {
					name := entry.Name()
					if strings.HasPrefix(name, lastWordWrittenSoFar) {
						rest := []rune(name)[len([]rune(lastWordWrittenSoFar)):]
						s.written = append(s.written, rest...)
						index += ulen(rest)
						found = true
						break
					}
				}
			}
			if !found {
			OUT:
				for _, p := range env.Path() {
					if entries, err := os.ReadDir(p); err == nil { // success
						for _, entry := range entries {
							name := entry.Name()
							if strings.HasPrefix(name, lastWordWrittenSoFar) && files.IsExecutable(filepath.Join(p, name)) && len(s.written) < len([]rune(name)) {
								rest := []rune(name)[len(s.written):]
								s.written = append(s.written, rest...)
								index += ulen(rest)
								break OUT
							}
						}
					}
				}
			}
			drawWritten()
		case "c:12": // ctrl-l
			c.Clear()
			clearAndPrepare()
		case "c:2": // ctrl-b : go up one directory
			if absPath, err := filepath.Abs(filepath.Join(s.Directories[s.dirIndex], "..")); err == nil { // success
				s.setPath(absPath)
				listDirectory()
			}
		case "c:16": // ctrl-p : cycle directory index backward
			if s.dirIndex == 0 {
				s.dirIndex = ulen(s.Directories) - 1
			} else {
				s.dirIndex--
			}
			listDirectory()
		case "c:20": // ctrl-t : tig
			run("tig", []string{}, s.Directories[s.dirIndex])
		case "c:7": // ctrl-g : lazygit
			run("lazygit", []string{}, s.Directories[s.dirIndex])
		case "c:6": // ctrl-f : find in files
			if len(s.written) == 0 {
				break
			}
			searchText := string(s.written)
			// Search for text in non-binary files recursively
			var foundPath string
			var foundFile string
			filepath.Walk(s.Directories[s.dirIndex], func(path string, info os.FileInfo, err error) error {
				if err != nil || foundPath != "" {
					return nil
				}
				if info.IsDir() {
					// Skip hidden directories unless showHidden is enabled
					if !s.ShowHidden && strings.HasPrefix(info.Name(), ".") {
						return filepath.SkipDir
					}
					return nil
				}
				// Skip binary files
				if files.IsBinary(path) {
					return nil
				}
				// Read and search file
				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				if strings.Contains(string(content), searchText) {
					foundPath = filepath.Dir(path)
					foundFile = filepath.Base(path)
					return filepath.SkipAll
				}
				return nil
			})
			if foundPath != "" {
				s.setPath(foundPath)
				s.filterPattern = ""
				s.written = []rune{}
				index = 0
				listDirectory()
				// Find and highlight the found file
				for i, entry := range s.fileEntries {
					if entry.realName == foundFile {
						s.clearHighlight()
						s.selectedIndex = i
						s.selectionMoved = true
						s.highlightSelection()
						break
					}
				}
			}
		//case "c:18": // ctrl-r : history search
		//run("fzf", []string{"a", "b", "c"}, s.Directories[s.dirIndex])
		case "c:3": // ctrl-c
			if len(s.written) == 0 {
				Cleanup(c)
				return s.currentAbsDir(), ErrExit
			}
			s.written = []rune{}
			index = 0
			s.selectedIndex = -1
			s.filterPattern = ""
			clearAndPrepare()
			s.ls(s.Directories[s.dirIndex])
			clearWritten()
			drawWritten() // for the cursor
		case "":
			continue
		default:
			if key != " " && strings.TrimSpace(key) == "" {
				continue
			}
			// Reset selection when typing
			s.clearHighlight()
			s.selectedIndex = -1
			clearWritten()
			tmp := append(s.written[:index], []rune(key)...)
			s.written = append(tmp, s.written[index:]...)
			index += ulen([]rune(key))
			// Update filter pattern and redraw file list
			s.filterPattern = string(s.written)
			clearAndPrepare()
			count, _ := s.ls(s.Directories[s.dirIndex])
			// If no matches, redraw without filter
			if count == 0 && s.filterPattern != "" {
				s.filterPattern = ""
				clearAndPrepare()
				s.ls(s.Directories[s.dirIndex])
			}
			clearWritten()
			drawWritten()
		}
		c.Draw()
	}

	Cleanup(c)
	return s.currentAbsDir(), nil
}
