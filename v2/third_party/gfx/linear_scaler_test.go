package gfx

import "testing"

func TestNewLinearScaler(t *testing.T) {
	linear := NewLinearScaler().Domain(200, 1000).Range(10, 20)

	var x float64 = 400

	if got, want := linear.ScaleFloat64(x), 12.5; got != want {
		t.Fatalf("linear.ScaleFloat64(%v) = %v, want %v", x, got, want)
	}
}

func TestInterpolateFloat64s(t *testing.T) {
	i := interpolateFloat64s(10, 20)

	for _, tc := range []struct {
		value float64
		want  float64
	}{
		{0.0, 10},
		{0.2, 12},
		{0.5, 15},
		{1.0, 20},
	} {
		if got := i(tc.value); got != tc.want {
			t.Fatalf("i(%v) = %v, want %v", tc.value, got, tc.want)
		}
	}
}
