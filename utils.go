package main

import (
	"fmt"
	"os"
)

// exists checks if the given path exists
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// errLog outputs a message to stderr
func errLog(s string) {
	fmt.Fprintf(os.Stderr, "%s\n", s)
}
