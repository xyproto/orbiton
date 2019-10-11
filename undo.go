package main

import (
	"errors"
	"sync"
)

// Undo is a struct that can store several states of the editor and position
type Undo struct {
	index            int
	size             int
	editorCopies     []Editor
	editorLineCopies []map[int][]rune
	hasSomething     []bool
	mut              *sync.RWMutex
}

// NewUndo takes arguments that are only for initializing the undo buffers.
// The *Position and *vt100.Canvas is used only as a default values for the elements in the undo buffers.
func NewUndo(size int) *Undo {
	return &Undo{0, size, make([]Editor, size, size), make([]map[int][]rune, size, size), make([]bool, size, size), &sync.RWMutex{}}
}

// Snapshot will store a snapshot, and move to the next position in the circular buffer
func (u *Undo) Snapshot(e *Editor) {
	u.mut.Lock()
	defer u.mut.Unlock()

	u.hasSomething[u.index] = true
	u.editorCopies[u.index] = e.Copy()
	u.editorLineCopies[u.index] = e.CopyLines()

	// Go forward 1 step in the circular buffer
	u.index++
	// Circular buffer wrap
	if u.index >= u.size {
		u.index = 0
	}
}

// Restore will restore a previous snapshot, and move to the previous position in the circular buffer
func (u *Undo) Restore(e *Editor) error {
	u.mut.Lock()
	defer u.mut.Unlock()

	// Go back 1 step in the circular buffer
	u.index--
	// Circular buffer wrap
	if u.index < 0 {
		u.index = u.size - 1
	}

	// Restore the state from this index, if there is something there
	if u.hasSomething[u.index] {
		*e = u.editorCopies[u.index]
		e.lines = u.editorLineCopies[u.index]
		return nil
	}
	return errors.New("no undo state at this index")
}

// Index will return the current undo index, in the undo buffers
func (u *Undo) Index() int {
	return u.index
}
