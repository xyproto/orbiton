//go:build darwin

package main

// Only isDarwin is true, for these build tags
const (
	isBSD    = false
	isDarwin = true
	isLinux  = false
)
