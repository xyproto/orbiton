package gfx

import "slices"

// SortSlice sorts s in place using the provided comparison function.
// cmp must return a negative number when a is less than b, zero when
// a equals b, and a positive number when a is greater than b — the
// same convention as slices.SortFunc and cmp.Compare.
func SortSlice[S ~[]E, E any](s S, cmp func(a, b E) int) {
	slices.SortFunc(s, cmp)
}
