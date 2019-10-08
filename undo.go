package main

import (
	"errors"
	"sync"

	"github.com/xyproto/vt100"
)

// Undo is a struct that can store several states of the editor and position
type Undo struct {
	index        int
	size         int
	canvasCopy   vt100.Canvas
	positionCopy Position
	hasSomething bool
	//canvasCopies   []vt100.Canvas
	//positionCopies []Position
	//hasSomething   []bool
	mut *sync.RWMutex
}

// NewUndo takes arguments that are only for initializing the undo buffers.
// The *Position and *vt100.Canvas is used only as a default values for the elements in the undo buffers.
func NewUndo(size int) *Undo {
	var (
		c vt100.Canvas
		p Position
	)
	return &Undo{0, size, c, p, false, &sync.RWMutex{}}
	//return &Undo{0, size, make([]vt100.Canvas, size, size), make([]Position, size, size), make([]bool, size, size), &sync.RWMutex{}}
}

// Snapshot will store a snapshot, and move to the next position in the circular buffer
func (u *Undo) Snapshot(c *vt100.Canvas, p *Position) {
	u.mut.Lock()
	defer u.mut.Unlock()

	u.canvasCopy = c.Copy()
	u.positionCopy = p.Copy()
	u.hasSomething = true

	//u.hasSomething[u.index] = true
	//u.canvasCopies[u.index] = c.Copy()
	//u.positionCopies[u.index] = p.Copy()

	// Go forward 1 step in the circular buffer
	u.index++
	// Circular buffer wrap
	if u.index >= u.size {
		u.index = 0
	}
}

// Restore will restore a previous snapshot, and move to the previous position in the circular buffer
func (u *Undo) Restore() (*vt100.Canvas, *Position, error) {
	u.mut.Lock()
	defer u.mut.Unlock()

	// Go back 1 step in the circular buffer
	u.index--
	// Circular buffer wrap
	if u.index < 0 {
		u.index = u.size - 1
	}

	if u.hasSomething {
		return &u.canvasCopy, &u.positionCopy, nil
	}
	return nil, nil, errors.New("got nothing")

	// Restore the state from this index, if there is something there
	//if u.hasSomething[u.index] {
	//	c := u.canvasCopies[u.index]
	//	p := u.positionCopies[u.index]
	//	return &c, &p, nil
	//}
	//return nil, nil, errors.New("no undo state at this index")
}

// Index will return the current undo index, in the undo buffers
func (u *Undo) Index() int {
	return u.index
}
