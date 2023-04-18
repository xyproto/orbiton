//go:build !tinygo
// +build !tinygo

package gfx

// Box is a 3D cuboid with a min and max Vec3
type Box struct {
	Min Vec3
	Max Vec3
}

// NewBox creates a new Box.
func NewBox(min, max Vec3) Box {
	return Box{
		Min: min,
		Max: max,
	}
}

// B returns a new Box with given the Min and Max coordinates.
func B(minX, minY, minZ, maxX, maxY, maxZ float64) Box {
	return NewBox(
		V3(minX, minY, minZ),
		V3(maxX, maxY, maxZ),
	)
}

// Overlaps checks if two boxes overlap or not.
func (b Box) Overlaps(a Box) bool {
	return (!(b.Min.X >= a.Max.X || a.Min.X >= b.Max.X) &&
		!(b.Min.Y >= a.Max.Y || a.Min.Y >= b.Max.Y) &&
		!(b.Min.Z >= a.Max.Z || a.Min.Z >= b.Max.Z))
}

// Behind checks if b is in front of the a box.
func (b Box) Behind(a Box) bool {
	// Test for intersection in X-axis (lower X value is in front)
	if b.Min.X >= a.Max.X {
		return true
	} else if a.Min.X >= b.Max.X {
		return false
	}

	// Test for intersection in Y-axis (lower Y value is in front)
	if b.Min.Y >= a.Max.Y {
		return true
	} else if a.Min.Y >= b.Max.Y {
		return false
	}

	// Test for intersection in Z-axis (higher Z value is in front)
	if b.Min.Z >= a.Max.Z {
		return false
	} else if a.Min.Z >= b.Max.Z {
		return true
	}

	return true
}
