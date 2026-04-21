package main

import (
	"github.com/xyproto/vt"
)

// Cursor abstracts cursor navigation operations so that text mode and
// graphical book mode can each implement the movement semantics that are
// correct for them. Text mode treats pos.sy as a canvas cell row and scrolls
// based on canvas height. Book mode treats the document as a stream of
// display rows (with soft-wrapped body lines consuming multiple rows per
// data line), so its movement and scrolling operate on display rows instead.
//
// Keyloop handlers should call e.Cursor().Up/Down/... rather than selecting
// between CursorUpward and bookCursorUp themselves — this keeps mode-specific
// details out of the hot path.
type Cursor interface {
	// Up/Down/Left/Right move one display position in the named direction.
	// They return true when the cursor position actually changed. In book
	// mode callers can use this to avoid scheduling an expensive image
	// re-encode when the user held an arrow key past a boundary.
	Up(c *vt.Canvas, status *StatusBar) bool
	Down(c *vt.Canvas, status *StatusBar) bool
	Left(c *vt.Canvas, status *StatusBar) bool
	Right(c *vt.Canvas, status *StatusBar) bool

	// Home moves to the start of the current (display) line.
	Home(c *vt.Canvas)
	// End moves to the end of the current (display) line.
	End(c *vt.Canvas)

	// EnsureVisible adjusts the scroll offset so the cursor is rendered
	// inside the visible area. Implementations may be a no-op when the
	// underlying Up/Down already guarantee visibility.
	EnsureVisible(c *vt.Canvas)
}

// Cursor returns the navigation implementation appropriate for the editor's
// current mode (book vs. text).
func (e *Editor) Cursor() Cursor {
	if e.bookMode.Load() {
		return bookCursor{e: e}
	}
	return textCursor{e: e}
}

// textCursor is the standard canvas-cell cursor used outside book mode.
type textCursor struct{ e *Editor }

func (t textCursor) Up(c *vt.Canvas, status *StatusBar) bool {
	beforeY, beforeX := t.e.DataY(), t.e.pos.sx
	t.e.CursorUpward(c, status)
	return t.e.DataY() != beforeY || t.e.pos.sx != beforeX
}

func (t textCursor) Down(c *vt.Canvas, status *StatusBar) bool {
	beforeY, beforeX := t.e.DataY(), t.e.pos.sx
	t.e.CursorDownward(c, status)
	return t.e.DataY() != beforeY || t.e.pos.sx != beforeX
}

func (t textCursor) Left(c *vt.Canvas, status *StatusBar) bool {
	beforeY, beforeX := t.e.DataY(), t.e.pos.sx
	t.e.CursorBackward(c, status)
	return t.e.DataY() != beforeY || t.e.pos.sx != beforeX
}

func (t textCursor) Right(c *vt.Canvas, status *StatusBar) bool {
	beforeY, beforeX := t.e.DataY(), t.e.pos.sx
	t.e.CursorForward(c, status)
	return t.e.DataY() != beforeY || t.e.pos.sx != beforeX
}
func (t textCursor) Home(c *vt.Canvas)          { t.e.Home() }
func (t textCursor) End(c *vt.Canvas)           { t.e.End(c) }
func (t textCursor) EnsureVisible(c *vt.Canvas) {}

// bookCursor navigates in display-row space used by book mode, accounting
// for soft-wrapped body/list lines and multi-row headers/images. After every
// movement it calls bookModeEnsureCursorVisible so offsetY stays consistent
// with the actual pixel layout — the regular canvas-cell scrolling inside
// CursorDownward/Upward doesn't know about display rows and would otherwise
// misplace the cursor when the viewport is about to scroll.
type bookCursor struct{ e *Editor }

func (b bookCursor) Up(c *vt.Canvas, status *StatusBar) bool {
	moved := b.e.bookCursorUp(c, status)
	if moved {
		b.e.bookModeEnsureCursorVisible(c)
	}
	return moved
}

func (b bookCursor) Down(c *vt.Canvas, status *StatusBar) bool {
	moved := b.e.bookCursorDown(c, status)
	if moved {
		b.e.bookModeEnsureCursorVisible(c)
	}
	return moved
}

func (b bookCursor) Left(c *vt.Canvas, status *StatusBar) bool {
	moved := b.e.bookCursorBackward(c, status)
	if moved {
		b.e.bookModeEnsureCursorVisible(c)
	}
	return moved
}

func (b bookCursor) Right(c *vt.Canvas, status *StatusBar) bool {
	moved := b.e.bookCursorForward(c, status)
	if moved {
		b.e.bookModeEnsureCursorVisible(c)
	}
	return moved
}

func (b bookCursor) Home(c *vt.Canvas) {
	b.e.bookHome(c)
	b.e.bookModeEnsureCursorVisible(c)
}

func (b bookCursor) End(c *vt.Canvas) {
	b.e.bookEnd(c)
	b.e.bookModeEnsureCursorVisible(c)
}

func (b bookCursor) EnsureVisible(c *vt.Canvas) {
	b.e.bookModeEnsureCursorVisible(c)
}
