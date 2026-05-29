package gfx

import (
	"image"
	"image/color"
	"image/draw"
)

// Turtle keeps track of the state of the Turtle.
//
// The default direction faces "up" (Y decreasing). The default line
// Width is 1 and the default Color is opaque black; both are set by
// NewTurtle and can be overridden either through functional options or
// by writing the fields directly between operations.
type Turtle struct {
	Position  Vec
	Direction Vec
	Width     float64
	Color     color.Color
	ops       []lineOp
}

// NewTurtle creates a new turtle used for drawing. The options are
// applied to the constructed Turtle in order before it is returned, so
// the very first Forward call already sees their effects.
func NewTurtle(p Vec, options ...func(*Turtle)) *Turtle {
	t := &Turtle{
		Position:  p,
		Direction: V(0, -1),
		Width:     1,
		Color:     color.Black,
	}

	for _, option := range options {
		option(t)
	}

	return t
}

// Bounds returns the union of the bounds of every line segment the
// Turtle has recorded. Returns the zero Rectangle when no segments
// have been recorded yet.
func (t *Turtle) Bounds() image.Rectangle {
	if len(t.ops) == 0 {
		return image.Rectangle{}
	}

	r := t.ops[0].Bounds()
	for _, op := range t.ops[1:] {
		r = r.Union(op.Bounds())
	}

	return r
}

// Draw all of the recorded line segments to the given image. Calling
// Draw does not consume the recorded operations; the same Turtle can
// be drawn onto multiple images, or onto the same image repeatedly.
func (t *Turtle) Draw(m draw.Image) {
	for _, op := range t.ops {
		op.Draw(m)
	}
}

// Reset discards any recorded line segments. Position, Direction,
// Width and Color are left unchanged.
func (t *Turtle) Reset() {
	t.ops = t.ops[:0]
}

// Resize scales the direction vector by the given factor. Subsequent
// Forward calls move proportionally further, and the lines they
// record are proportionally thicker.
func (t *Turtle) Resize(s float64) {
	t.Direction = t.Direction.Scaled(s)
}

// Turn the given number of arc degrees.
func (t *Turtle) Turn(d Degrees) {
	t.Direction = t.Direction.Rotated(d.Radians())
}

// Move the given number of steps without recording a line segment.
func (t *Turtle) Move(steps float64) {
	t.Position = t.Position.Add(t.Direction.Scaled(steps))
}

// Forward the given number of steps, recording a line segment from
// the previous Position to the new one.
//
// Width and Color are read from the Turtle at call time. If the
// Turtle was constructed by NewTurtle they default to 1 and opaque
// black respectively; if Width is non-positive or Color is nil at the
// time of the call (for example because the Turtle was constructed
// with a struct literal) the same defaults are used for that single
// segment without mutating the Turtle.
func (t *Turtle) Forward(steps float64) {
	from := t.Position

	t.Move(steps)

	width := t.Width
	if width <= 0 {
		width = 1
	}
	c := t.Color
	if c == nil {
		c = color.Black
	}

	t.ops = append(t.ops, lineOp{
		From:      from,
		To:        t.Position,
		Thickness: width * t.Direction.Len(),
		Color:     c,
	})
}

type lineOp struct {
	From      Vec
	To        Vec
	Thickness float64
	Color     color.Color
}

func (lo lineOp) Draw(m draw.Image) {
	DrawLine(m, lo.From, lo.To, lo.Thickness, lo.Color)
}

func (lo lineOp) Bounds() image.Rectangle {
	if lo.Thickness <= 1 {
		return NewRect(lo.From, lo.To).Bounds()
	}

	return polylineFromTo(lo.From, lo.To, lo.Thickness).Rect().Bounds()
}
