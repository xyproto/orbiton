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
	count                int
	maxSize              int
	maxMemoryUse         uint64 // can be <= 0 to not check for memory use
	ignoreSnapshots      bool   // used when playing back macros
}

const (
	initialUndoSize     = 8
	defaultMaxUndoCount = 512
	undoGrowthFactor    = 2
	defaultUndoMemory   = 0 // 32 * 1024 * 1024
)

var (
	// Circular undo buffer that starts small and grows as needed
	undo = NewUndo(defaultMaxUndoCount, defaultUndoMemory)

	// Save the contents of one switch.
	// Used when switching between a .c or .cpp file to the corresponding .h file.
	switchBuffer = NewUndo(1, defaultUndoMemory)

	// Save a copy of the undo stack when switching between files
	switchUndoBackup = NewUndo(defaultMaxUndoCount, defaultUndoMemory)
)

// NewUndo takes arguments that are only for initializing the undo buffers
func NewUndo(maxSize int, maxMemoryUse uint64) *Undo {
	// Start with a small initial size or the max size if it's very small
	initialSize := initialUndoSize
	if maxSize < initialSize {
		initialSize = maxSize
	}

	return &Undo{
		mut:                  &sync.RWMutex{},
		editorCopies:         make([]Editor, initialSize),
		editorLineCopies:     make([]map[int][]rune, initialSize),
		editorPositionCopies: make([]Position, initialSize),
		index:                0,
		count:                0,
		maxSize:              maxSize,
		maxMemoryUse:         maxMemoryUse,
		ignoreSnapshots:      false,
	}
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
func (u *Undo) MemoryFootprint() uint64 {
	var sum uint64
	for i := 0; i < len(u.editorLineCopies); i++ {
		sum += lineMapMemoryFootprint(u.editorLineCopies[i])
	}
	sum += uint64(unsafe.Sizeof(u.index))
	sum += uint64(unsafe.Sizeof(u.count))
	sum += uint64(unsafe.Sizeof(u.maxSize))
	sum += uint64(unsafe.Sizeof(u.editorCopies))
	sum += uint64(unsafe.Sizeof(u.editorPositionCopies))
	sum += uint64(unsafe.Sizeof(u.mut))
	sum += uint64(unsafe.Sizeof(u.maxMemoryUse))
	// Add the actual capacity of the slices
	sum += uint64(cap(u.editorCopies)) * uint64(unsafe.Sizeof(Editor{}))
	sum += uint64(cap(u.editorLineCopies)) * uint64(unsafe.Sizeof(map[int][]rune{}))
	sum += uint64(cap(u.editorPositionCopies)) * uint64(unsafe.Sizeof(Position{}))
	return sum
}

// grow expands the circular buffer when more space is needed
func (u *Undo) grow() {
	currentSize := len(u.editorCopies)
	newSize := currentSize * undoGrowthFactor
	if newSize > u.maxSize {
		newSize = u.maxSize
	}

	// If we can't grow anymore, we're at max capacity
	if newSize <= currentSize {
		return
	}

	// Create new slices with larger capacity
	newEditorCopies := make([]Editor, newSize)
	newEditorLineCopies := make([]map[int][]rune, newSize)
	newEditorPositionCopies := make([]Position, newSize)

	// Copy existing data to new slices
	// Handle the circular nature of the buffer
	if u.count > 0 {
		// Find the oldest entry in the circular buffer
		oldestIndex := u.index - u.count
		if oldestIndex < 0 {
			oldestIndex += currentSize
		}

		const withLines = true

		copied := 0
		for i := 0; i < u.count; i++ {
			srcIndex := (oldestIndex + i) % currentSize
			newEditorCopies[copied] = *(u.editorCopies[srcIndex].Copy(withLines))
			newEditorLineCopies[copied] = u.editorLineCopies[srcIndex]
			newEditorPositionCopies[copied] = u.editorPositionCopies[srcIndex]
			copied++
		}

		// Reset index to point to the next free slot
		u.index = u.count
	}

	// Replace the slices
	u.editorCopies = newEditorCopies
	u.editorLineCopies = newEditorLineCopies
	u.editorPositionCopies = newEditorPositionCopies
}

// Snapshot will store a snapshot, and move to the next position in the circular buffer
func (u *Undo) Snapshot(e *Editor) {
	if u.ignoreSnapshots {
		return
	}

	u.mut.Lock()
	defer u.mut.Unlock()

	// Check if we need to grow the buffer
	if u.count >= len(u.editorCopies) && len(u.editorCopies) < u.maxSize {
		u.grow()
	}

	const withLines = false

	u.editorCopies[u.index] = *(e.Copy(withLines))
	u.editorLineCopies[u.index] = e.CopyLines()
	u.editorPositionCopies[u.index] = e.pos

	// Go forward 1 step in the circular buffer
	u.index++

	// Circular buffer wrap
	if u.index >= len(u.editorCopies) {
		u.index = 0
	}

	// Update count (don't exceed buffer size)
	if u.count < len(u.editorCopies) {
		u.count++
	}

	// If the undo buffer uses too much memory, reduce the size
	if u.maxMemoryUse > 0 && u.MemoryFootprint() > u.maxMemoryUse {
		u.shrinkForMemory()
	}
}

// shrinkForMemory reduces the buffer size when memory usage is too high
func (u *Undo) shrinkForMemory() {
	newSize := len(u.editorCopies) / 2
	if newSize < initialUndoSize {
		newSize = initialUndoSize
	}

	// Keep only the most recent entries
	keepCount := newSize
	if u.count < keepCount {
		keepCount = u.count
	}

	newEditorCopies := make([]Editor, newSize)
	newEditorLineCopies := make([]map[int][]rune, newSize)
	newEditorPositionCopies := make([]Position, newSize)

	const withLines = true

	// Copy the most recent entries
	for i := 0; i < keepCount; i++ {
		srcIndex := u.index - keepCount + i
		if srcIndex < 0 {
			srcIndex += len(u.editorCopies)
		}

		newEditorCopies[i] = *(u.editorCopies[srcIndex].Copy(withLines))
		newEditorLineCopies[i] = u.editorLineCopies[srcIndex]
		newEditorPositionCopies[i] = u.editorPositionCopies[srcIndex]
	}

	// Update the undo struct
	u.editorCopies = newEditorCopies
	u.editorLineCopies = newEditorLineCopies
	u.editorPositionCopies = newEditorPositionCopies
	u.index = keepCount % newSize
	u.count = keepCount
}

// Restore will restore a previous snapshot, and move to the previous position in the circular buffer
func (u *Undo) Restore(e *Editor) error {
	u.mut.Lock()
	defer u.mut.Unlock()

	if u.count == 0 {
		return errors.New("no undo state available")
	}

	// Go back 1 step in the circular buffer
	u.index--
	if u.index < 0 {
		u.index = len(u.editorCopies) - 1
	}

	// Decrease count since we're moving backwards
	u.count--

	const withLines = true

	// Restore the state from this index
	if lines := u.editorLineCopies[u.index]; len(lines) > 0 || u.index == 0 {
		*e = *(u.editorCopies[u.index].Copy(withLines))
		e.lines = lines
		e.pos = u.editorPositionCopies[u.index]
		return nil
	}

	return errors.New("no undo state at this index")
}

// Len will return the current number of stored undo snapshots
func (u *Undo) Len() int {
	u.mut.RLock()
	defer u.mut.RUnlock()
	return u.count
}
