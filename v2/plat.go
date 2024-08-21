//go:build !linux && !darwin && !netbsd && !freebsd && !openbsd && !dragonfly

package main

// All functions return false as none of the specified platforms match
func isLinux() bool {
	return false
}

func isDarwin() bool {
	return false
}

func isBSD() bool {
	return false
}
