package gfx

import (
	"fmt"
	"os"
)

// Stdout, and Stderr are open Files pointing to the standard output,
// and standard error file descriptors.
var (
	Stdout = os.Stdout
	Stderr = os.Stderr
)

// Log to standard output.
func Log(format string, a ...interface{}) {
	fmt.Printf(format+"\n", a...)
}

// Fatal prints to os.Stderr, followed by a call to os.Exit(1).
func Fatal(v ...interface{}) {
	fmt.Fprintln(Stderr, v...)
	os.Exit(1)
}

// Dump all of the arguments to standard output.
func Dump(a ...interface{}) {
	for _, v := range a {
		Log("%+v", v)
	}
}

// Printf formats according to a format specifier and writes to standard output.
func Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Printf(format, a...)
}

// Sprintf formats according to a format specifier and returns the resulting string.
func Sprintf(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}
