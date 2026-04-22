package gfx

import (
	"math"
	"math/rand"
)

// SimplexNoise is a speed-improved simplex noise algorithm for 2D, 3D and 4D.
//
// Based on example code by Stefan Gustavson (stegu@itn.liu.se).
// Optimisations by Peter Eastman (peastman@drizzle.stanford.edu).
// Better rank ordering method by Stefan Gustavson in 2012.
//
// This could be speeded up even further, but it's useful as it is.
//
// # Version 2012-03-09
//
// This code was placed in the public domain by its original author,
// Stefan Gustavson. You may use it as you see fit, but
// attribution is appreciated.
type SimplexNoise struct {
	perm      []uint8
	permMod12 []uint8
}

// NewSimplexNoise creates a new simplex noise instance with the given seed
func NewSimplexNoise(seed int64) *SimplexNoise {
	var p [256]uint8

	perm := make([]uint8, 512)
	permMod12 := make([]uint8, 512)

	for i := 0; i < 256; i++ {
		p[i] = uint8(i)
	}

	r := rand.New(rand.NewSource(seed))

	for i := 0; i < 256; i++ {
		si := r.Int() & 255
		p[i], p[si] = p[si], p[i]
	}

	// To remove the need for index wrapping, double the permutation table length

	for i := 0; i < 512; i++ {
		perm[i] = p[i&255]
		permMod12[i] = perm[i] % 12
	}

	return &SimplexNoise{perm, permMod12}
}

// Noise2D performs 2D simplex noise
func (sn *SimplexNoise) Noise2D(xin, yin float64) float64 {
	var n0, n1, n2 float64 // Noise contributions from the three corners

	// Skew the input space to determine which simplex cell we're in
	s := (xin + yin) * f2 // Hairy factor for 2D
	i := fastfloor(xin + s)
	j := fastfloor(yin + s)
	t := float64(i+j) * g2
	X0 := float64(i) - t // Unskew the cell origin back to (x,y) space
	Y0 := float64(j) - t
	x0 := xin - X0 // The x,y distances from the cell origin
	y0 := yin - Y0

	// For the 2D case, the simplex shape is an equilateral triangle.
	// Determine which simplex we are in.
	var i1, j1 int32 // Offsets for second (middle) corner of simplex in (i,j) coords

	if x0 > y0 {
		i1, j1 = 1, 0 // lower triangle, XY order: (0,0)->(1,0)->(1,1)
	} else {
		i1, j1 = 0, 1 // upper triangle, YX order: (0,0)->(0,1)->(1,1)
	}

	// A step of (1,0) in (i,j) means a step of (1-c,-c) in (x,y), and
	// a step of (0,1) in (i,j) means a step of (-c,1-c) in (x,y), where
	// c = (3-sqrt(3))/6

	x1 := x0 - float64(i1) + g2 // Offsets for middle corner in (x,y) unskewed coords
	y1 := y0 - float64(j1) + g2
	x2 := x0 - 1.0 + 2.0*g2 // Offsets for last corner in (x,y) unskewed coords
	y2 := y0 - 1.0 + 2.0*g2

	// Work out the hashed gradient indices of the three simplex corners
	ii := i & 255
	jj := j & 255
	perm := sn.perm
	m12 := sn.permMod12
	gi0 := m12[ii+int32(perm[jj])]
	gi1 := m12[ii+i1+int32(perm[jj+j1])]
	gi2 := m12[ii+1+int32(perm[jj+1])]

	// Calculate the contribution from the three corners
	t0 := 0.5 - x0*x0 - y0*y0

	if t0 < 0.0 {
		n0 = 0.0
	} else {
		t0 *= t0
		n0 = t0 * t0 * grad3[gi0].dot2(x0, y0) // (x,y) of grad3 used for 2D gradient
	}

	t1 := 0.5 - x1*x1 - y1*y1

	if t1 < 0.0 {
		n1 = 0.0
	} else {
		t1 *= t1
		n1 = t1 * t1 * grad3[gi1].dot2(x1, y1)
	}

	t2 := 0.5 - x2*x2 - y2*y2

	if t2 < 0.0 {
		n2 = 0.0
	} else {
		t2 *= t2
		n2 = t2 * t2 * grad3[gi2].dot2(x2, y2)
	}

	// Add contributions from each corner to get the final noise value.
	// The result is scaled to return values in the interval [-1,1].
	return 70.0 * (n0 + n1 + n2)
}

