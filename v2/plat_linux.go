//go:build linux

package main

// isLinux checks if the current OS is Linux.
func isLinux() bool {
	return true
}

// Other functions can be set to return false since they won't be relevant on Linux
func isDarwin() bool {
	return false
}

func isBSD() bool {
	return false
}
