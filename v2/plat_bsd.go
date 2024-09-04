//go:build netbsd || freebsd || openbsd || dragonfly

package main

// Only isBSD is true, for these build tags
const (
	isBSD    = true
	isDarwin = false
	isLinux  = false
)
