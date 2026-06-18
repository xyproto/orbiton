//go:build windows

package main

// Windows has no concept of an init process (PID 1), so these are stubs that
// keep main.go compiling. runningAsInit always reports false, which means none
// of the other init helpers are ever called on Windows.

// runningAsInit always returns false on Windows.
func runningAsInit() bool { return false }

// reapZombies is a no-op on Windows.
func reapZombies() {}

// runInitShell is a no-op on Windows.
func runInitShell() {}

// initBrowseDir returns the current directory on Windows.
func initBrowseDir() string { return "." }
