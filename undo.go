package main

import (
	"sync"
)

type Undo struct {
	index          int
	size           int
	editorCopies   []Editor
	positionCopies []Position
	mut            *sync.RWMutex
}

// New undo take arguments that are only for initializing the undo buffers.
// The editor and position is used only as a default value for the structs.
func NewUndo(e *Editor, p *Position, size int) *Undo {
	u := &Undo{0, size, make([]Editor, size, size), make([]Position, size, size), &sync.RWMutex{}}
	for i := 0; i < size; i++ {
		u.editorCopies[i] = *e
		u.positionCopies[i] = *p
	}
	return u
}

// Save a snapshot, and move to the next position in the circular buffer
func (u *Undo) Snapshot(p *Position) {
	u.mut.Lock()

	u.positionCopies[u.index] = *p
	u.editorCopies[u.index] = *(p.e)
	// Go forward 1 step in the circular buffer
	u.index++
	// Circular buffer wrap
	if u.index == u.size {
		u.index = 0
	}

	u.mut.Unlock()
}

// Restore the previous snapshot, and move to the previous position in the circular buffer
func (u *Undo) Back() (*Editor, *Position) {
	u.mut.Lock()

	// Go back 1 step in the circular buffer
	u.index--
	// Circular buffer wrap
	if u.index < 0 {
		u.index = u.size - 1
	}
	// Restore the state from this index
	e := u.editorCopies[u.index]
	p := u.positionCopies[u.index]
	// link the Position and Editor structs
	p.e = &e

	u.mut.Unlock()
	return &e, &p
}

func (u *Undo) Position() int {
	return u.index
}
