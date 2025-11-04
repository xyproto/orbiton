package util

import "golang.org/x/exp/constraints"

func IfThenElse[T any](condition bool, a T, b T) T {
	if condition {
		return a
	}
	return b
}

func MakeMatrix2D[T any, S constraints.Integer](a S, b S) [][]T {
	matrix := make([][]T, a)
	for i, _ := range matrix {
		matrix[i] = make([]T, b)
	}
	return matrix
}

func MakeMatrix3D[T any](a int, b int, c int) [][][]T {
	matrix := make([][][]T, a)
	for i, _ := range matrix {
		matrix[i] = make([][]T, b)
		for j, _ := range matrix[i] {
			matrix[i][j] = make([]T, c)
		}
	}
	return matrix
}

func MakeMatrix4D[T any](a int, b int, c int, d int) [][][][]T {
	matrix := make([][][][]T, a)
	for i, _ := range matrix {
		matrix[i] = make([][][]T, b)
		for j, _ := range matrix[i] {
			matrix[i][j] = make([][]T, c)
			for k, _ := range matrix[i][j] {
				matrix[i][j][k] = make([]T, d)
			}
		}
	}
	return matrix
}

func CompareMatrix2D[T any](a [][]T, b [][]T, compare func(T, T) bool) bool {

	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := 0; j < len(a[i]); j++ {
			if !compare(a[i][j], b[i][j]) {
				return false
			}
		}
	}

	return true
}

func CompareMatrix3D[T any](a [][][]T, b [][][]T, compare func(T, T) bool) bool {

	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := 0; j < len(a[i]); j++ {
			if len(a[i][j]) != len(b[i][j]) {
				return false
			}
			for k := 0; k < len(a[i][j]); k++ {
				if !compare(a[i][j][k], b[i][j][k]) {
					return false
				}
			}
		}
	}

	return true
}

func FillFloat32(a []float32, fromIndex uint32, toIndex uint32, val float32) {
	for i := fromIndex; i < toIndex; i++ {
		a[i] = val
	}
}

func Add[T any](slice []T, index int, elem T) []T {
	newSlice := append(slice[:index], elem)
	newSlice = append(newSlice, slice[index:]...)
	return newSlice
}

type Dimension struct {
	Width  uint32
	Height uint32
}

type Rectangle struct {
	Origin Point
	Size   Dimension
}

func (r Rectangle) ComputeLowerCorner() Point {
	return Point{
		X: r.Origin.X + int32(r.Size.Width),
		Y: r.Origin.Y + int32(r.Size.Height),
	}
}
