//go:build windows

package main

import "log"

func stopwin() {
	log.Fatalln(versionString + " does not support Windows yet, because the keyboard handling needs more work. Pull requests are welcome.")
}
