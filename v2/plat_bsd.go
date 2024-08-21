//go:build netbsd || freebsd || openbsd || dragonfly

package main

// isBSD checks if the current OS is a BSD variant or Dragonfly.
func isBSD() bool {
	return true
}

// Other functions can be set to return false since they won't be relevant on BSD
func isLinux() bool {
	return false
}

func isDarwin() bool {
	return false
}
