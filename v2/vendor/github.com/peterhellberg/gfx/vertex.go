package gfx

import "image/color"

// Vertex holds Position, Color, Picture and Intensity.
type Vertex struct {
	Position  Vec
	Color     color.NRGBA
	Picture   Vec
	Intensity float64
}

// NewVertex returns a new vertex with the given position.
func NewVertex(pos Vec, args ...interface{}) Vertex {
	vx := Vertex{Position: pos}

	for _, a := range args {
		switch v := a.(type) {
		case color.NRGBA:
			vx.Color = v
		case Vec:
			vx.Picture = v
		case float64:
			vx.Intensity = v
		}
	}

	return vx
}

// Vx returns a new vertex with the given coordinates.
func Vx(pos Vec, args ...interface{}) Vertex {
	return NewVertex(pos, args...)
}
