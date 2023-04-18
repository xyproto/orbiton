package gfx

import (
	"fmt"
	"image/color"
)

// TrianglesData specifies a list of Triangles vertices with three common properties:
// TrianglesPosition, TrianglesColor and TrianglesPicture.
type TrianglesData []Vertex

var (
	_ TrianglesPosition = (*TrianglesData)(nil)
	_ TrianglesColor    = (*TrianglesData)(nil)
	_ TrianglesPicture  = (*TrianglesData)(nil)
)

// MakeTrianglesData creates Vertexes of length len initialized with default property values.
//
// Prefer this function to make(Vertexes, len), because make zeros them, while this function
// does the correct intialization.
func MakeTrianglesData(len int) *TrianglesData {
	td := &TrianglesData{}
	td.SetLen(len)

	return td
}

// Len returns the number of vertices in Vertexes.
func (td *TrianglesData) Len() int {
	return len(*td)
}

// SetLen resizes Vertexes to len, while keeping the original content.
//
// If len is greater than Vertexes's current length, the new data is filled with default
// values ((0, 0), white, (0, 0), 0).
func (td *TrianglesData) SetLen(length int) {
	switch current := td.Len(); {
	case length > current:
		for i := 0; i < length-current; i++ {
			*td = append(*td, Vertex{Color: ColorWhite})
		}
	case length < current:
		*td = (*td)[:length]
	}
}

// Slice returns a sub-Triangles of this TrianglesData.
func (td *TrianglesData) Slice(i, j int) Triangles {
	s := (*td)[i:j]

	return &s
}

func (td *TrianglesData) updateData(t Triangles) {
	// fast path optimization
	if t, ok := t.(*TrianglesData); ok {
		copy(*td, *t)

		return
	}

	// slow path manual copy
	if t, ok := t.(TrianglesPosition); ok {
		for i := range *td {
			(*td)[i].Position = t.Position(i)
		}
	}

	if t, ok := t.(TrianglesColor); ok {
		for i := range *td {
			(*td)[i].Color = t.Color(i)
		}
	}

	if t, ok := t.(TrianglesPicture); ok {
		for i := range *td {
			(*td)[i].Picture, (*td)[i].Intensity = t.Picture(i)
		}
	}
}

// Update copies vertex properties from the supplied Triangles into this Vertexes.
//
// TrianglesPosition, TrianglesColor and TrianglesTexture are supported.
func (td *TrianglesData) Update(t Triangles) {
	if td.Len() != t.Len() {
		panic(fmt.Errorf("(%T).Update: invalid triangles length", td))
	}

	td.updateData(t)
}

// Copy returns an exact independent copy of this Vertexes.
func (td *TrianglesData) Copy() Triangles {
	copyTd := TrianglesData{}
	copyTd.SetLen(td.Len())
	copyTd.Update(td)

	return &copyTd
}

// Position returns the position property of i-th vertex.
func (td *TrianglesData) Position(i int) Vec {
	return (*td)[i].Position
}

// Color returns the color property of i-th vertex.
func (td *TrianglesData) Color(i int) color.NRGBA {
	return (*td)[i].Color
}

// Picture returns the picture property of i-th vertex.
func (td *TrianglesData) Picture(i int) (pic Vec, intensity float64) {
	return (*td)[i].Picture, (*td)[i].Intensity
}
