package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Sudoers represents an /etc/sudoers file while it is being edited
type Sudoers struct {
	origModTime  time.Time
	fd           *os.File
	originalPath string
	tempPath     string
	origSize     int64
}

// NewSudoers creates a new Sudoers instance for safe editing
// It requires root privileges and locks the sudoers file
func NewSudoers(sudoersPath string) (*Sudoers, error) {
	if os.Geteuid() != 0 {
		return nil, fmt.Errorf("osudo: only root can run %s", filepath.Base(os.Args[0]))
	}

	file, err := os.OpenFile(sudoersPath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s: %w", sudoersPath, err)
	}

	// Non-blocking exclusive lock
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, fmt.Errorf("%s busy, try again later", sudoersPath)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("unable to stat %s: %w", sudoersPath, err)
	}

	sudoers := &Sudoers{
		origModTime:  info.ModTime(),
		fd:           file,
		originalPath: sudoersPath,
		tempPath:     sudoersPath + ".tmp",
		origSize:     info.Size(),
	}

	sudoers.setupSignalHandlers()

	if err := sudoers.createTempFile(); err != nil {
		sudoers.cleanup()
		return nil, err
	}

	return sudoers, nil
}

// TempPath returns the path to the temporary file for editing
func (s *Sudoers) TempPath() string {
	return s.tempPath
}

// Finalize validates and installs the edited sudoers file
func (s *Sudoers) Finalize() error {
	defer s.cleanup()

	if !s.wasModified() {
		if !isQuietMode() {
			fmt.Printf("%s unchanged\n", s.originalPath)
		}
		return nil
	}

	return s.handleValidationAndInstall()
}

func (s *Sudoers) handleValidationAndInstall() error {
	for !validateSudoersSyntax(s.tempPath) {
		switch askWhatNow() {
		case 'e':
			return nil // Re-edit
		case 'x':
			fmt.Fprintf(os.Stderr, "sudoers file unchanged\n")
			return nil
		case 'Q':
			fmt.Fprintf(os.Stderr, "Warning: installing sudoers file with syntax errors!\n")
			return s.commitChanges()
		}
	}
	return s.commitChanges()
}

func (s *Sudoers) createTempFile() error {
	tempFile, err := os.OpenFile(s.tempPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("unable to create temp file %s: %w", s.tempPath, err)
	}
	defer tempFile.Close()

	// Copy original content if file is not empty
	if s.origSize > 0 {
		if _, err := s.fd.Seek(0, 0); err != nil {
			return fmt.Errorf("unable to seek in original file: %w", err)
		}

		if _, err := io.Copy(tempFile, s.fd); err != nil {
			return fmt.Errorf("unable to copy to temp file: %w", err)
		}
	}

	// Preserve original timestamp
	if err := os.Chtimes(s.tempPath, s.origModTime, s.origModTime); err != nil {
		return fmt.Errorf("unable to preserve timestamp: %w", err)
	}

	return nil
}

func (s *Sudoers) wasModified() bool {
	info, err := os.Stat(s.tempPath)
	if err != nil {
		return false
	}

	// Check if size or modification time changed
	if info.Size() != s.origSize || !info.ModTime().Equal(s.origModTime) {
		// Empty file when original wasn't is suspicious
		if info.Size() == 0 && s.origSize > 0 {
			fmt.Fprintf(os.Stderr, "zero length temporary file, %s unchanged\n", s.originalPath)
			return false
		}
		return true
	}
	return false
}

func validateSudoersSyntax(filepath string) bool {
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to open %s for validation: %v\n", filepath, err)
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lineNo int
	var hasValidRules bool

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Include directives
		if strings.HasPrefix(line, "@include") || strings.HasPrefix(line, "#include") {
			if len(strings.Fields(line)) < 2 {
				fmt.Fprintf(os.Stderr, "parse error in %s near line %d\n", filepath, lineNo)
				return false
			}
			continue
		}

		// Basic rule detection
		if strings.Contains(line, "ALL") || strings.Contains(line, "=") {
			hasValidRules = true
		}

		// Common syntax errors
		if strings.Contains(line, "\t\t\t") || strings.Contains(line, ",,") {
			fmt.Fprintf(os.Stderr, "parse error in %s near line %d\n", filepath, lineNo)
			return false
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", filepath, err)
		return false
	}

	if lineNo > 0 && !hasValidRules {
		fmt.Fprintf(os.Stderr, "warning: %s contains no rules\n", filepath)
	}

	return true
}

func askWhatNow() rune {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("What now? ")
		input, _ := reader.ReadString('\n')
		if len(input) > 0 {
			switch choice := rune(input[0]); choice {
			case 'e', 'x', 'Q':
				return choice
			default:
				fmt.Println("Options are:")
				fmt.Println("  (e)dit sudoers file again")
				fmt.Println("  e(x)it without saving changes to sudoers file")
				fmt.Println("  (Q)uit and save changes to sudoers file (DANGER!)")
			}
		}
	}
}

func (s *Sudoers) commitChanges() error {
	// Set ownership
	if err := os.Chown(s.tempPath, 0, 0); err != nil {
		return fmt.Errorf("unable to set ownership of %s: %w", s.tempPath, err)
	}
	// Set permissions
	if err := os.Chmod(s.tempPath, 0o440); err != nil {
		return fmt.Errorf("unable to set permissions of %s: %w", s.tempPath, err)
	}
	// Atomic move (or copy+remove if cross-filesystem)
	if err := os.Rename(s.tempPath, s.originalPath); err != nil {
		if copyErr := copyFile(s.tempPath, s.originalPath); copyErr != nil {
			return fmt.Errorf("unable to install %s: %w", s.originalPath, err)
		}
		os.Remove(s.tempPath)
	}
	s.tempPath = "" // Mark as installed
	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (s *Sudoers) setupSignalHandlers() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	go func() {
		<-c
		s.cleanup()
		os.Exit(1)
	}()
}

func (s *Sudoers) cleanup() {
	if s == nil {
		return
	}
	if s.tempPath != "" {
		os.Remove(s.tempPath)
	}
	if s.fd != nil {
		s.fd.Close()
	}
}

func isQuietMode() bool {
	stat, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}
