package gfx

import (
	"image"
	"math"
	"testing"
)

func TestV(t *testing.T) {
	for _, tc := range []struct {
		x    float64
		y    float64
		want Vec
	}{
		{123, 456, Vec{123, 456}},
		{1.1, 2.2, Vec{1.1, 2.2}},
	} {
		if got := V(tc.x, tc.y); got != tc.want {
			t.Fatalf("unexpected vector: %v", got)
		}
	}
}

func TestIV(t *testing.T) {
	for _, tc := range []struct {
		x    int
		y    int
		want Vec
	}{
		{123, 456, Vec{123, 456}},
		{789, 333, Vec{789, 333}},
	} {
		if got := IV(tc.x, tc.y); got != tc.want {
			t.Fatalf("unexpected vector: %v", got)
		}
	}
}

func TestPV(t *testing.T) {
	for _, tc := range []struct {
		p    image.Point
		want Vec
	}{
		{Pt(123, 456), Vec{123, 456}},
		{Pt(789, 333), Vec{789, 333}},
	} {
		if got := PV(tc.p); got != tc.want {
			t.Fatalf("unexpected vector: %v", got)
		}
	}
}

func TestUnit(t *testing.T) {
	for angle, want := range map[float64]Vec{
		1: V(0.5403023058681398, 0.8414709848078965),
		2: V(-0.4161468365471424, 0.9092974268256816),
		9: V(-0.9111302618846769, 0.4121184852417566),
	} {
		if got := Unit(angle); got != want {
			t.Fatalf("Unit(%v) = %v, want %v", angle, got, want)
		}
	}
}

func TestVecEq(t *testing.T) {
	for _, tc := range []struct {
		u    Vec
		v    Vec
		want bool
	}{
		{V(1, 1), V(1, 1), true},
		{V(1, 1), V(2, 2), false},
	} {
		if got := tc.u.Eq(tc.v); got != tc.want {
			t.Fatalf("%v.Eq(%v) = %v, want %v", tc.u, tc.v, got, tc.want)
		}
	}
}

func ExampleVec_Add() {
	Dump(
		V(1, 1).Add(V(2, 3)),
		V(3, 3).Add(V(-1, -2)),
	)

	// Output:
	// gfx.V(3.00000000, 4.00000000)
	// gfx.V(2.00000000, 1.00000000)
}

func ExampleVec_AddXY() {
	Dump(
		V(1, 1).AddXY(2, 3),
		V(3, 3).AddXY(-1, -2),
	)

	// Output:
	// gfx.V(3.00000000, 4.00000000)
	// gfx.V(2.00000000, 1.00000000)
}

func ExampleVec_Sub() {
	Dump(
		V(1, 1).Sub(V(2, 3)),
		V(3, 3).Sub(V(-1, -2)),
	)

	// Output:
	// gfx.V(-1.00000000, -2.00000000)
	// gfx.V(4.00000000, 5.00000000)
}

func ExampleVec_To() {
	Dump(
		V(1, 1).To(V(2, 3)),
		V(3, 3).To(V(-1, -2)),
	)

	// Output:
	// gfx.V(1.00000000, 2.00000000)
	// gfx.V(-4.00000000, -5.00000000)
}

func ExampleVec_Mod() {
	Dump(
		V(1, 1).Mod(V(2.5, 3)),
		V(2, 5.5).Mod(V(2, 3)),
	)

	// Output:
	// gfx.V(1.00000000, 1.00000000)
	// gfx.V(0.00000000, 2.50000000)
}

func ExampleVec_Abs() {
	Dump(
		V(1, -1).Abs(),
		V(-2, -2).Abs(),
		V(3, 6).Abs(),
	)

	// Output:
	// gfx.V(1.00000000, 1.00000000)
	// gfx.V(2.00000000, 2.00000000)
	// gfx.V(3.00000000, 6.00000000)
}

func ExampleVec_Max() {
	Dump(
		V(1, 1).Max(V(2.5, 3)),
		V(2, 5.5).Max(V(2, 3)),
	)

	// Output:
	// gfx.V(2.50000000, 3.00000000)
	// gfx.V(2.00000000, 5.50000000)
}

