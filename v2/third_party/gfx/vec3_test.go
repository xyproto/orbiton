package gfx

import (
	"math"
	"testing"
)

func TestV3(t *testing.T) {
	x, y, z := 1.1, 2.2, 3.3

	want := Vec3{X: 1.1, Y: 2.2, Z: 3.3}

	if got := V3(x, y, z); !got.Eq(want) {
		t.Fatalf("V3(%v, %v, %v) = %v, want %v", x, y, z, got, want)
	}
}

func TestIV3(t *testing.T) {
	x, y, z := 1, 2, 3

	if got, want := IV3(x, y, z), V3(1, 2, 3); !got.Eq(want) {
		t.Fatalf("IV3(%d, %d, %d) = %v, want %v", x, y, z, got, want)
	}
}

func TestVec3String(t *testing.T) {
	if got, want := V3(1, 2, 3).String(), "gfx.V3(1, 2, 3)"; got != want {
		t.Fatalf("V3(1,2,3) = %q, want %q", got, want)
	}
}

func TestVec3XYZ(t *testing.T) {
	v := V3(1.1, 2.2, 3.3)

	x, y, z := v.XYZ()

	if x != v.X {
		t.Fatalf("x = %v, want %v", x, v.X)
	}

	if y != v.Y {
		t.Fatalf("y = %v, want %v", y, v.Y)
	}

	if z != v.Z {
		t.Fatalf("z = %v, want %v", y, v.Z)
	}
}

func TestVec3Eq(t *testing.T) {
	for _, tc := range []struct {
		u    Vec3
		v    Vec3
		want bool
	}{
		{V3(1, 2, 3), V3(1, 2, 3), true},
		{V3(1, 2, 3), V3(4, 5, 6), false},
	} {
		if got := tc.u.Eq(tc.v); got != tc.want {
			t.Fatalf("u.Eq(%v) = %v, want %v", tc.v, got, tc.want)
		}
	}
}

func TestVec3Add(t *testing.T) {
	for _, tc := range []struct {
		u    Vec3
		v    Vec3
		want Vec3
	}{
		{V3(1, 2, 3), V3(0, 0, 0), V3(1, 2, 3)},
		{V3(1, 2, 3), V3(5, 6, 7), V3(6, 8, 10)},
		{V3(1, 2, 3), V3(-1, -1, -1), V3(0, 1, 2)},
	} {
		if got := tc.u.Add(tc.v); !got.Eq(tc.want) {
			t.Fatalf("u.Add(%v) = %v, want %v", tc.v, got, tc.want)
		}
	}
}

func TestVec3AddXYZ(t *testing.T) {
	for _, tc := range []struct {
		u       Vec3
		x, y, z float64
		want    Vec3
	}{
		{V3(1, 2, 3), 0, 0, 0, V3(1, 2, 3)},
		{V3(1, 2, 3), 5, 6, 7, V3(6, 8, 10)},
		{V3(1, 2, 3), -1, -1, -1, V3(0, 1, 2)},
	} {
		if got := tc.u.AddXYZ(tc.x, tc.y, tc.z); !got.Eq(tc.want) {
			t.Fatalf("u.AddXYZ(%v, %v, %v) = %v, want %v", tc.x, tc.y, tc.z, got, tc.want)
		}
	}
}

func TestVec3Sub(t *testing.T) {
	for _, tc := range []struct {
		u    Vec3
		v    Vec3
		want Vec3
	}{
		{V3(1, 2, 3), V3(0, 0, 0), V3(1, 2, 3)},
		{V3(1, 2, 3), V3(5, 6, 7), V3(-4, -4, -4)},
		{V3(1, 2, 3), V3(-1, -1, -1), V3(2, 3, 4)},
	} {
		if got := tc.u.Sub(tc.v); !got.Eq(tc.want) {
			t.Fatalf("u.Sub(%v) = %v, want %v", tc.v, got, tc.want)
		}
	}
}

