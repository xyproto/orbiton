package main

import (
	"fmt"
	"github.com/xyproto/vt100"
	"os"
)

func errorMessageQuit(tty *vt100.TTY, err error) {
	tty.Close()
	vt100.Reset()
	vt100.Clear()
	vt100.Close()
	fmt.Fprintln(os.Stderr, "error: "+err.Error())
	vt100.SetXY(uint(0), uint(1))
	os.Exit(1)
}
