package main

import (
	"github.com/xyproto/vt100"
)

type Orbiton struct {
	e        *Editor
	c        *vt100.Canvas
	tty      *vt100.TTY
	status   *StatusBar
	bookmark *Position
	undo     *Undo
	fileLock *LockKeeper
}