func TestVec3Scaled(t *testing.T) {
	for _, tc := range []struct {
		u    Vec3
		s    float64
		want Vec3
	}{
		{V3(1, 2, 3), 0, V3(0, 0, 0)},
		{V3(1, 2, 3), 0.5, V3(0.5, 1, 1.5)},
		{V3(1, 2, 3), 5, V3(5, 10, 15)},
	} {
		if got := tc.u.Scaled(tc.s); !got.Eq(tc.want) {
			t.Fatalf("u.Scaled(%v) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

func TestVec3ScaledXYZ(t *testing.T) {
	for _, tc := range []struct {
		u    Vec3
		v    Vec3
		want Vec3
	}{
		{V3(1, 2, 3), V3(0, 0, 0), V3(0, 0, 0)},
		{V3(1, 2, 3), V3(0.5, 0.3, 0), V3(0.5, 0.6, 0)},
		{V3(1, 2, 3), V3(2, 3, 4), V3(2, 6, 12)},
	} {
		if got := tc.u.ScaledXYZ(tc.v); !got.Eq(tc.want) {
			t.Fatalf("u.ScaledXYZ(%v) = %v, want %v", tc.v, got, tc.want)
		}
	}
}

func TestVec3Len(t *testing.T) {
	for _, tc := range []struct {
		u    Vec3
		want float64
	}{
		{V3(1, 0, 0), 1},
		{V3(1, 2, 2), 3},
		{V3(2, 1, 2), 3},
		{V3(1, 1, 0), 1.4142135623730951},
		{V3(1, 1, 1), 1.7320508075688772},
		{V3(1, 2, 1), 2.449489742783178},
		{V3(1, 2, 3), 3.7416573867739413},
	} {
		if got := tc.u.Len(); got != tc.want {
			t.Fatalf("u.Len() = %v, want %v", got, tc.want)
		}
	}
}

func TestVec3Div(t *testing.T) {
	for _, tc := range []struct {
		u    Vec3
		s    float64
		want Vec3
	}{
		{V3(2, 3, 4), 2, V3(1, 1.5, 2)},
	} {
		if got := tc.u.Div(tc.s); !got.Eq(tc.want) {
			t.Fatalf("u.Div(%v) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

func TestVec3SqDist(t *testing.T) {
	for _, tc := range []struct {
		u    Vec3
		v    Vec3
		want float64
	}{
		{V3(1, 2, 3), V3(1, 2, 3), 0},
		{V3(1, 2, 3), V3(2, 3, 4), 3},
		{V3(1, 2, 3), V3(4, 3, 2), 11},
		{V3(1, 2, 3), V3(4, 5, 6), 27},
	} {
		if got := tc.u.SqDist(tc.v); got != tc.want {
			t.Fatalf("u.SqDist(%v) = %v, want %v", tc.v, got, tc.want)
		}
	}
}

func TestVec3Dist(t *testing.T) {
	for _, tc := range []struct {
		u    Vec3
		v    Vec3
		want float64
	}{
		{V3(1, 2, 3), V3(1, 2, 3), 0},
		{V3(1, 2, 3), V3(2, 3, 4), 1.7320508075688772},
		{V3(1, 2, 3), V3(4, 3, 2), 3.3166247903554},
		{V3(1, 2, 3), V3(4, 5, 6), 5.196152422706632},
	} {
		if got := tc.u.Dist(tc.v); got != tc.want {
			t.Fatalf("u.Dist(%v) = %v, want %v", tc.v, got, tc.want)
		}
	}
}

func TestVec3Unit(t *testing.T) {
	for _, tc := range []struct {
		u    Vec3
		want Vec3
	}{
		{V3(1, 0, 0), V3(1, 0, 0)},
		{V3(1, 1, 0), V3(0.7071067811865475, 0.7071067811865475, 0)},
		{V3(1.1, 2.2, 3.3), V3(0.2672612419124244, 0.5345224838248488, 0.8017837257372732)},
	} {
		if got := tc.u.Unit(); !got.Eq(tc.want) {
			t.Fatalf("u.Unit() = %v, want %v", got, tc.want)
		}
	}
}

func TestVec3Map(t *testing.T) {
	for _, tc := range []struct {
		u    Vec3
		f    func(float64) float64
		want Vec3
	}{
		{V3(1.1, 2.2, 3.3), math.Ceil, V3(2, 3, 4)},
		{V3(1.1, 2.2, 3.3), math.Floor, V3(1, 2, 3)},
	} {
		if got := tc.u.Map(tc.f); !got.Eq(tc.want) {
			t.Fatalf("u.Map(%T) = %v, want %v", tc.f, got, tc.want)
		}
	}
}

func TestVec3Lerp(t *testing.T) {
	for _, tc := range []struct {
		u    Vec3
		v    Vec3
		t    float64
		want Vec3
	}{
		{V3(1, 2, 3), V3(2, 4, 6), 0.5, V3(1.5, 3, 4.5)},
		{V3(1, 1, 1), V3(4, 5, 6), 0.2, V3(1.6, 1.8, 2)},
		{V3(2, 2, 2), V3(5, 7, 9), 0.321, V3(2.963, 3.605, 4.247)},
	} {
		if got := tc.u.Lerp(tc.v, tc.t); !got.Eq(tc.want) {
			t.Fatalf("u.Lerp(%v, %v) = %v, want %v", tc.v, tc.t, got, tc.want)
		}
	}
}
