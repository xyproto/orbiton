//go:build !tinygo
// +build !tinygo

package gfx

import "math"

// CIELab represents a color in CIE-L*ab.
type CIELab struct {
	L float64
	A float64
	B float64
}

// DeltaC calculates Delta C* for two CIE-L*ab colors.
//
// CIE-a*1, CIE-b*1                   //Color #1 CIE-L*ab values
// CIE-a*2, CIE-b*2                   //Color #2 CIE-L*ab values
//
// Delta C* = sqrt( ( CIE-a*2 ^ 2 ) + ( CIE-b*2 ^ 2 ) )
//   - sqrt( ( CIE-a*1 ^ 2 ) + ( CIE-b*1 ^ 2 ) )
func (c1 CIELab) DeltaC(c2 CIELab) float64 {
	return math.Sqrt(math.Pow(c2.A, 2)+math.Pow(c2.B, 2)) -
		math.Sqrt(math.Pow(c1.A, 2)+math.Pow(c1.B, 2))
}

// DeltaH calculates Delta H* for two CIE-L*ab colors.
//
// CIE-a*1, CIE-b*1                   //Color #1 CIE-L*ab values
// CIE-a*2, CIE-b*2                   //Color #2 CIE-L*ab values
//
// xDE =  sqrt( ( CIE-a*2 ^ 2 ) + ( CIE-b*2 ^ 2 ) )
//   - sqrt( ( CIE-a*1 ^ 2 ) + ( CIE-b*1 ^ 2 ) )
//
// Delta H* = sqrt( ( CIE-a*2 - CIE-a*1 ) ^ 2
//   - ( CIE-b*2 - CIE-b*1 ) ^ 2 - ( xDE ^ 2 ) )
func (c1 CIELab) DeltaH(c2 CIELab) float64 {
	xDE := math.Sqrt(math.Pow(c2.A, 2)+math.Pow(c2.B, 2)) -
		math.Sqrt(math.Pow(c1.A, 2)+math.Pow(c1.B, 2))

	return math.Sqrt(math.Pow(c2.A-c1.A, 2) +
		math.Pow(c2.B-c1.B, 2) - math.Pow(xDE, 2))
}

// DeltaE calculates Delta E* for two CIE-L*ab colors.
//
// CIE-L*1, CIE-a*1, CIE-b*1          //Color #1 CIE-L*ab values
// CIE-L*2, CIE-a*2, CIE-b*2          //Color #2 CIE-L*ab values
//
// Delta E* = sqrt( ( ( CIE-L*1 - CIE-L*2 ) ^ 2 )
//   - ( ( CIE-a*1 - CIE-a*2 ) ^ 2 )
//   - ( ( CIE-b*1 - CIE-b*2 ) ^ 2 ) )
func (c1 CIELab) DeltaE(c2 CIELab) float64 {
	return math.Sqrt(math.Pow(c1.L*1-c2.L*2, 2) +
		math.Pow(c1.A*1-c2.A*2, 2) + math.Pow(c1.B*1-c2.B*2, 2),
	)
}

// CIELab converts from XYZ to CIE-L*ab.
//
// Reference-X, Y and Z refer to specific illuminants and observers.
// Common reference values are available below in this same page.
//
// var_X = X / Reference-X
// var_Y = Y / Reference-Y
// var_Z = Z / Reference-Z
//
// if ( var_X > 0.008856 ) var_X = var_X ^ ( 1/3 )
// else                    var_X = ( 7.787 * var_X ) + ( 16 / 116 )
// if ( var_Y > 0.008856 ) var_Y = var_Y ^ ( 1/3 )
// else                    var_Y = ( 7.787 * var_Y ) + ( 16 / 116 )
// if ( var_Z > 0.008856 ) var_Z = var_Z ^ ( 1/3 )
// else                    var_Z = ( 7.787 * var_Z ) + ( 16 / 116 )
//
// CIE-L* = ( 116 * var_Y ) - 16
// CIE-a* = 500 * ( var_X - var_Y )
// CIE-b* = 200 * ( var_Y - var_Z )
func (xyz XYZ) CIELab(ref XYZ) CIELab {
	X := xyz.X / ref.X
	Y := xyz.Y / ref.Y
	Z := xyz.Z / ref.Z

	if X > 0.008856 {
		X = math.Pow(X, (1.0 / 3))
	} else {
		X = (7.787 * X) + (16.0 / 116)
	}

	if Y > 0.008856 {
		Y = math.Pow(Y, (1.0 / 3))
	} else {
		Y = (7.787 * Y) + (16.0 / 116)
	}

	if Z > 0.008856 {
		Z = math.Pow(Z, (1.0 / 3))
	} else {
		Z = (7.787 * Z) + (16.0 / 116)
	}

	return CIELab{
		L: (116.0 * Y) - 16,
		A: 500.0 * (X - Y),
		B: 200.0 * (Y - Z),
	}
}
