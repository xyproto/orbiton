package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/xyproto/vt100"
)

// Undo is a struct that can store several states of the editor and position
type Undo struct {
	index            int
	size             int
	canvasCopies     []vt100.Canvas
	positionCopies   []Position
	editorCopies     []Editor
	editorLineCopies []map[int][]rune
	hasSomething     []bool
	mut              *sync.RWMutex
}

// NewUndo takes arguments that are only for initializing the undo buffers.
// The *Position and *vt100.Canvas is used only as a default values for the elements in the undo buffers.
func NewUndo(size int) *Undo {
	return &Undo{0, size, make([]vt100.Canvas, size, size), make([]Position, size, size), make([]Editor, size, size), make([]map[int][]rune, size, size), make([]bool, size, size), &sync.RWMutex{}}
}

// Snapshot will store a snapshot, and move to the next position in the circular buffer
func (u *Undo) Snapshot(c *vt100.Canvas, e *Editor) {
	u.mut.Lock()
	defer u.mut.Unlock()

	var sb strings.Builder
	sb.WriteString("New undo snapshot\n")
	sb.WriteString(fmt.Sprintf("index: %d\n", u.index))
	sb.WriteString(fmt.Sprintf("c.Copy(): %v\n---\n", c.Copy()))
	sb.WriteString(fmt.Sprintf("e.pos.Copy(): %v\n---\n", e.pos.Copy()))
	sb.WriteString(fmt.Sprintf("e.Copy(): %v\n---\n", e.Copy()))
	sb.WriteString(fmt.Sprintf("e.CopyLines(): %v\n---\n", e.CopyLines()))

	err := ioutil.WriteFile("/tmp/undo.log", []byte(sb.String()), 0644)
	if err != nil {
		panic(err)
	}

	u.hasSomething[u.index] = true
	u.canvasCopies[u.index] = c.Copy()
	u.positionCopies[u.index] = e.pos.Copy()
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
func (u *Undo) Restore(c *vt100.Canvas, e *Editor) error {
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
		c = &(u.canvasCopies[u.index])
		e = &(u.editorCopies[u.index])
		e.pos = u.positionCopies[u.index]
		e.lines = u.editorLineCopies[u.index]
		return nil
	}
	return errors.New("no undo state at this index")
}

// Index will return the current undo index, in the undo buffers
func (u *Undo) Index() int {
	return u.index
}
