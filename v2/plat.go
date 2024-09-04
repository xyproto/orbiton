//go:build !linux && !darwin && !netbsd && !freebsd && !openbsd && !dragonfly

package main

// None of these are true, since none of the build tags matches
const (
	isBSD    = false
	isDarwin = false
	isLinux  = false
)
