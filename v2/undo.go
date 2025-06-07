package main

import (
	"errors"
	"sync"
	"unsafe"
)

// Undo is a struct that can store several states of the editor and position
type Undo struct {
	mut                  *sync.RWMutex
	editorCopies         []Editor
	editorLineCopies     []map[int][]rune
	editorPositionCopies []Position
	index                int
	size                 int
	maxMemoryUse         uint64 // can be <= 0 to not check for memory use
	ignoreSnapshots      bool   // used when playing back macros
}

const (
	// number of undo actions possible to store in the circular buffer
	defaultUndoCount = 1024

	// maximum amount of memory the undo buffers can use before re-using buffers, 0 to disable
	defaultUndoMemory = 0 // 32 * 1024 * 1024
)

var (
	// Circular undo buffer with room for N actions, change false to true to check for too limit memory use
	undo = NewUndo(defaultUndoCount, defaultUndoMemory)

	// Save the contents of one switch.
	// Used when switching between a .c or .cpp file to the corresponding .h file.
	switchBuffer = NewUndo(1, defaultUndoMemory)

	// Save a copy of the undo stack when switching between files
	switchUndoBackup = NewUndo(defaultUndoCount, defaultUndoMemory)
)

// NewUndo takes arguments that are only for initializing the undo buffers.
// The *Position and *vt100.Canvas is used only as a default values for the elements in the undo buffers.
func NewUndo(size int, maxMemoryUse uint64) *Undo {
	return &Undo{&sync.RWMutex{}, make([]Editor, size), make([]map[int][]rune, size), make([]Position, size), 0, size, maxMemoryUse, false}
}

// IgnoreSnapshots is used when playing back macros, to snapshot the macro playback as a whole instead
func (u *Undo) IgnoreSnapshots(b bool) {
	u.ignoreSnapshots = b
}

func lineMapMemoryFootprint(m map[int][]rune) uint64 {
	var sum uint64
	for _, v := range m {
		sum += uint64(cap(v))
	}
	return sum
}

// MemoryFootprint returns how much memory one Undo struct is using
// TODO: Check if the size of the slices that contains structs are correct
func (u *Undo) MemoryFootprint() uint64 {
	var sum uint64
	for _, m := range u.editorLineCopies {
		sum += lineMapMemoryFootprint(m)
	}
	sum += uint64(unsafe.Sizeof(u.index))
	sum += uint64(unsafe.Sizeof(u.size))
	sum += uint64(unsafe.Sizeof(u.editorCopies))
	sum += uint64(unsafe.Sizeof(u.editorPositionCopies))
	sum += uint64(unsafe.Sizeof(u.mut))
	sum += uint64(unsafe.Sizeof(u.maxMemoryUse))
	return sum
}

// Snapshot will store a snapshot, and move to the next position in the circular buffer
func (u *Undo) Snapshot(e *Editor) {
	if u.ignoreSnapshots {
		return
	}

	u.mut.Lock()
	defer u.mut.Unlock()

	eCopy := e.Copy()
	eCopy.lines = nil
	u.editorCopies[u.index] = *eCopy
	u.editorLineCopies[u.index] = e.CopyLines()
	u.editorPositionCopies[u.index] = e.pos

	// Go forward 1 step in the circular buffer
	u.index++
	// Circular buffer wrap
	if u.index >= u.size {
		u.index = 0
	}

	// If the undo buffer uses too much memory, reduce the size to half of the current size, but use a minimum of 10
	if u.maxMemoryUse > 0 && u.MemoryFootprint() > u.maxMemoryUse {
		newSize := u.size / 2
		if newSize < 10 {
			newSize = 10
		}

		smallest := newSize
		if u.size < smallest {
			smallest = u.size
		}

		newUndo := NewUndo(newSize, u.maxMemoryUse)
		newUndo.index = u.index
		if newUndo.index >= newUndo.size {
			newUndo.index = 0
		}
		newUndo.mut = u.mut

		u.mut.Lock()
		defer u.mut.Unlock()

		// Copy over the contents to the new undo struct
		offset := u.index
		for i := 0; i < smallest; i++ {
			copyFromPos := i + offset
			if copyFromPos > u.size {
				copyFromPos -= u.size
			}
			copyToPos := i

			newUndo.editorCopies[copyToPos] = u.editorCopies[copyFromPos]
			newUndo.editorLineCopies[copyToPos] = u.editorLineCopies[copyFromPos]
			newUndo.editorPositionCopies[copyToPos] = u.editorPositionCopies[copyFromPos]
		}

		// Replace the undo struct
		*u = *newUndo

		// Adjust the index after the size has been changed
		if u.index >= u.size {
			u.index = 0
		}
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

	// Restore the state from this index, if there is something there OR if the index is 0
	if lines := u.editorLineCopies[u.index]; len(lines) > 0 || u.index == 0 {

		*e = u.editorCopies[u.index]
		e.lines = lines
		e.pos = u.editorPositionCopies[u.index]

		return nil
	}
	return errors.New("no undo state at this index")
}

// Len will return the current number of stored undo snapshots.
// This is the same as the index int that points to the next free slot.
func (u *Undo) Len() int {
	return u.index
}
