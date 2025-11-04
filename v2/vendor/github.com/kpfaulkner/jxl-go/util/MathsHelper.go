package util

import (
	"cmp"
	"errors"
	"math"
	"math/bits"

	"golang.org/x/exp/constraints"
)

var (
	cosineLUT = generateCosineLUT()
)

type signedInts interface {
	int8 | int16 | int32 | int64
}

func generateCosineLUT() [][][]float32 {

	tempCosineLUT := MakeMatrix3D[float32](9, 0, 0)
	root2 := math.Sqrt(2.0)
	for l := 0; l < len(tempCosineLUT); l++ {
		s := 1 << l
		tempCosineLUT[l] = MakeMatrix2D[float32](s-1, s)
		for n := 0; n < len(tempCosineLUT[l]); n++ {
			for k := 0; k < len(tempCosineLUT[l][n]); k++ {
				tempCosineLUT[l][n][k] = float32(root2 * math.Cos(float64(math.Pi*(float32(n)+1.0)*(float32(k)+0.5)/float32(s))))
			}
		}
	}
	return tempCosineLUT
}

func SignedPow(base float32, exponent float32) float32 {
	if base < 0 {
		return -float32(math.Pow(float64(-base), float64(exponent)))
	}
	return float32(math.Pow(float64(base), float64(exponent)))
}

func CeilLog1p[T constraints.Integer](x T) int {
	return 64 - bits.LeadingZeros64(uint64(x))
}

func CeilLog1pUint64(x uint64) int {
	return 64 - bits.LeadingZeros64(x)
}

func FloorLog1p[T constraints.Integer](x T) int64 {
	c := int64(CeilLog1p[T](x))
	if (x+1)&x != 0 {
		return c - 1
	}
	return c
}

func FloorLog1pUint64(x uint64) int64 {
	c := int64(CeilLog1pUint64(x))
	if (x+1)&x != 0 {
		return c - 1
	}
	return c
}

func CeilLog2[T constraints.Integer](x T) int {
	return CeilLog1p[T](x - 1)
}

func Max[T cmp.Ordered](args ...T) T {
	if len(args) == 0 {
		return *new(T)
	}

	if isNan(args[0]) {
		return args[0]
	}

	max := args[0]
	for _, arg := range args[1:] {

		if isNan(arg) {
			return arg
		}

		if arg > max {
			max = arg
		}
	}
	return max
}

func Min[T cmp.Ordered](args ...T) T {
	if len(args) == 0 {
		return *new(T)
	}

	if isNan(args[0]) {
		return args[0]
	}

	min := args[0]
	for _, arg := range args[1:] {

		if isNan(arg) {
			return arg
		}

		if arg < min {
			min = arg
		}
	}
	return min
}

func Clamp3(v int32, a int32, b int32) int32 {
	var lower int32
	if a < b {
		lower = a
	} else {
		lower = b
	}

	upper := lower ^ a ^ b
	if v < lower {
		return lower
	}
	if v > upper {
		return upper
	}
	return v
}

func Clamp3Float32(v float32, a float32, b float32) float32 {
	var lower float32
	if a < b {
		lower = a
	} else {
		lower = b
	}
	var upper float32
	if a < b {
		upper = b
	} else {
		upper = a
	}
	if v < lower {
		return lower
	}
	if v > upper {
		return upper
	}
	return v
}

func Clamp(v int32, a int32, b int32, c int32) int32 {
	var lower int32
	if a < b {
		lower = a
	} else {
		lower = b
	}
	upper := lower ^ a ^ b
	if lower < c {
		lower = lower
	} else {
		lower = c
	}

	if upper > c {
		upper = upper
	} else {
		upper = c
	}

	if v < lower {
		return lower
	}

	if v > upper {
		return upper
	}

	return v

}
func isNan[T cmp.Ordered](arg T) bool {
	return arg != arg
}

func Abs[T signedInts](a T) T {
	if a < 0 {
		return -a
	}
	return a
}

func MakeSliceWithDefault[T any](length int, defaultVal T) []T {
	if length < 0 {
		return nil
	}
	m := make([]T, length)
	for i := 0; i < length; i++ {
		m[i] = defaultVal
	}
	return m
}

func MatrixIdentity(i int) [][]float32 {
	matrix := make([][]float32, i)
	for j := 0; j < i; j++ {
		matrix[j] = make([]float32, i)
		matrix[j][j] = 1
	}
	return matrix
}

