//go:build windows

package main

// Only isWindows is true, for these build tags
const (
	isBSD     = false
	isDarwin  = false
	isLinux   = false
	isWindows = true
)
