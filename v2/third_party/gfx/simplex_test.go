package gfx

import "testing"

func TestNewSimplexNoise(t *testing.T) {
	sn := NewSimplexNoise(1234)

	if got, want := len(sn.perm), 512; got != want {
		t.Fatalf("len(sn.perm) = %d, want %d", got, want)
	}

	if got, want := len(sn.permMod12), 512; got != want {
		t.Fatalf("len(sn.permMod12) = %d, want %d", got, want)
	}
}

func TestSimplexNoiseNoise2D(t *testing.T) {
	for _, tc := range []struct {
		seed int64
		x, y float64
		want float64
	}{
		{1234, 1, 2, 0.23526496123584156},
		{1234, 2, 5, -0.49876571260155433},
	} {
		sn := NewSimplexNoise(tc.seed)

		if got := sn.Noise2D(tc.x, tc.y); !inDelta(t, tc.want, got, 0.001) {
			t.Fatalf("sn.Noise2D(%v, %v) = %v, want %v", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestSimplexNoiseNoise3D(t *testing.T) {
	for _, tc := range []struct {
		seed    int64
		x, y, z float64
		want    float64
	}{
		{1234, 1, 2, 3, 0},
		{1234, 1, 2, 4, -0.760099588477367},
		{1234, 2, 5, 9, -0.6522213991769574},
		{1234, 9, 7, 1, 0.7600995884773635},
	} {
		sn := NewSimplexNoise(tc.seed)

		if got := sn.Noise3D(tc.x, tc.y, tc.z); !inDelta(t, tc.want, got, 0.001) {
			t.Fatalf("sn.Noise3D(%v, %v, %v) = %v, want %v", tc.x, tc.y, tc.z, got, tc.want)
		}
	}
}

func TestSimplexNoiseNoise4D(t *testing.T) {
	for _, tc := range []struct {
		seed       int64
		x, y, z, w float64
		want       float64
	}{
		{1234, 1, 2, 3, 1, -0.2209468512526074},
		{1234, 1, 2, 4, 2, 0.2615450624752106},
		{1234, 2, 5, 9, 3, -0.5524905255577035},
		{1234, 9, 7, 1, 4, 0.059165223461962874},
	} {
		sn := NewSimplexNoise(tc.seed)

		if got := sn.Noise4D(tc.x, tc.y, tc.z, tc.w); !inDelta(t, tc.want, got, 0.001) {
			t.Fatalf("sn.Noise4D(%v, %v, %v, %v) = %v, want %v", tc.x, tc.y, tc.z, tc.w, got, tc.want)
		}
	}
}