func ExampleVec_Min() {
	Dump(
		V(1, 1).Min(V(2.5, 3)),
		V(2, 5.5).Min(V(2, 3)),
	)

	// Output:
	// gfx.V(1.00000000, 1.00000000)
	// gfx.V(2.00000000, 3.00000000)
}

func ExampleVec_Dot() {
	Dump(
		V(1, 1).Dot(V(2.5, 3)),
		V(2, 5.5).Dot(V(2, 3)),
	)

	// Output:
	// 5.5
	// 20.5
}

func ExampleVec_Cross() {
	Dump(
		V(1, 1).Cross(V(2.5, 3)),
		V(2, 5.5).Cross(V(2, 3)),
	)

	// Output:
	// 0.5
	// -5
}

func ExampleVec_Project() {
	Dump(
		V(1, 1).Project(V(2.5, 3)),
		V(2, 5.5).Project(V(2, 3)),
	)

	// Output:
	// gfx.V(0.90163934, 1.08196721)
	// gfx.V(3.15384615, 4.73076923)
}

func ExampleVec_Map() {
	Dump(
		V(1.1, 1).Map(math.Ceil),
		V(1.1, 2.5).Map(math.Round),
	)

	// Output:
	// gfx.V(2.00000000, 1.00000000)
	// gfx.V(1.00000000, 3.00000000)
}

func ExampleVec_Vec3() {
	Dump(
		V(1, 2).Vec3(3),
		V(4, 5).Vec3(6),
	)

	// Output:
	// gfx.V3(1, 2, 3)
	// gfx.V3(4, 5, 6)
}

func ExampleVec_Pt() {
	Dump(
		V(1, 2).Pt(),
		V(3, 4).Pt(),
	)

	// Output:
	// (1,2)
	// (3,4)
}

func ExampleVec_R() {
	Dump(
		V(1, 2).R(V(3, 4)),
		V(5, 2).R(V(3, 4)),
	)

	// Output:
	// gfx.R(1, 2, 3, 4)
	// gfx.R(5, 2, 3, 4)
}

func ExampleVec_B() {
	Dump(
		V(1, 2).B(V(3, 4)),
		V(5, 2).B(V(3, 4)),
	)

	// Output:
	// (1,2)-(3,4)
	// (5,2)-(3,4)
}

func ExampleVec_Rect() {
	Dump(
		V(10, 10).Rect(-1, -2, 3, 4),
		V(3, 4).Rect(1.5, 2.2, 3.3, 4.5),
	)

	// Output:
	// gfx.R(9, 8, 13, 14)
	// gfx.R(4.5, 6.2, 6.3, 8.5)
}

func ExampleVec_Bounds() {
	Dump(
		V(10, 10).Bounds(-1, -2, 3, 4),
		V(3, 4).Bounds(1.5, 2.2, 3.3, 4.5),
	)

	// Output:
	// (9,8)-(13,14)
	// (4,6)-(6,8)
}

func ExampleVec_Lerp() {
	a, b := V(1, 2), V(30, 40)

	Dump(
		a.Lerp(b, 0),
		a.Lerp(b, 0.1),
		a.Lerp(b, 0.5),
		a.Lerp(b, 0.9),
		a.Lerp(b, 1),
	)

	// Output:
	// gfx.V(1.00000000, 2.00000000)
	// gfx.V(3.90000000, 5.80000000)
	// gfx.V(15.50000000, 21.00000000)
	// gfx.V(27.10000000, 36.20000000)
	// gfx.V(30.00000000, 40.00000000)
}

func ExampleCentroid() {
	Dump(
		Centroid(V(1, 1), V(6, 1), V(3, 4)),
		Centroid(V(0, 0), V(10, 0), V(5, 10)),
	)

	// Output:
	// gfx.V(3.33333333, 2.00000000)
	// gfx.V(5.00000000, 3.33333333)
}
