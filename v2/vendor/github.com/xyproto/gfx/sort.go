//go:build !tinygo
// +build !tinygo

package gfx

import "sort"

// SortSlice sorts the provided slice given the provided less function.
func SortSlice(slice interface{}, less func(i, j int) bool) {
	sort.Slice(slice, less)
}
