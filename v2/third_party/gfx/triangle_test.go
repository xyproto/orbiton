package gfx

import (
	"image/color"
	"testing"
)

func TestNewTriangle(t *testing.T) {
	td := &TrianglesData{
		{Position: V(0, 0), Color: ColorRed},
		{Position: V(1, 0), Color: ColorGreen},
		{Position: V(2, 4), Color: ColorBlue},
	}

	tri := NewTriangle(0, td)

	if got, want := tri.Centroid(), V(1, 1.3333333333333333); got != want {
		t.Fatalf("tri.Centroid() = %v, want %v", got, want)
	}
}

func TestTrianglePositions(t *testing.T) {
	a, b, c := V(0, 0), V(1, 0), V(2, 4)

	tri := NewTriangle(0, &TrianglesData{
		{Position: a},
		{Position: b},
		{Position: c},
	})

	pa, pb, pc := tri.Positions()

	if pa != a {
		t.Fatalf("pa = %v, want %v", pa, a)
	}

	if pb != b {
		t.Fatalf("pb = %v, want %v", pb, b)
	}

	if pc != c {
		t.Fatalf("pc = %v, want %v", pc, c)
	}
}

func TestTriangleColors(t *testing.T) {
	a, b, c := ColorRed, ColorGreen, ColorBlue

	tri := NewTriangle(0, &TrianglesData{
		{Color: a},
		{Color: b},
		{Color: c},
	})

	ca, cb, cc := tri.Colors()

	if ca != a {
		t.Fatalf("ca = %v, want %v", ca, a)
	}

	if cb != b {
		t.Fatalf("cb = %v, want %v", cb, b)
	}

	if cc != c {
		t.Fatalf("cc = %v, want %v", cc, c)
	}
}

func TestTriangleColor(t *testing.T) {
	r, g, b := ColorRed, ColorGreen, ColorBlue

	tri := NewTriangle(0, &TrianglesData{
		{Position: V(0, 0), Color: r},
		{Position: V(10, 0), Color: g},
		{Position: V(5, 10), Color: b},
	})

	for v, want := range map[Vec]color.NRGBA{
		V(0, 0): g,
		V(1, 1): r,
		V(2, 2): r,
		V(6, 6): b,
	} {
		if got := tri.Color(v); got != want {
			t.Fatalf("tri.Color(%v) = %v, want %v", v, got, want)
		}
	}
}

func TestTriangleContains(t *testing.T) {
	a, b, c := ColorRed, ColorGreen, ColorBlue

	tri := NewTriangle(0, &TrianglesData{
		{Position: V(0, 0), Color: a},
		{Position: V(10, 0), Color: b},
		{Position: V(5, 10), Color: c},
	})

	for v, want := range map[Vec]bool{
		V(0, 0): true,
		V(1, 1): true,
		V(6, 6): true,
		V(0, 6): false,
		V(9, 9): false,
	} {
		if got := tri.Contains(v); got != want {
			t.Fatalf("tri.Contains(%v) = %v, want %v", v, got, want)
		}
	}
}

func TestTriangleBounds(t *testing.T) {
	tri := NewTriangle(0, &TrianglesData{
		{Position: V(0, 0)},
		{Position: V(1, 0)},
		{Position: V(2, 4)},
	})

	if got, want := tri.Bounds(), IR(0, 0, 2, 4); got != want {
		t.Fatalf("tri.Bounds() = %v, want %v", got, want)
	}
}

func ExampleT() {
	t := T(
		Vx(V(1, 2), ColorRed),
		Vx(V(3, 4), ColorGreen, V(1, 1)),
		Vx(V(5, 6), ColorBlue, 0.5),
	)

	Log("%v\n%v\n%v", t[0], t[1], t[2])

	// Output:
	// {gfx.V(1.00000000, 2.00000000) {255 0 0 255} gfx.V(0.00000000, 0.00000000) 0}
	// {gfx.V(3.00000000, 4.00000000) {0 255 0 255} gfx.V(1.00000000, 1.00000000) 0}
	// {gfx.V(5.00000000, 6.00000000) {0 0 255 255} gfx.V(0.00000000, 0.00000000) 0.5}
}
