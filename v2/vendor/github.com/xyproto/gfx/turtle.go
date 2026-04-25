package gfx

import (
	"image"
	"image/color"
	"image/draw"
)

// Turtle keeps track of the state of the Turtle.
type Turtle struct {
	Position  Vec
	Direction Vec
	Width     float64
	Color     color.Color
	ops       turtleOperations
}

// NewTurtle creates a new turtle used for drawing.
func NewTurtle(p Vec, options ...func(*Turtle)) *Turtle {
	t := &Turtle{
		Position:  p,
		Direction: V(0, -1),
	}

	for _, option := range options {
		option(t)
	}

	return t
}

// Bounds return the bounds of the drawing.
func (t *Turtle) Bounds() image.Rectangle {
	if len(t.ops) == 0 {
		return image.ZR
	}

	return t.ops.Bounds()
}

// Draw all of the operations to the given image.
func (t *Turtle) Draw(m draw.Image) {
	for _, op := range t.ops {
		op.Draw(m)
	}
}

// Resize scaled to the given value.
func (t *Turtle) Resize(s float64) {
	t.Direction = t.Direction.Scaled(s)
}

// Turn the given number of arc degrees
func (t *Turtle) Turn(d Degrees) {
	t.Direction = t.Direction.Rotated(d.Radians())
}

// Move the given number of steps without drawing.
func (t *Turtle) Move(steps float64) {
	t.Position = t.Position.Add(t.Direction.Scaled(steps))
}

// Forward the given number of steps.
func (t *Turtle) Forward(steps float64) {
	from := t.Position

	t.Move(steps)

	to := t.Position

	if t.Width <= 0 {
		t.Width = 1
	}

	if t.Color == nil {
		t.Color = color.Black
	}

	t.addOp(lineOp{
		From:      from,
		To:        to,
		Thickness: t.Width * t.Direction.Len(),
		Color:     t.Color,
	})
}

func (t *Turtle) addOp(top turtleOperation) {
	t.ops = append(t.ops, top)
}

type turtleOperation interface {
	turtleDrawer
	Bounds() image.Rectangle
}

type turtleDrawer interface {
	Draw(draw.Image)
}

type turtleOperations []turtleOperation

func (tops turtleOperations) Bounds() image.Rectangle {
	switch len(tops) {
	case 0:
		return image.ZR
	case 1:
		return tops[0].Bounds()
	default:
		r := tops[0].Bounds()

		for _, top := range tops[1:] {
			r = r.Union(top.Bounds())
		}

		return r
	}
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

func (lo lineOp) Rect() Rect {
	switch lo.Thickness {
	case 1:
		return NewRect(lo.From, lo.To)
	default:
		return polylineFromTo(lo.From, lo.To, lo.Thickness).Rect()
	}
}

func (lo lineOp) Bounds() image.Rectangle {
	return lo.Rect().Bounds()
}
