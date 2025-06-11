package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
)

func osudo() {
	// Build the environment with the EDITOR variable set to "o"
	env := append(env.Environ(), "EDITOR=o")
	// Get the path to the visudo executable
	visudoPath := files.Which("visudo")
	if visudoPath != "" { // success
		// Replace the current process with visudo
		if err := syscall.Exec(visudoPath, []string{"visudo"}, env); err != nil {
			// Could not exec visudo
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		// No need to return here, because syscall.Exec replaces the current process
	}
}
