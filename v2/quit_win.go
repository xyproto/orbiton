//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/vt"
)

// quitExecShellCommand executes a shell command and exits on Windows
func quitExecShellCommand(tty *vt.TTY, workDir string, shellCommand string) {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	if tty != nil {
		tty.Close()
	}

	vt.Reset()
	vt.SetNoColor()
	vt.Clear()
	vt.Close()
	vt.ShowCursor(true)
	vt.SetXY(uint(0), uint(1))

	// On Windows, use cmd.exe to execute shell commands
	_ = os.Chdir(workDir)

	// Use cmd.exe with /C flag to execute the command and exit
	cmd := exec.Command("cmd.exe", "/C", shellCommand)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = workDir
	cmd.Env = env.Environ()

	// Start the command and exit the current process
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting shell command: %v\n", err)
		os.Exit(1)
	}

	// Wait for command to complete before exiting
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
}

// quitBat executes bat and exits on Windows
func quitBat(filename string) error {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	// ORBITON_BAT environment variable allows configuring the bat command and flags
	batCommandLine := env.Str("ORBITON_BAT", "bat")
	batExecutable := batCommandLine
	args := []string{}

	if strings.Contains(batCommandLine, " ") {
		batCommandLine = strings.ReplaceAll(batCommandLine, "\\ ", "\\")
		fields := strings.Split(batCommandLine, " ")
		batExecutable = files.Which(fields[0])
		args = append(args, fields[1:]...)
		for i, arg := range args {
			if strings.Contains(arg, "\\") {
				args[i] = strings.ReplaceAll(arg, "\\", " ")
			}
		}
	} else {
		batExecutable = files.Which(batExecutable)
	}

	if batExecutable == "" {
		return fmt.Errorf("%q is not available in the PATH", batExecutable)
	}

	args = append(args, filename)
	vt.ShowCursor(true)

	// Use os/exec instead of syscall.Exec on Windows
	cmd := exec.Command(batExecutable, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env.Environ()

	if err := cmd.Start(); err != nil {
		return err
	}

	// Exit after starting the command
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
	return nil // never reached
}

// quitToMan executes man and exits on Windows (if available)
func quitToMan(tty *vt.TTY, workDir, nroffFilename string, w, h uint) error {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	if tty != nil {
		tty.Close()
	}

	vt.Reset()
	vt.SetNoColor()
	vt.Clear()
	vt.Close()
	vt.ShowCursor(true)

	_ = os.Chdir(workDir)

	editorExecutable := files.WhichCached(env.Str("EDITOR"))
	if editorExecutable == "" {
		return fmt.Errorf("could not find %s in PATH", env.Str("EDITOR"))
	}

	manExecutable := files.WhichCached("man")
	if manExecutable == "" {
		// man is not typically available on Windows
		// Fall back to opening the file directly in the editor
		cmd := exec.Command(editorExecutable, nroffFilename)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env.Environ()
		if err := cmd.Start(); err != nil {
			return err
		}
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			os.Exit(1)
		}
		os.Exit(0)
		return nil
	}

	args := []string{}
	absManPageFilename, err := filepath.Abs(nroffFilename)
	if err != nil {
		return err
	}
	args = append(args, absManPageFilename)

	env.Set("NROFF_FILENAME", nroffFilename)
	env.Set("MANPAGER", editorExecutable)
	env.Set("COLUMNS", strconv.Itoa(int(w)))
	env.Set("LINES", strconv.Itoa(int(h)))

	cmd := exec.Command(manExecutable, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env.Environ()

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
	return nil
}

// quitToNroff executes orbiton again and exits on Windows
func quitToNroff(tty *vt.TTY, backupDirectory string, w, h uint) error {
	quitMut.Lock()
	defer quitMut.Unlock()

	stopBackgroundProcesses()

	if tty != nil {
		tty.Close()
	}

	vt.Reset()
	vt.SetNoColor()
	vt.Clear()
	vt.Close()
	vt.ShowCursor(true)

	// Try to find orbiton or o in PATH
	oExecutable := files.WhichCached("orbiton")
	if oExecutable == "" {
		oExecutable = files.WhichCached("o")
	}
	if oExecutable == "" {
		return fmt.Errorf("could not find orbiton or o in PATH")
	}

	args := globalArgs

	// Change to backup directory if needed
	if backupDirectory != "" {
		if files.IsDir(backupDirectory) {
			if err := os.Chdir(backupDirectory); err != nil {
				return err
			}
		} else if filepath.Dir(backupDirectory) != "." {
			if err := os.Chdir(filepath.Dir(backupDirectory)); err != nil {
				return err
			}
		} else {
			if err := os.Chdir(backupDirectory); err != nil {
				return err
			}
		}
	}

	env.Unset("NROFF_FILENAME")
	env.Unset("MANPAGER")
	env.Set("ORBITON_SPACE") // sets ORBITON_SPACE to "1"

	env.Set("COLUMNS", strconv.Itoa(int(w)))
	env.Set("LINES", strconv.Itoa(int(h)))

	cmd := exec.Command(oExecutable, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env.Environ()

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
	return nil
}