// Noise3D performs 3D simplex noise
func (sn *SimplexNoise) Noise3D(xin, yin, zin float64) float64 {
	var n0, n1, n2, n3 float64 // Noise contributions from the four corners

	// Skew the input space to determine which simplex cell we're in
	s := (xin + yin + zin) * f3 // Very nice and simple skew factor for 3D
	i := fastfloor(xin + s)
	j := fastfloor(yin + s)
	k := fastfloor(zin + s)
	t := float64(i+j+k) * g3
	X0 := float64(i) - t // Unskew the cell origin back to (x,y,z) space
	Y0 := float64(j) - t
	Z0 := float64(k) - t
	x0 := xin - X0 // The x,y,z distances from the cell origin
	y0 := yin - Y0
	z0 := zin - Z0

	// For the 3D case, the simplex shape is a slightly irregular tetrahedron.
	// Determine which simplex we are in.
	var i1, j1, k1 int32 // Offsets for second corner of simplex in (i,j,k) coords
	var i2, j2, k2 int32 // Offsets for third corner of simplex in (i,j,k) coords

	if x0 >= y0 {
		if y0 >= z0 {
			i1, j1, k1, i2, j2, k2 = 1, 0, 0, 1, 1, 0 // X Y Z order
		} else if x0 >= z0 {
			i1, j1, k1, i2, j2, k2 = 1, 0, 0, 1, 0, 1 // X Z Y order
		} else {
			i1, j1, k1, i2, j2, k2 = 0, 0, 1, 1, 0, 1 // Z X Y order
		}
	} else { // x0<y0
		if y0 < z0 {
			i1, j1, k1, i2, j2, k2 = 0, 0, 1, 0, 1, 1 // Z Y X order
		} else if x0 < z0 {
			i1, j1, k1, i2, j2, k2 = 0, 1, 0, 0, 1, 1 // Y Z X order
		} else {
			i1, j1, k1, i2, j2, k2 = 0, 1, 0, 1, 1, 0 // Y X Z order
		}
	}

	// A step of (1,0,0) in (i,j,k) means a step of (1-c,-c,-c) in (x,y,z),
	// a step of (0,1,0) in (i,j,k) means a step of (-c,1-c,-c) in (x,y,z), and
	// a step of (0,0,1) in (i,j,k) means a step of (-c,-c,1-c) in (x,y,z), where
	// c = 1/6.
	x1 := x0 - float64(i1) + g3 // Offsets for second corner in (x,y,z) coords
	y1 := y0 - float64(j1) + g3
	z1 := z0 - float64(k1) + g3
	x2 := x0 - float64(i2) + 2.0*g3 // Offsets for third corner in (x,y,z) coords
	y2 := y0 - float64(j2) + 2.0*g3
	z2 := z0 - float64(k2) + 2.0*g3
	x3 := x0 - 1.0 + 3.0*g3 // Offsets for last corner in (x,y,z) coords
	y3 := y0 - 1.0 + 3.0*g3
	z3 := z0 - 1.0 + 3.0*g3

	// Work out the hashed gradient indices of the four simplex corners
	ii := i & 255
	jj := j & 255
	kk := k & 255
	perm := sn.perm
	m12 := sn.permMod12
	gi0 := m12[ii+int32(perm[jj+int32(perm[kk])])]
	gi1 := m12[ii+i1+int32(perm[jj+j1+int32(perm[kk+k1])])]
	gi2 := m12[ii+i2+int32(perm[jj+j2+int32(perm[kk+k2])])]
	gi3 := m12[ii+1+int32(perm[jj+1+int32(perm[kk+1])])]

	// Calculate the contribution from the four corners
	t0 := 0.6 - x0*x0 - y0*y0 - z0*z0

	if t0 < 0.0 {
		n0 = 0.0
	} else {
		t0 *= t0
		n0 = t0 * t0 * grad3[gi0].dot3(x0, y0, z0)
	}

	t1 := 0.6 - x1*x1 - y1*y1 - z1*z1

	if t1 < 0.0 {
		n1 = 0.0
	} else {
		t1 *= t1
		n1 = t1 * t1 * grad3[gi1].dot3(x1, y1, z1)
	}

	t2 := 0.6 - x2*x2 - y2*y2 - z2*z2

	if t2 < 0.0 {
		n2 = 0.0
	} else {
		t2 *= t2
		n2 = t2 * t2 * grad3[gi2].dot3(x2, y2, z2)
	}

	t3 := 0.6 - x3*x3 - y3*y3 - z3*z3

	if t3 < 0.0 {
		n3 = 0.0
	} else {
		t3 *= t3
		n3 = t3 * t3 * grad3[gi3].dot3(x3, y3, z3)
	}

	// Add contributions from each corner to get the final noise value.
	// The result is scaled to stay just inside [-1,1]
	return 32.0 * (n0 + n1 + n2 + n3)
}

