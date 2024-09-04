//go:build linux

package main

// Only isLinux is true, for these build tags
const (
	isBSD    = false
	isDarwin = false
	isLinux  = true
)
