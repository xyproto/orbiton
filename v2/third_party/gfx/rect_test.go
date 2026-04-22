package gfx

import "testing"

func TestNewRect(t *testing.T) {
	for _, tc := range []struct {
		min Vec
		max Vec
	}{
		{V(1, 2), V(3, 4)},
		{V(5, 6), V(7, 8)},
	} {
		r := NewRect(tc.min, tc.max)

		if r.Min != tc.min || r.Max != tc.max {
			t.Fatalf("unexpected rect: %v", r)
		}
	}
}

func TestBoundsToRect(t *testing.T) {
	got, want := BoundsToRect(IR(0, 0, 10, 5)), R(0, 0, 10, 5)

	if got != want {
		t.Fatalf("BoundsToRect(IR(0, 0, 10, 5)) = %v, want %v", got, want)
	}
}

func TestBoundsCenter(t *testing.T) {
	if got, want := BoundsCenter(IR(0, 0, 10, 5)), V(5, 2.5); got != want {
		t.Fatalf("BoundsCenter(IR(0, 0, 10, 5)) = %v, want %v", got, want)
	}
}

func TestRectString(t *testing.T) {
	if got, want := R(1, 2, 3, 4).String(), "gfx.R(1, 2, 3, 4)"; got != want {
		t.Fatalf("R(1, 2, 3, 4).String() = %q, want %q", got, want)
	}
}

func TestRectNorm(t *testing.T) {
	if got, want := R(10, 5, 7, 6).Norm(), R(7, 5, 10, 6); got != want {
		t.Fatalf("R(10, 5, 7, 6).Norm() = %v, want %v", got, want)
	}
}

func TestRectW(t *testing.T) {
	if got, want := R(1, 2, 10, 5).W(), float64(9); got != want {
		t.Fatalf("R(1, 2, 10, 5).W() = %v, want %v", got, want)
	}
}

func TestRectH(t *testing.T) {
	if got, want := R(1, 2, 10, 5).H(), float64(3); got != want {
		t.Fatalf("R(1, 2, 10, 5).H() = %v, want %v", got, want)
	}
}

func TestRectSize(t *testing.T) {
	if got, want := R(1, 2, 10, 5).Size(), V(9, 3); got != want {
		t.Fatalf("R(1, 2, 10, 5).Size() = %v, want %v", got, want)
	}
}

func TestRectArea(t *testing.T) {
	if got, want := R(1, 2, 10, 5).Area(), float64(27); got != want {
		t.Fatalf("R(1, 2, 10, 5).Area() = %v, want %v", got, want)
	}
}

func TestRectCenter(t *testing.T) {
	if got, want := R(0, 0, 4, 4).Center(), V(2, 2); got != want {
		t.Fatalf("R(0, 0, 4, 4).Center() = %v, want %v", got, want)
	}
}

func TestRectMoved(t *testing.T) {
	if got, want := R(0, 0, 4, 4).Moved(V(2, -1)), R(2, -1, 6, 3); got != want {
		t.Fatalf("R(0, 0, 4, 4).Moved(V(2, -1)) = %v, want %v", got, want)
	}
}

func TestRectResized(t *testing.T) {
	r := R(10, 10, 20, 20)

	if got, want := r.Resized(r.Center(), V(5, 2)), R(12.5, 14, 17.5, 16); got != want {
		t.Fatalf("r.Resized(r.Center(), V(5,2)) = %v, want %v", got, want)
	}

}

func TestRectResizedMin(t *testing.T) {
	r := R(10, 10, 20, 20)

	if got, want := r.ResizedMin(V(5, 15)), R(10, 10, 15, 25); got != want {
		t.Fatalf("r.ResizedMin(V(5, 15)) = %v, want %v", got, want)
	}
}

func TestRectContains(t *testing.T) {
	for _, tc := range []struct {
		r    Rect
		v    Vec
		want bool
	}{
		{R(0, 0, 5, 5), V(3, 3), true},
		{R(0, 0, 5, 5), V(6, 3), false},
	} {
		if got := tc.r.Contains(tc.v); got != tc.want {
			t.Fatalf("%v.Contains(%v) = %v, want %v", tc.r, tc.v, got, tc.want)
		}
	}
}

func TestRectOverlaps(t *testing.T) {
	for _, tc := range []struct {
		r1   Rect
		r2   Rect
		want bool
	}{
		{R(10, 10, 25, 25), R(20, 20, 30, 30), true},
		{R(10, 10, 20, 20), R(30, 30, 40, 40), false},
	} {
		if got := tc.r1.Overlaps(tc.r2); got != tc.want {
			t.Fatalf("%v.Overlaps(%v) = %v, want %v", tc.r1, tc.r2, got, tc.want)
		}
	}
}

func TestRectUnion(t *testing.T) {
	if got, want := R(0, 1, 2, 3).Union(R(3, 4, 5, 6)), R(0, 1, 5, 6); got != want {
		t.Fatalf("R(0, 1, 2, 3).Union(R(3, 4, 5, 6)) = %v, want %v", got, want)
	}
}