// Noise4D performs 4D simplex noise, better simplex rank ordering method 2012-03-09
func (sn *SimplexNoise) Noise4D(x, y, z, w float64) float64 {
	var n0, n1, n2, n3, n4 float64 // Noise contributions from the five corners

	// Skew the (x,y,z,w) space to determine which cell of 24 simplices we're in
	s := (x + y + z + w) * f4 // Factor for 4D skewing
	i := fastfloor(x + s)
	j := fastfloor(y + s)
	k := fastfloor(z + s)
	l := fastfloor(w + s)
	t := float64(i+j+k+l) * g4 // Factor for 4D unskewing
	X0 := float64(i) - t       // Unskew the cell origin back to (x,y,z,w) space
	Y0 := float64(j) - t
	Z0 := float64(k) - t
	W0 := float64(l) - t
	x0 := x - X0 // The x,y,z,w distances from the cell origin
	y0 := y - Y0
	z0 := z - Z0
	w0 := w - W0

	// For the 4D case, the simplex is a 4D shape I won't even try to describe.
	// To find out which of the 24 possible simplices we're in, we need to
	// determine the magnitude ordering of x0, y0, z0 and w0.
	// Six pair-wise comparisons are performed between each possible pair
	// of the four coordinates, and the results are used to rank the numbers.
	rankx := 0
	ranky := 0
	rankz := 0
	rankw := 0

	if x0 > y0 {
		rankx++
	} else {
		ranky++
	}

	if x0 > z0 {
		rankx++
	} else {
		rankz++
	}

	if x0 > w0 {
		rankx++
	} else {
		rankw++
	}

	if y0 > z0 {
		ranky++
	} else {
		rankz++
	}

	if y0 > w0 {
		ranky++
	} else {
		rankw++
	}

	if z0 > w0 {
		rankz++
	} else {
		rankw++
	}

	// simplex[c] is a 4-vector with the numbers 0, 1, 2 and 3 in some order.
	// Many values of c will never occur, since e.g. x>y>z>w makes x<z, y<w and x<w
	// impossible. Only the 24 indices which have non-zero entries make any sense.
	// We use a thresholding to set the coordinates in turn from the largest magnitude.
	// Rank 3 denotes the largest coordinate.
	i1 := one(rankx >= 3) // The integer offsets for the second simplex corner
	j1 := one(ranky >= 3)
	k1 := one(rankz >= 3)
	l1 := one(rankw >= 3)

	// Rank 2 denotes the second largest coordinate.
	i2 := one(rankx >= 2) // The integer offsets for the third simplex corner
	j2 := one(ranky >= 2)
	k2 := one(rankz >= 2)
	l2 := one(rankw >= 2)

	// Rank 1 denotes the second smallest coordinate.
	i3 := one(rankx >= 1) // The integer offsets for the fourth simplex corner
	j3 := one(ranky >= 1)
	k3 := one(rankz >= 1)
	l3 := one(rankw >= 1)

	// The fifth corner has all coordinate offsets = 1, so no need to compute that.
	x1 := x0 - float64(i1) + g4 // Offsets for second corner in (x,y,z,w) coords
	y1 := y0 - float64(j1) + g4
	z1 := z0 - float64(k1) + g4
	w1 := w0 - float64(l1) + g4
	x2 := x0 - float64(i2) + 2.0*g4 // Offsets for third corner in (x,y,z,w) coords
	y2 := y0 - float64(j2) + 2.0*g4
	z2 := z0 - float64(k2) + 2.0*g4
	w2 := w0 - float64(l2) + 2.0*g4
	x3 := x0 - float64(i3) + 3.0*g4 // Offsets for fourth corner in (x,y,z,w) coords
	y3 := y0 - float64(j3) + 3.0*g4
	z3 := z0 - float64(k3) + 3.0*g4
	w3 := w0 - float64(l3) + 3.0*g4
	x4 := x0 - 1.0 + 4.0*g4 // Offsets for last corner in (x,y,z,w) coords
	y4 := y0 - 1.0 + 4.0*g4
	z4 := z0 - 1.0 + 4.0*g4
	w4 := w0 - 1.0 + 4.0*g4

	// Work out the hashed gradient indices of the five simplex corners
	ii := i & 255
	jj := j & 255
	kk := k & 255
	ll := l & 255
	p := sn.perm
	gi0 := p[ii+int32(p[jj+int32(p[kk+int32(p[ll])])])] % 32
	gi1 := p[ii+i1+int32(p[jj+j1+int32(p[kk+k1+int32(p[ll+l1])])])] % 32
	gi2 := p[ii+i2+int32(p[jj+j2+int32(p[kk+k2+int32(p[ll+l2])])])] % 32
	gi3 := p[ii+i3+int32(p[jj+j3+int32(p[kk+k3+int32(p[ll+l3])])])] % 32
	gi4 := p[ii+1+int32(p[jj+1+int32(p[kk+1+int32(p[ll+1])])])] % 32

	// Calculate the contribution from the five corners
	t0 := 0.6 - x0*x0 - y0*y0 - z0*z0 - w0*w0

	if t0 < 0.0 {
		n0 = 0.0
	} else {
		t0 *= t0
		n0 = t0 * t0 * grad4[gi0].dot4(x0, y0, z0, w0)
	}

	t1 := 0.6 - x1*x1 - y1*y1 - z1*z1 - w1*w1

	if t1 < 0.0 {
		n1 = 0.0
	} else {
		t1 *= t1
		n1 = t1 * t1 * grad4[gi1].dot4(x1, y1, z1, w1)
	}

	t2 := 0.6 - x2*x2 - y2*y2 - z2*z2 - w2*w2

	if t2 < 0.0 {
		n2 = 0.0
	} else {
		t2 *= t2
		n2 = t2 * t2 * grad4[gi2].dot4(x2, y2, z2, w2)
	}

	t3 := 0.6 - x3*x3 - y3*y3 - z3*z3 - w3*w3

	if t3 < 0.0 {
		n3 = 0.0
	} else {
		t3 *= t3
		n3 = t3 * t3 * grad4[gi3].dot4(x3, y3, z3, w3)
	}

	t4 := 0.6 - x4*x4 - y4*y4 - z4*z4 - w4*w4

	if t4 < 0.0 {
		n4 = 0.0
	} else {
		t4 *= t4
		n4 = t4 * t4 * grad4[gi4].dot4(x4, y4, z4, w4)
	}

	// Sum up and scale the result to cover the range [-1,1]
	return 27.0 * (n0 + n1 + n2 + n3 + n4)
}

