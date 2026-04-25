//go:build !tinygo
// +build !tinygo

package gfx

import "math"

// HunterLab represents a color in Hunter-Lab.
type HunterLab struct {
	L float64
	A float64
	B float64
}

// XYZ converts from HunterLab to XYZ.
//
// Reference-X, Y and Z refer to specific illuminants and observers.
// Common reference values are available below in this same page.
//
// var_Ka = ( 175.0 / 198.04 ) * ( Reference-Y + Reference-X )
// var_Kb = (  70.0 / 218.11 ) * ( Reference-Y + Reference-Z )
//
// Y = ( ( Hunter-L / Reference-Y ) ^ 2 ) * 100.0
// X =   ( Hunter-a / var_Ka * sqrt( Y / Reference-Y ) + ( Y / Reference-Y ) ) * Reference-X
// Z = - ( Hunter-b / var_Kb * sqrt( Y / Reference-Y ) - ( Y / Reference-Y ) ) * Reference-Z
func (h HunterLab) XYZ(ref XYZ) XYZ {
	Ka := (175.0 / 198.04) * (ref.Y + ref.X)
	Kb := (70.0 / 218.11) * (ref.Y + ref.Z)

	Y := math.Pow((h.L/ref.Y), 2) * 100.0
	X := (h.A/Ka*math.Sqrt(Y/ref.Y) + (Y / ref.Y)) * ref.X
	Z := -(h.B/Kb*math.Sqrt(Y/ref.Y) - (Y / ref.Y)) * ref.Z

	return XYZ{X, Y, Z}
}

// HunterLab converts from XYZ to HunterLab.
//
// Reference-X, Y and Z refer to specific illuminants and observers.
// Common reference values are available below in this same page.
//
// var_Ka = ( 175.0 / 198.04 ) * ( Reference-Y + Reference-X )
// var_Kb = (  70.0 / 218.11 ) * ( Reference-Y + Reference-Z )
//
// Hunter-L = 100.0 * sqrt( Y / Reference-Y )
// Hunter-a = var_Ka * ( ( ( X / Reference-X ) - ( Y / Reference-Y ) ) / sqrt( Y / Reference-Y ) )
// Hunter-b = var_Kb * ( ( ( Y / Reference-Y ) - ( Z / Reference-Z ) ) / sqrt( Y / Reference-Y ) )
func (xyz XYZ) HunterLab(ref XYZ) HunterLab {
	Ka := (175.0 / 198.04) * (ref.Y + ref.X)
	Kb := (70.0 / 218.11) * (ref.Y + ref.Z)

	return HunterLab{
		L: 100.0 * math.Sqrt(xyz.Y/ref.Y),
		A: Ka * (((xyz.X / ref.X) - (xyz.Y / ref.Y)) / math.Sqrt(xyz.Y/ref.Y)),
		B: Kb * (((xyz.Y / ref.Y) - (xyz.Z / ref.Z)) / math.Sqrt(xyz.Y/ref.Y)),
	}
}
