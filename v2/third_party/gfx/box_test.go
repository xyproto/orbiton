package gfx

import "testing"

func TestBoxOverlaps(t *testing.T) {
	for _, tc := range []struct {
		a    Box
		b    Box
		want bool
	}{
		{B(-2, -2, -2, 2, 2, 2), B(-1, -1, -1, 1, 1, 1), true},
		{B(-2, -2, -2, 2, 2, 2), B(3, 3, 3, 4, 4, 4), false},
	} {
		if got := tc.a.Overlaps(tc.b); got != tc.want {
			t.Fatalf("a.Overlaps(b) = %v, want %v", got, tc.want)
		}
	}
}