func one(x bool) int32 {
	if x {
		return 1
	}
	return 0
}

func fastfloor(x float64) int32 {
	if x >= 0.0 {
		return int32(x)
	}
	return int32(x) - 1
}

type grad struct {
	x float64
	y float64
	z float64
	w float64
}

func (g *grad) dot2(x, y float64) float64 {
	return g.x*x + g.y*y
}

func (g *grad) dot3(x, y, z float64) float64 {
	return g.x*x + g.y*y + g.z*z
}

func (g *grad) dot4(x, y, z, w float64) float64 {
	return g.x*x + g.y*y + g.z*z + g.w*w
}

var grad3 = []grad{
	{1, 1, 0, 0}, {-1, 1, 0, 0}, {1, -1, 0, 0}, {-1, -1, 0, 0},
	{1, 0, 1, 0}, {-1, 0, 1, 0}, {1, 0, -1, 0}, {-1, 0, -1, 0},
	{0, 1, 1, 0}, {0, -1, 1, 0}, {0, 1, -1, 0}, {0, -1, -1, 0},
}

var grad4 = []grad{
	{0, 1, 1, 1}, {0, 1, 1, -1}, {0, 1, -1, 1}, {0, 1, -1, -1},
	{0, -1, 1, 1}, {0, -1, 1, -1}, {0, -1, -1, 1}, {0, -1, -1, -1},
	{1, 0, 1, 1}, {1, 0, 1, -1}, {1, 0, -1, 1}, {1, 0, -1, -1},
	{-1, 0, 1, 1}, {-1, 0, 1, -1}, {-1, 0, -1, 1}, {-1, 0, -1, -1},
	{1, 1, 0, 1}, {1, 1, 0, -1}, {1, -1, 0, 1}, {1, -1, 0, -1},
	{-1, 1, 0, 1}, {-1, 1, 0, -1}, {-1, -1, 0, 1}, {-1, -1, 0, -1},
	{1, 1, 1, 0}, {1, 1, -1, 0}, {1, -1, 1, 0}, {1, -1, -1, 0},
	{-1, 1, 1, 0}, {-1, 1, -1, 0}, {-1, -1, 1, 0}, {-1, -1, -1, 0},
}

// Skewing and unskewing factors for 2, 3, and 4 dimensions
var (
	f2 = 0.5 * (math.Sqrt(3.0) - 1.0)
	g2 = (3.0 - math.Sqrt(3.0)) / 6.0
	f3 = 1.0 / 3.0
	g3 = 1.0 / 6.0
	f4 = (math.Sqrt(5.0) - 1.0) / 4.0
	g4 = (5.0 - math.Sqrt(5.0)) / 20.0
)
