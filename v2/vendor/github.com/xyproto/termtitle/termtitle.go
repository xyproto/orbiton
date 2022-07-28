package termtitle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xyproto/env"
)

// Read the HOME environment variable and default to /home/$LOGNAME if it isn't set
var homeDir = env.Str("HOME", "/home/"+env.Str("LOGNAME"))

// hasE checks if the given environment variable name is set
func hasE(envVar string) bool {
	_, ok := os.LookupEnv(envVar)
	return ok
}

// TitleString returns a string that can be used for setting the title
// for the currently running terminal emulator, or an error if not supported.
func TitleString(title string) (string, error) {
	// This is possibly the widest supported string for setting the title
	formatString := "\033]0;%s\a"
	if hasE("ALACRITTY_LOG") { // alacritty?
		formatString = "\033]2;%s\007"
	} else if hasE("KONSOLE_VERSION") { // konsole?
		formatString = "\033]30;%s\007"
	} else if hasE("GNOME_TERMINAL_SERVICE") { // gnome-terminal?
		// ok
	} else if hasE("ZUTTY_VERSION") { // zutty?
		return "", errors.New("this terminal emulator currently does not support changing the title")
	} else {
		return "", errors.New("found no supported terminal emulator")
	}
	return fmt.Sprintf(formatString, title), nil
}

// Set tries to set the title of the currently running terminal emulator.
// An error is returned if no supported terminal emulator is found.
func Set(title string) error {
	s, err := TitleString(title)
	if err != nil {
		return err
	}
	fmt.Print(s)
	return nil
}

// MustSet will do a best effort of setting the terminal emulator title.
// No error is returned.
func MustSet(title string) {
	s, _ := TitleString(title)
	fmt.Print(s)
}

// GenerateTitle tries to find a suitable terminal emulator title text for a given filename,
// that is not too long (ideally <30 characters)
func GenerateTitle(filename string) string {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return filepath.Base(filename)
	}
	// First try to find the relative path to the home directory
	relPath, err := filepath.Rel(homeDir, absPath)
	if err != nil {
		// If the relative directory to $HOME could not be found, then just use the base filename
		return filepath.Base(filename)
	}
	title := filepath.Join("~", relPath)
	// If the relative directory path is short enough, use that
	if len(title) < 30 {
		return title
	}
	// Just use the base filename
	return filepath.Base(filename)
}