func MatrixVectorMultiply(matrix [][]float32, columnVector []float32) ([]float32, error) {

	if len(matrix) == 0 {
		return columnVector, nil
	}

	if len(matrix[0]) > len(columnVector) || len(columnVector) == 0 {
		return nil, errors.New("Invalid argument")
	}
	extra := len(columnVector) - len(matrix[0])
	total := make([]float32, len(matrix)+extra)

	for y := 0; y < len(matrix); y++ {
		row := matrix[y]

		for x := 0; x < len(row); x++ {
			total[y] += row[x] * columnVector[x]
		}
	}
	if extra != 0 {
		copy(total[len(matrix):], columnVector[len(matrix[0]):])
	}

	return total, nil
}

// multiply any number of matrices
func MatrixMultiply(matrix ...[][]float32) ([][]float32, error) {

	var err error
	left := matrix[0]
	for i := 1; i < len(matrix); i++ {
		right := matrix[i]
		left, err = MatrixMatrixMultiply(left, right)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func MatrixMatrixMultiply(left [][]float32, right [][]float32) ([][]float32, error) {

	if left == nil {
		return right, nil
	}
	if right == nil {
		return left, nil
	}

	if len(left[0]) != len(right) {
		return nil, errors.New("Invalid argument")
	}

	result := make([][]float32, len(left))
	for i := 0; i < len(left); i++ {
		result[i] = make([]float32, len(right[0]))
	}

	for i := 0; i < len(left); i++ {
		for j := 0; j < len(right[0]); j++ {
			for k := 0; k < len(right); k++ {
				result[i][j] += left[i][k] * right[k][j]
			}
		}
	}
	return result, nil
}

func InvertMatrix3x3(matrix [][]float32) [][]float32 {
	det := matrix[0][0]*matrix[1][1]*matrix[2][2] + matrix[0][1]*matrix[1][2]*matrix[2][0] + matrix[0][2]*matrix[1][0]*matrix[2][1] - matrix[0][2]*matrix[1][1]*matrix[2][0] - matrix[0][1]*matrix[1][0]*matrix[2][2] - matrix[0][0]*matrix[1][2]*matrix[2][1]
	if det == 0 {
		return nil
	}
	invDet := 1.0 / det
	return [][]float32{
		{(matrix[1][1]*matrix[2][2] - matrix[1][2]*matrix[2][1]) * invDet, (matrix[0][2]*matrix[2][1] - matrix[0][1]*matrix[2][2]) * invDet, (matrix[0][1]*matrix[1][2] - matrix[0][2]*matrix[1][1]) * invDet},
		{(matrix[1][2]*matrix[2][0] - matrix[1][0]*matrix[2][2]) * invDet, (matrix[0][0]*matrix[2][2] - matrix[0][2]*matrix[2][0]) * invDet, (matrix[0][2]*matrix[1][0] - matrix[0][0]*matrix[1][2]) * invDet},
		{(matrix[1][0]*matrix[2][1] - matrix[1][1]*matrix[2][0]) * invDet, (matrix[0][1]*matrix[2][0] - matrix[0][0]*matrix[2][1]) * invDet, (matrix[0][0]*matrix[1][1] - matrix[0][1]*matrix[1][0]) * invDet},
	}
}

func CeilDiv(numerator uint32, denominator uint32) uint32 {
	return ((numerator - 1) / denominator) + 1
}

func TransposeMatrix(matrix [][]float32, inSize Point) [][]float32 {
	if inSize.X == 0 || inSize.Y == 0 {
		return nil
	}
	dest := MakeMatrix2D[float32](inSize.X, inSize.Y)
	TransposeMatrixInto(matrix, dest, ZERO, ZERO, inSize)
	return dest
}

func TransposeMatrixInto(src [][]float32, dest [][]float32, srcStart Point, destStart Point, srcSize Point) {
	for y := int32(0); y < srcSize.Y; y++ {
		srcY := src[y+srcStart.Y]
		for x := int32(0); x < srcSize.X; x++ {
			dest[destStart.Y+x][destStart.X+y] = srcY[srcStart.X+x]
		}
	}
}

func Matrix3Equal[T comparable](a [][][]T, b [][][]T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		for j := 0; j < len(a[i]); j++ {
			for k := 0; k < len(a[i][j]); k++ {
				if a[i][j][k] != b[i][j][k] {
					return false
				}
			}
		}
	}
	return true
}

func DeepCopy3[T comparable](a [][][]T) [][][]T {

	if a == nil {
		return nil
	}
	matrixCopy := make([][][]T, len(a))
	for i := 0; i < len(a); i++ {
		if a[i] == nil {
			continue
		}
		matrixCopy[i] = make([][]T, len(a[i]))
		for j := 0; j < len(a[i]); j++ {
			if a[i][j] == nil {
				continue
			}
			matrixCopy[i][j] = make([]T, len(a[i][j]))
			for k := 0; k < len(a[i][j]); k++ {
				matrixCopy[i][j][k] = a[i][j][k]
			}
		}
	}
	return matrixCopy
}

func InverseDCT2D(src [][]float32, dest [][]float32, startIn Point, startOut Point, size Dimension, scratchSpace0 [][]float32, scratchSpace1 [][]float32, transposed bool) error {

	logHeight := CeilLog2(size.Height)
	logWidth := CeilLog2(size.Width)
	if transposed {
		for y := int32(0); y < int32(size.Height); y++ {
			if err := inverseDCTHorizontal(src[startIn.Y+y], scratchSpace1[y], startIn.X, 0, logWidth, int32(size.Width)); err != nil {
				return err
			}
		}
		TransposeMatrixInto(scratchSpace1, scratchSpace0, ZERO, ZERO, Point{X: int32(size.Width), Y: int32(size.Height)})
		for y := int32(0); y < int32(size.Width); y++ {
			if err := inverseDCTHorizontal(scratchSpace0[y], dest[startOut.Y+y], 0, startOut.X, logHeight, int32(size.Height)); err != nil {
				return err
			}
		}
	} else {
		TransposeMatrixInto(src, scratchSpace0, startIn, ZERO, Point{X: int32(size.Width), Y: int32(size.Height)})
		for y := int32(0); y < int32(size.Width); y++ {
			if err := inverseDCTHorizontal(scratchSpace0[y], scratchSpace1[y],
				0, 0, logHeight, int32(size.Height)); err != nil {
				return err
			}
		}
		TransposeMatrixInto(scratchSpace1, scratchSpace0, ZERO, ZERO, Point{X: int32(size.Height), Y: int32(size.Width)})
		for y := int32(0); y < int32(size.Height); y++ {
			if err := inverseDCTHorizontal(scratchSpace0[y], dest[startOut.Y+y],
				0, startOut.X, logWidth, int32(size.Width)); err != nil {
				return err
			}
		}
	}
	return nil
}

func inverseDCTHorizontal(src []float32, dest []float32, xStartIn int32, xStartOut int32, xLogLength int,
	xLength int32) error {

	// fill dest with initial data
	for i := xStartOut; i < xStartOut+xLength; i++ {
		dest[i] = src[xStartIn]
	}

	lutX := cosineLUT[xLogLength]
	for n := int32(1); n < xLength; n++ {
		lut := lutX[n-1]
		s2 := src[xStartIn+n]
		for k := int32(0); k < xLength; k++ {
			dest[xStartOut+k] += s2 * lut[k]
		}
	}

	return nil
}

func ForwardDCT2D(src [][]float32, dest [][]float32, startIn Point, startOut Point, length Dimension,
	scratchSpace0 [][]float32, scratchSpace1 [][]float32, b bool) error {

	yLogLength := CeilLog2(length.Height)
	xLogLength := CeilLog2(length.Width)
	for y := int32(0); y < int32(length.Height); y++ {
		if err := forwardDCTHorizontal(src[y+startIn.Y], scratchSpace0[y], startIn.X, 0, xLogLength, int32(length.Width)); err != nil {
			return err
		}
	}
	TransposeMatrixInto(scratchSpace0, scratchSpace1, ZERO, ZERO, Point{X: int32(length.Width), Y: int32(length.Height)})
	for x := int32(0); x < int32(length.Width); x++ {
		if err := forwardDCTHorizontal(scratchSpace1[x], scratchSpace0[x], 0, 0, yLogLength, int32(length.Height)); err != nil {
			return err
		}
	}

	TransposeMatrixInto(scratchSpace0, dest, ZERO, startOut, Point{X: int32(length.Height), Y: int32(length.Width)})
	return nil
}

func forwardDCTHorizontal(src []float32, dest []float32, xStartIn int32, xStartOut int32, xLogLength int, xLength int32) error {

	invLength := 1.0 / float32(xLength)
	d2 := src[xStartIn]
	for x := int32(1); x < xLength; x++ {
		d2 += src[xStartIn+x]
	}
	dest[xStartOut] = d2 * invLength
	for k := int32(1); k < xLength; k++ {
		lut := cosineLUT[xLogLength][k-1]
		d2 = src[xStartIn] * lut[0]
		for n := int32(1); n < xLength; n++ {
			d2 += src[xStartIn+n] * lut[n]
		}
		dest[xStartOut+k] = d2 * invLength
	}

	return nil
}

func MirrorCoordinate(coordinate int32, size int32) int32 {
	for coordinate < 0 || coordinate >= size {
		tc := ^coordinate
		if tc >= 0 {
			coordinate = tc
		} else {
			coordinate = (size << 1) + tc
		}
	}
	return coordinate
}
